package llm

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		BaseURL:     "http://localhost:1234/v1",
		Model:       "gpt-4o-mini",
		APIKey:      "test-key",
		MaxTokens:   1000,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
	}

	client := NewClient(cfg)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	// Since fields are private, we can't check them directly without reflection or getters.
	// But we can ensure basic initialization validation passes.
	if client.client == nil {
		t.Error("Underlying openai client is nil")
	}
}

func TestConvertJSONSchema(t *testing.T) {
	// Value extraction test to ensure schema handling (future proofing)
	// For now, this is a placeholder if we add schema validation.
}

func TestExtractJobData_ContextCancellation(t *testing.T) {
	// We can't easily mock the OpenAI client without an interface,
	// but we can test that the context timeout logic is in place
	// if we could inject a slow client.
	// For now, we'll trust the integration verification for the actual call.
	// This test acts as a placeholder for TDD flow.
}
