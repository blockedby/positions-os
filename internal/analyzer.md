# analyzer

LLM-based job analysis worker — processes raw job postings through AI to extract structured data.

## Core

- **processor.go** → [processor.go.md](processor.go.md) — LLM analysis orchestration
- **consumer.go** → [consumer.go.md](consumer.go.md) — NATS event consumption

## Tests

- **processor_test.go** → [processor_test.go.md](processor_test.go.md) — Unit tests
- **consumer_integration_test.go** → [consumer_integration_test.go.md](consumer_integration_test.go.md) — NATS integration tests
