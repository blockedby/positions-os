# E2E Database Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add proper database cleanup to E2E tests so orphaned test data doesn't accumulate between runs.

**Architecture:** Implement cleanup in global-teardown.ts that deletes all test-created resources via the existing REST API. No new backend endpoints needed. Clean up targets (the only resource API tests create).

**Tech Stack:** Playwright, TypeScript, REST API

---

## Background

**Current State:**
- API tests create targets with names like `E2E Test Target 1737012345678`
- Each test deletes its own data on success
- If a test fails mid-execution, orphaned data remains
- Global teardown is a stub that does nothing

**Tables that could have test data:**
- `scraping_targets` - API tests create these
- `jobs` - Not created by E2E tests (only via scraping)
- Others - Not touched by E2E tests

---

### Task 1: Implement Global Teardown Cleanup

**Files:**
- Modify: `frontend/e2e/global-teardown.ts`

**Step 1: Read current implementation**

Current code (stub):
```typescript
import { FullConfig } from '@playwright/test'

async function globalTeardown(_config: FullConfig) {
  console.log('Global Teardown: Cleaning up...')
  // Add cleanup logic if needed (e.g., reset database state)
}

export default globalTeardown
```

**Step 2: Implement cleanup logic**

Replace with:
```typescript
import { FullConfig } from '@playwright/test'

async function globalTeardown(config: FullConfig) {
  const apiBaseUrl = process.env.API_BASE_URL || 'http://localhost:3100'

  console.log('Global Teardown: Cleaning up test data...')

  try {
    // Fetch all targets
    const response = await fetch(`${apiBaseUrl}/api/v1/targets`, {
      signal: AbortSignal.timeout(5000)
    })

    if (!response.ok) {
      console.log('Global Teardown: Could not fetch targets, skipping cleanup')
      return
    }

    const targets = await response.json()

    if (!Array.isArray(targets) || targets.length === 0) {
      console.log('Global Teardown: No targets to clean up')
      return
    }

    // Delete targets that look like test data (contain "E2E" or "Test")
    const testTargets = targets.filter((t: { name: string }) =>
      /e2e|test/i.test(t.name)
    )

    if (testTargets.length === 0) {
      console.log('Global Teardown: No test targets to clean up')
      return
    }

    console.log(`Global Teardown: Deleting ${testTargets.length} test target(s)...`)

    for (const target of testTargets) {
      try {
        await fetch(`${apiBaseUrl}/api/v1/targets/${target.id}`, {
          method: 'DELETE',
          signal: AbortSignal.timeout(2000)
        })
      } catch (err) {
        console.log(`Global Teardown: Failed to delete target ${target.id}`)
      }
    }

    console.log('Global Teardown: Cleanup complete')
  } catch (err) {
    console.log('Global Teardown: Cleanup failed, continuing anyway')
  }
}

export default globalTeardown
```

**Step 3: Verify by running tests twice**

Run:
```bash
docker-compose up frontend-e2e-tests
```

Expected: Tests pass, teardown logs show cleanup activity.

Run again to verify no orphaned data:
```bash
docker-compose up frontend-e2e-tests
```

Expected: "No test targets to clean up" or clean deletion.

**Step 4: Commit**

```bash
git add frontend/e2e/global-teardown.ts
git commit -m "feat(e2e): add database cleanup in global teardown

Deletes test-created targets (names containing 'E2E' or 'Test')
after test run completes. Prevents orphaned data accumulation
when tests fail mid-execution."
```

---

### Task 2: Add Cleanup Before Tests (Optional Safety Net)

**Files:**
- Modify: `frontend/e2e/global-setup.ts`

**Step 1: Add pre-test cleanup**

This ensures a clean slate even if previous run crashed without teardown.

Add after the backend readiness check (around line 25):

```typescript
// Clean up any leftover test data from previous runs
try {
  const apiBaseUrl = process.env.API_BASE_URL || baseURL.replace(':80', ':3100').replace('frontend', 'collector')
  const targetsResponse = await fetch(`${apiBaseUrl}/api/v1/targets`, {
    signal: AbortSignal.timeout(5000)
  })

  if (targetsResponse.ok) {
    const targets = await targetsResponse.json()
    const testTargets = targets.filter((t: { name: string }) => /e2e|test/i.test(t.name))

    if (testTargets.length > 0) {
      console.log(`Global Setup: Cleaning ${testTargets.length} leftover test target(s)...`)
      for (const target of testTargets) {
        await fetch(`${apiBaseUrl}/api/v1/targets/${target.id}`, {
          method: 'DELETE',
          signal: AbortSignal.timeout(2000)
        }).catch(() => {})
      }
    }
  }
} catch {
  // Ignore cleanup errors in setup
}
```

