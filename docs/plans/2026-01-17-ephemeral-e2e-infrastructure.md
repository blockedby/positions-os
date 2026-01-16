# Ephemeral E2E Test Infrastructure Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create isolated, ephemeral Docker infrastructure for E2E tests with automatic cleanup - no test data persists between runs.

**Architecture:** Separate `docker-compose.e2e.yml` with its own network and ephemeral containers. PostgreSQL uses `tmpfs` (RAM disk) so data is automatically destroyed when containers stop. Each test run starts with a fresh database, migrations run automatically, and everything is torn down after tests complete.

**Tech Stack:** Docker Compose, PostgreSQL (tmpfs), NATS, Playwright

---

## How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│  E2E Test Network (jhos-e2e-network) - ISOLATED                │
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│  │  postgres   │    │    nats     │    │  collector  │        │
│  │   (tmpfs)   │◄───│             │◄───│             │        │
│  │  RAM-based  │    │  ephemeral  │    │  ephemeral  │        │
│  │  auto-wipe  │    │             │    │             │        │
│  └─────────────┘    └─────────────┘    └──────┬──────┘        │
│                                               │                │
│  ┌─────────────┐                      ┌──────▼──────┐        │
│  │   e2e-run   │─────────────────────►│  frontend   │        │
│  │  Playwright │  BASE_URL            │   (nginx)   │        │
│  │   tests     │                      │             │        │
│  └─────────────┘                      └─────────────┘        │
│                                                                 │
│  On stop: ALL DATA DESTROYED (tmpfs wiped, containers removed) │
└─────────────────────────────────────────────────────────────────┘
```

**Benefits:**
- **Perfect isolation**: E2E tests never touch dev/prod database
- **Auto-cleanup**: RAM disk (tmpfs) is wiped when container stops
- **Fresh migrations**: Schema always matches latest migration files
- **Fast**: tmpfs is faster than disk-based PostgreSQL
- **Reproducible**: Same starting state every run

---

### Task 1: Create docker-compose.e2e.yml

**Files:**
- Create: `docker-compose.e2e.yml`

**Step 1: Create the E2E compose file**

```yaml
# docker-compose.e2e.yml
# ============================================================
# Ephemeral E2E test infrastructure
# Usage: docker compose -f docker-compose.e2e.yml up --abort-on-container-exit
# ============================================================

services:
  # ------------------------------------------------------------
  # Ephemeral PostgreSQL with RAM-based storage
  # ------------------------------------------------------------
  postgres:
    image: postgres:16-alpine
    container_name: jhos-e2e-postgres
    environment:
      POSTGRES_USER: e2e_user
      POSTGRES_PASSWORD: e2e_pass
      POSTGRES_DB: e2e_db
    tmpfs:
      - /var/lib/postgresql/data:size=256M
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U e2e_user -d e2e_db"]
      interval: 2s
      timeout: 2s
      retries: 10

  # ------------------------------------------------------------
  # Ephemeral NATS (no persistence needed for tests)
  # ------------------------------------------------------------
  nats:
    image: nats:2.10-alpine
    container_name: jhos-e2e-nats
    command: ["--jetstream"]
    healthcheck:
      test: ["CMD", "nats-server", "--help"]
      interval: 2s
      timeout: 2s
      retries: 5

  # ------------------------------------------------------------
  # Collector (Go backend) - runs migrations on startup
  # ------------------------------------------------------------
  collector:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: jhos-e2e-collector
    depends_on:
      postgres:
        condition: service_healthy
      nats:
        condition: service_healthy
    environment:
      DATABASE_URL: postgres://e2e_user:e2e_pass@postgres:5432/e2e_db?sslmode=disable
      NATS_URL: nats://nats:4222
      HTTP_PORT: 3100
      LOG_LEVEL: warn
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://127.0.0.1:3100/api/v1/stats"]
      interval: 2s
      timeout: 2s
      retries: 15

  # ------------------------------------------------------------
  # Frontend (nginx serving React SPA)
  # ------------------------------------------------------------
  frontend:
    build:
      context: .
      dockerfile: Dockerfile.frontend
    container_name: jhos-e2e-frontend
    depends_on:
      collector:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://127.0.0.1/"]
      interval: 2s
      timeout: 2s
      retries: 10

  # ------------------------------------------------------------
  # E2E Test Runner (Playwright)
  # ------------------------------------------------------------
  e2e-runner:
    build:
      context: ./frontend
      dockerfile: Dockerfile.test
    container_name: jhos-e2e-runner
    command: ["bun", "playwright", "test", "--reporter=list"]
    environment:
      BASE_URL: http://frontend:80
      API_BASE_URL: http://collector:3100
      # Skip backend check in global-setup (we use depends_on)
      SKIP_BACKEND_CHECK: "1"
    depends_on:
      frontend:
        condition: service_healthy
    volumes:
      # Mount test results for viewing after run
      - ./frontend/test-results:/app/test-results
      - ./frontend/playwright-report:/app/playwright-report

networks:
  default:
    name: jhos-e2e-network
```

**Step 2: Verify file syntax**

Run:
```bash
docker compose -f docker-compose.e2e.yml config
```

Expected: YAML parsed successfully, services listed.

**Step 3: Commit**

```bash
git add docker-compose.e2e.yml
git commit -m "feat(e2e): add ephemeral test infrastructure

- Separate docker-compose for E2E tests
- PostgreSQL with tmpfs (RAM disk) for auto-cleanup
- Isolated network (jhos-e2e-network)
- All containers ephemeral, destroyed after tests"
```

---

### Task 2: Add Convenience Scripts

**Files:**
- Create: `scripts/e2e.sh`

**Step 1: Create the E2E runner script**

```bash
#!/bin/bash
# scripts/e2e.sh - Run E2E tests with ephemeral infrastructure

