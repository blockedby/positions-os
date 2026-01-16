# How to Run E2E Tests

## Prerequisites

1. Docker installed
2. Go 1.21+ installed
3. Bun installed

## Step-by-Step

### 1. Start Infrastructure

```bash
docker compose up -d postgres nats
```

Wait for services to be healthy:
```bash
docker compose ps
# Both should show "healthy"
```

### 2. Run Migrations

```bash
docker compose --profile tools run --rm migrate
```

### 3. Start Backend (Collector)

Terminal 1:
```bash
export DATABASE_URL="postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable"
export NATS_URL="nats://localhost:4222"
export HTTP_PORT=3100
go run ./cmd/collector/main.go
```

Verify backend is running:
```bash
curl http://localhost:3100/api/v1/stats
# Should return JSON
```

### 4. Run E2E Tests

Terminal 2:
```bash
cd frontend
BASE_URL=http://localhost:3100 bunx playwright test
```

## One-Liner (if backend already running)

```bash
cd frontend && BASE_URL=http://localhost:3100 bunx playwright test
```

## Common Issues

1. **"Backend did not start in time"** - Backend not running on port 3100
2. **Empty output** - Check if playwright is installed: `cd frontend && bunx playwright install chromium`
3. **Timeout** - Increase timeout in playwright.config.ts

## Cleanup

```bash
docker compose down
```
