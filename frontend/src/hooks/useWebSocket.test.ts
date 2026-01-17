import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act } from '@testing-library/react'
import { useWebSocket, useScrapeStatus } from './useWebSocket'
import { renderHookWithWebSocket, createTestQueryClient } from '@/test/test-utils'

// ============================================================================
// WebSocket Mock
// ============================================================================

class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  url: string
  readyState: number = MockWebSocket.CONNECTING

  onopen: ((event: Event) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null

  private static instances: MockWebSocket[] = []

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }

  static getLastInstance(): MockWebSocket | undefined {
    return MockWebSocket.instances[MockWebSocket.instances.length - 1]
  }

  static clearInstances(): void {
    MockWebSocket.instances = []
  }

  send = vi.fn()
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED
  })

  // Test helpers
  simulateOpen(): void {
    this.readyState = MockWebSocket.OPEN
    this.onopen?.(new Event('open'))
  }

  simulateMessage(data: unknown): void {
    this.onmessage?.(new MessageEvent('message', { data: JSON.stringify(data) }))
  }

  simulateClose(wasClean = true): void {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.(new CloseEvent('close', { wasClean }))
  }

  simulateError(): void {
    this.onerror?.(new Event('error'))
  }
}

// ============================================================================
// Tests
// ============================================================================

