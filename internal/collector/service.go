package collector

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	GetTopics(ctx context.Context, channel *telegram.Channel) ([]telegram.Topic, error)
	GetStatus() telegram.Status
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

// ListTopics returns list of topics for a forum channel
func (s *Service) ListTopics(ctx context.Context, channelURL string) ([]telegram.Topic, error) {
	// resolve channel
	channel, err := s.tgClient.ResolveChannel(ctx, channelURL)
	if err != nil {
		return nil, fmt.Errorf("resolve channel: %w", err)
	}

	if !channel.IsForum {
		return nil, fmt.Errorf("channel is not a forum")
	}

	// fetch topics
	topics, err := s.tgClient.GetTopics(ctx, channel)
	if err != nil {
		return nil, fmt.Errorf("get topics: %w", err)
	}

	return topics, nil
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

	s.log.Info().
		Interface("options", opts).
		Msg("scrape: starting")

	// get or create target
	s.log.Debug().Msg("scrape: getting or creating target")
	target, err := s.getOrCreateTarget(ctx, opts)
	if err != nil {
		s.log.Error().Err(err).Msg("scrape: failed to get target")
		return nil, fmt.Errorf("get target: %w", err)
	}

	s.log.Info().
		Str("target_id", target.ID.String()).
		Str("channel", target.URL).
		Msg("scrape: target resolved")

	// resolve channel if needed
	s.log.Debug().Str("channel", target.URL).Msg("scrape: resolving channel")
	channel, err := s.tgClient.ResolveChannel(ctx, target.URL)
	if err != nil {
		s.log.Error().Err(err).Str("channel", target.URL).Msg("scrape: failed to resolve channel")
		return nil, fmt.Errorf("resolve channel: %w", err)
	}

	s.log.Info().
		Int64("channel_id", channel.ID).
		Int64("access_hash", channel.AccessHash).
		Bool("is_forum", channel.IsForum).
		Msg("scrape: channel resolved")

	// update target with telegram info
	if err := s.targets.UpdateTelegramInfo(ctx, target.ID, channel.ID, channel.AccessHash); err != nil {
		s.log.Warn().Err(err).Msg("scrape: failed to update telegram info")
	}

	// get existing job message IDs for smart filtering
	s.log.Debug().Msg("scrape: fetching existing job message IDs")
	existingJobIDs, err := s.jobs.GetExistingMessageIDs(ctx, target.ID)
	if err != nil {
		s.log.Error().Err(err).Msg("scrape: failed to get existing job IDs")
		return nil, fmt.Errorf("get existing job IDs: %w", err)
	}
	s.log.Debug().Int("existing_jobs", len(existingJobIDs)).Msg("scrape: found existing jobs")

	// get smart message filter for deduplication (checks range AND job existence)
	s.log.Debug().Msg("scrape: creating smart message filter")
	filter, err := s.ranges.NewSmartFilter(ctx, target.ID, existingJobIDs)
	if err != nil {
		s.log.Error().Err(err).Msg("scrape: failed to create filter")
		return nil, fmt.Errorf("create filter: %w", err)
	}

	// determine limit
	limit := opts.Limit
	if limit <= 0 {
		limit = 100 // default batch size
	}

	s.log.Info().Int("batch_size", limit).Msg("scrape: starting message fetch loop")

	// Safety limits to prevent infinite loops
	const maxBatches = 100 // Maximum 100 batches = 10,000 messages max

	var minMsgID, maxMsgID int64
	offsetID := 0
	previousOffsetID := -1 // Track previous offset to detect stuck loops
	batchNum := 0

	// fetch messages in batches
	for batchNum < maxBatches {
		batchNum++
		s.log.Info().
			Int("batch", batchNum).
			Int("max_batches", maxBatches).
			Int("offset_id", offsetID).
			Int("limit", min(limit, 100)).
			Msg("scrape: fetching messages batch")

		// Detect if we're stuck on the same offset (infinite loop protection)
		if offsetID == previousOffsetID && offsetID != 0 {
			s.log.Warn().
				Int("offset_id", offsetID).
				Msg("scrape: offset not changing, exiting to prevent infinite loop")
			break
		}
		previousOffsetID = offsetID

		select {
		case <-ctx.Done():
			s.log.Info().Msg("scrape: cancelled by context")
			return result, nil
		default:
		}

		messages, err := s.tgClient.GetMessages(ctx, channel, offsetID, min(limit, 100))
		if err != nil {
			s.log.Error().
				Err(err).
				Int("batch", batchNum).
				Int("offset_id", offsetID).
				Msg("scrape: failed to get messages")
			result.Errors++
			break
		}

		s.log.Info().
			Int("batch", batchNum).
			Int("messages_received", len(messages)).
			Msg("scrape: received messages")

		if len(messages) == 0 {
			s.log.Info().Msg("scrape: no more messages, exiting loop")
			break
		}

		result.TotalFetched += len(messages)

		// extract message IDs for filtering
		var msgIDs []int64
		for _, msg := range messages {
			msgIDs = append(msgIDs, int64(msg.ID))
		}

		s.log.Debug().
			Int("batch", batchNum).
			Int("msg_count", len(msgIDs)).
			Msg("scrape: extracted message IDs")

		// filter out already processed messages
		newIDs := filter.FilterNew(msgIDs)
		newIDSet := make(map[int64]bool)
		for _, id := range newIDs {
			newIDSet[id] = true
		}

		s.log.Info().
			Int("batch", batchNum).
			Int("total_messages", len(messages)).
			Int("new_messages", len(newIDs)).
			Int("already_processed", len(msgIDs)-len(newIDs)).
			Msg("scrape: filtered messages")

		// process new messages
		processedInBatch := 0
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
				s.log.Debug().Int64("msg_id", msgID).Msg("scrape: skipped empty message")
				continue
			}

			// check until date
			if opts.Until != nil && msg.Date.Before(*opts.Until) {
				s.log.Debug().
					Int64("msg_id", msgID).
					Time("msg_date", msg.Date).
					Time("until", *opts.Until).
					Msg("scrape: skipped message older than until date")
				continue
			}

			// create job
			s.log.Debug().Int64("msg_id", msgID).Msg("scrape: creating job for message")
			if err := s.createJob(ctx, target.ID, target.URL, &msg); err != nil {
				s.log.Error().Err(err).Int("message_id", msg.ID).Msg("scrape: failed to create job")
				result.Errors++
				continue
			}

			result.NewJobs++
			processedInBatch++
		}

		s.log.Info().
			Int("batch", batchNum).
			Int("processed", processedInBatch).
			Int("total_new_jobs", result.NewJobs).
			Msg("scrape: batch processed")

		// update offset for next batch
		oldOffsetID := offsetID
		if len(messages) > 0 {
			offsetID = messages[len(messages)-1].ID
		}

		s.log.Info().
			Int("batch", batchNum).
			Int("old_offset", oldOffsetID).
			Int("new_offset", offsetID).
			Msg("scrape: updated offset for next batch")

		// check if we've fetched enough
		if opts.Limit > 0 && result.TotalFetched >= opts.Limit {
			s.log.Info().
				Int("total_fetched", result.TotalFetched).
				Int("limit", opts.Limit).
				Msg("scrape: reached fetch limit, exiting loop")
			break
		}

		// small delay to avoid rate limiting
		s.log.Debug().Msg("scrape: sleeping 100ms to avoid rate limiting")
		time.Sleep(100 * time.Millisecond)
	}

	// Check if we exited due to max batch limit
	if batchNum >= maxBatches {
		s.log.Warn().
			Int("batches_processed", batchNum).
			Int("max_batches", maxBatches).
			Msg("scrape: reached maximum batch limit, stopping for safety")
	}

	// update parsed range
	s.log.Info().
		Int64("min_msg_id", minMsgID).
		Int64("max_msg_id", maxMsgID).
		Msg("scrape: updating parsed range")

	if maxMsgID > 0 {
		if err := s.ranges.UpdateRange(ctx, target.ID, minMsgID, maxMsgID); err != nil {
			s.log.Warn().Err(err).Msg("scrape: failed to update parsed range")
		}
	}

	// update target last scraped
	if err := s.targets.UpdateLastScraped(ctx, target.ID, maxMsgID); err != nil {
		s.log.Warn().Err(err).Msg("scrape: failed to update last scraped")
	}

	s.log.Info().
		Int("total", result.TotalFetched).
		Int("new", result.NewJobs).
		Int("skipped_old", result.SkippedOld).
		Int("skipped_empty", result.SkippedEmpty).
		Int("errors", result.Errors).
		Msg("scrape: completed successfully")

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
func (s *Service) createJob(ctx context.Context, targetID uuid.UUID, channelURL string, msg *telegram.Message) error {
	msgID := int64(msg.ID)
	sourceDate := msg.Date
	sourceURL := buildSourceURL(channelURL, msg.ID)

	job := &repository.Job{
		TargetID:    targetID,
		ExternalID:  strconv.FormatInt(msgID, 10),
		RawContent:  msg.Text,
		SourceURL:   &sourceURL,
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

// GetTelegramStatus returns the current status of the telegram connection
func (s *Service) GetTelegramStatus() telegram.Status {
	// If the client wrapper exposes status, we use it.
	// Otherwise, we might need to cast or if the interface changes.
	// The TelegramClient interface in service.go needs update too.
	return s.tgClient.GetStatus()
}

// helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// buildSourceURL creates a Telegram message URL from channel and message ID
func buildSourceURL(channelURL string, messageID int) string {
	// Extract channel name from various formats
	channel := channelURL

	// Remove @ prefix
	channel = strings.TrimPrefix(channel, "@")

	// Remove https://t.me/ prefix
	channel = strings.TrimPrefix(channel, "https://t.me/")
	channel = strings.TrimPrefix(channel, "http://t.me/")

	return fmt.Sprintf("https://t.me/%s/%d", channel, messageID)
}
