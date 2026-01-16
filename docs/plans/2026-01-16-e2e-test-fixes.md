# E2E Test Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all failing E2E tests by addressing WebSocket multiple connections, API response issues, and UI test selectors.

**Architecture:** Centralize WebSocket connection management at App level using React Context to prevent multiple connections. Ensure API responses match test expectations. Update E2E tests to use flexible selectors.

**Tech Stack:** React 19, TypeScript, TanStack Query, Playwright, Go Chi router

---

## Summary of Issues (from PR #15 Review)

1. **WebSocket Multiple Connections**: Multiple components calling `useWebSocket()` independently create separate connections
2. **API Response Schema Mismatches**: Some fields undefined in responses
3. **UI Test Timeouts**: Tests waiting for elements that don't appear as expected

---

## Task 1: Create WebSocket Context Provider

**Files:**
- Create: `frontend/src/contexts/WebSocketContext.tsx`
- Modify: `frontend/src/App.tsx`

**Step 1: Write the failing test**

```typescript
// frontend/src/contexts/WebSocketContext.test.tsx
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { WebSocketProvider, useWebSocketContext } from './WebSocketContext'

describe('WebSocketContext', () => {
  it('should provide a single WebSocket connection', () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <WebSocketProvider>{children}</WebSocketProvider>
    )

    const { result } = renderHook(() => useWebSocketContext(), { wrapper })

    expect(result.current).toBeDefined()
    expect(result.current.isConnected).toBeDefined()
    expect(result.current.subscribe).toBeInstanceOf(Function)
  })

  it('should throw error when used outside provider', () => {
    expect(() => {
      renderHook(() => useWebSocketContext())
    }).toThrow()
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bun run test -- src/contexts/WebSocketContext.test.tsx`
Expected: FAIL with "Cannot find module"

**Step 3: Write the WebSocket Context implementation**

```typescript
// frontend/src/contexts/WebSocketContext.tsx
import { createContext, useContext, useEffect, useRef, useState, useCallback, type ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { WSClient } from '@/lib/api'
import { queryKeys } from '@/lib/types'
import type { WSEvent } from '@/lib/types'

type EventSubscriber = (event: WSEvent) => void

interface WebSocketContextValue {
  isConnected: boolean
  subscribe: (callback: EventSubscriber) => () => void
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null)

export function useWebSocketContext() {
  const context = useContext(WebSocketContext)
  if (!context) {
    throw new Error('useWebSocketContext must be used within WebSocketProvider')
  }
  return context
}

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const clientRef = useRef<WSClient | null>(null)
  const subscribersRef = useRef<Set<EventSubscriber>>(new Set())
  const [isConnected, setIsConnected] = useState(false)
  const queryClient = useQueryClient()

  // Subscribe function that returns unsubscribe
  const subscribe = useCallback((callback: EventSubscriber) => {
    subscribersRef.current.add(callback)
    return () => {
      subscribersRef.current.delete(callback)
    }
  }, [])

  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      try {
        const wsEvent: WSEvent = JSON.parse(event.data)

        // Notify all subscribers
        subscribersRef.current.forEach((callback) => callback(wsEvent))

        // Invalidate queries based on event type
        switch (wsEvent.type) {
          case 'job.new':
          case 'job.updated':
          case 'job.analyzed':
            queryClient.invalidateQueries({ queryKey: ['jobs'] })
            queryClient.invalidateQueries({ queryKey: queryKeys.job(wsEvent.job_id) })
            queryClient.invalidateQueries({ queryKey: queryKeys.stats() })
            break

          case 'target.created':
          case 'target.updated':
          case 'target.deleted':
            queryClient.invalidateQueries({ queryKey: queryKeys.targets() })
            queryClient.invalidateQueries({ queryKey: queryKeys.stats() })
            break

          case 'scrape.started':
          case 'scrape.progress':
          case 'scrape.completed':
            queryClient.invalidateQueries({ queryKey: ['scrape-status'] })
            queryClient.invalidateQueries({ queryKey: ['jobs'] })
            queryClient.invalidateQueries({ queryKey: queryKeys.stats() })
            break

          case 'stats.updated':
            queryClient.invalidateQueries({ queryKey: queryKeys.stats() })
            break
        }
      } catch (error) {
        console.error('Failed to parse WebSocket event:', error)
      }
    }

    const handleOpen = () => setIsConnected(true)
    const handleClose = () => setIsConnected(false)

    const client = new WSClient({
      onMessage: handleMessage,
      onOpen: handleOpen,
      onClose: handleClose,
    })

    clientRef.current = client
    client.connect()

    return () => {
      client.disconnect()
    }
  }, [queryClient])

  return (
    <WebSocketContext.Provider value={{ isConnected, subscribe }}>
      {children}
    </WebSocketContext.Provider>
  )
}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bun run test -- src/contexts/WebSocketContext.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/contexts/
git commit -m "feat(frontend): add WebSocket context provider for single connection

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 2: Update App.tsx to Use WebSocket Provider

**Files:**
- Modify: `frontend/src/App.tsx`

**Step 1: Update App.tsx**

```typescript
// frontend/src/App.tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/query-client'
import { WebSocketProvider } from '@/contexts/WebSocketContext'
import { Sidebar } from '@/components/layout/Sidebar'
import { Main } from '@/components/layout/Main'
import Dashboard from '@/pages/Dashboard'
import Jobs from '@/pages/Jobs'
import Settings from '@/pages/Settings'

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <WebSocketProvider>
        <BrowserRouter>
          <div className="app-layout">
            <Sidebar />
            <Routes>
              <Route
                path="/"
                element={
                  <Main>
                    <Dashboard />
                  </Main>
                }
              />
              <Route
                path="/jobs"
                element={
                  <Main>
                    <Jobs />
                  </Main>
                }
              />
              <Route
                path="/settings"
                element={
                  <Main>
                    <Settings />
                  </Main>
                }
              />
            </Routes>
          </div>
        </BrowserRouter>
      </WebSocketProvider>
    </QueryClientProvider>
  )
}

