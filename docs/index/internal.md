# internal

Private application packages — business logic and infrastructure.

## Business Logic

- **analyzer/** → [analyzer.md](../../internal/analyzer.md) — LLM job analysis worker
- **collector/** → [collector.md](../../internal/collector.md) — Telegram scraping service

## Data Layer

- **models/** → [models.md](../../internal/models.md) — Domain entities
- **repository/** → [repository.md](../../internal/repository.md) — Database CRUD operations

## Infrastructure

- **config/** → [config.md](../../internal/config.md) — Environment configuration
- **database/** → [database.md](../../internal/database.md) — Connection management
- **llm/** → [llm.md](../../internal/llm.md) — OpenAI-compatible LLM client
- **logger/** → [logger.md](../../internal/logger.md) — Structured logging
- **nats/** → [nats.md](../../internal/nats.md) — NATS pub/sub client
- **publisher/** → [publisher.md](../../internal/publisher.md) — Event publishing
- **telegram/** → [telegram.md](../../internal/telegram.md) — Telegram API wrapper
- **web/** → [web.md](../../internal/web.md) — HTTP server + WebSocket hub
