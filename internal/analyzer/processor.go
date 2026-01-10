package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/blockedby/positions-os/internal/llm"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LLMClient abstracts the LLM provider
type LLMClient interface {
	ExtractJobData(ctx context.Context, rawContent, systemPrompt, userPrompt string) (string, error)
}

// JobsRepository defines required DB operations
type JobsRepository interface {
	UpdateStructuredData(ctx context.Context, id uuid.UUID, data []byte, status string) error
	GetByID(ctx context.Context, id uuid.UUID) (*repository.Job, error)
}

// Processor handles the analysis of raw job data
type Processor struct {
	llm     LLMClient
	repo    JobsRepository
	prompts *llm.PromptConfig
	log     *zerolog.Logger
}

// NewProcessor creates a new job processor
func NewProcessor(
	llm LLMClient,
	repo JobsRepository,
	prompts *llm.PromptConfig,
	log *zerolog.Logger,
) *Processor {
	return &Processor{
		llm:     llm,
		repo:    repo,
		prompts: prompts,
		log:     log,
	}
}

// ProcessJob analyzes a single job by ID
func (p *Processor) ProcessJob(ctx context.Context, jobID uuid.UUID) error {
	// 1. Fetch job
	job, err := p.repo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("fetch job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// 2. Prepare prompt
	userPrompt := p.prompts.BuildUserPrompt(job.RawContent)

	// 3. Call LLM
	jsonStr, err := p.llm.ExtractJobData(ctx, job.RawContent, p.prompts.System, userPrompt)
	if err != nil {
		return fmt.Errorf("llm extraction: %w", err)
	}

	// 4. Clean and Validate JSON
	// LLMs sometimes wrap code in ```json ... ```
	cleaned := cleanJSON(jsonStr)

	if !json.Valid([]byte(cleaned)) {
		// Attempt to recover or just fail? For now fail.
		return fmt.Errorf("invalid json received from llm")
	}

	// 5. Update DB
	if err := p.repo.UpdateStructuredData(ctx, jobID, []byte(cleaned), "ANALYZED"); err != nil {
		return fmt.Errorf("update db: %w", err)
	}

	p.log.Info().Str("job_id", jobID.String()).Msg("job analyzed successfully")
	return nil
}

// cleanJSON removes markdown code blocks if present
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