export default App
```

**Step 2: Run tests to verify no regressions**

Run: `cd frontend && bun run test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add frontend/src/App.tsx
git commit -m "feat(frontend): wrap App in WebSocketProvider

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 3: Refactor useWebSocket Hook to Use Context

**Files:**
- Modify: `frontend/src/hooks/useWebSocket.ts`
- Modify: `frontend/src/hooks/useWebSocket.test.ts`

**Step 1: Update the hook to use context**

```typescript
// frontend/src/hooks/useWebSocket.ts
import { useEffect, useState, useCallback } from 'react'
import { useWebSocketContext } from '@/contexts/WebSocketContext'
import type { WSEvent } from '@/lib/types'

interface WebSocketOptions {
  enabled?: boolean
  onEvent?: (event: WSEvent) => void
}

export function useWebSocket({ enabled = true, onEvent }: WebSocketOptions = {}) {
  const context = useWebSocketContext()

  useEffect(() => {
    if (!enabled || !onEvent) return
    return context.subscribe(onEvent)
  }, [enabled, onEvent, context])

  return {
    isConnected: context.isConnected,
  }
}

// Scrape Status Hook remains the same but uses context-based useWebSocket
export function useScrapeStatus(enabled = true) {
  const [isScraping, setIsScraping] = useState(false)
  const [scrapeTarget, setScrapeTarget] = useState<string>()
  const [scrapeProgress, setScrapeProgress] = useState<{
    processed: number
    total: number
    newJobs: number
  }>()

  const handleScrapeEvent = useCallback((event: WSEvent) => {
    switch (event.type) {
      case 'scrape.started':
        setIsScraping(true)
        setScrapeTarget(event.target)
        setScrapeProgress(undefined)
        break

      case 'scrape.progress':
        setScrapeProgress({
          processed: event.processed,
          total: event.processed,
          newJobs: event.new_jobs,
        })
        break

      case 'scrape.completed':
      case 'scrape.failed':
      case 'scrape.cancelled':
        setIsScraping(false)
        setScrapeTarget(undefined)
        setScrapeProgress(undefined)
        break
    }
  }, [])

  const { isConnected } = useWebSocket({
    enabled,
    onEvent: handleScrapeEvent,
  })

  return {
    isScraping,
    target: scrapeTarget,
    progress: scrapeProgress,
    isConnected,
  }
}
```

**Step 2: Update tests to provide context**

Update `frontend/src/hooks/useWebSocket.test.ts` to wrap hooks in `WebSocketProvider`.

**Step 3: Run tests**

Run: `cd frontend && bun run test`
Expected: All tests pass

