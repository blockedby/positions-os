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
   # Start core infra (Postgres, NATS)
   docker compose up -d

   # OR start full application (including Collector and Analyzer)
   docker compose --profile app up -d
   ```

   **Apply migrations**:

   ```powershell
   docker compose --profile tools run --rm migrate
   ```

4. **Launch Unified Service**:
   The collector now serves both the Scraping API and the Web UI.

   ```powershell
   go run cmd/collector/main.go
   ```

5. **Access Web UI**:
   Open your browser at [http://localhost:3100](http://localhost:3100)

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

### 1. Authentication

Generate a Telegram session string required for `.env`:

```powershell
go run cmd/tg-auth/main.go
```

_Follow the interactive prompts (Option 2 for SMS is recommended)._

### 2. Inspect Forum Topics

If you need to scrape specific sub-chats (topics) from a supergroup:

```powershell
go run cmd/tg-topics/main.go @some_forum_username
```

_This will output a list of topics and their IDs (e.g., `id: 15`). Use these IDs in your scrape request._

### 3. Run Collector

Start the collector service locally:

```powershell
go run cmd/collector/main.go
```

### 4. Tests

Run integration and unit tests:

```powershell
go test ./...
```

## AI Prompts

- [Chain of Thoughts](docs/prompts/chain-of-thoughts.xml) — Reasoning guidelines.
- [Job Extraction](docs/prompts/job-extraction.xml) — Data extraction schema.

## Services

| Service          | Port | Description                      |
| ---------------- | ---- | -------------------------------- |
| PostgreSQL       | 5432 | Main database                    |
| NATS             | 4222 | Message broker                   |
| NATS Monitor     | 8222 | NATS monitoring                  |
| **Web UI & API** | 3100 | Dashboard and Scraping (Unified) |
| Analyzer         | -    | Background AI Analysis service   |

```powershell
# install dependencies
go mod tidy

# run migrations (via Docker if migrate CLI is not installed)
docker compose --profile tools run --rm migrate

# start core infrastructure
docker compose up -d

# start full app (Collector + Analyzer)
docker compose --profile app up -d

# build all services
go build -o bin/ ./cmd/...

# run tests
go test -v ./...
```

### Windows-Specific Notes

- **LM Studio**: If using LM Studio on the host, ensure "CORS" is allowed or set `LLM_BASE_URL=http://localhost:1234/v1`. For Docker compatibility, use `http://host.docker.internal:1234/v1`.
- **Make on Windows**: If you don't have `make`, use the direct `docker compose` or `go run` commands shown above.
- **Log Files**: Logs are stored in `./logs/`. Ensure the directory exists or the service has permissions to create it.

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

## Limits & Rate Limiting

### Telegram API Limits

- **Messages per batch**: 100 (Telegram API maximum)
- **Rate limit delay**: 100ms between batches (auto-applied)
- **FloodWait handling**: Automatic backoff when Telegram returns FLOOD_WAIT errors

### Application Safety Limits

- **Maximum batches per scrape**: 100 (prevents infinite loops, ~ 10,000 messages max)
- **Duplicate offset detection**: Scrape exits if offset doesn't change between batches
- **Context timeout**: Scrape jobs run in background with cancellable contexts

### Configurable Limits

| Parameter     | Request Field | Default | Description                                        |
| ------------- | ------------- | ------- | -------------------------------------------------- |
| Message limit | `limit`       | 100     | Maximum messages to fetch                          |
| Until date    | `until`       | -       | Stop at messages older than this date (YYYY-MM-DD) |

### Recommendations

- Use `"limit": 10-50` for testing new channels
- Use `"limit": 100-500` for regular scraping
- Avoid unlimited scrapes on channels with 1000+ messages

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

For a complete guide on how data flows through the system and detailed descriptions of all variables, see **[Environment Variables Reference](docs/environment-variables.md)**.

Key setup requirements:

- `TG_API_ID` - Telegram API ID (from https://my.telegram.org)
- `TG_API_HASH` - Telegram API Hash
- `TG_SESSION_STRING` - Generated via `tg-auth` tool
- `DATABASE_URL` - PostgreSQL connection string
- `NATS_URL` - NATS server URL
- `LLM_BASE_URL` - LLM API endpoint (e.g. `http://localhost:1234/v1` or `https://api.openai.com/v1`)
- `LLM_API_KEY` - API Key for LLM service
- `LLM_MODEL` - Model name (e.g. `gpt-4o-mini`, `qwen2.5-coder-7b-instruct`)

## Development Phases

- [x] **Phase 0**: Infrastructure (PostgreSQL, NATS, migrations)
- [x] **Phase 1**: Collector (Telegram scraping, REST API)
- [x] **Phase 2**: Analyzer (LLM-based job analysis)
- [x] **Phase 3**: Web UI (User interface, Unified Service)
- [ ] **Phase 4**: Brain (Resume tailoring, PDF generation)

## License

Private
