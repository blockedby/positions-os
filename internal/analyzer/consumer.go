package analyzer

import (
	"context"
	"encoding/json"

	"github.com/blockedby/positions-os/internal/nats"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
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

	if err := c.processor.ProcessJob(ctx, event.JobID); err != nil {
		c.log.Error().Str("job_id", event.JobID.String()).Err(err).Msg("failed to process job")
		// Return error to Nak and trigger retry
		return err
	}

	return nil
}
