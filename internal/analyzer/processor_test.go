package analyzer

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/blockedby/positions-os/internal/llm"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MockJobsRepo implements JobsRepository interface for testing
type MockJobsRepo struct {
	Jobs        map[uuid.UUID]*repository.Job
	UpdatedData map[string]interface{}
	Err         error
	mu          sync.Mutex
}

// ... (MockLLMClient stays same)

// MockLLMClient implements LLMClient for testing
type MockLLMClient struct {
	ExtractFunc func(ctx context.Context, raw, sys, user string) (string, error)
}

func (m *MockLLMClient) ExtractJobData(ctx context.Context, rawContent, systemPrompt, userPrompt string) (string, error) {
	if m.ExtractFunc != nil {
		return m.ExtractFunc(ctx, rawContent, systemPrompt, userPrompt)
	}
	return "{}", nil
}

func (m *MockJobsRepo) GetByID(ctx context.Context, id uuid.UUID) (*repository.Job, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Jobs[id], nil
}

func (m *MockJobsRepo) UpdateStructuredData(ctx context.Context, id uuid.UUID, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Err != nil {
		return m.Err
	}
	m.UpdatedData = data
	return nil
}

func (m *MockJobsRepo) GetUpdatedData() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.UpdatedData
}

func TestProcessor_ProcessJob(t *testing.T) {
	// Setup simple logger
	logger := zerolog.Nop()

	// Setup Prompts
	prompts := &llm.PromptConfig{
		System: "sys",
		User:   "content: {{RAW_CONTENT}}",
	}

	// Test Case 1: Success
	t.Run("Success", func(t *testing.T) {
		jobID := uuid.New()

		// Setup Mock LLM
		mockLLM := &MockLLMClient{
			ExtractFunc: func(ctx context.Context, raw, sys, user string) (string, error) {
				// Verify inputs if needed
				if !strings.Contains(user, "Go Developer") {
					t.Errorf("User prompt missing content: %s", user)
				}
				return `{"title": "Go Developer"}`, nil
			},
		}

		// Setup Mock Repo
		mockRepo := &MockJobsRepo{
			Jobs: map[uuid.UUID]*repository.Job{
				jobID: {
					ID:         jobID,
					RawContent: "Go Developer",
					Status:     "RAW",
				},
			},
		}

		// Initialize Processor
		proc := NewProcessor(mockLLM, mockRepo, prompts, &logger)

		// Execute
		err := proc.ProcessJob(context.Background(), jobID)
		if err != nil {
			t.Fatalf("ProcessJob failed: %v", err)
		}

		// Verify
		if mockRepo.UpdatedData["title"] != "Go Developer" {
			t.Errorf("Unexpected data: %v", mockRepo.UpdatedData)
		}
	})

	// Test Case 2: LLM Validation Error (Invalid JSON)
	t.Run("InvalidJSON", func(t *testing.T) {
		jobID := uuid.New()

		mockLLM := &MockLLMClient{
			ExtractFunc: func(ctx context.Context, raw, sys, user string) (string, error) {
				return `INVALID JSON`, nil
			},
		}

		mockRepo := &MockJobsRepo{
			Jobs: map[uuid.UUID]*repository.Job{
				jobID: {ID: jobID, RawContent: "Test"},
			},
		}

		proc := NewProcessor(mockLLM, mockRepo, prompts, &logger)
		err := proc.ProcessJob(context.Background(), jobID)
		if err == nil {
			t.Error("Expected error for invalid JSON, got nil")
		}
	})

	// Test Case 3: Markdown JSON Cleanup
	t.Run("MarkdownCleanup", func(t *testing.T) {
		jobID := uuid.New()

		mockLLM := &MockLLMClient{
			ExtractFunc: func(ctx context.Context, raw, sys, user string) (string, error) {
				return "```json\n{\"key\": \"val\"}\n```", nil
			},
		}

		mockRepo := &MockJobsRepo{
			Jobs: map[uuid.UUID]*repository.Job{
				jobID: {ID: jobID, RawContent: "Test"},
			},
		}

		proc := NewProcessor(mockLLM, mockRepo, prompts, &logger)
		err := proc.ProcessJob(context.Background(), jobID)
		if err != nil {
			t.Fatalf("Failed to process markdown json: %v", err)
		}

		if mockRepo.UpdatedData["key"] != "val" {
			t.Errorf("JSON cleanup failed. Got: %v", mockRepo.UpdatedData)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr // simplistic check or stdlib strings.Contains
}
