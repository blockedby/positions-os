# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Positions OS** is an automated job search system built in Go with a React frontend. It scrapes job postings from Telegram channels, performs AI-powered analysis using LLMs, and provides a modern web interface for managing the workflow. The architecture is event-driven using NATS for inter-service communication.

**Tech Stack:**
- **Backend**: Go 1.21+, Chi router, JSON REST API
- **Frontend**: React 19 + TypeScript, Vite, React Router, TanStack Query
- **Styling**: Pico.css (dark mode only) + custom CSS
- **Database**: PostgreSQL with golang-migrate
- **Message Broker**: NATS JetStream
- **LLM**: OpenAI-compatible API (LM Studio, Ollama, OpenAI)
- **Telegram**: MTProto via gotgproto (not Bot API)
- **Testing**: Vitest (unit), Playwright (E2E)

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
task collector               # Run collector service (main backend service)
task tg-auth                 # Generate Telegram session string
task tg-topics channel=@X    # List forum topics for a channel

# Testing
task test                    # Run Go tests
task test-unit               # Run unit tests only
task test-coverage           # Run tests with coverage
task e2e                     # Run Playwright E2E tests
task e2e-headed              # Run E2E with visible browser
task e2e-ui                  # Open Playwright UI mode
task e2e-debug               # Run E2E in debug mode

# Code Quality
task lint                    # Run golangci-lint
task fmt                     # Format code
task check                   # Verify compilation
task build                   # Build all binaries
```

### Frontend Development

```bash
cd frontend

# Development
bun install                  # Install dependencies
bun dev                      # Start dev server (http://localhost:5173)
bun build                    # Build for production (outputs to ../static/dist)

# Testing
bun test                     # Run unit tests with Vitest
bun test:coverage            # Run tests with coverage
bun test:e2e                 # Run Playwright E2E tests
bun test:e2e:ui              # Open Playwright UI

# Linting
bun lint                     # Run ESLint
bun type-check               # TypeScript type checking
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

1. **Collector Service** (`cmd/collector/`): Main backend service exposing JSON REST API. Handles Telegram scraping, manages scraping targets, and serves the built React app as static files.

2. **Analyzer Service** (`cmd/analyzer/`): Background worker that subscribes to NATS `jobs.new` events, processes raw job content through an LLM, and extracts structured data (title, salary, skills, etc.).

3. **Frontend SPA** (`frontend/`): React TypeScript application with routing (React Router), state management (TanStack Query), and real-time updates (WebSocket). Built with Vite and styled with Pico.css dark theme.

### Data Flow

```
Telegram Channel → Collector → PostgreSQL (RAW jobs)
                                        ↓
                                      NATS (jobs.new)
                                        ↓
                              Analyzer subscribes + processes with LLM
                                        ↓
                              PostgreSQL (ANALYZED jobs with structured_data)
                                        ↓
                              WebSocket → React frontend updates
```

### Key Internal Packages (Backend)

- `internal/collector/` - Scraping business logic, scrape request management
- `internal/analyzer/` - LLM processing and NATS consumption
- `internal/telegram/` - Telegram API client wrapper (gotgproto)
- `internal/repository/` - Database access layer (Jobs, Targets, Ranges, Stats)
- `internal/models/` - Core data models (Job, JobData, JobStatus enum)
- `internal/web/` - HTTP server, WebSocket hub, Chi router
- `internal/web/handlers/` - API endpoint handlers (Jobs, Targets, Stats, Auth, Collector)
- `internal/nats/` - NATS client wrapper for pub/sub
- `internal/llm/` - OpenAI-compatible LLM client
- `internal/config/` - Environment-based configuration

### Frontend Structure