**Step 2: Run tests to verify**

Run:
```bash
docker-compose up frontend-e2e-tests
```

Expected: Setup logs show cleanup if orphaned data exists.

**Step 3: Commit**

```bash
git add frontend/e2e/global-setup.ts
git commit -m "feat(e2e): add pre-test cleanup for orphaned test data

Cleans up leftover test targets before running tests.
Ensures clean slate even if previous run crashed."
```

---

### Task 3: Extract Shared Cleanup Function (DRY)

**Files:**
- Create: `frontend/e2e/utils/cleanup.ts`
- Modify: `frontend/e2e/global-setup.ts`
- Modify: `frontend/e2e/global-teardown.ts`

**Step 1: Create shared utility**

```typescript
// frontend/e2e/utils/cleanup.ts

export async function cleanupTestTargets(apiBaseUrl: string): Promise<number> {
  try {
    const response = await fetch(`${apiBaseUrl}/api/v1/targets`, {
      signal: AbortSignal.timeout(5000)
    })

    if (!response.ok) {
      return 0
    }

    const targets = await response.json()

    if (!Array.isArray(targets)) {
      return 0
    }

    // Match test data patterns: "E2E", "Test", or names with timestamps
    const testTargets = targets.filter((t: { name: string }) =>
      /e2e|test|\d{13}/i.test(t.name)
    )

    for (const target of testTargets) {
      await fetch(`${apiBaseUrl}/api/v1/targets/${target.id}`, {
        method: 'DELETE',
        signal: AbortSignal.timeout(2000)
      }).catch(() => {})
    }

    return testTargets.length
  } catch {
    return 0
  }
}
```

**Step 2: Update global-setup.ts**

```typescript
import { FullConfig } from '@playwright/test'
import { cleanupTestTargets } from './utils/cleanup'

async function globalSetup(config: FullConfig) {
  const baseURL = process.env.BASE_URL || config.projects[0].use.baseURL || 'http://localhost:3100'
  const apiBaseUrl = process.env.API_BASE_URL || 'http://localhost:3100'

  if (process.env.SKIP_BACKEND_CHECK) {
    console.log('Global Setup: Skipping backend check (SKIP_BACKEND_CHECK=1)')
    return
  }

  console.log(`Global Setup: Checking backend at ${baseURL}...`)

  const maxRetries = process.env.CI ? 10 : 3
  const retryDelay = 1000

  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch(`${baseURL}/api/v1/stats`, {
        signal: AbortSignal.timeout(2000)
      })
      if (response.ok) {
        console.log('Global Setup: Backend is ready!')

        // Clean up leftover test data
        const cleaned = await cleanupTestTargets(apiBaseUrl)
        if (cleaned > 0) {
          console.log(`Global Setup: Cleaned up ${cleaned} leftover test target(s)`)
        }

        return
      }
    } catch {
      if (i < maxRetries - 1) {
        console.log(`Global Setup: Backend not ready, retry ${i + 1}/${maxRetries}...`)
        await new Promise(r => setTimeout(r, retryDelay))
      }
    }
  }

  throw new Error(`Backend not available at ${baseURL}/api/v1/stats after ${maxRetries} attempts.`)
}

export default globalSetup
```

**Step 3: Update global-teardown.ts**

```typescript
import { FullConfig } from '@playwright/test'
import { cleanupTestTargets } from './utils/cleanup'

async function globalTeardown(_config: FullConfig) {
  const apiBaseUrl = process.env.API_BASE_URL || 'http://localhost:3100'

  console.log('Global Teardown: Cleaning up test data...')

  const cleaned = await cleanupTestTargets(apiBaseUrl)

  if (cleaned > 0) {
    console.log(`Global Teardown: Deleted ${cleaned} test target(s)`)
  } else {
    console.log('Global Teardown: No test data to clean up')
  }
}

export default globalTeardown
```

**Step 4: Run tests to verify**

```bash
docker-compose up frontend-e2e-tests
```

Expected: All 22 tests pass, cleanup logs appear in setup and teardown.

**Step 5: Commit**

```bash
git add frontend/e2e/utils/cleanup.ts frontend/e2e/global-setup.ts frontend/e2e/global-teardown.ts
git commit -m "refactor(e2e): extract shared cleanup utility

DRY: Both setup and teardown now use cleanupTestTargets().
Matches test data by name patterns: E2E, Test, or timestamps."
```

---

## Verification Checklist

- [ ] `docker-compose up frontend-e2e-tests` passes all 22 tests
- [ ] Global Setup logs cleanup of leftover data (if any)
- [ ] Global Teardown logs cleanup completion
- [ ] Running tests twice shows no orphaned data
- [ ] Manually failing a test mid-run, then re-running cleans up orphaned data
