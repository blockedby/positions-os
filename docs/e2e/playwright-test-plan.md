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

## Local Test Runner Setup

### Overview

This section provides a comprehensive setup guide for running Playwright E2E tests locally using Task runner. The setup uses existing Docker infrastructure (`task docker-up`) and adds Playwright testing on top.

### Prerequisites

#### 1. System Requirements

| Tool | Version | Installation |
|------|---------|--------------|
| **Task** | 3.x+ | `brew install go-task` or [taskfile.dev](https://taskfile.dev/installation/) |
| **Bun** | 1.x+ | `curl -fsSL https://bun.sh/install \| bash` |
| **Docker** | 24.x+ | [docker.com](https://docs.docker.com/get-docker/) |
| **Go** | 1.21+ | `brew install go` or [go.dev](https://go.dev/dl/) |

#### 2. Verify Installation

```bash
# Check all tools are available
task --version    # Task version 3.x
bun --version     # Bun 1.x
docker --version  # Docker 24.x+
go version        # go1.21+
```

### Step-by-Step Setup

#### Step 1: Install Playwright in Frontend

```bash
# Navigate to frontend directory
cd frontend

# Install Playwright and dependencies
bun add -D @playwright/test

# Install browser binaries (Chromium only for speed)
bunx playwright install chromium

# Verify installation
bunx playwright --version
```

#### Step 2: Create Playwright Configuration

```typescript
// frontend/playwright.config.ts
import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  // Test directory
  testDir: './e2e',

  // Run tests in parallel
  fullyParallel: true,

  // Fail build on CI if test.only left in code
  forbidOnly: !!process.env.CI,

  // Retry failed tests (more on CI)
  retries: process.env.CI ? 2 : 0,

  // Limit workers for stability
  workers: process.env.CI ? 1 : 4,

  // Reporter configuration
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
  ],

  // Shared settings for all projects
  use: {
    // Base URL for navigation
    baseURL: process.env.BASE_URL || 'http://localhost:3100',

    // Capture trace on first retry
    trace: 'on-first-retry',

    // Screenshot on failure
    screenshot: 'only-on-failure',

    // Video on failure
    video: 'on-first-retry',

    // Timeout for actions
    actionTimeout: 10000,

    // Timeout for navigation
    navigationTimeout: 30000,
  },

  // Test timeout
  timeout: 60000,

  // Expect timeout
  expect: {
    timeout: 5000,
  },

  // Browser projects
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    // Uncomment for cross-browser testing
    // {
    //   name: 'firefox',
    //   use: { ...devices['Desktop Firefox'] },
    // },
    // {
    //   name: 'webkit',
    //   use: { ...devices['Desktop Safari'] },
    // },
  ],

  // Output directory for test artifacts
  outputDir: 'test-results',

  // Global setup/teardown
  globalSetup: './e2e/global-setup.ts',
  globalTeardown: './e2e/global-teardown.ts',
})
```

#### Step 3: Create Directory Structure

```bash
# Create E2E test structure
mkdir -p frontend/e2e/{fixtures,pages,specs}

# Create required files
touch frontend/e2e/global-setup.ts
touch frontend/e2e/global-teardown.ts
touch frontend/e2e/fixtures/index.ts
touch frontend/e2e/fixtures/mock-data.ts
touch frontend/e2e/fixtures/database.fixture.ts
touch frontend/e2e/fixtures/api-mock.fixture.ts
touch frontend/e2e/pages/settings.page.ts
touch frontend/e2e/specs/websocket.spec.ts
touch frontend/e2e/specs/targets.spec.ts
touch frontend/e2e/specs/api.spec.ts
```

**Final Structure:**

```
frontend/
├── e2e/
│   ├── fixtures/
│   │   ├── index.ts              # Export all fixtures
│   │   ├── mock-data.ts          # Test data constants
│   │   ├── database.fixture.ts   # DB seeding fixture
│   │   └── api-mock.fixture.ts   # API mocking fixture
│   ├── pages/
│   │   ├── settings.page.ts      # Settings Page Object
│   │   ├── dashboard.page.ts     # Dashboard Page Object
│   │   └── jobs.page.ts          # Jobs Page Object
│   ├── specs/
│   │   ├── websocket.spec.ts     # WS-* tests
│   │   ├── targets.spec.ts       # TG-* tests
│   │   ├── api.spec.ts           # API-* tests
│   │   └── auth.spec.ts          # AUTH-* tests
│   ├── global-setup.ts           # Runs before all tests
│   └── global-teardown.ts        # Runs after all tests
├── playwright.config.ts
├── playwright-report/            # Generated HTML reports
└── test-results/                 # Test artifacts (screenshots, videos)
```

#### Step 4: Create Global Setup/Teardown

```typescript
// frontend/e2e/global-setup.ts
import { chromium, FullConfig } from '@playwright/test'

async function globalSetup(config: FullConfig) {
  const baseURL = config.projects[0].use.baseURL || 'http://localhost:3100'

  console.log('🚀 Global Setup: Waiting for backend...')

  // Wait for backend to be ready
  const browser = await chromium.launch()
  const page = await browser.newPage()

  let retries = 30
  while (retries > 0) {
    try {
      const response = await page.goto(`${baseURL}/api/v1/stats`, {
        timeout: 5000,
      })
      if (response?.ok()) {
        console.log('✅ Backend is ready!')
        break
      }
    } catch {
      retries--
      if (retries === 0) {
        throw new Error('Backend did not start in time')
      }
      console.log(`⏳ Waiting for backend... (${retries} retries left)`)
      await page.waitForTimeout(1000)
    }
  }

  await browser.close()
}

export default globalSetup
```

```typescript
// frontend/e2e/global-teardown.ts
import { FullConfig } from '@playwright/test'

async function globalTeardown(config: FullConfig) {
  console.log('🧹 Global Teardown: Cleaning up...')
  // Add cleanup logic if needed (e.g., reset database state)
}

export default globalTeardown
```

#### Step 5: Add Taskfile Configuration

Add these tasks to your `Taskfile.yml`:

```yaml
# Taskfile.yml - E2E Testing Tasks
version: '3'

tasks:
  # ==========================================================================
  # E2E Test Setup
  # ==========================================================================

  e2e-setup:
    desc: Install Playwright and browsers
    dir: frontend
    cmds:
      - bun add -D @playwright/test
      - bunx playwright install chromium
    status:
      - test -d node_modules/@playwright

  e2e-setup-all-browsers:
    desc: Install all Playwright browsers (Chromium, Firefox, WebKit)
    dir: frontend
    cmds:
      - bunx playwright install

  # ==========================================================================
  # Running Tests
  # ==========================================================================

  e2e:
    desc: Run all E2E tests (requires backend running)
    dir: frontend
    deps: [docker-up]
    cmds:
      - bunx playwright test
    env:
      BASE_URL: http://localhost:3100

  e2e-headed:
    desc: Run E2E tests with visible browser
    dir: frontend
    cmds:
      - bunx playwright test --headed
    env:
      BASE_URL: http://localhost:3100

  e2e-ui:
    desc: Open Playwright UI mode for interactive testing
    dir: frontend
    cmds:
      - bunx playwright test --ui
    env:
      BASE_URL: http://localhost:3100

  e2e-debug:
    desc: Run E2E tests in debug mode (step through)
    dir: frontend
    cmds:
      - bunx playwright test --debug
    env:
      BASE_URL: http://localhost:3100
      PWDEBUG: '1'

  # ==========================================================================
  # Filtered Test Runs
  # ==========================================================================

  e2e-websocket:
    desc: Run only WebSocket tests
    dir: frontend
    cmds:
      - bunx playwright test specs/websocket.spec.ts
    env:
      BASE_URL: http://localhost:3100

  e2e-targets:
    desc: Run only Targets CRUD tests
    dir: frontend
    cmds:
      - bunx playwright test specs/targets.spec.ts
    env:
      BASE_URL: http://localhost:3100

  e2e-api:
    desc: Run only API tests
    dir: frontend
    cmds:
      - bunx playwright test specs/api.spec.ts
    env:
      BASE_URL: http://localhost:3100

  e2e-grep:
    desc: Run tests matching pattern (usage: task e2e-grep -- "pattern")
    dir: frontend
    cmds:
      - bunx playwright test --grep "{{.CLI_ARGS}}"
    env:
      BASE_URL: http://localhost:3100

  # ==========================================================================
  # Reports & Artifacts
  # ==========================================================================

  e2e-report:
    desc: Open last E2E test report in browser
    dir: frontend
    cmds:
      - bunx playwright show-report

  e2e-clean:
    desc: Clean test artifacts (reports, screenshots, videos)
    dir: frontend
    cmds:
      - rm -rf playwright-report test-results

  # ==========================================================================
  # CI Integration
  # ==========================================================================

  e2e-ci:
    desc: Run E2E tests in CI mode (single worker, retries)
    dir: frontend
    cmds:
      - bunx playwright test --reporter=github,html
    env:
      BASE_URL: http://localhost:3100
      CI: 'true'

  # ==========================================================================
  # Development Workflow
  # ==========================================================================

  e2e-watch:
    desc: Watch mode - re-run tests on file changes
    dir: frontend
    cmds:
      - |
        echo "Watching for changes in e2e/ directory..."
        while true; do
          inotifywait -r -e modify,create,delete ./e2e/ 2>/dev/null || fswatch -1 ./e2e/
          bunx playwright test --reporter=list
        done
    env:
      BASE_URL: http://localhost:3100

  e2e-new-test:
    desc: Generate test file from template (usage: task e2e-new-test -- feature-name)
    dir: frontend
    cmds:
      - |
        cat > e2e/specs/{{.CLI_ARGS}}.spec.ts << 'EOF'
        import { test, expect } from '@playwright/test'

        test.describe('{{.CLI_ARGS}}', () => {
          test.beforeEach(async ({ page }) => {
            // Setup before each test
          })

          test('should work', async ({ page }) => {
            // Test implementation
            await page.goto('/')
            await expect(page).toHaveTitle(/Positions OS/)
          })
        })
        EOF
      - echo "Created e2e/specs/{{.CLI_ARGS}}.spec.ts"

  # ==========================================================================
  # Full Test Cycle
  # ==========================================================================

  e2e-full:
    desc: Full E2E cycle - start services, run tests, show report
    cmds:
      - task: docker-up
      - task: migrate-up
      - task: e2e
      - task: e2e-report

  e2e-dev:
    desc: Start backend and open Playwright UI
    cmds:
      - task: docker-up
      - task: e2e-ui
```

#### Step 6: Add Package.json Scripts

```json
// frontend/package.json (add to scripts)
{
  "scripts": {
    "test:e2e": "playwright test",
    "test:e2e:headed": "playwright test --headed",
    "test:e2e:ui": "playwright test --ui",
    "test:e2e:debug": "PWDEBUG=1 playwright test --debug",
    "test:e2e:report": "playwright show-report"
  }
}
```

### Development Workflow

#### TDD Cycle with Task Runner

```
┌─────────────────────────────────────────────────────────────────┐
│                    RED-GREEN-REFACTOR CYCLE                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. RED: Write failing test                                     │
│     $ task e2e-grep -- "should create target"                   │
│     ❌ Test fails (feature not implemented)                     │
│                                                                 │
│  2. GREEN: Implement minimal fix                                │
│     $ task e2e-grep -- "should create target"                   │
│     ✅ Test passes                                              │
│                                                                 │
│  3. REFACTOR: Improve code quality                              │
│     $ task e2e-targets                                          │
│     ✅ All target tests pass                                    │
│                                                                 │
│  4. VALIDATE: Run full suite                                    │
│     $ task e2e                                                  │
│     ✅ All tests pass                                           │
│                                                                 │
│  5. COMMIT & PUSH                                               │
│     $ git add . && git commit -m "feat: add target creation"    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

#### Quick Reference Commands

| Task | Command | Description |
|------|---------|-------------|
| **Setup** | `task e2e-setup` | Install Playwright & browsers |
| **Run all** | `task e2e` | Run all E2E tests |
| **UI mode** | `task e2e-ui` | Interactive test runner |
| **Debug** | `task e2e-debug` | Step-through debugging |
| **Headed** | `task e2e-headed` | Watch tests in browser |
| **Specific** | `task e2e-grep -- "pattern"` | Run matching tests |
| **Report** | `task e2e-report` | View HTML report |
| **Clean** | `task e2e-clean` | Remove artifacts |
| **Full cycle** | `task e2e-full` | Services + tests + report |

### Environment Configuration

#### .env.test

```bash
# frontend/.env.test
# Base URL for tests
BASE_URL=http://localhost:3100

# Database for seeding (optional, for DB fixtures)
TEST_DATABASE_URL=postgres://postgres:postgres@localhost:5432/positions_dev

# Playwright settings
PWDEBUG=0
CI=false

# Telegram mocking - NEVER use real credentials!
TG_API_ID=0
TG_API_HASH=mock
TG_SESSION_STRING=
```

#### Loading Environment in Tests

```typescript
// frontend/e2e/fixtures/index.ts
import { test as base } from '@playwright/test'
import * as dotenv from 'dotenv'

// Load test environment
dotenv.config({ path: '.env.test' })

// Re-export base test
export { expect } from '@playwright/test'
export const test = base
```

### Troubleshooting

#### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `Backend not ready` | Services not started | Run `task docker-up` first |
| `Browser not found` | Playwright not installed | Run `task e2e-setup` |
| `Timeout waiting` | Slow backend startup | Increase timeout in global-setup |
| `Port already in use` | Previous run didn't cleanup | Run `task docker-down && task docker-up` |
| `ECONNREFUSED` | Wrong BASE_URL | Check `.env.test` has correct URL |

#### Debug Mode

```bash
# Enable Playwright Inspector
PWDEBUG=1 task e2e-grep -- "failing test"

# Verbose logging
DEBUG=pw:api task e2e

# Trace viewer for failed tests
bunx playwright show-trace test-results/*/trace.zip
```

#### Check Backend Health

```bash
# Verify backend is responding
curl http://localhost:3100/api/v1/stats

# Check WebSocket endpoint
websocat ws://localhost:3100/ws

# View backend logs
docker compose logs -f collector
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
