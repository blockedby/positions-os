# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Positions OS** is an automated job search system built in Go with a React frontend. It scrapes job postings from Telegram channels, performs AI-powered analysis using LLMs, and provides a modern web interface for managing the workflow.

**Tech Stack:**
- **Backend**: Go 1.23+, Chi router, JSON REST API
- **Frontend**: React 19 + TypeScript, Vite, TanStack Query
- **Styling**: Pico.css (dark mode only)
- **Database**: PostgreSQL with golang-migrate
- **Message Broker**: NATS JetStream
- **LLM**: OpenAI-compatible API (LM Studio, Ollama, OpenAI)
- **Telegram**: MTProto via gotgproto (not Bot API)
- **Testing**: Vitest (unit), Playwright (E2E)

## Development Commands

### Task Runner (Recommended)

```bash
# Infrastructure
task docker-up               # Start PostgreSQL + NATS
task docker-down             # Stop Docker services

# Development
task collector               # Run main backend service
task setup                   # Install dev tools (golangci-lint, lefthook)

# Testing & Quality
task test                    # Run Go tests
task lint                    # Run golangci-lint
task e2e                     # Run Playwright E2E tests (requires backend running)
```

### Frontend (in frontend/ directory)

```bash
bun install                  # Install dependencies
bun dev                      # Dev server at http://localhost:5173
bun build                    # Build to ../static/dist
bun test                     # Unit tests with Vitest
bun lint                     # ESLint
```

### Running a Single Test

```bash
# Go - run specific test
go test -v -run TestFunctionName ./internal/package/...

# Frontend unit test
cd frontend && bun test -- --grep "test name"

# Frontend E2E - specific file
task e2e-targets             # Just targets.spec.ts
bunx playwright test --grep "test name"  # By test name
```

## Architecture

### Service Components

1. **Collector** (`cmd/collector/`): Main backend - REST API, Telegram scraping, serves React app
2. **Analyzer** (`cmd/analyzer/`): Background worker - subscribes to NATS, processes jobs through LLM

### Data Flow

```
Telegram Channel → Collector → PostgreSQL (RAW jobs)
                                    ↓
                                  NATS (jobs.new)
                                    ↓
                          Analyzer → LLM processing
                                    ↓
                          PostgreSQL (ANALYZED jobs)
                                    ↓
                          WebSocket → React frontend
```

### Key Packages

- `internal/collector/` - Scraping orchestration and request management
- `internal/analyzer/` - LLM processing and NATS consumption
- `internal/telegram/` - Telegram MTProto client wrapper
- `internal/repository/` - Database access (Jobs, Targets, Ranges, Stats)
- `internal/models/` - Core entities (Job, JobData, JobStatus)
- `internal/web/handlers/` - API endpoint handlers
- `internal/llm/` - OpenAI-compatible LLM client

### Job Status Flow

`RAW` → `ANALYZED` → `INTERESTED`/`REJECTED` → `TAILORED` → `SENT` → `RESPONDED`

## Configuration

Required environment variables (see `.env.example`):

- `TG_API_ID`, `TG_API_HASH`, `TG_SESSION_STRING` - from https://my.telegram.org
- `DATABASE_URL` - PostgreSQL connection string
- `NATS_URL` - default: `nats://localhost:4222`
- `LLM_BASE_URL`, `LLM_MODEL`, `LLM_API_KEY` - OpenAI-compatible API
- `HTTP_PORT` - default: 3100

Generate session: `task tg-auth`

## Important Implementation Details

### Telegram

- Uses MTProto (userbot), not Bot API - can read any public channel
- Max 100 messages per batch, 100ms delay between batches
- FloodWait errors handled automatically with backoff
- Forum topics: use `task tg-topics channel=@forum` to list topic IDs

### Frontend

- Vite dev server proxies `/api` and `/ws` to backend at port 3100
- Production build outputs to `../static/dist` (served by Go)
- Path alias: `@/` maps to `src/`
- Real-time updates via WebSocket at `/ws`

### NATS

- Subject `jobs.new` - published when raw job created
- Stream: `jobs`, Consumer: `analyzer_processor`

### Database Migrations

Migrations run **automatically** when collector or analyzer services start:
- Uses golang-migrate with embedded SQL files (`migrations/` package)
- Services fail fast if migrations fail, ensuring schema correctness
- Migration version logged on startup: `{"version":6,"dirty":false,"message":"migrations complete"}`

Manual operations (development only):
```bash
task migrate-create name=X   # Create new migration
task migrate-down            # Rollback last migration
task migrate-reset           # Rollback all (destroys data!)
```

## Services and Ports

| Service      | Port | Description                   |
| ------------ | ---- | ----------------------------- |
| PostgreSQL   | 5432 | Main database                 |
| NATS         | 4222 | Message broker                |
| Backend API  | 3100 | REST API + WebSocket + Static |
| Frontend Dev | 5173 | Vite dev server               |

## Code Quality

- **golangci-lint v2**: `task lint` before committing
- **Lefthook**: Pre-commit hooks run automatically
- **ESLint**: Frontend linting with React hooks plugin
- Dark mode only - no light mode support needed

## Documentation

See `docs/` for detailed documentation:
- `docs/implementation-order.md` - Phase breakdown
- `docs/telegram-integration.md` - Telegram technical details
- `docs/design-system.md` - UI design reference
- `docs/prompts/` - LLM prompt templates (XML)
