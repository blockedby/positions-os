# E2E Test Execution Order

## Why Tests Hang

The `global-setup.ts` waits up to 30 seconds for backend (`http://localhost:3100/api/v1/stats`) to respond. Without running backend, tests will hang until timeout.

## Prerequisites (MUST be running first)

```bash
# 1. Start infrastructure
docker compose up -d postgres nats

# 2. Wait for healthy status
docker compose ps  # both should be "healthy"

# 3. Start backend in separate terminal
export DATABASE_URL="postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable"
export NATS_URL="nats://localhost:4222"
export HTTP_PORT=3100
go run ./cmd/collector/main.go

# 4. Verify backend responds
curl http://localhost:3100/api/v1/stats
```

## Test Files (add one at a time)

### Order of Addition

1. **api.spec.ts** - Pure API tests, no WebSocket
   - Tests REST endpoints
   - No UI interaction
   - Fastest to debug

2. **targets.spec.ts** - CRUD operations
   - Creates/reads/updates/deletes targets
   - Uses API + some UI
   - Medium complexity

3. **websocket.spec.ts** - WebSocket stability
   - Tests real-time connection
   - Most complex, add last
   - Tests multiple connections issue

## How to Run Individual Test Files

```bash
cd frontend

# Run only API tests
BASE_URL=http://localhost:3100 bunx playwright test api.spec.ts

# Run only targets tests
BASE_URL=http://localhost:3100 bunx playwright test targets.spec.ts

# Run only WebSocket tests
BASE_URL=http://localhost:3100 bunx playwright test websocket.spec.ts

# Run all
BASE_URL=http://localhost:3100 bunx playwright test
```

## Debugging Hangs

1. Check backend: `curl http://localhost:3100/api/v1/stats`
2. Check docker: `docker compose ps`
3. Run with debug: `PWDEBUG=1 bunx playwright test api.spec.ts`
4. Check console: `bunx playwright test --reporter=line`
