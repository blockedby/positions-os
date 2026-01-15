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