```
frontend/src/
├── main.tsx                   # Entry point
├── App.tsx                    # Root component with routing
├── components/
│   ├── layout/                # Sidebar, Main layout
│   ├── ui/                    # Reusable components (Button, Card, Input, Badge, etc.)
│   ├── dashboard/             # Dashboard-specific components
│   ├── jobs/                  # Jobs page components (FilterBar, JobsTable, JobDetail)
│   └── settings/              # Settings page components (TargetList, TelegramAuth)
├── pages/
│   ├── Dashboard.tsx          # Dashboard page
│   ├── Jobs.tsx               # Jobs listing page
│   └── Settings.tsx           # Settings page
├── hooks/
│   ├── useJobs.ts             # TanStack Query hooks for jobs
│   ├── useTargets.ts          # TanStack Query hooks for targets
│   ├── useStats.ts            # TanStack Query hooks for stats
│   └── useWebSocket.ts        # WebSocket connection management
├── lib/
│   ├── api.ts                 # API client & WebSocket client
│   ├── types.ts               # TypeScript type definitions
│   └── query-client.ts        # React Query configuration
└── styles/
    └── globals.css            # Global styles + Pico.css customization
```

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
The project uses [golang-migrate](https://github.com/golang-migrate/migrate). Migration files are in `migrations/`.

### NATS Subjects
- `jobs.new` - Published when a new raw job is created
- Stream: `jobs`, Consumer: `analyzer_processor`

### Frontend Development
- Vite dev server: `http://localhost:5173` (proxies `/api` and `/ws` to backend)
- Production build: outputs to `../static/dist` (served by Go backend)
- Path alias: `@/` maps to `src/`
- Real-time updates via WebSocket connection
- State management: TanStack Query (server state) + React hooks (local state)

## Testing

### Backend Tests
Run `task test` to execute all Go tests. The project includes unit tests and integration tests.

### Frontend Tests
- **Unit Tests**: Vitest with Testing Library (`bun test`)
- **E2E Tests**: Playwright (`task e2e` or `bun test:e2e`)
  - Tests include: WebSocket stability, Targets CRUD, API endpoints
  - Located in `frontend/e2e/`
  - Global setup/teardown for test isolation

## Development Phases

- ✅ Phase 0: Infrastructure (PostgreSQL, NATS, migrations)
- ✅ Phase 1: Collector (Telegram scraping, REST API)
- ✅ Phase 2: Analyzer (LLM-based job analysis)
- ✅ Phase 3: Web UI (React SPA with real-time updates)
- 🚧 Phase 4: Brain (Resume tailoring, PDF generation)

See `docs/implementation-order.md` for detailed phase breakdown.

## Code Quality

- **golangci-lint v2**: Run `task lint` before committing
- **Lefthook**: Pre-commit hooks automatically run linting
- **Format**: Use `task fmt` (includes `goimports`)
- **ESLint**: Frontend linting with React hooks plugin
- Use `git commit --no-verify` to bypass hooks if needed (not recommended)

## Services and Ports

| Service          | Port  | Description                      |
| ---------------- | ----- | -------------------------------- |
| PostgreSQL       | 5432  | Main database                    |
| NATS             | 4222  | Message broker                   |
| NATS Monitor     | 8222  | NATS monitoring UI               |
| Backend API      | 3100  | Go REST API + WebSocket + Static |
| Frontend Dev     | 5173  | Vite dev server (development)    |

---

## Design System

The web UI uses a dark-themed design system built on Pico.css with custom variables.

### Color Palette

**Background Colors:**
```css
--pico-background-color: #1a1f2e     // Main background
--pico-card-background-color: #242c3d  // Cards/panels
--pico-card-separator-color: #334155   // Borders/dividers
```

**Text Colors:**
```css
--pico-color: #e2e8f0           // Primary text (high contrast)
--pico-muted-color: #94a3b8     // Secondary text
```

**Accent Colors:**
```css
--pico-primary: #60a5fa          // Blue - primary actions
--pico-success: #22c55e          // Green - success states
--pico-warning: #f59e0b          // Orange - warnings
--pico-error: #ef4444            // Red - errors, rejected
```

**Status Badge Colors:**

| Status     | Background              | Border                  | Text      |
|------------|-------------------------|-------------------------|-----------|
| RAW        | rgba(245, 158, 11, 0.2) | rgba(245, 158, 11, 0.3) | #f59e0b   |
| ANALYZED   | rgba(96, 165, 250, 0.2) | rgba(96, 165, 250, 0.3) | #60a5fa   |
| INTERESTED | rgba(34, 197, 94, 0.2)  | rgba(34, 197, 94, 0.3)  | #22c55e   |
| REJECTED   | rgba(239, 68, 68, 0.2)  | rgba(239, 68, 68, 0.3)  | #ef4444   |
| TAILORED   | rgba(168, 85, 247, 0.2) | rgba(168, 85, 247, 0.3) | #a855f7   |
| SENT       | rgba(6, 182, 212, 0.2)  | rgba(6, 182, 212, 0.3)  | #06b6d4   |
| RESPONDED  | rgba(236, 72, 153, 0.2) | rgba(236, 72, 153, 0.3) | #ec4899   |

### Layout

- **Sidebar Width**: 16rem (256px)
- **Card Padding**: 1.5rem
- **Main Content Padding**: 1.5rem
- **Border Radius**: 0.5rem (8px)

### Typography

| Level | Size            | Weight | Usage         |
|-------|-----------------|--------|---------------|
| h1    | 1.875rem (30px) | 700    | Page titles   |
| h2    | 1.5rem (24px)   | 600    | Section heads |
| h3    | 1.25rem (20px)  | 600    | Card titles   |
| small | 0.875rem (14px) | 400    | Secondary     |

See `docs/design-system.md` for full design system reference.

---

## LLM Integration

### Configuration

The Analyzer uses an OpenAI-compatible LLM client (`sashabaranov/go-openai`).

```env
# LLM settings
LLM_BASE_URL=http://localhost:1234/v1  # LM Studio, Ollama, OpenAI
LLM_MODEL=gpt-4o-mini                  # Model name
LLM_API_KEY=                           # Empty for local models
LLM_MAX_TOKENS=2048
LLM_TEMPERATURE=0.1
LLM_TIMEOUT_SECONDS=60
```

### Prompt Templates

Prompts are stored in XML format in `docs/prompts/`:

- `job-extraction.xml` - Extract structured data from job postings
- `resume-tailoring.xml` - Adapt resume to job requirements
- `cover-letter.xml` - Generate cover letters (3 templates)

### Extraction Output Schema

```json
{
  "title": string | null,
  "description": string | null,
  "salary_min": number | null,
  "salary_max": number | null,
  "currency": "RUB" | "USD" | "EUR" | null,
  "location": string | null,
  "is_remote": boolean,
  "language": "RU" | "EN",
  "technologies": string[],
  "experience_years": number | null,
  "company": string | null,
  "contacts": string[]
}
```

---

## Telegram Integration

### Why MTProto (not Bot API)?

| Criterion       | Bot API               | MTProto (Userbot)    |
|-----------------|-----------------------|----------------------|
| Read channels   | Only if bot is admin  | Any public channel   |
| Read groups     | Only if bot added     | Any public group     |
| Send DMs        | Only if user started  | Any user*            |
| History         | Limited               | Full                 |
| Rate limits     | Strict                | More flexible        |

### Getting Credentials

1. Go to [my.telegram.org](https://my.telegram.org)
2. Sign in with phone number
3. Select "API development tools"
4. Create an app (any name works)
5. Save `api_id` and `api_hash` to `.env`

### Session Management

Generate session string once, reuse forever:

```bash
task tg-auth
# Follow interactive prompts
# Copy session string to .env
```

**Security Rules:**
- Never commit session strings (only in `.env`)
- Use a dedicated Telegram account (not your main)
- Don't run from multiple devices simultaneously

### Rate Limits

| Action                | Limit        |
|-----------------------|--------------|
| Messages (different)  | ~30/sec      |
| Messages (same chat)  | ~1/sec       |
| Messages (group)      | ~20/min      |
| GetMessages (history) | ~300-500 req |
| ResolveUsername       | ~50/min      |

FloodWait errors are handled automatically with exponential backoff.

See `docs/telegram-integration.md` for complete Telegram integration details.

---

## Database Schema

### Tables

| Migration | Table                | Description                        |
|-----------|---------------------|------------------------------------|
| 0001      | `scraping_targets`  | Source configuration               |
| 0002      | `jobs`              | Job postings with structured data   |
| 0003      | `job_applications`  | Application tracking               |
| 0004      | (triggers)          | `updated_at` auto-update           |
| 0005      | `parsed_ranges`     | Track last scraped message IDs     |

### Jobs Table Columns

Key columns:
- `id` - UUID primary key
- `external_id` - Telegram message ID (deduplication)
- `status` - ENUM: RAW, ANALYZED, INTERESTED, REJECTED, TAILORED, SENT, RESPONDED
- `raw_content` - Original message text
- `structured_data` - JSONB from LLM extraction
- `source_channel` - Origin channel username

---

## NATS Event Flow

### Subjects

| Subject         | Publisher | Subscriber  | Payload              |
|-----------------|-----------|-------------|----------------------|
| `jobs.new`      | Collector | Analyzer    | `{job_id}`           |
| `jobs.analyzed` | Analyzer  | Web (WS)    | `{job_id, data}`     |
| `brain.prepare` | Web UI    | Brain       | `{job_id}`           |
| `jobs.prepared` | Brain     | Web (WS)    | `{job_id, docs}`     |

### Why Only job_id in NATS?

Passing only the `job_id` keeps messages small and ensures the analyzer fetches fresh data from the database (single source of truth).

---

## CLI Tools

### tg-auth

Generate Telegram session string for authentication.

```bash
task tg-auth
# Interactive: enter phone, code from Telegram
# Outputs: TG_SESSION_STRING for .env
```

### tg-topics

List forum topics for a Telegram channel/group.

```bash
task tg-topics channel=@forum_name
# Shows: topic ID, title, color
```

---

## WebSocket Events

The web UI uses WebSocket for real-time updates. Connect at `ws://localhost:3100/ws`.

### Event Types

**Scraping Events:**
```json
{"type": "scrape.started", "target": "@golang_jobs", "limit": 100}
{"type": "scrape.progress", "target": "@golang_jobs", "processed": 45, "new_jobs": 12}
{"type": "scrape.completed", "target": "@golang_jobs", "total": 100, "new": 23}
{"type": "scrape.failed", "target": "@golang_jobs", "error": "..."}
{"type": "scrape.cancelled", "target": "@golang_jobs"}
```

**Job Events:**
```json
{"type": "job.new", "job_id": "uuid", "title": "Go Developer", "company": "Yandex"}
{"type": "job.analyzed", "job_id": "uuid", "technologies": ["go", "postgres"]}
{"type": "job.updated", "job_id": "uuid", "status": "INTERESTED"}
```

**Target Events:**
```json
{"type": "target.created", "target_id": "uuid"}
{"type": "target.updated", "target_id": "uuid"}
{"type": "target.deleted", "target_id": "uuid"}
```

**Auth Events:**
```json
{"type": "tg_qr", "url": "tg://login?token=..."}
{"type": "tg_auth_success"}
```

**System Events:**
```json
{"type": "stats.updated"}
{"type": "error", "message": "..."}
```

---

## REST API Endpoints

### Authentication

```
GET  /api/v1/auth/status
POST /api/v1/auth/qr
```

### Scraping

```
POST /api/v1/scrape/telegram
{
  "channel": "@go_jobs",
  "limit": 100,
  "until": "2025-01-01",
  "topic_ids": [1, 2, 3]
}

DELETE /api/v1/scrape/current
GET    /api/v1/scrape/status
```

### Jobs

```
GET  /api/v1/jobs?page=1&limit=50&status=ANALYZED&tech=go&q=remote&salary_min=100000
GET  /api/v1/jobs/{id}
PATCH /api/v1/jobs/{id}/status
{
  "status": "INTERESTED"
}
```

### Targets

```
GET    /api/v1/targets
POST   /api/v1/targets
GET    /api/v1/targets/{id}
PUT    /api/v1/targets/{id}
DELETE /api/v1/targets/{id}
```

### Stats

```
GET /api/v1/stats
```

### Brain (Phase 4 - Planned)

```
POST /api/v1/jobs/{id}/prepare
GET  /api/v1/jobs/{id}/documents
GET  /api/v1/jobs/{id}/documents/resume.pdf
GET  /api/v1/jobs/{id}/documents/cover.pdf
```

---

## File Structure Reference

```
positions-os/
├── cmd/                    # Service entry points
│   ├── analyzer/           # LLM analysis worker
│   ├── collector/          # Main service (backend API + serves React app)
│   ├── tg-auth/            # Telegram session generator
│   ├── tg-topics/          # Forum topics lister
│   └── validate-yaml/      # YAML validator
├── internal/               # Private packages
│   ├── analyzer/           # LLM processing logic
│   ├── collector/          # Scraping orchestration
│   ├── config/             # Environment configuration
│   ├── database/           # Connection management
│   ├── llm/                # OpenAI-compatible client
│   ├── logger/             # Structured logging
│   ├── models/             # Domain entities
│   ├── nats/               # Message broker client
│   ├── publisher/          # Event publishing
│   ├── repository/         # Database CRUD
│   ├── telegram/           # MTProto wrapper
│   ├── web/                # HTTP server, WebSocket hub
│   └── web/handlers/       # API endpoint handlers
├── frontend/               # React TypeScript SPA
│   ├── src/
│   │   ├── components/     # React components
│   │   ├── pages/          # Page components
│   │   ├── hooks/          # Custom React hooks
│   │   ├── lib/            # API client, types, utilities
│   │   └── styles/         # CSS stylesheets
│   ├── e2e/                # Playwright E2E tests
│   ├── vite.config.ts      # Vite configuration
│   ├── playwright.config.ts # Playwright configuration
│   └── package.json        # Frontend dependencies
├── migrations/             # Database schema migrations
├── static/                 # Static assets
│   └── dist/               # Built React app (served by backend)
├── storage/                # Resume storage (Phase 4)
│   ├── resume.md           # Base resume
│   └── outputs/            # Generated documents
├── docs/                   # Project documentation
│   ├── prompts/            # LLM prompt templates (XML)
│   └── index/              # Code index (links to detailed docs)
├── Dockerfile              # Collector service Docker image
├── Dockerfile.analyzer     # Analyzer service Docker image
├── Dockerfile.frontend     # Frontend production Docker image
├── docker-compose.yml      # Multi-service Docker setup
└── Taskfile.yml            # Task runner configuration
```

---

## Development Principles

1. **Working code > perfect code** - Make it work, then refactor
2. **One source first, add more later** - Telegram works perfectly before adding other sources
3. **API-first approach** - All actions through REST, UI uses same endpoints
4. **Docker Compose everywhere** - No "works on my machine"
5. **Dark mode only** - Optimize for dark theme, no light mode needed
6. **Local LLM for filtering, powerful LLM for generation** - Cost efficiency
7. **Type safety** - TypeScript on frontend, strict typing on backend

---

## Frontend Tech Stack Details

### Core Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| React | 19.2.0 | UI framework |
| React Router DOM | 7.12.0 | Client-side routing |
| @tanstack/react-query | 5.90.17 | Server state management |
| @picocss/pico | 2.1.1 | Minimal CSS framework |
| qrcode.react | 4.2.0 | QR code generation for Telegram auth |

### Dev Dependencies

| Package | Purpose |
|---------|---------|
| Vite | Fast build tool and dev server |
| TypeScript | Type safety |
| ESLint | Code linting |
| Vitest | Unit test runner |
| @playwright/test | E2E test framework |
| Testing Library | React component testing utilities |

### Build Configuration

- **Build output**: `../static/dist` (Go backend serves this)
- **Dev proxy**: `/api` → `http://localhost:3100`, `/ws` → `ws://localhost:3100`
- **Path alias**: `@/` → `src/`
- **TypeScript**: Strict mode with separate configs for app and build tools

---

## Further Documentation

| Document                   | Description                              |
|---------------------------|------------------------------------------|
| `docs/implementation-order.md` | Detailed phase breakdown (Russian)   |
| `docs/design-system.md`    | Full design system specification         |
| `docs/environment-variables.md` | Complete environment reference      |
| `docs/telegram-integration.md` | Telegram technical specification     |
| `docs/phase-2-analyzer.md`  | Analyzer implementation details        |
| `docs/phase-3-webui.md`     | Web UI implementation details         |
| `docs/phase-4-brain.md`     | Brain/resume tailoring plan            |
| `docs/index/`               | Code documentation index                |
