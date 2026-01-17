package brain

import (
	"context"
	"fmt"
	"time"

	"github.com/blockedby/positions-os/internal/logger"
)

// LLMClient defines the interface for LLM operations
type LLMClient interface {
	ExtractJobData(ctx context.Context, rawContent, systemPrompt, userPrompt string) (string, error)
}

// BrainLLM wraps the LLM client with rate limiting for brain operations.
// Rate limit is hardcoded to 1 request per second.
type BrainLLM struct {
	client      LLMClient
	rateLimiter *time.Ticker
}

// NewBrainLLM creates a new BrainLLM with 1 req/sec rate limiting.
func NewBrainLLM(client LLMClient) *BrainLLM {
	return &BrainLLM{
		client:      client,
		rateLimiter: time.NewTicker(time.Second),
	}
}

// waitForRateLimit waits for the rate limiter before proceeding.
func (b *BrainLLM) waitForRateLimit() {
	<-b.rateLimiter.C
}

// TailorResume adapts the base resume to a specific job using the LLM.
func (b *BrainLLM) TailorResume(ctx context.Context, baseResume, jobData string) (string, error) {
	logger.Info("calling LLM for resume tailoring")

	b.waitForRateLimit()

	systemPrompt := `You are an HR consultant. Adapt the resume for the job:
- Highlight relevant skills first
- Emphasize relevant experience
- DO NOT add anything that isn't there
- Keep facts, change emphasis
- DO NOT change resume structure (sections, order)
- Language: if job is English → resume in English, otherwise Russian
Return ONLY the Markdown resume, no comments.`

	userPrompt := fmt.Sprintf("## Job:\n%s\n\n## Base Resume:\n%s", jobData, baseResume)

	result, err := b.client.ExtractJobData(ctx, "", systemPrompt, userPrompt)
	if err != nil {
		logger.Error("LLM tailoring failed", err)
		return "", fmt.Errorf("LLM tailoring failed: %w", err)
	}

	logger.Info("resume tailoring complete")
	return result, nil
}

// GenerateCover generates a cover letter using the LLM with a template.
func (b *BrainLLM) GenerateCover(ctx context.Context, jobData, tailoredResume, templateID string) (string, error) {
	logger.Info("calling LLM for cover letter")

	b.waitForRateLimit()

	systemPrompt := `Write a cover letter based on the template.
Adapt to the specific job while keeping structure.
Tone: professional but not formal.
Language: matches the template.`

	templates := map[string]string{
		"formal_ru": `Уважаемый(-ая) {{CONTACT_NAME}},

Меня заинтересовала позиция {{POSITION}} в {{COMPANY}}.

{{RELEVANT_EXPERIENCE}}

{{WHY_COMPANY}}

Буду рад обсудить возможное сотрудничество.

С уважением,
{{MY_NAME}}`,
		"modern_ru": `Привет!

Увидел вакансию {{POSITION}} и понял — это то, что ищу.

{{RELEVANT_EXPERIENCE}}

{{WHY_COMPANY}}

Давайте созвонимся?

{{MY_NAME}}`,
		"professional_en": `Dear Hiring Manager,

I am writing to express my interest in the {{POSITION}} role at {{COMPANY}}.

{{RELEVANT_EXPERIENCE}}

{{WHY_COMPANY}}

I look forward to discussing this opportunity.

Best regards,
{{MY_NAME}}`,
	}

	template, ok := templates[templateID]
	if !ok {
		template = templates["professional_en"] // default
	}

	userPrompt := fmt.Sprintf("## Job:\n%s\n\n## My Resume (tailored):\n%s\n\n## Use template:\n%s\n\nGenerate a personalized cover letter.",
		jobData, tailoredResume, template)

	result, err := b.client.ExtractJobData(ctx, "", systemPrompt, userPrompt)
	if err != nil {
		logger.Error("LLM cover generation failed", err)
		return "", fmt.Errorf("LLM cover generation failed: %w", err)
	}

	logger.Info("cover letter generated")
	return result, nil
}

// Close stops the rate limiter.
func (b *BrainLLM) Close() {
	b.rateLimiter.Stop()
}
