# Fix Docker E2E Test Infrastructure Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the Docker E2E test infrastructure so all 22 Playwright tests pass.

**Architecture:** The E2E tests run in a Docker container and load pages via Playwright. Currently tests point to the Go backend (`collector:3100`) which doesn't serve static files. The fix is to point tests to the nginx frontend container (`frontend:80`) which serves the React SPA and proxies API/WebSocket requests to the collector.

**Tech Stack:** Docker Compose, Playwright, nginx, Go

---

## Root Cause Analysis

**Issues Found:**
1. ✅ FIXED: Health check uses `localhost` → IPv6 `::1` → connection refused (changed to `127.0.0.1`)
2. ✅ FIXED: `API_BASE_URL` not set for direct API tests (added to docker-compose.yml)
3. ❌ PENDING: `BASE_URL=http://collector:3100` but collector doesn't serve static files

**Current State:**
- 14 tests pass (all API tests + some WebSocket tests)
- 8 tests fail (UI tests can't load pages because collector returns 404)

---

### Task 1: Update E2E Test BASE_URL to Use Frontend Container

**Files:**
- Modify: `docker-compose.yml:168-170`

**Step 1: Change BASE_URL from collector to frontend**

The frontend nginx container serves the React SPA at port 80 and proxies `/api/` and `/ws` to the collector. Tests should use this entry point.

```yaml
# In docker-compose.yml, change:
    environment:
      BASE_URL: http://collector:3100
      API_BASE_URL: http://collector:3100

# To:
    environment:
      BASE_URL: http://frontend:80
      API_BASE_URL: http://collector:3100
```

**Step 2: Run E2E tests to verify**

Run: `docker-compose up frontend-e2e-tests`

Expected: More tests should pass now that pages can load from nginx.

**Step 3: Commit**

```bash
git add docker-compose.yml
git commit -m "fix(e2e): point BASE_URL to frontend nginx container

The collector container doesn't serve static files in Docker.
Tests should use the frontend nginx container which serves the
React SPA and proxies API/WS requests to the collector."
```

---

### Task 2: Fix Dockerfile.frontend Health Check IPv6 Issue

**Files:**
- Modify: `Dockerfile.frontend:40-41`

**Step 1: Check if health check has IPv6 issue**

The Dockerfile.frontend health check uses `localhost` which may resolve to IPv6:

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1
```

Note: This is already overridden in docker-compose.yml, so this fix is for standalone Docker builds.

**Step 2: Update health check to use 127.0.0.1**

```dockerfile
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://127.0.0.1/ || exit 1
```

**Step 3: Commit**

```bash
git add Dockerfile.frontend
git commit -m "fix(docker): use 127.0.0.1 in health check to avoid IPv6 issues

Alpine wget resolves localhost to IPv6 ::1 but nginx only listens
on IPv4. Using explicit 127.0.0.1 ensures the health check works."
```

---

### Task 3: Verify All E2E Tests Pass

**Files:**
- None (verification only)

**Step 1: Rebuild and run full E2E suite**

```bash
docker-compose build frontend frontend-e2e-tests
docker-compose up -d postgres nats collector frontend
docker-compose up frontend-e2e-tests
```

**Step 2: Verify results**

Expected output:
```
Running 22 tests using 4 workers
...
22 passed
```

**Step 3: If tests still fail, debug**

If UI tests still fail:
1. Check screenshot artifacts in `test-results/` directory
2. Verify frontend can reach collector: `docker exec jhos-frontend curl http://collector:3100/api/v1/stats`
3. Check browser console errors in error-context.md files

---

### Task 4: Clean Up and Final Commit

**Files:**
- None (already committed)

**Step 1: Verify git status is clean**

```bash
git status
```

**Step 2: Run tests one final time**

```bash
docker-compose down
docker-compose up -d
docker-compose up frontend-e2e-tests
```

**Step 3: Create summary commit if any additional fixes were needed**

---

## Verification Checklist

- [ ] `docker-compose up frontend-e2e-tests` runs without errors
- [ ] All 22 E2E tests pass
- [ ] Frontend health check shows "healthy" status
- [ ] API tests still pass (using collector directly)
- [ ] WebSocket tests pass (connection established)

## Rollback Plan

If tests regress, revert to using collector by changing `BASE_URL` back:

```yaml
environment:
  BASE_URL: http://collector:3100
```

And consider alternative: add multi-stage build to include static files in collector container.
