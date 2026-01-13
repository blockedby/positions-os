# internal

Private application packages — business logic and infrastructure.

## Business Logic

- **analyzer/** → [analyzer.md](analyzer.md) — LLM job analysis worker
- **collector/** → [collector.md](collector.md) — Telegram scraping service

## Data Layer

- **models/** → [models.md](models.md) — Domain entities
- **repository/** → [repository.md](repository.md) — Database CRUD operations

## Infrastructure

- **config/** → [config.md](config.md) — Environment configuration
- **database/** → [database.md](database.md) — Connection management
- **llm/** → [llm.md](llm.md) — OpenAI-compatible LLM client
- **logger/** → [logger.md](logger.md) — Structured logging
- **nats/** → [nats.md](nats.md) — NATS pub/sub client
- **publisher/** → [publisher.md](publisher.md) — Event publishing
- **telegram/** → [telegram.md](telegram.md) — Telegram API wrapper
- **web/** → [web.md](web.md) — HTTP server + WebSocket hub
