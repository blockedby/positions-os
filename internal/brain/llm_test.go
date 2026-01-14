package brain

import (
	"context"
	"testing"
	"time"
)

// MockLLMClient is a test double for LLM operations
type MockLLMClient struct {
	ExtractFunc func(ctx context.Context, rawContent, systemPrompt, userPrompt string) (string, error)
	CallCount   int
	Delay       time.Duration // Simulates LLM latency
}

func (m *MockLLMClient) ExtractJobData(ctx context.Context, rawContent, systemPrompt, userPrompt string) (string, error) {
	m.CallCount++
	if m.Delay > 0 {
		time.Sleep(m.Delay)
	}
	if m.ExtractFunc != nil {
		return m.ExtractFunc(ctx, rawContent, systemPrompt, userPrompt)
	}
	return "mock response", nil
}

// TestBrainLLM_RateLimiting
func TestBrainLLM_RateLimiting(t *testing.T) {
	// Setup
	mockLLM := &MockLLMClient{Delay: 10 * time.Millisecond}
	brain := NewBrainLLM(mockLLM)
	ctx := context.Background()

	// Execute: Make 3 rapid calls
	start := time.Now()
	for i := 0; i < 3; i++ {
		_, _ = brain.TailorResume(ctx, "base resume", "job data")
	}
	elapsed := time.Since(start)

	// Assert: With rate limiter at 1 req/sec, 3 calls should take at least 2 seconds
	minExpected := 2 * time.Second
	if elapsed < minExpected {
		t.Errorf("Rate limiting not working: 3 calls took %v, want at least %v", elapsed, minExpected)
	}

	// Assert all calls were made
	if mockLLM.CallCount != 3 {
		t.Errorf("Expected 3 LLM calls, got %d", mockLLM.CallCount)
	}
}

// TestBrainLLM_TailorResume_CallsLLM
func TestBrainLLM_TailorResume_CallsLLM(t *testing.T) {
	// Setup
	mockLLM := &MockLLMClient{
		ExtractFunc: func(ctx context.Context, rawContent, systemPrompt, userPrompt string) (string, error) {
			return "# Tailored Resume", nil
		},
	}
	brain := NewBrainLLM(mockLLM)
	ctx := context.Background()

	// Execute
	result, err := brain.TailorResume(ctx, "base resume", "job data")

	// Assert
	if err != nil {
		t.Errorf("TailorResume() error = %v", err)
	}
	if result != "# Tailored Resume" {
		t.Errorf("TailorResume() = %q, want %q", result, "# Tailored Resume")
	}
}

// TestBrainLLM_GenerateCover_CallsLLM
func TestBrainLLM_GenerateCover_CallsLLM(t *testing.T) {
	// Setup
	mockLLM := &MockLLMClient{
		ExtractFunc: func(ctx context.Context, rawContent, systemPrompt, userPrompt string) (string, error) {
			return "Dear Hiring Manager,", nil
		},
	}
	brain := NewBrainLLM(mockLLM)
	ctx := context.Background()

	// Execute
	result, err := brain.GenerateCover(ctx, "job data", "tailored resume", "formal_ru")

	// Assert
	if err != nil {
		t.Errorf("GenerateCover() error = %v", err)
	}
	if result != "Dear Hiring Manager," {
		t.Errorf("GenerateCover() = %q, want %q", result, "Dear Hiring Manager,")
	}
}
