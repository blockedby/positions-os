# internal

Private application packages — business logic and infrastructure.

## Packages

- **analyzer/** → [analyzer.md](../../internal/analyzer.md) — LLM job analysis worker
- **collector/** → [collector.md](../../internal/collector.md) — Telegram scraping service
- **config/** — Environment-based configuration
- **llm/** — OpenAI-compatible LLM client
- **logger/** — Zerolog wrapper
- **models/** — Domain entities (Job, Target, etc.)
- **nats/** — NATS pub/sub client
- **repository/** — Database CRUD operations
- **telegram/** — Telegram API client wrapper
- **web/** — HTTP server, WebSocket hub, templates
