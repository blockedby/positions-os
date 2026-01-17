package analyzer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/blockedby/positions-os/internal/llm"
	"github.com/blockedby/positions-os/internal/nats"
	"github.com/blockedby/positions-os/internal/repository"
)

func TestConsumer_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") == "" {
		t.Skip("Skipping integration test; set INTEGRATION_TEST=1 to run")
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		t.Skip("NATS_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Connect to NATS
	client, err := nats.New(ctx, natsURL)
	if err != nil {
		t.Fatalf("failed to connect to nats: %v", err)
	}
	defer client.Close()

	// Ensure stream exists
	err = client.EnsureStream(ctx, "jobs", []string{"jobs.new"})
	if err != nil {
		t.Fatalf("failed to ensure stream: %v", err)
	}

	// 2. Setup Processor with Mocks
	jobID := uuid.New()
	mockRepo := &MockJobsRepo{
		Jobs: map[uuid.UUID]*repository.Job{
			jobID: {
				ID:         jobID,
				RawContent: "Go Developer",
				Status:     "RAW",
			},
		},
	}
	mockLLM := &MockLLMClient{
		ExtractFunc: func(ctx context.Context, raw, sys, user string) (string, error) {
			return `{"title": "Go Developer"}`, nil
		},
	}
	prompts := &llm.PromptConfig{User: "{{RAW_CONTENT}}"}
	logger := zerolog.Nop()

	proc := NewProcessor(mockLLM, mockRepo, prompts, &logger)

	// 3. Start Consumer
	consumer := NewConsumer(client, proc, &logger)
	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("failed to start consumer: %v", err)
	}

	// 4. Publish Event
	event := struct {
		JobID uuid.UUID `json:"job_id"`
	}{
		JobID: jobID,
	}

	// We use the same client to publish
	err = client.Publish(ctx, "jobs.new", event)
	if err != nil {
		t.Fatalf("failed to publish event: %v", err)
	}

	// 5. Wait for processing (polling)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				// Check if Repo was updated
				if len(mockRepo.UpdatedData) > 0 {
					done <- true
					return
				}
			}
		}
	}()

	select {
	case <-done:
		// Success
		if mockRepo.UpdatedData["title"] != "Go Developer" {
			t.Errorf("Unexpected title in repo: %v", mockRepo.UpdatedData)
		}
	case <-ctx.Done():
		t.Fatal("Timeout waiting for message processing")
	}
}
