# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Positions OS** is an automated job search system built in Go. It scrapes job postings from Telegram channels, performs AI-powered analysis using LLMs, and provides a web interface for managing the workflow. The architecture is event-driven using NATS for inter-service communication.

## Development Commands

### Task Runner (Recommended)
The project uses [Task](https://taskfile.dev/) as a cross-platform task runner.

```bash
# Setup
task setup                   # Install development tools (golangci-lint, lefthook)
task deps                    # Install Go dependencies

# Infrastructure
task docker-up               # Start PostgreSQL + NATS
task docker-down             # Stop Docker services
task docker-app              # Start full app stack in Docker

# Database
task migrate-up              # Run all migrations
task migrate-down            # Rollback last migration
task migrate-create name=X   # Create new migration

# Development
task collector               # Run collector service (main service with web UI)
task tg-auth                 # Generate Telegram session string
task tg-topics channel=@X    # List forum topics for a channel

# Testing
task test                    # Run all tests
task test-unit               # Run unit tests only
task test-coverage           # Run tests with coverage

# Code Quality
task lint                    # Run golangci-lint
task fmt                     # Format code
task check                   # Verify compilation
task build                   # Build all binaries
```

### Direct Go Commands

```bash
go run ./cmd/collector/main.go    # Main service (web UI + scraping API)
go run ./cmd/analyzer/main.go     # AI analysis background worker
go run ./cmd/tg-auth/main.go      # Telegram session generator
go test ./...                     # Run tests
```

## Architecture

### Service Components

1. **Collector Service** (`cmd/collector/`): Unified service serving both the scraping API and the web UI. Handles Telegram scraping, manages scraping targets, and exposes REST endpoints.

2. **Analyzer Service** (`cmd/analyzer/`): Background worker that subscribes to NATS `jobs.new` events, processes raw job content through an LLM, and extracts structured data (title, salary, skills, etc.).

3. **Web UI** (`internal/web/`): Dashboard for viewing jobs, managing targets, and monitoring scraping status. Uses Go templates + HTMX + Pico.css (dark mode).

### Data Flow

```
Telegram Channel → Collector → PostgreSQL (RAW jobs)
                                        ↓
                                      NATS (jobs.new)
                                        ↓
                              Analyzer subscribes + processes with LLM
                                        ↓
                              PostgreSQL (ANALYZED jobs with structured_data)
```

### Key Internal Packages

- `internal/collector/` - Scraping business logic, scrape request management
- `internal/analyzer/` - LLM processing and NATS consumption
- `internal/telegram/` - Telegram API client wrapper (gotgproto)
- `internal/repository/` - Database access layer (Jobs, Targets, Ranges, Stats)
- `internal/models/` - Core data models (Job, JobData, JobStatus enum)
- `internal/web/` - HTTP server, WebSocket hub, template engine
- `internal/nats/` - NATS client wrapper for pub/sub
- `internal/llm/` - OpenAI-compatible LLM client
- `internal/config/` - Environment-based configuration

### Job Status Flow

Jobs progress through these statuses: `RAW` → `ANALYZED` → `INTERESTED`/`REJECTED` → `TAILORED` → `SENT` → `RESPONDED`

## Configuration

Required environment variables (see `.env.example`):

- **Telegram**: `TG_API_ID`, `TG_API_HASH`, `TG_SESSION_STRING` (get from https://my.telegram.org)
- **Database**: `DATABASE_URL` (PostgreSQL connection string)
- **NATS**: `NATS_URL` (default: `nats://localhost:4222`)
- **LLM**: `LLM_BASE_URL`, `LLM_MODEL`, `LLM_API_KEY` (OpenAI-compatible API)
- **Web**: `HTTP_PORT` (default: 3100)

Generate `TG_SESSION_STRING` by running `task tg-auth` and following the interactive prompts.

## Important Implementation Details

### Telegram Scraping Limits
- **Max batch size**: 100 messages (Telegram API limit)
- **Max batches**: 100 (safety limit = ~10,000 messages max per scrape)
- **Rate limit**: 100ms delay between batches (auto-applied)
- **FloodWait**: Automatic backoff when Telegram returns rate limit errors

### Forum Topics
Telegram supergroups can be configured as forums with topics. When scraping forums:
- Use `task tg-topics channel=@forum` to list available topics
- Pass `topic_ids` array in scrape request to filter specific topics
- "General" topic always has `id=1`

### Database Migrations
The project uses [golang-migrate](https://github.com/golangci/golangci-lint). Migration files are in `migrations/`.

### NATS Subjects
- `jobs.new` - Published when a new raw job is created
- Stream: `jobs`, Consumer: `analyzer_processor`

## Testing

Run `task test` to execute all tests. The project includes unit tests and integration tests.

## Development Phases

- ✅ Phase 0: Infrastructure (PostgreSQL, NATS, migrations)
- ✅ Phase 1: Collector (Telegram scraping, REST API)
- ✅ Phase 2: Analyzer (LLM-based job analysis)
- ✅ Phase 3: Web UI (User interface, unified service)
- 🚧 Phase 4: Brain (Resume tailoring, PDF generation)

See `docs/implementation-order.md` for detailed phase breakdown.

## Code Quality

- **golangci-lint v2**: Run `task lint` before committing
- **Lefthook**: Pre-commit hooks automatically run linting
- **Format**: Use `task fmt` (includes `goimports`)
- Use `git commit --no-verify` to bypass hooks if needed (not recommended)

## Services and Ports

| Service          | Port  | Description                      |
| ---------------- | ----- | -------------------------------- |
| PostgreSQL       | 5432  | Main database                    |
| NATS             | 4222  | Message broker                   |
| NATS Monitor     | 8222  | NATS monitoring UI               |
| Web UI & API     | 3100  | Dashboard and scraping endpoints |
