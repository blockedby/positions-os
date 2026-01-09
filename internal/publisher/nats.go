package publisher

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/blockedby/positions-os/internal/collector"
	"github.com/nats-io/nats.go"
)

// NATSClient interface to allow mocking
type NATSClient interface {
	Publish(subject string, data []byte) error
}

// NATSPublisher implements collector.EventPublisher
type NATSPublisher struct {
	js NATSClient
}

// NewNATSPublisher creates a new publisher
func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
	return &NATSPublisher{js: conn}
}

// PublishJobNew publishes a new job event
func (p *NATSPublisher) PublishJobNew(ctx context.Context, event collector.JobNewEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if err := p.js.Publish("jobs.new", data); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	return nil
}
