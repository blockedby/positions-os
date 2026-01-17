// Package llm provides OpenAI-compatible LLM client.
package llm

import (
	"context"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// Client is a wrapper around go-openai with specific configurations.
type Client struct {
	client      *openai.Client
	model       string
	maxTokens   int
	temperature float32
	timeout     time.Duration
}

// Config holds the configuration for the LLM client.
type Config struct {
	BaseURL     string
	Model       string
	APIKey      string
	MaxTokens   int
	Temperature float32
	Timeout     time.Duration
}

// NewClient creates a new LLM client with the provided configuration.
func NewClient(cfg Config) *Client {
	config := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	}

	return &Client{
		client:      openai.NewClientWithConfig(config),
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		timeout:     cfg.Timeout,
	}
}

// ExtractJobData extracts structured data from the raw job content.
// It uses the system and user prompts provided to guide the LLM.
func (c *Client) ExtractJobData(ctx context.Context, rawContent string, systemPrompt, userPrompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		MaxTokens:   c.maxTokens,
		Temperature: c.temperature,
	})
	if err != nil {
		return "", fmt.Errorf("llm completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}
