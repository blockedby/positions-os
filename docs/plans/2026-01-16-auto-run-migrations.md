# Auto-Run Database Migrations Implementation Plan

> **Status:** ✅ COMPLETED
> **For Claude:** Use superpowers:executing-plans to verify implementation if needed.

**Goal:** Automatically run database migrations on service startup for both collector and analyzer services, eliminating manual migration steps and ensuring schema is always up-to-date in all environments.

**Architecture:** Use a dedicated `migrations` package that embeds SQL files via Go's `embed` package. Both collector and analyzer import this package and run migrations synchronously during initialization, before connecting to repositories. Services fail fast if migrations fail.

**Tech Stack:**
- golang-migrate/migrate (v4)
- embed package (Go 1.16+)
- pgx driver for migrations
- Dedicated `migrations/embed.go` package for centralized embedding

---

## Implementation Summary

### Package Structure

```
migrations/
├── embed.go              # Embeds all .sql files as migrations.FS
├── 0001_*.up.sql
├── 0001_*.down.sql
├── ...
internal/migrator/
├── migrator.go           # Migration runner using golang-migrate
├── migrator_test.go      # Unit tests
```

### Key Design Decisions

1. **Centralized Embedding**: Instead of embedding migrations in each cmd, we use a dedicated `migrations/embed.go` package that exports `migrations.FS`. This avoids path issues with `//go:embed ../../` and follows DRY principles.

2. **Shared Migrator Package**: `internal/migrator` provides `NewWithFS(fs.FS)` that accepts any filesystem, making it testable and reusable.

3. **Fail Fast**: Services exit immediately if migrations fail, preventing operation with incorrect schema.

4. **Idempotent**: Running migrations multiple times is safe - already-applied migrations are skipped.

---

## Task 1: Add Migration Dependencies ✅

**Files:** `go.mod`, `go.sum`

Dependencies added:
```go
github.com/golang-migrate/migrate/v4 v4.19.1
```

Includes pgx/v5 database driver and iofs source driver.

**Commit:** `build: add golang-migrate dependencies for auto-migrations`

---

## Task 2: Create Migrations Package ✅

**Files:**
- `migrations/embed.go`
- `internal/migrator/migrator.go`
- `internal/migrator/migrator_test.go`

### migrations/embed.go

```go
// Package migrations embeds database migration files for use by services.
package migrations

import "embed"

// FS contains all migration SQL files.
//
//go:embed *.sql
var FS embed.FS
```

### internal/migrator/migrator.go

```go
package migrator

import (
    "context"
    "errors"
    "fmt"
    "io/fs"

    "github.com/golang-migrate/migrate/v4"
    _ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
    "github.com/golang-migrate/migrate/v4/source/iofs"
)

type Migrator struct {
    migrationsFS fs.FS
}

func NewWithFS(migrationsFS fs.FS) (*Migrator, error) {
    if migrationsFS == nil {
        return nil, errors.New("migrationsFS cannot be nil")
    }
    return &Migrator{migrationsFS: migrationsFS}, nil
}

func (m *Migrator) Up(ctx context.Context, databaseURL string) error {
    // Creates source driver, runs migrations, handles ErrNoChange
}

func (m *Migrator) Version(ctx context.Context, databaseURL string) (uint, bool, error) {
    // Returns current version and dirty state
}
```

**Commit:** `feat(migrator): add migration runner package with embed support`

---

## Task 3: Integrate Migrations in Collector ✅

**Files:** `cmd/collector/main.go`

### Changes

1. Add imports:
```go
import (
    "github.com/blockedby/positions-os/internal/migrator"
    "github.com/blockedby/positions-os/migrations"
)
```

2. Run migrations after logger init, before database connection:
```go
// 2a. Run database migrations
log.Info().Msg("running database migrations")
m, err := migrator.NewWithFS(migrations.FS)
if err != nil {
    log.Fatal().Err(err).Msg("failed to create migrator")
}

if err := m.Up(context.Background(), cfg.DatabaseURL); err != nil {
    log.Fatal().Err(err).Msg("failed to run migrations")
}

version, dirty, err := m.Version(context.Background(), cfg.DatabaseURL)
if err != nil {
    log.Warn().Err(err).Msg("failed to get migration version")
} else {
    log.Info().Uint("version", version).Bool("dirty", dirty).Msg("migrations complete")
}
```

**Commit:** `feat(collector): auto-run migrations on startup with embedded files`

