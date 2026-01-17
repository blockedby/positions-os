// Package nats provides a client for NATS JetStream pub/sub messaging.
package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client wraps nats connection and jetstream context.
type Client struct {
	Conn *nats.Conn
	js   jetstream.JetStream
}

// New creates a new nats client with jetstream support.
func New(_ context.Context, natsURL string) (*Client, error) {
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("connect to nats: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create jetstream context: %w", err)
	}

	return &Client{Conn: conn, js: js}, nil
}

// EnsureStream creates a stream if it doesn't exist.
func (c *Client) EnsureStream(ctx context.Context, name string, subjects []string) error {
	_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     name,
		Subjects: subjects,
	})
	if err != nil {
		return fmt.Errorf("create stream %s: %w", name, err)
	}
	return nil
}

// Publish publishes a message to a subject.
func (c *Client) Publish(ctx context.Context, subject string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = c.js.Publish(ctx, subject, payload)
	if err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}

	return nil
}

// Subscribe creates a durable consumer and starts consuming messages.
func (c *Client) Subscribe(ctx context.Context, stream, consumer, subject string, handler func([]byte) error) error {
	cons, err := c.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:       consumer,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	_, err = cons.Consume(func(msg jetstream.Msg) {
		if err := handler(msg.Data()); err != nil {
			// negative acknowledgement - will be redelivered
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})

	return err
}

// Close closes the nats connection.
func (c *Client) Close() {
	c.Conn.Close()
}

// IsConnected returns true if connected to nats.
func (c *Client) IsConnected() bool {
	return c.Conn.IsConnected()
}