**Step 4: Commit**

```bash
git add frontend/src/hooks/useWebSocket.ts frontend/src/hooks/useWebSocket.test.ts
git commit -m "refactor(frontend): useWebSocket now uses centralized context

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 4: Remove Duplicate useWebSocket Calls from Pages

**Files:**
- Modify: `frontend/src/pages/Dashboard.tsx`
- Modify: `frontend/src/pages/Jobs.tsx`
- Modify: `frontend/src/pages/Settings.tsx`

**Step 1: Update Dashboard.tsx**

Remove direct `useWebSocket()` call - context handles it.

```typescript
// frontend/src/pages/Dashboard.tsx
import { StatsCards, RecentJobs } from '@/components/dashboard'

export default function Dashboard() {
  // WebSocket connection is managed by App-level WebSocketProvider
  return (
    <div className="dashboard">
      <h1>Dashboard</h1>
      <StatsCards />
      <RecentJobs />
    </div>
  )
}
```

**Step 2: Update Jobs.tsx**

Remove direct `useWebSocket()` call.

**Step 3: Update Settings.tsx**

Remove duplicate `useWebSocket()` call - keep only `useScrapeStatus()`.

```typescript
// Remove: useWebSocket({ enabled: true })
// Keep: const { isScraping, target, progress } = useScrapeStatus()
```

**Step 4: Run tests**

Run: `cd frontend && bun run test`
Expected: All tests pass

**Step 5: Commit**

```bash
git add frontend/src/pages/
git commit -m "refactor(frontend): remove duplicate WebSocket connections from pages

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 5: Fix TelegramAuth WebSocket Usage

**Files:**
- Modify: `frontend/src/components/settings/TelegramAuth.tsx`

**Step 1: Update TelegramAuth to use context subscription**

The component needs to subscribe to specific events via the context.

```typescript
// Update to use context-based subscription
const context = useWebSocketContext()

useEffect(() => {
  return context.subscribe(handleWSEvent)
}, [context, handleWSEvent])
```

**Step 2: Run tests**

Run: `cd frontend && bun run test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add frontend/src/components/settings/TelegramAuth.tsx
git commit -m "refactor(frontend): TelegramAuth uses WebSocket context

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 6: Fix Backend Delete Handler (if needed)

**Files:**
- Verify: `internal/web/handlers/targets.go`

**Step 1: Verify Delete returns 204**

Check that `Delete` handler returns `http.StatusNoContent` (204).

Current code at line 158: `w.WriteHeader(http.StatusNoContent)` - Already correct.

**Step 2: Run backend tests**

Run: `go test ./internal/web/handlers/...`
Expected: All tests pass

---

## Task 7: Run All Unit Tests

**Step 1: Run Go tests**

Run: `go test ./...`
Expected: All tests pass

**Step 2: Run Frontend tests**

Run: `cd frontend && bun run test`
Expected: All tests pass

**Step 3: Commit any fixes**

```bash
git add .
git commit -m "test: ensure all unit tests pass

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 8: E2E Test Adjustments (if needed)

**Files:**
- Potentially modify: `frontend/e2e/websocket.spec.ts`
- Potentially modify: `frontend/e2e/targets.spec.ts`
- Potentially modify: `frontend/e2e/api.spec.ts`

**Step 1: Start services for E2E testing**

```bash
docker compose up -d
cd frontend && bun run dev &
```

**Step 2: Run E2E tests**

Run: `cd frontend && bun run test:e2e`

**Step 3: Fix any remaining issues based on actual failures**

- Update selectors if UI elements have different attributes
- Adjust timeouts if needed
- Update expected values based on actual API responses

**Step 4: Commit E2E test fixes**

```bash
git add frontend/e2e/
git commit -m "fix(e2e): align tests with WebSocket context architecture

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Task 9: Final Verification

**Step 1: Run all tests**

```bash
go test ./...
cd frontend && bun run test
cd frontend && bun run test:e2e
```

**Step 2: Update PR**

```bash
git push origin fix/e2e-test-alignment
```

---

## Success Criteria

- [ ] All Go unit tests pass
- [ ] All frontend unit tests pass (212 tests)
- [ ] All E2E tests pass (22 tests)
- [ ] WebSocket connections: maximum 2 (one + StrictMode remount)
- [ ] No WebSocket reconnection loops
- [ ] API responses match expected schemas
