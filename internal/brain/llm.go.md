# llm.go

LLM client wrapper for brain operations — resume tailoring and cover letter generation with rate limiting.

## Type

- **`BrainLLM`** — Wraps LLMClient with 1 req/sec rate limiting (hardcoded)

## Functions

| Function | Description |
|----------|-------------|
| `NewBrainLLM(client LLMClient) *BrainLLM` | Creates wrapper with rate limiter |
| `TailorResume(ctx, baseResume, jobData) -> (string, error)` | Adapts resume to job |
| `GenerateCover(ctx, jobData, tailoredResume, templateID) -> (string, error)` | Generates cover letter |
| `Close()` | Stops rate limiter ticker |

## Interfaces

- **`LLMClient`** — Contract for LLM operations (ExtractJobData method)

## Templates

| ID | Language | Style |
|----|----------|-------|
| `formal_ru` | Russian | Formal |
| `modern_ru` | Russian | Modern/casual |
| `professional_en` | English | Professional |

## Acceptance Status

- [x] Rate limiter limits to 1 req/sec (hardcoded)
- [x] TailorResume calls LLM with correct prompts
- [x] GenerateCover uses templates
- [x] All operations logged
- [x] Tests verify rate limiting behavior

## Test Coverage

| Test | Validates |
|------|-----------|
| `TestBrainLLM_RateLimiting` | 3 calls take ≥2 seconds |
| `TestBrainLLM_TailorResume_CallsLLM` | LLM called with correct params |
| `TestBrainLLM_GenerateCover_CallsLLM` | Template-based generation |
