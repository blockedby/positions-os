# Playwright E2E Test Plan

**Related PR:** [#12 - fix(frontend): prevent WebSocket infinite reconnection loop](https://github.com/blockedby/positions-os/pull/12)

## Overview

This document outlines the end-to-end testing strategy for the React frontend, focusing on the WebSocket stability fixes and Targets API functionality introduced in PR #12.

## Test Scope

### 1. WebSocket Stability Tests

**Goal:** Verify that the WebSocket connection is stable and does not cause infinite reconnection loops.

#### Test Cases

| ID | Test Name | Description | Expected Result |
|----|-----------|-------------|-----------------|
| WS-01 | `websocket-single-connection` | Navigate to Settings page and monitor WebSocket connections | Only ONE WebSocket connection should be established |
| WS-02 | `websocket-no-reconnect-loop` | Open Settings page and wait 10 seconds | No repeated "WebSocket is closed" errors in console |
| WS-03 | `websocket-survives-navigation` | Navigate between Dashboard -> Settings -> Jobs | WebSocket should not disconnect/reconnect unnecessarily |
| WS-04 | `websocket-reconnect-on-disconnect` | Simulate server disconnect | WebSocket should reconnect once after server comes back |

#### Implementation Notes

```typescript
// WS-01: Monitor WebSocket connections
test('should establish only one WebSocket connection', async ({ page }) => {
  const wsConnections: string[] = []

  page.on('websocket', ws => {
    wsConnections.push(ws.url())
  })

  await page.goto('/settings')
  await page.waitForTimeout(5000)

  // Should have exactly one connection
  expect(wsConnections.length).toBe(1)
})

// WS-02: Monitor console for errors
test('should not produce WebSocket reconnection errors', async ({ page }) => {
  const errors: string[] = []

  page.on('console', msg => {
    if (msg.type() === 'error' && msg.text().includes('WebSocket')) {
      errors.push(msg.text())
    }
  })

  await page.goto('/settings')
  await page.waitForTimeout(10000)

  expect(errors).toHaveLength(0)
})
```

---

### 2. Targets CRUD Tests

**Goal:** Verify that targets can be created, read, updated, and deleted via the React UI.

#### Test Cases

| ID | Test Name | Description | Expected Result |
|----|-----------|-------------|-----------------|
| TG-01 | `targets-list-empty-state` | Load Settings with no targets | "No targets configured" message displayed |
| TG-02 | `targets-list-displays-items` | Load Settings with existing targets | Target items displayed with name, type, URL |
| TG-03 | `targets-create-channel` | Create a new TG_CHANNEL target | Target appears in list, API returns 201 |
| TG-04 | `targets-create-validation` | Submit form with missing fields | Validation errors displayed |
| TG-05 | `targets-edit-target` | Edit existing target name | Target updated in list |
| TG-06 | `targets-delete-target` | Delete a target with confirmation | Target removed from list |
| TG-07 | `targets-toggle-active` | Toggle is_active checkbox on edit | Active/Paused badge updates |

#### Implementation Notes

```typescript
// TG-01: Empty state
test('should display empty state when no targets exist', async ({ page }) => {
  await page.route('**/api/v1/targets', route => {
    route.fulfill({ status: 200, body: '[]' })
  })

  await page.goto('/settings')
  await expect(page.getByText('No targets configured')).toBeVisible()
})

// TG-03: Create target
test('should create a new target', async ({ page }) => {
  await page.goto('/settings')
  await page.getByRole('button', { name: 'Add Target' }).click()

  await page.getByLabel('Name').fill('Go Jobs Channel')
  await page.getByLabel('Type').selectOption('TG_CHANNEL')
  await page.getByLabel('URL').fill('@golang_jobs')

  await page.getByRole('button', { name: 'Create' }).click()

  await expect(page.getByText('Go Jobs Channel')).toBeVisible()
})

// TG-06: Delete with confirmation
test('should delete target after confirmation', async ({ page }) => {
  page.on('dialog', dialog => dialog.accept())

  await page.goto('/settings')
  await page.getByRole('button', { name: 'Delete' }).first().click()

  // Target should be removed
})
```

---

### 3. API Response Tests

**Goal:** Verify that API endpoints return correct JSON responses.

#### Test Cases

| ID | Test Name | Description | Expected Result |
|----|-----------|-------------|-----------------|
| API-01 | `api-targets-returns-array` | GET /api/v1/targets when empty | Returns `[]` not null |
| API-02 | `api-targets-create-json` | POST /api/v1/targets with JSON body | Returns created target with 201 |
| API-03 | `api-targets-update-json` | PUT /api/v1/targets/:id with JSON body | Returns updated target with 200 |
| API-04 | `api-targets-get-by-id` | GET /api/v1/targets/:id | Returns single target object |

#### Implementation Notes

```typescript
// API-01: Empty array response
test('should return empty array for empty targets list', async ({ request }) => {
  const response = await request.get('/api/v1/targets')
  const body = await response.json()

  expect(response.ok()).toBeTruthy()
  expect(Array.isArray(body)).toBe(true)
  expect(body).toEqual([])
})

// API-02: Create with JSON
test('should create target via JSON API', async ({ request }) => {
  const response = await request.post('/api/v1/targets', {
    data: {
      name: 'Test Channel',
      type: 'TG_CHANNEL',
      url: '@test_channel',
      is_active: true
    }
  })

  expect(response.status()).toBe(201)
  const body = await response.json()
  expect(body.name).toBe('Test Channel')
  expect(body.id).toBeDefined()
})
```

---

### 4. Real-Time Updates Tests

**Goal:** Verify that WebSocket events properly update the UI.

#### Test Cases

| ID | Test Name | Description | Expected Result |
|----|-----------|-------------|-----------------|
| RT-01 | `realtime-target-created` | Create target via API | UI updates without refresh |
| RT-02 | `realtime-scrape-progress` | Start scrape operation | Progress indicator updates |
| RT-03 | `realtime-job-analyzed` | New job analyzed | Jobs list updates |

---

## Test Environment Setup

### Prerequisites

```bash
# Install Playwright
cd frontend
bun add -D @playwright/test
bunx playwright install chromium

# Create playwright.config.ts
```

### Configuration

```typescript
// frontend/playwright.config.ts
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',

  use: {
    baseURL: 'http://localhost:3100',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  webServer: {
    command: 'cd .. && task docker-app',
    url: 'http://localhost:3100',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },
})
```

### Directory Structure

```
frontend/
├── e2e/
│   ├── fixtures/
│   │   └── test-data.ts        # Mock data for tests
│   ├── pages/
│   │   ├── settings.page.ts    # Page Object Model
│   │   └── dashboard.page.ts
│   ├── websocket.spec.ts       # WS-* tests
│   ├── targets.spec.ts         # TG-* tests
│   └── api.spec.ts             # API-* tests
├── playwright.config.ts
└── package.json
```

---

## Test Execution

### Commands

```bash
# Run all E2E tests
bun run test:e2e

# Run specific test file
bunx playwright test websocket.spec.ts

# Run with UI mode (debugging)
bunx playwright test --ui

# Run headed (see browser)
bunx playwright test --headed

# Generate report
bunx playwright show-report
```

### CI Integration

```yaml
# .github/workflows/e2e.yml
name: E2E Tests

on:
  pull_request:
    branches: [main]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: oven-sh/setup-bun@v2

      - name: Install dependencies
        run: cd frontend && bun install

      - name: Install Playwright Browsers
        run: cd frontend && bunx playwright install --with-deps chromium

      - name: Start services
        run: docker compose up -d

      - name: Run E2E tests
        run: cd frontend && bun run test:e2e

      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: playwright-report
          path: frontend/playwright-report/
```

---

## 5. Telegram Authentication Tests (Mocked)

**Goal:** Test Telegram QR authentication flow without real Telegram API calls.

> **IMPORTANT:** Telegram authentication MUST be mocked in all E2E tests. Never use real Telegram credentials in automated tests.

### Why Mock Telegram Auth?

1. **Security** - Real session strings should never be in test code or CI
2. **Rate limits** - Telegram aggressively rate-limits authentication attempts
3. **Stability** - External API calls make tests flaky
4. **Speed** - Mocked responses are instant vs real API latency

### Test Cases

| ID | Test Name | Description | Expected Result |
|----|-----------|-------------|-----------------|
| AUTH-01 | `tg-auth-qr-display` | Open TelegramAuth component | QR code placeholder displayed |
| AUTH-02 | `tg-auth-qr-refresh` | Click refresh QR button | New QR code generated (mocked) |
| AUTH-03 | `tg-auth-success-flow` | Simulate successful auth via WS event | Success message displayed |
| AUTH-04 | `tg-auth-error-handling` | Simulate auth error via WS event | Error message displayed |

### Mocking Strategy

```typescript
// e2e/fixtures/telegram-auth.fixture.ts
import { test as base } from '@playwright/test'

// Mock Telegram auth responses
export const test = base.extend({
  mockTelegramAuth: async ({ page }, use) => {
    // Intercept Telegram auth API calls
    await page.route('**/api/v1/telegram/auth/**', route => {
      const url = route.request().url()

      if (url.includes('/qr')) {
        // Mock QR code generation
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            qr_link: 'tg://login?token=MOCK_TOKEN_12345',
            expires_at: new Date(Date.now() + 60000).toISOString()
          })
        })
      } else if (url.includes('/status')) {
        // Mock auth status check
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'pending' })
        })
      }
    })

    await use()
  }
})

// Test implementation
test('should display QR code for Telegram auth', async ({ page, mockTelegramAuth }) => {
  await page.goto('/settings')

  // Find TelegramAuth section
  const authSection = page.locator('[data-testid="telegram-auth"]')
  await expect(authSection).toBeVisible()

  // QR should be displayed (mocked)
  await expect(page.getByText('Scan QR')).toBeVisible()
})
```

### WebSocket Event Mocking for Auth

```typescript
// Simulate auth success via WebSocket
test('should handle successful Telegram auth', async ({ page }) => {
  await page.goto('/settings')

  // Inject mock WebSocket event
  await page.evaluate(() => {
    window.dispatchEvent(new CustomEvent('ws-mock', {
      detail: {
        type: 'tg_auth.success',
        user: { id: 12345, username: 'test_user' }
      }
    }))
  })

  await expect(page.getByText('Connected as @test_user')).toBeVisible()
})
```

---

## Data Mocks Plan

### Overview

All E2E tests should use consistent mock data to ensure reproducibility. Mock data is seeded before tests and cleaned up afterward.

### Core Database Entities

#### 1. Scraping Targets

```typescript
// e2e/fixtures/mock-data.ts
export const mockTargets = {
  channel: {
    id: '550e8400-e29b-41d4-a716-446655440001',
    name: 'Go Jobs Channel',
    type: 'TG_CHANNEL',
    url: '@golang_jobs',
    is_active: true,
    metadata: {},
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    last_scraped_at: null
  },
  forum: {
    id: '550e8400-e29b-41d4-a716-446655440002',
    name: 'Rust Forum',
    type: 'TG_FORUM',
    url: '@rust_jobs',
    is_active: true,
    metadata: { topic_ids: [1, 42, 100] },
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    last_scraped_at: '2025-01-10T12:00:00Z'
  },
  inactive: {
    id: '550e8400-e29b-41d4-a716-446655440003',
    name: 'Paused Target',
    type: 'TG_CHANNEL',
    url: '@paused_channel',
    is_active: false,
    metadata: {},
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-05T00:00:00Z',
    last_scraped_at: null
  }
}
```

#### 2. Jobs

```typescript
export const mockJobs = {
  raw: {
    id: '660e8400-e29b-41d4-a716-446655440001',
    external_id: 'tg_12345',
    source_channel: '@golang_jobs',
    status: 'RAW',
    raw_content: 'Looking for Go developer with 3+ years experience...',
    structured_data: null,
    created_at: '2025-01-15T10:00:00Z',
    updated_at: '2025-01-15T10:00:00Z'
  },
  analyzed: {
    id: '660e8400-e29b-41d4-a716-446655440002',
    external_id: 'tg_12346',
    source_channel: '@golang_jobs',
    status: 'ANALYZED',
    raw_content: 'Senior Backend Engineer needed. Remote OK. 150-250k RUB.',
    structured_data: {
      title: 'Senior Backend Engineer',
      description: 'Senior Backend Engineer needed',
      salary_min: 150000,
      salary_max: 250000,
      currency: 'RUB',
      location: 'Remote',
      is_remote: true,
      language: 'RU',
      technologies: ['Go', 'PostgreSQL', 'Docker'],
      experience_years: 5,
      company: null,
      contacts: ['@hr_contact']
    },
    created_at: '2025-01-15T09:00:00Z',
    updated_at: '2025-01-15T09:30:00Z'
  },
  interested: {
    id: '660e8400-e29b-41d4-a716-446655440003',
    external_id: 'tg_12347',
    source_channel: '@rust_jobs',
    status: 'INTERESTED',
    raw_content: 'Rust developer for fintech startup...',
    structured_data: {
      title: 'Rust Developer',
      salary_min: 200000,
      salary_max: 350000,
      currency: 'RUB',
      is_remote: true,
      technologies: ['Rust', 'Tokio', 'PostgreSQL']
    },
    created_at: '2025-01-14T08:00:00Z',
    updated_at: '2025-01-14T12:00:00Z'
  }
}
```

#### 3. Stats

```typescript
export const mockStats = {
  total_jobs: 156,
  by_status: {
    RAW: 45,
    ANALYZED: 78,
    INTERESTED: 23,
    REJECTED: 10
  },
  by_source: {
    '@golang_jobs': 89,
    '@rust_jobs': 67
  },
  active_targets: 2,
  total_targets: 3
}
```

### Database Seeding Strategy

Using [playwright-postgres-seeder](https://www.npmjs.com/package/playwright-postgres-seeder) pattern:

```typescript
// e2e/fixtures/database.fixture.ts
import { test as base } from '@playwright/test'
import { Pool } from 'pg'
import { mockTargets, mockJobs } from './mock-data'

type DatabaseFixture = {
  db: Pool
  seedTargets: () => Promise<void>
  seedJobs: () => Promise<void>
  cleanup: () => Promise<void>
}

export const test = base.extend<DatabaseFixture>({
  db: async ({}, use) => {
    const pool = new Pool({
      connectionString: process.env.TEST_DATABASE_URL ||
        'postgres://postgres:postgres@localhost:5432/positions_test'
    })
    await use(pool)
    await pool.end()
  },

  seedTargets: async ({ db }, use) => {
    const seed = async () => {
      for (const target of Object.values(mockTargets)) {
        await db.query(`
          INSERT INTO scraping_targets (id, name, type, url, is_active, metadata, created_at, updated_at, last_scraped_at)
          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
          ON CONFLICT (id) DO NOTHING
        `, [target.id, target.name, target.type, target.url, target.is_active,
            JSON.stringify(target.metadata), target.created_at, target.updated_at, target.last_scraped_at])
      }
    }
    await use(seed)
  },

  seedJobs: async ({ db }, use) => {
    const seed = async () => {
      for (const job of Object.values(mockJobs)) {
        await db.query(`
          INSERT INTO jobs (id, external_id, source_channel, status, raw_content, structured_data, created_at, updated_at)
          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
          ON CONFLICT (id) DO NOTHING
        `, [job.id, job.external_id, job.source_channel, job.status, job.raw_content,
            job.structured_data ? JSON.stringify(job.structured_data) : null, job.created_at, job.updated_at])
      }
    }
    await use(seed)
  },

  cleanup: async ({ db }, use) => {
    const clean = async () => {
      await db.query('DELETE FROM jobs WHERE id LIKE $1', ['660e8400%'])
      await db.query('DELETE FROM scraping_targets WHERE id LIKE $1', ['550e8400%'])
    }
    await use(clean)
  }
})

// Usage in tests
test.describe('Jobs page', () => {
  test.beforeEach(async ({ seedTargets, seedJobs }) => {
    await seedTargets()
    await seedJobs()
  })

  test.afterEach(async ({ cleanup }) => {
    await cleanup()
  })

  test('should display jobs list', async ({ page }) => {
    await page.goto('/jobs')
    await expect(page.getByText('Senior Backend Engineer')).toBeVisible()
  })
})
```

### API Mocking for Isolated Tests

For tests that don't need database:

```typescript
// e2e/fixtures/api-mock.fixture.ts
import { test as base } from '@playwright/test'
import { mockTargets, mockJobs, mockStats } from './mock-data'

export const test = base.extend({
  mockApi: async ({ page }, use) => {
    // Mock all API endpoints
    await page.route('**/api/v1/targets', route => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(Object.values(mockTargets))
        })
      }
    })

    await page.route('**/api/v1/jobs*', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: Object.values(mockJobs),
          total: 3,
          page: 1,
          per_page: 20
        })
      })
    })

    await page.route('**/api/v1/stats', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockStats)
      })
    })

    await use()
  }
})
```

### Mock Data Summary Table

| Entity | Mock Count | Variants |
|--------|------------|----------|
| Targets | 3 | channel, forum, inactive |
| Jobs | 3 | raw, analyzed, interested |
| Stats | 1 | aggregated counts |
| TG Auth | N/A | QR code, status events |

---

## Local CI/CD Runner

### Overview

For local development, we use a Docker-based test runner that mirrors CI behavior. This ensures tests pass locally before pushing.

### Option 1: Docker Compose (Recommended)

```yaml
# docker-compose.test.yml
services:
  # Test database (isolated from dev)
  postgres-test:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: positions_test
    ports:
      - "5433:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  # NATS for WebSocket testing
  nats-test:
    image: nats:2.10-alpine
    command: ["--jetstream"]
    ports:
      - "4223:4222"

  # Backend API
  api-test:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres-test:5432/positions_test?sslmode=disable
      NATS_URL: nats://nats-test:4222
      HTTP_PORT: 3100
      # Mock Telegram - don't use real credentials!
      TG_API_ID: "0"
      TG_API_HASH: "mock"
      TG_SESSION_STRING: ""
    ports:
      - "3100:3100"
    depends_on:
      postgres-test:
        condition: service_healthy
      nats-test:
        condition: service_started

  # Playwright test runner
  playwright:
    image: mcr.microsoft.com/playwright:v1.50.0-noble
    working_dir: /app/frontend
    volumes:
      - ./frontend:/app/frontend
      - ./playwright-report:/app/frontend/playwright-report
    environment:
      BASE_URL: http://api-test:3100
      TEST_DATABASE_URL: postgres://postgres:postgres@postgres-test:5432/positions_test
    command: ["npx", "playwright", "test", "--reporter=html"]
    depends_on:
      - api-test
```

### Running Tests Locally

```bash
# Run all E2E tests in Docker
docker compose -f docker-compose.test.yml up --build --abort-on-container-exit

# View HTML report after tests complete
open frontend/playwright-report/index.html

# Clean up
docker compose -f docker-compose.test.yml down -v
```

### Option 2: Task Runner Integration

Add to `Taskfile.yml`:

```yaml
# Taskfile.yml additions
tasks:
  test-e2e:
    desc: Run E2E tests in Docker
    cmds:
      - docker compose -f docker-compose.test.yml up --build --abort-on-container-exit
      - docker compose -f docker-compose.test.yml down -v
    env:
      COMPOSE_PROJECT_NAME: positions-test

  test-e2e-local:
    desc: Run E2E tests against local services
    dir: frontend
    cmds:
      - bunx playwright test {{.CLI_ARGS}}
    env:
      BASE_URL: http://localhost:3100

  test-e2e-ui:
    desc: Run E2E tests with Playwright UI
    dir: frontend
    cmds:
      - bunx playwright test --ui

  test-e2e-report:
    desc: Open last E2E test report
    dir: frontend
    cmds:
      - bunx playwright show-report
```

### Option 3: Act (GitHub Actions Locally)

Run GitHub Actions workflows locally using [act](https://github.com/nektos/act):

```bash
# Install act
brew install act  # macOS
# or
curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Run E2E workflow locally
act -j e2e -P ubuntu-latest=catthehacker/ubuntu:act-latest

# Run with secrets
act -j e2e --secret-file .secrets
```

### Comparison of Local CI Options

| Option | Pros | Cons | Best For |
|--------|------|------|----------|
| **Docker Compose** | Full isolation, mirrors prod | Slower startup, more resources | Full integration tests |
| **Task Runner** | Fast, simple | Requires local services running | Quick iteration |
| **Act** | Exact CI parity | Complex setup, slow | Debugging CI failures |

### Recommended Workflow

```
Development Cycle:
1. Write test (RED)           → task test-e2e-local -- --grep "my test"
2. Implement fix (GREEN)      → task test-e2e-local -- --grep "my test"
3. Refactor                   → task test-e2e-local
4. Full validation            → task test-e2e (Docker)
5. Push to PR                 → GitHub Actions runs automatically
```

### Environment Variables for Testing

```bash
# .env.test
DATABASE_URL=postgres://postgres:postgres@localhost:5433/positions_test
NATS_URL=nats://localhost:4223
HTTP_PORT=3100
BASE_URL=http://localhost:3100

# Telegram mocking - NEVER use real credentials in tests!
TG_API_ID=0
TG_API_HASH=mock
TG_SESSION_STRING=
```

---

## Priority Order

Based on PR #12's fixes, tests should be implemented in this order:

1. **Critical (Phase 1)**
   - WS-01, WS-02: WebSocket stability
   - TG-01, TG-03: Basic target CRUD
   - API-01: Empty array fix

2. **Important (Phase 2)**
   - TG-04, TG-05, TG-06: Full CRUD cycle
   - WS-03, WS-04: WebSocket resilience
   - API-02, API-03: JSON API

3. **Enhancement (Phase 3)**
   - RT-01, RT-02, RT-03: Real-time updates
   - TG-07: Edge cases

---

## Success Criteria

Tests pass when:
- WebSocket establishes exactly 1 connection per page
- No WebSocket errors in console after 10 seconds
- Targets API returns `[]` for empty list
- CRUD operations work via JSON API
- UI updates in real-time via WebSocket events

---

## References

### Playwright Documentation
- [Mock APIs | Playwright](https://playwright.dev/docs/mock) - Official mocking guide
- [Authentication | Playwright](https://playwright.dev/docs/auth) - Auth testing patterns
- [Fixtures | Playwright](https://playwright.dev/docs/test-fixtures) - Custom fixtures
- [Continuous Integration | Playwright](https://playwright.dev/docs/ci) - CI setup

### Database Testing
- [playwright-postgres-seeder](https://www.npmjs.com/package/playwright-postgres-seeder) - PostgreSQL seeding plugin
- [Database Rollback Strategies in Playwright](https://www.thegreenreport.blog/articles/database-rollback-strategies-in-playwright/database-rollback-strategies-in-playwright.html) - Transaction rollback patterns
- [Managing Database Integration With Playwright](https://medium.com/@Amr.sa/managing-database-integration-with-playwright-4b7484e98615) - Integration strategies

### Docker & CI/CD
- [Running Playwright Tests in Docker](https://www.neovasolutions.com/2024/10/03/running-playwright-tests-in-a-docker-container/) - Docker containerization
- [End-to-End Testing with Playwright and Docker | BrowserStack](https://www.browserstack.com/guide/playwright-docker) - Docker best practices
- [Built a Full CI/CD Pipeline with Playwright + Docker](https://dev.to/deftoexplore/built-a-full-cicd-pipeline-with-playwright-docker-allure-in-just-2-days-155m) - Complete pipeline example

### Best Practices
- [How to Mock APIs with Playwright | BrowserStack](https://www.browserstack.com/guide/how-to-mock-api-with-playwright) - Comprehensive mocking guide
- [How to Manage Authentication in Playwright | Checkly](https://www.checklyhq.com/docs/learn/playwright/authentication/) - Auth management patterns
