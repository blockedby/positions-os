# TDD Error-Fixing Guide: Red-Green-Refactor

**Related PR:** [#12 - fix(frontend): prevent WebSocket infinite reconnection loop](https://github.com/blockedby/positions-os/pull/12)

## Overview

This guide applies the **Red-Green-Refactor** TDD pattern to systematically fix errors discovered during Playwright E2E testing. Following Uncle Bob's three laws of TDD ensures every fix is test-driven and verified.

## The Three Laws of TDD

```xml
<law order="1">
  You are not allowed to write any production code unless it is to make a failing
  unit test pass.
</law>
<law order="2">
  You are not allowed to write any more of a unit test than is sufficient to fail;
  and compilation failures are failures.
</law>
<law order="3">
  You are not allowed to write any more production code than is sufficient to pass
  the one failing unit test.
</law>
```

---

## Error-Fixing Workflow

### Phase 1: RED - Write a Failing Test

When you encounter an error in E2E testing:

1. **Isolate the failure** - Identify the minimal reproduction case
2. **Write a test that fails** - The test proves the bug exists
3. **Verify it fails for the right reason** - Not due to test setup issues

#### Example: WebSocket Infinite Loop Bug

```typescript
// e2e/websocket.spec.ts
test('should not reconnect WebSocket infinitely', async ({ page }) => {
  const wsConnections: number[] = []

  page.on('websocket', () => {
    wsConnections.push(Date.now())
  })

  await page.goto('/settings')
  await page.waitForTimeout(5000)

  // RED: This test FAILS before the fix
  // Multiple connections happen within seconds
  expect(wsConnections.length).toBeLessThanOrEqual(1)
})
```

**Expected Output (Before Fix):**
```
Error: expect(received).toBeLessThanOrEqual(expected)
Expected: <= 1
Received: 47
```

### Phase 2: GREEN - Make the Test Pass

Write the **minimum code** necessary to make the failing test pass.

#### The Fix (from PR #12)

```typescript
// frontend/src/hooks/useWebSocket.ts

// BEFORE (causes infinite loop):
useEffect(() => {
  const client = new WSClient({ onMessage: handleMessage })
  client.connect()
  return () => client.disconnect()
}, [enabled, handleMessage, handleOpen, handleClose]) // Problem!

// AFTER (minimal fix - use refs):
const onEventRef = useRef(onEvent)
useEffect(() => { onEventRef.current = onEvent }, [onEvent])

useEffect(() => {
  const handleMessage = (event) => {
    onEventRef.current?.(event)
  }
  const client = new WSClient({ onMessage: handleMessage })
  client.connect()
  return () => client.disconnect()
}, [enabled]) // Only 'enabled' triggers reconnection
```

**Run the test again:**
```bash
bunx playwright test websocket.spec.ts
# PASSED
```

### Phase 3: REFACTOR - Improve Without Breaking

Now that tests are green, improve code quality:

1. **Remove duplication**
2. **Improve naming**
3. **Extract methods**
4. **Clean up design**

**All tests must stay green throughout refactoring.**

#### Refactoring Example

```typescript
// Consolidate duplicate useWebSocket calls in useScrapeStatus

// BEFORE (two useWebSocket calls):
export function useScrapeStatus(enabled = true) {
  const { isConnected } = useWebSocket({ enabled })
  // ...
  useWebSocket({
    enabled,
    onEvent: (event) => { /* inline handler */ }
  })
}

// AFTER (single call with memoized handler):
export function useScrapeStatus(enabled = true) {
  const handleScrapeEvent = useCallback((event: WSEvent) => {
    // Handle event
  }, [])

  const { isConnected } = useWebSocket({
    enabled,
    onEvent: handleScrapeEvent,
  })
}
```

**Verify tests still pass:**
```bash
bunx playwright test
# All tests PASSED
```

---

## Error Categories and TDD Approach

### Category 1: WebSocket Errors

| Error | RED Test | GREEN Fix | REFACTOR |
|-------|----------|-----------|----------|
| Infinite reconnection | Count connections over 5s | Use refs for callbacks | Extract stable callback pattern |
| "Connection closed" spam | Monitor console errors | Proper cleanup in useEffect | Add error boundary |
| State update after unmount | Track component lifecycle | Add mounted check | Use useIsMounted hook |

### Category 2: API Errors

| Error | RED Test | GREEN Fix | REFACTOR |
|-------|----------|-----------|----------|
| Empty body instead of `[]` | Assert `Array.isArray(response)` | Return `[]` explicitly | Add `respondJSON` helper |
| 500 on invalid JSON | Send malformed JSON | Add JSON decode error handling | Centralize error responses |
| Missing Content-Type | Check response headers | Set header in handler | Use middleware |

### Category 3: UI Errors

| Error | RED Test | GREEN Fix | REFACTOR |
|-------|----------|-----------|----------|
| Form validation missing | Submit empty form | Add validation checks | Extract validator function |
| Button state incorrect | Click disabled button | Disable during loading | Use loading state hook |
| List not updating | Create item via API | Invalidate query cache | Add optimistic updates |

---

## Step-by-Step: Fixing a Test Failure

### 1. Identify the Failing Test

```bash
$ bunx playwright test targets.spec.ts

  FAIL  targets.spec.ts > should return empty array for empty targets
  Error: Expected array but got null
```

### 2. Write the Minimal Failing Unit Test

```go
// internal/web/handlers/targets_test.go
func TestTargetsHandler_List_ReturnsEmptyArray(t *testing.T) {
    mockRepo := new(MockTargetsRepository)
    handler, _ := setupTargetsHandler(t, mockRepo)

    // Return nil from repo (no targets)
    mockRepo.On("List", mock.Anything).Return(nil, nil)

    req := httptest.NewRequest("GET", "/targets", nil)
    rec := httptest.NewRecorder()
    handler.List(rec, req)

    // RED: This fails before fix
    assert.Equal(t, "[]", strings.TrimSpace(rec.Body.String()))
}
```

### 3. Run Test - Confirm RED

```bash
$ go test ./internal/web/handlers/... -run TestTargetsHandler_List_ReturnsEmptyArray
--- FAIL: TestTargetsHandler_List_ReturnsEmptyArray
    Expected: []
    Actual:   null
```

### 4. Write Minimal Fix - Go GREEN

```go
// internal/web/handlers/targets.go
func (h *TargetsHandler) List(w http.ResponseWriter, r *http.Request) {
    targets, err := h.repo.List(r.Context())
    if err != nil {
        respondError(w, http.StatusInternalServerError, err.Error())
        return
    }

    // GREEN: Ensure empty array, not null
    if targets == nil {
        targets = []repository.ScrapingTarget{}
    }

    respondJSON(w, http.StatusOK, targets)
}
```

### 5. Run Test - Confirm GREEN

```bash
$ go test ./internal/web/handlers/... -run TestTargetsHandler_List_ReturnsEmptyArray
PASS
```

### 6. Refactor If Needed

```go
// Extract helper to ensure non-null slices
func ensureSlice[T any](slice []T) []T {
    if slice == nil {
        return []T{}
    }
    return slice
}

// Use in handler
respondJSON(w, http.StatusOK, ensureSlice(targets))
```

### 7. Run All Tests - Confirm Still GREEN

```bash
$ go test ./...
$ bunx playwright test
# All tests PASS
```

---

## Checklist for Each Error Fix

```markdown
## Error Fix Checklist

- [ ] **RED Phase**
  - [ ] Error identified and isolated
  - [ ] Failing test written
  - [ ] Test fails for the right reason
  - [ ] Test is minimal (no extra assertions)

- [ ] **GREEN Phase**
  - [ ] Minimal code written to pass test
  - [ ] No premature optimization
  - [ ] No extra features added
  - [ ] Test now passes

- [ ] **REFACTOR Phase**
  - [ ] Code duplication removed
  - [ ] Names are clear and descriptive
  - [ ] Functions are single-purpose
  - [ ] All tests still pass
```

---

## Benefits of This Approach

1. **Near-100% test coverage** - Every line of production code is demanded by a test
2. **Minimal waste** - No code written that isn't needed for the current fix
3. **Instant feedback** - The cycle is fast; you always know where a bug was introduced
4. **Better design** - Writing tests first forces thinking about the interface, not implementation

---

## Related Documents

- [Playwright Test Plan](./playwright-test-plan.md) - Full E2E test specifications
- [TDD Red-Green-Refactor Pattern](../tdd-red-green-refactor-order.xml) - Original pattern reference
