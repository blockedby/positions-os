package collector

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/google/uuid"
)

// TelegramClient defines interface for telegram operations
type TelegramClient interface {
	ResolveChannel(ctx context.Context, username string) (*telegram.Channel, error)
	GetMessages(ctx context.Context, channel *telegram.Channel, offsetID int, limit int) ([]telegram.Message, error)
}

// Service orchestrates the scraping process
type Service struct {
	tgClient  TelegramClient
	targets   *repository.TargetsRepository
	jobs      *repository.JobsRepository
	ranges    *repository.RangesRepository
	publisher EventPublisher
	log       *logger.Logger
}

// EventPublisher publishes job events
type EventPublisher interface {
	PublishJobNew(ctx context.Context, event JobNewEvent) error
}

// JobNewEvent represents a new job event for NATS
type JobNewEvent struct {
	JobID      uuid.UUID `json:"job_id"`
	TargetID   uuid.UUID `json:"target_id"`
	ExternalID string    `json:"external_id"`
	RawContent string    `json:"raw_content"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewService creates a new collector service
func NewService(
	tgClient TelegramClient,
	targets *repository.TargetsRepository,
	jobs *repository.JobsRepository,
	ranges *repository.RangesRepository,
	publisher EventPublisher,
	log *logger.Logger,
) *Service {
	return &Service{
		tgClient:  tgClient,
		targets:   targets,
		jobs:      jobs,
		ranges:    ranges,
		publisher: publisher,
		log:       log,
	}
}

// ScrapeResult contains scraping statistics
type ScrapeResult struct {
	TotalFetched int
	NewJobs      int
	SkippedOld   int
	SkippedEmpty int
	Errors       int
}

// Scrape performs scraping for given options
func (s *Service) Scrape(ctx context.Context, opts ScrapeOptions) (*ScrapeResult, error) {
	result := &ScrapeResult{}

	// get or create target
	target, err := s.getOrCreateTarget(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("get target: %w", err)
	}

	s.log.Info().
		Str("target_id", target.ID.String()).
		Str("channel", target.URL).
		Msg("starting scrape")

	// resolve channel if needed
	channel, err := s.tgClient.ResolveChannel(ctx, target.URL)
	if err != nil {
		return nil, fmt.Errorf("resolve channel: %w", err)
	}

	// update target with telegram info
	if err := s.targets.UpdateTelegramInfo(ctx, target.ID, channel.ID, channel.AccessHash); err != nil {
		s.log.Warn().Err(err).Msg("failed to update telegram info")
	}

	// get message filter for deduplication
	filter, err := s.ranges.NewFilter(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("create filter: %w", err)
	}

	// determine limit
	limit := opts.Limit
	if limit <= 0 {
		limit = 100 // default batch size
	}

	var minMsgID, maxMsgID int64
	offsetID := 0

	// fetch messages in batches
	for {
		select {
		case <-ctx.Done():
			s.log.Info().Msg("scrape cancelled")
			return result, nil
		default:
		}

		messages, err := s.tgClient.GetMessages(ctx, channel, offsetID, min(limit, 100))
		if err != nil {
			s.log.Error().Err(err).Msg("failed to get messages")
			result.Errors++
			break
		}

		if len(messages) == 0 {
			break
		}

		result.TotalFetched += len(messages)

		// extract message IDs for filtering
		var msgIDs []int64
		for _, msg := range messages {
			msgIDs = append(msgIDs, int64(msg.ID))
		}

		// filter out already processed messages
		newIDs := filter.FilterNew(msgIDs)
		newIDSet := make(map[int64]bool)
		for _, id := range newIDs {
			newIDSet[id] = true
		}

		// process new messages
		for _, msg := range messages {
			msgID := int64(msg.ID)

			// track min/max for range update
			if minMsgID == 0 || msgID < minMsgID {
				minMsgID = msgID
			}
			if msgID > maxMsgID {
				maxMsgID = msgID
			}

			// skip if already processed
			if !newIDSet[msgID] {
				result.SkippedOld++
				continue
			}

			// skip empty messages
			if msg.Text == "" {
				result.SkippedEmpty++
				continue
			}

			// check until date
			if opts.Until != nil && msg.Date.Before(*opts.Until) {
				continue
			}

			// create job
			if err := s.createJob(ctx, target.ID, &msg); err != nil {
				s.log.Error().Err(err).Int("message_id", msg.ID).Msg("failed to create job")
				result.Errors++
				continue
			}

			result.NewJobs++
		}

		// update offset for next batch
		if len(messages) > 0 {
			offsetID = messages[len(messages)-1].ID
		}

		// check if we've fetched enough
		if opts.Limit > 0 && result.TotalFetched >= opts.Limit {
			break
		}

		// small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	// update parsed range
	if maxMsgID > 0 {
		if err := s.ranges.UpdateRange(ctx, target.ID, minMsgID, maxMsgID); err != nil {
			s.log.Warn().Err(err).Msg("failed to update parsed range")
		}
	}

	// update target last scraped
	if err := s.targets.UpdateLastScraped(ctx, target.ID, maxMsgID); err != nil {
		s.log.Warn().Err(err).Msg("failed to update last scraped")
	}

	s.log.Info().
		Int("total", result.TotalFetched).
		Int("new", result.NewJobs).
		Int("skipped_old", result.SkippedOld).
		Int("skipped_empty", result.SkippedEmpty).
		Int("errors", result.Errors).
		Msg("scrape completed")

	return result, nil
}

// getOrCreateTarget gets existing target or creates new one
func (s *Service) getOrCreateTarget(ctx context.Context, opts ScrapeOptions) (*repository.ScrapingTarget, error) {
	// if target ID is provided, use it
	if opts.TargetID != uuid.Nil {
		target, err := s.targets.GetByID(ctx, opts.TargetID)
		if err != nil {
			return nil, err
		}
		if target == nil {
			return nil, fmt.Errorf("target not found: %s", opts.TargetID)
		}
		return target, nil
	}

	// try to find by channel URL
	target, err := s.targets.GetByURL(ctx, opts.Channel)
	if err != nil {
		return nil, err
	}
	if target != nil {
		return target, nil
	}

	// create new target
	target = &repository.ScrapingTarget{
		Name:     opts.Channel,
		Type:     "TG_CHANNEL", // default, will be updated if it's a forum
		URL:      opts.Channel,
		IsActive: true,
	}
	if err := s.targets.Create(ctx, target); err != nil {
		return nil, fmt.Errorf("create target: %w", err)
	}

	return target, nil
}

// createJob creates a new job from a telegram message
func (s *Service) createJob(ctx context.Context, targetID uuid.UUID, msg *telegram.Message) error {
	msgID := int64(msg.ID)
	sourceDate := msg.Date

	job := &repository.Job{
		TargetID:    targetID,
		ExternalID:  strconv.FormatInt(msgID, 10),
		RawContent:  msg.Text,
		SourceDate:  &sourceDate,
		TgMessageID: &msgID,
		Status:      "RAW",
	}

	if msg.TopicID != nil {
		topicID := int64(*msg.TopicID)
		job.TgTopicID = &topicID
	}

	if err := s.jobs.Create(ctx, job); err != nil {
		return err
	}

	// publish event
	if s.publisher != nil {
		event := JobNewEvent{
			JobID:      job.ID,
			TargetID:   targetID,
			ExternalID: job.ExternalID,
			RawContent: job.RawContent,
			CreatedAt:  job.CreatedAt,
		}
		if err := s.publisher.PublishJobNew(ctx, event); err != nil {
			s.log.Warn().Err(err).Msg("failed to publish job event")
		}
	}

	return nil
}

// helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