set -e

echo "🧪 Starting E2E tests with ephemeral infrastructure..."

# Build and run, exit when e2e-runner completes
docker compose -f docker-compose.e2e.yml up \
  --build \
  --abort-on-container-exit \
  --exit-code-from e2e-runner

EXIT_CODE=$?

echo "🧹 Cleaning up containers..."
docker compose -f docker-compose.e2e.yml down --volumes --remove-orphans

if [ $EXIT_CODE -eq 0 ]; then
  echo "✅ E2E tests passed!"
else
  echo "❌ E2E tests failed with exit code $EXIT_CODE"
  echo "📁 Check frontend/test-results/ for screenshots"
fi

exit $EXIT_CODE
```

**Step 2: Make script executable**

Run:
```bash
chmod +x scripts/e2e.sh
```

**Step 3: Test the script**

Run:
```bash
./scripts/e2e.sh
```

Expected: All containers start, tests run, containers destroyed, exit code 0.

**Step 4: Commit**

```bash
git add scripts/e2e.sh
git commit -m "feat(e2e): add convenience script for running E2E tests

Usage: ./scripts/e2e.sh
- Builds and starts ephemeral infrastructure
- Runs Playwright tests
- Cleans up all containers on completion"
```

---

### Task 3: Add Taskfile Entry

**Files:**
- Modify: `Taskfile.yml`

**Step 1: Find the testing section in Taskfile**

Look for existing e2e task around line with `e2e:` or similar.

**Step 2: Add/update e2e-docker task**

Add this task:

```yaml
  e2e-docker:
    desc: Run E2E tests with ephemeral Docker infrastructure
    cmds:
      - ./scripts/e2e.sh
```

**Step 3: Verify task works**

Run:
```bash
task e2e-docker
```

Expected: Same as running `./scripts/e2e.sh` directly.

**Step 4: Commit**

```bash
git add Taskfile.yml
git commit -m "feat(e2e): add task e2e-docker for ephemeral E2E tests"
```

---

### Task 4: Update global-setup.ts to Support SKIP_BACKEND_CHECK

**Files:**
- Modify: `frontend/e2e/global-setup.ts:6-10`

**Step 1: Verify SKIP_BACKEND_CHECK is already supported**

Current code should have:
```typescript
if (process.env.SKIP_BACKEND_CHECK) {
  console.log('Global Setup: Skipping backend check (SKIP_BACKEND_CHECK=1)')
  return
}
```

If present, this task is complete. If not, add it after line 5.

**Step 2: No commit needed if already present**

---

### Task 5: Update Documentation

**Files:**
- Modify: `CLAUDE.md` (testing section)

**Step 1: Find testing commands section**

Look for section with `task e2e` or similar.

**Step 2: Add ephemeral E2E documentation**

Add after existing E2E commands:

```markdown
# Ephemeral E2E Tests (Recommended for CI)
task e2e-docker           # Run E2E with isolated, ephemeral containers
./scripts/e2e.sh          # Direct script (same as above)

# Benefits:
# - Fresh database every run (no cleanup needed)
# - Isolated network (doesn't touch dev DB)
# - Auto-cleanup on completion
```

**Step 3: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: add ephemeral E2E test documentation"
```

---

### Task 6: Remove Old E2E Service from Main docker-compose.yml (Optional)
no, keep it for now
**Files:**
- Modify: `docker-compose.yml:163-179`

**Step 1: Decide whether to keep or remove**

Options:
- **Keep**: Developers can still run `docker-compose up frontend-e2e-tests` for quick iteration
- **Remove**: Single source of truth for E2E tests

Recommendation: Keep for now, add deprecation comment. -- yes

**Step 2: Add deprecation notice**

```yaml
  # DEPRECATED: Use docker-compose.e2e.yml for ephemeral E2E tests
  # This service shares the dev database and may leave orphaned test data.
  # Usage: docker compose -f docker-compose.e2e.yml up --abort-on-container-exit
  frontend-e2e-tests:
    ...
```

**Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "docs(docker): add deprecation notice for shared-db E2E tests

Recommend using docker-compose.e2e.yml for isolated E2E tests."
```

---

### Task 7: Final Verification

**Step 1: Run full E2E suite**

```bash
task e2e-docker
```

Expected:
```
🧪 Starting E2E tests with ephemeral infrastructure...
[+] Building ...
[+] Running ...
Running 22 tests using 4 workers
✓ ... (all 22 tests)
22 passed
🧹 Cleaning up containers...
✅ E2E tests passed!
```

**Step 2: Verify cleanup**

```bash
docker ps -a --filter "name=jhos-e2e"
```

Expected: No containers listed (all removed).

**Step 3: Verify dev database untouched**

```bash
docker exec jhos-postgres psql -U jhos -d jhos -c "SELECT COUNT(*) FROM scraping_targets;"
```

Expected: Same count as before E2E run (tests used separate DB).

---

## Verification Checklist

- [ ] `docker compose -f docker-compose.e2e.yml config` validates successfully
- [ ] `./scripts/e2e.sh` runs all 22 tests
- [ ] All containers are removed after tests complete
- [ ] Dev database (`jhos-postgres`) is not modified by E2E tests
- [ ] Test results are available in `frontend/test-results/`
- [ ] `task e2e-docker` works as expected

## Rollback Plan

If issues arise, E2E tests can still run with the original approach:

```bash
docker-compose up frontend-e2e-tests
```

The original service remains in `docker-compose.yml` with a deprecation notice.