---

## Task 4: Integrate Migrations in Analyzer ✅

**Files:** `cmd/analyzer/main.go`

Identical pattern to collector - add imports and migration code after logger initialization.

**Commit:** `feat(analyzer): auto-run migrations on startup with embedded files`

---

## Task 5: Update Dockerfiles ✅

**Files:** `Dockerfile`, `Dockerfile.analyzer`

Both Dockerfiles copy migrations for inspection/debugging:
```dockerfile
# Copy migrations for inspection (embedded in binary, but useful for debugging)
COPY --from=builder /app/migrations ./migrations
```

Note: Migrations are embedded in binaries at build time, so runtime files are optional but useful for debugging.

**Commit:** `build(docker): ensure migrations are copied to both service images`

---

## Task 6: Update Docker Compose ✅

**Files:** `docker-compose.yml`

Added deprecation notice to migrate service:
```yaml
# DEPRECATED: Migrations now run automatically on collector/analyzer startup.
# This service is kept only for manual operations (rollback, force version, etc.)
# Usage: docker compose --profile tools run migrate down 1
migrate:
  ...
```

**Commit:** `docs(docker): mark manual migrate service as deprecated`

---

## Task 7: Update Documentation ✅

**Files:** `CLAUDE.md`

Added Database Migrations section:
```markdown
### Database Migrations

Migrations run **automatically** when collector or analyzer services start:
- Uses golang-migrate with embedded SQL files (`migrations/` package)
- Services fail fast if migrations fail, ensuring schema correctness
- Migration version logged on startup: `{"version":6,"dirty":false,"message":"migrations complete"}`

Manual operations (development only):
- task migrate-create name=X   # Create new migration
- task migrate-down            # Rollback last migration
- task migrate-reset           # Rollback all (destroys data!)
```

Removed `task migrate-up` from common commands since migrations are automatic.

**Commit:** `docs: update migration documentation for auto-run feature`

---

## Task 8: Update Taskfile ✅

**Files:** `Taskfile.yml`

Added deprecation notices:
```yaml
migrate-up:
  desc: "[DEPRECATED] Run all migrations - now auto-run on service startup"

migrate-down:
  desc: "[DEV] Rollback last migration"

migrate-reset:
  desc: "[DANGER] Rollback all migrations - destroys all data"
```

**Commit:** `chore(tasks): mark manual migration tasks as deprecated`

---

## Testing Checklist

- [x] Clean database startup (migrations run successfully)
- [x] Collector starts and runs migrations
- [x] Analyzer starts and runs migrations
- [x] Docker images build with embedded migrations
- [x] Docker compose stack starts successfully
- [x] Migrations are idempotent (running twice is safe)
- [x] Migration version is logged on startup
- [x] Failed migrations cause service to exit
- [x] Manual migration commands still work (for development)
- [x] Documentation is updated and accurate

---

## Verification Commands

```bash
# Build both services
go build ./cmd/collector ./cmd/analyzer

# Start fresh (clean database)
docker compose down -v
docker compose up -d postgres nats
sleep 3

# Run collector - should show migration output
go run ./cmd/collector/main.go
# Expected: {"level":"info","version":6,"dirty":false,"message":"migrations complete"}

# Run analyzer (in another terminal) - should skip (already migrated)
go run ./cmd/analyzer/main.go
# Expected: {"level":"info","version":6,"dirty":false,"message":"migrations complete"}

# Docker test
docker compose build collector analyzer
docker compose up -d
docker compose logs collector analyzer | grep migration
```

---

## Future Considerations

Not included but worth considering:

1. **Migration Health Check**: Add `/health` endpoint that includes migration status
2. **Migration Metrics**: Track execution time and failures in monitoring
3. **Pre-deployment Validation**: CI job that validates migrations against schema snapshot
4. **Migration Locking**: Ensure only one service runs migrations when scaling horizontally

---

## Summary

This implementation provides automatic database migrations for both collector and analyzer services:

1. **Centralized `migrations/` package** with embedded SQL files
2. **Reusable `internal/migrator`** package for running migrations
3. **Both services** run migrations on startup before connecting to database
4. **Fail-fast behavior** ensures services don't run with incorrect schema
5. **Documentation updated** to reflect automatic migration behavior
6. **Manual tools preserved** for development rollback operations

The result is a production-ready migration system where developers never need to remember manual migration steps, and deployments are safer with automatic schema updates.
