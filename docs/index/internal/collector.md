# collector

Telegram scraping service — fetches job postings from channels and forums.

## Core

- **service.go** → [service.go.md](../../internal/collector/service.go.md) — Scraping orchestration
- **manager.go** → [manager.go.md](../../internal/collector/manager.go.md) — Scrape job lifecycle

## API

- **handler.go** → [handler.go.md](../../internal/collector/handler.go.md) — HTTP endpoints
- **router.go** → [router.go.md](../../internal/collector/router.go.md) — Route setup
- **validation.go** → [validation.go.md](../../internal/collector/validation.go.md) — Request validation

## Tests

- **handler_test.go** → [handler_test.go.md](../../internal/collector/handler_test.go.md)
- **manager_test.go** → [manager_test.go.md](../../internal/collector/manager_test.go.md)
- **validation_test.go** → [validation_test.go.md](../../internal/collector/validation_test.go.md)
