# collector

Telegram scraping service — fetches job postings from channels and forums.

## Core

- **service.go** → [service.go.md](service.go.md) — Scraping orchestration
- **manager.go** → [manager.go.md](manager.go.md) — Scrape job lifecycle

## API

- **handler.go** → [handler.go.md](handler.go.md) — HTTP endpoints
- **router.go** → [router.go.md](router.go.md) — Route setup
- **validation.go** → [validation.go.md](validation.go.md) — Request validation

## Tests

- **handler_test.go** → [handler_test.go.md](handler_test.go.md)
- **manager_test.go** → [manager_test.go.md](manager_test.go.md)
- **validation_test.go** → [validation_test.go.md](validation_test.go.md)
