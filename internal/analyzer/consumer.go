// Package analyzer provides NATS consumer for job analysis.
package analyzer

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/blockedby/positions-os/internal/nats"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	maxRetries    = 3
	retryInterval = 500 * time.Millisecond
)

// Consumer handles consuming NATS events
type Consumer struct {
	client    *nats.Client
	processor *Processor
	log       *zerolog.Logger
}

// NewConsumer creates a new NATS consumer
func NewConsumer(client *nats.Client, processor *Processor, log *zerolog.Logger) *Consumer {
	return &Consumer{
		client:    client,
		processor: processor,
		log:       log,
	}
}

// Start subscribes to jobs.new and starts processing
func (c *Consumer) Start(ctx context.Context) error {
	// Subject: jobs.new
	// Stream: jobs (created by collector)
	// Durable Consumer Name: analyzer_processor
	c.log.Info().Msg("starting analyzer consumer")
	return c.client.Subscribe(ctx, "jobs", "analyzer_processor", "jobs.new", c.handleMessage)
}

// handleMessage processes a single message
func (c *Consumer) handleMessage(data []byte) error {
	var event struct {
		JobID uuid.UUID `json:"job_id"`
	}
	if err := json.Unmarshal(data, &event); err != nil {
		c.log.Error().Err(err).Msg("invalid nats message format, skipping")
		return nil // Return nil to Ack and move on (poison message)
	}

	c.log.Debug().Str("job_id", event.JobID.String()).Msg("received job event")

	// Create a new context for processing
	// We use background because the message handler might be called with a context that behaves differently
	// or we want to ensure independent execution. However, nats library Consume usually uses closure.
	ctx := context.Background()

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		lastErr = c.processor.ProcessJob(ctx, event.JobID)
		if lastErr == nil {
			return nil
		}

		// If not found, retry after delay (race condition with DB commit)
		if strings.Contains(lastErr.Error(), "not found") {
			if attempt < maxRetries {
				c.log.Debug().
					Str("job_id", event.JobID.String()).
					Int("attempt", attempt).
					Msg("job not found, retrying after delay")
				time.Sleep(retryInterval)
				continue
			}
			// Max retries reached, skip the message
			c.log.Warn().Str("job_id", event.JobID.String()).Msg("job not found after retries, skipping")
			return nil
		}

		// Other error - don't retry internally, let NATS handle it
		break
	}

	c.log.Error().Str("job_id", event.JobID.String()).Err(lastErr).Msg("failed to process job")
	return lastErr
}
