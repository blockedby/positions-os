//go:build integration

package llm_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/llm"
	"github.com/joho/godotenv"
)

func TestIntegration_ExtractJobData(t *testing.T) {
	// Load .env from project root
	_ = godotenv.Load("../../.env")

	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		t.Skip("Skipping integration test: LLM_BASE_URL not set")
	}

	cfg := llm.Config{
		BaseURL:     baseURL,
		Model:       os.Getenv("LLM_MODEL"),
		APIKey:      os.Getenv("LLM_API_KEY"),
		MaxTokens:   1000,
		Temperature: 0.1,
		Timeout:     60 * time.Second,
	}

	client := llm.NewClient(cfg)

	// Load prompt
	prompt, err := llm.LoadPrompt("../../docs/prompts/job-extraction.xml")
	if err != nil {
		t.Fatalf("Failed to load prompt: %v", err)
	}

	// Sample raw content
	rawContent := `
We are looking for a Senior Go Developer.
Salary: $5000 - $7000.
Stack: Go, PostgreSQL, Kafka, Redis.
Contact: hiring@company.com or @hiring_manager using Telegram.
Remote only.
`

	ctx := context.Background()
	userPrompt := prompt.BuildUserPrompt(rawContent)

	t.Logf("Sending request to LLM at %s...", baseURL)
	response, err := client.ExtractJobData(ctx, rawContent, prompt.System, userPrompt)
	if err != nil {
		t.Fatalf("LLM Request failed: %v", err)
	}

	t.Logf("LLM Response:\n%s", response)

	if len(response) == 0 {
		t.Error("Received empty response from LLM")
	}
}
