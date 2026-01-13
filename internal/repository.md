# repository

Data access layer — PostgreSQL CRUD operations via pgx and GORM.

## Repositories

- **jobs.go** → [jobs.go.md](jobs.go.md) — Job CRUD, filtering, status updates
- **targets.go** → [targets.go.md](targets.go.md) — Scraping target management
- **ranges.go** → [ranges.go.md](ranges.go.md) — Parsed range tracking
- **stats.go** → [stats.go.md](stats.go.md) — Aggregated statistics

## Tests

- **jobs_test.go** → [jobs_test.go.md](jobs_test.go.md) — Business logic tests
- **jobs_db_test.go** → [jobs_db_test.go.md](jobs_db_test.go.md) — DB integration tests
- **targets_test.go** — Target repository tests
- **ranges_test.go** — Range tracking tests
