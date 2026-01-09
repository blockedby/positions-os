# Positions OS

Automated job search system: vacancy scraping, AI analysis, and application automation.

## Quick Start

### Requirements

- Docker & Docker Compose
- Go 1.21+
- Telegram API credentials (get from https://my.telegram.org)

### Setup

1. **Prepare Environment**:

   ```powershell
   copy .env.example .env
   # Fill TG_API_ID and TG_API_HASH from https://my.telegram.org
   ```

2. **Generate Session**:

   ```powershell
   go run cmd/tg-auth/main.go
   # Follow prompts. For session, you can use TDesktop (if installed) or SMS.
   # Copy the result string to TG_SESSION_STRING in .env
   ```

3. **Start Infrastructure**:

   ```powershell
   docker compose up -d
   # Apply migrations (requires migrate tool or use docker profile if configured in Makefile)
   # For Windows without 'make', you can run:
   docker compose --profile tools run --rm migrate
   ```

4. **Launch Service**:
   ```powershell
   go run cmd/collector/main.go
   ```

## Project Structure

```
positions-os/
├── cmd/                       # service entry points
│   ├── tg-auth/              # telegram authentication cli tool
│   ├── tg-topics/            # telegram forum topics lister
│   └── collector/            # collector service (phase 1)
├── internal/                  # internal packages
│   ├── config/               # configuration
│   ├── database/             # postgresql client
│   ├── logger/               # structured logging
│   ├── models/               # data models
│   ├── nats/                 # nats pub/sub client
│   ├── telegram/             # telegram api client
│   ├── repository/           # data access layer
│   └── collector/            # collector business logic
├── migrations/                # sql database migrations
├── docs/                      # documentation
└── docker-compose.yml         # infrastructure setup
```

## Scripts

- **Auth**: `go run cmd/tg-auth/main.go` — Generate Telegram session string.
- **Topics**: `go run cmd/tg-topics/main.go @channel` — List forum topics.
- **Collector**: `go run cmd/collector/main.go` — Start service locally.
- **Infrastructure**: `docker compose up -d` — Start DB and NATS.
- **Migrations**: `make migrate-up` — Apply database schema.
- **Tests**: `go test ./...` — Run all tests.

## AI Prompts

- [Chain of Thoughts](docs/prompts/chain-of-thoughts.xml) — Reasoning guidelines.
- [Job Extraction](docs/prompts/job-extraction.xml) — Data extraction schema.

## Services

| Service       | Port | Description                |
| ------------- | ---- | -------------------------- |
| PostgreSQL    | 5432 | Main database              |
| NATS          | 4222 | Message broker             |
| NATS Monitor  | 8222 | NATS monitoring            |
| Collector API | 3100 | Scraping service (Phase 1) |

## Commands (PowerShell on Windows)

```powershell
# install dependencies
go mod tidy

# run migrations (requires migrate cli)
migrate -path migrations -database "postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable" up

# build all services
go build -o bin/ ./cmd/...

# run tests
go test -v ./...

# format code
go fmt ./...
```

## API Endpoints (Phase 1: Collector)

### Scraping Control

```bash
# start scraping a channel
POST /api/v1/scrape/telegram
{
  "channel": "@golang_jobs",
  "limit": 100,
  "topic_ids": [1, 15, 28]  # optional, for forums only
}

# stop current scraping task
DELETE /api/v1/scrape/current

# get scraping status
GET /api/v1/scrape/status

# health check
GET /health
```

### Target Management

```bash
# list all scraping targets
GET /api/v1/targets

# create new target
POST /api/v1/targets
{
  "name": "Go Jobs",
  "type": "TG_CHANNEL",
  "url": "@golang_jobs"
}
```

## Documentation

- [Implementation Plan](docs/implementation-order.md)
- [Phase 0: Infrastructure](docs/phase-0-infrastructure.md)
- [Phase 1: Collector](docs/phase-1-collector.md)

## Environment Variables

See `.env.example` for all available configuration options.

Key variables:

- `TG_API_ID` - Telegram API ID (from https://my.telegram.org)
- `TG_API_HASH` - Telegram API Hash
- `TG_SESSION_STRING` - Generated via `tg-auth` tool
- `DATABASE_URL` - PostgreSQL connection string
- `NATS_URL` - NATS server URL

## Development Phases

- [x] **Phase 0**: Infrastructure (PostgreSQL, NATS, migrations)
- [x] **Phase 1**: Collector (Telegram scraping, REST API)
- [ ] **Phase 2**: Analyzer (LLM-based job analysis)
- [ ] **Phase 3**: Brain (Resume tailoring, application automation)
- [ ] **Phase 4**: Web UI (User interface)

## License

Private