describe('useWebSocket', () => {
  beforeEach(() => {
    vi.stubGlobal('WebSocket', MockWebSocket)
    MockWebSocket.clearInstances()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  describe('connection state', () => {
    it('should return isConnected from context', async () => {
      const { result } = renderHookWithWebSocket(() => useWebSocket({ enabled: true }))

      const ws = MockWebSocket.getLastInstance()
      expect(ws).toBeDefined()
      expect(result.current.isConnected).toBe(false)

      act(() => {
        ws?.simulateOpen()
      })

      expect(result.current.isConnected).toBe(true)
    })

    it('should update isConnected on close', () => {
      const { result } = renderHookWithWebSocket(() => useWebSocket({ enabled: true }))

      const ws = MockWebSocket.getLastInstance()

      act(() => {
        ws?.simulateOpen()
      })

      expect(result.current.isConnected).toBe(true)

      act(() => {
        ws?.simulateClose(true)
      })

      expect(result.current.isConnected).toBe(false)
    })
  })

  describe('event handling', () => {
    it('should call onEvent callback with parsed event', () => {
      const onEvent = vi.fn()

      renderHookWithWebSocket(() => useWebSocket({ enabled: true, onEvent }))

      const ws = MockWebSocket.getLastInstance()

      act(() => {
        ws?.simulateOpen()
        ws?.simulateMessage({
          type: 'job.new',
          job_id: 'job-1',
          timestamp: '2026-01-15T10:00:00Z',
        })
      })

      expect(onEvent).toHaveBeenCalledWith({
        type: 'job.new',
        job_id: 'job-1',
        timestamp: '2026-01-15T10:00:00Z',
      })
    })

    it('should not subscribe when disabled', () => {
      const onEvent = vi.fn()

      renderHookWithWebSocket(() => useWebSocket({ enabled: false, onEvent }))

      const ws = MockWebSocket.getLastInstance()

      act(() => {
        ws?.simulateOpen()
        ws?.simulateMessage({
          type: 'job.new',
          job_id: 'job-1',
          timestamp: '2026-01-15T10:00:00Z',
        })
      })

      expect(onEvent).not.toHaveBeenCalled()
    })

    it('should not subscribe when no onEvent provided', () => {
      // This should work without errors
      const { result } = renderHookWithWebSocket(() => useWebSocket({ enabled: true }))

      const ws = MockWebSocket.getLastInstance()

      act(() => {
        ws?.simulateOpen()
        ws?.simulateMessage({
          type: 'job.new',
          job_id: 'job-1',
          timestamp: '2026-01-15T10:00:00Z',
        })
      })

      // Should still have isConnected
      expect(result.current.isConnected).toBe(true)
    })
  })

  describe('query invalidation (via context)', () => {
    it('should invalidate jobs queries on job.new event', () => {
      const queryClient = createTestQueryClient()
      queryClient.setQueryData(['jobs'], { jobs: [] })
      queryClient.setQueryData(['stats'], {})

      renderHookWithWebSocket(() => useWebSocket({ enabled: true }), { queryClient })

      const ws = MockWebSocket.getLastInstance()

      act(() => {
        ws?.simulateOpen()
        ws?.simulateMessage({
          type: 'job.new',
          job_id: 'job-1',
          timestamp: '2026-01-15T10:00:00Z',
        })
      })

      expect(queryClient.getQueryState(['jobs'])?.isInvalidated).toBe(true)
      expect(queryClient.getQueryState(['stats'])?.isInvalidated).toBe(true)
    })

    it('should invalidate targets queries on target.created event', () => {
      const queryClient = createTestQueryClient()
      queryClient.setQueryData(['targets'], [])
      queryClient.setQueryData(['stats'], {})

      renderHookWithWebSocket(() => useWebSocket({ enabled: true }), { queryClient })

      const ws = MockWebSocket.getLastInstance()

      act(() => {
        ws?.simulateOpen()
        ws?.simulateMessage({
          type: 'target.created',
          target: { id: 'target-1', name: 'New Target' },
          timestamp: '2026-01-15T10:00:00Z',
        })
      })

      expect(queryClient.getQueryState(['targets'])?.isInvalidated).toBe(true)
      expect(queryClient.getQueryState(['stats'])?.isInvalidated).toBe(true)
    })

    it('should invalidate scrape-status on scrape.started event', () => {
      const queryClient = createTestQueryClient()
      queryClient.setQueryData(['scrape-status'], { is_scraping: false })
      queryClient.setQueryData(['jobs'], { jobs: [] })
      queryClient.setQueryData(['stats'], {})

      renderHookWithWebSocket(() => useWebSocket({ enabled: true }), { queryClient })

      const ws = MockWebSocket.getLastInstance()

      act(() => {
        ws?.simulateOpen()
        ws?.simulateMessage({
          type: 'scrape.started',
          target: '@go_jobs',
          limit: 100,
          timestamp: '2026-01-15T10:00:00Z',
        })
      })

      expect(queryClient.getQueryState(['scrape-status'])?.isInvalidated).toBe(true)
      expect(queryClient.getQueryState(['jobs'])?.isInvalidated).toBe(true)
      expect(queryClient.getQueryState(['stats'])?.isInvalidated).toBe(true)
    })

    it('should invalidate stats on stats.updated event', () => {
      const queryClient = createTestQueryClient()
      queryClient.setQueryData(['stats'], {})

      renderHookWithWebSocket(() => useWebSocket({ enabled: true }), { queryClient })

      const ws = MockWebSocket.getLastInstance()

      act(() => {
        ws?.simulateOpen()
        ws?.simulateMessage({
          type: 'stats.updated',
          stats: { total_jobs: 100 },
          timestamp: '2026-01-15T10:00:00Z',
        })
      })

      expect(queryClient.getQueryState(['stats'])?.isInvalidated).toBe(true)
    })
  })
})

describe('useScrapeStatus', () => {
  beforeEach(() => {
    vi.stubGlobal('WebSocket', MockWebSocket)
    MockWebSocket.clearInstances()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.useRealTimers()
  })

  it('should start with default values', () => {
    const { result } = renderHookWithWebSocket(() => useScrapeStatus())

    expect(result.current.isScraping).toBe(false)
    expect(result.current.target).toBeUndefined()
    expect(result.current.progress).toBeUndefined()
  })

  it('should update state on scrape.started event', () => {
    const { result } = renderHookWithWebSocket(() => useScrapeStatus())

    const ws = MockWebSocket.getLastInstance()

    act(() => {
      ws?.simulateOpen()
      ws?.simulateMessage({
        type: 'scrape.started',
        target: '@go_jobs',
        limit: 100,
        timestamp: '2026-01-15T10:00:00Z',
      })
    })

    expect(result.current.isScraping).toBe(true)
    expect(result.current.target).toBe('@go_jobs')
    expect(result.current.progress).toBeUndefined()
  })

  it('should update progress on scrape.progress event', () => {
    const { result } = renderHookWithWebSocket(() => useScrapeStatus())

    const ws = MockWebSocket.getLastInstance()

    act(() => {
      ws?.simulateOpen()
      ws?.simulateMessage({
        type: 'scrape.started',
        target: '@go_jobs',
        limit: 100,
        timestamp: '2026-01-15T10:00:00Z',
      })
      ws?.simulateMessage({
        type: 'scrape.progress',
        target: '@go_jobs',
        processed: 50,
        new_jobs: 10,
        timestamp: '2026-01-15T10:01:00Z',
      })
    })

    expect(result.current.isScraping).toBe(true)
    expect(result.current.progress).toEqual({
      processed: 50,
      total: 50, // Same as processed since API doesn't return total
      newJobs: 10,
    })
  })

  it('should reset state on scrape.completed event', () => {
    const { result } = renderHookWithWebSocket(() => useScrapeStatus())

    const ws = MockWebSocket.getLastInstance()

    act(() => {
      ws?.simulateOpen()
      ws?.simulateMessage({
        type: 'scrape.started',
        target: '@go_jobs',
        limit: 100,
        timestamp: '2026-01-15T10:00:00Z',
      })
      ws?.simulateMessage({
        type: 'scrape.completed',
        target: '@go_jobs',
        total: 100,
        new: 25,
        timestamp: '2026-01-15T10:05:00Z',
      })
    })

    expect(result.current.isScraping).toBe(false)
    expect(result.current.target).toBeUndefined()
    expect(result.current.progress).toBeUndefined()
  })

  it('should reset state on scrape.failed event', () => {
    const { result } = renderHookWithWebSocket(() => useScrapeStatus())

    const ws = MockWebSocket.getLastInstance()

    act(() => {
      ws?.simulateOpen()
      ws?.simulateMessage({
        type: 'scrape.started',
        target: '@go_jobs',
        limit: 100,
        timestamp: '2026-01-15T10:00:00Z',
      })
      ws?.simulateMessage({
        type: 'scrape.failed',
        target: '@go_jobs',
        error: 'Connection lost',
        timestamp: '2026-01-15T10:05:00Z',
      })
    })

    expect(result.current.isScraping).toBe(false)
    expect(result.current.target).toBeUndefined()
  })

  it('should reset state on scrape.cancelled event', () => {
    const { result } = renderHookWithWebSocket(() => useScrapeStatus())

    const ws = MockWebSocket.getLastInstance()

    act(() => {
      ws?.simulateOpen()
      ws?.simulateMessage({
        type: 'scrape.started',
        target: '@go_jobs',
        limit: 100,
        timestamp: '2026-01-15T10:00:00Z',
      })
      ws?.simulateMessage({
        type: 'scrape.cancelled',
        target: '@go_jobs',
        timestamp: '2026-01-15T10:02:00Z',
      })
    })

    expect(result.current.isScraping).toBe(false)
    expect(result.current.target).toBeUndefined()
  })

  it('should not subscribe when disabled', () => {
    const { result } = renderHookWithWebSocket(() => useScrapeStatus(false))

    const ws = MockWebSocket.getLastInstance()

    act(() => {
      ws?.simulateOpen()
      ws?.simulateMessage({
        type: 'scrape.started',
        target: '@go_jobs',
        limit: 100,
        timestamp: '2026-01-15T10:00:00Z',
      })
    })

    // State should remain at defaults since subscription is disabled
    expect(result.current.isScraping).toBe(false)
    expect(result.current.target).toBeUndefined()
  })

  it('should expose isConnected state', () => {
    const { result } = renderHookWithWebSocket(() => useScrapeStatus())

    expect(result.current.isConnected).toBe(false)

    const ws = MockWebSocket.getLastInstance()

    act(() => {
      ws?.simulateOpen()
    })

    expect(result.current.isConnected).toBe(true)
  })
})
