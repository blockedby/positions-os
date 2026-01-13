# analyzer

LLM-based job analysis worker — processes raw job postings through AI to extract structured data.

## Core

- **processor.go** → [processor.go.md](../../internal/analyzer/processor.go.md) — LLM analysis orchestration
- **consumer.go** → [consumer.go.md](../../internal/analyzer/consumer.go.md) — NATS event consumption

## Tests

- **processor_test.go** → [processor_test.go.md](../../internal/analyzer/processor_test.go.md) — Unit tests
- **consumer_integration_test.go** → [consumer_integration_test.go.md](../../internal/analyzer/consumer_integration_test.go.md) — NATS integration tests
