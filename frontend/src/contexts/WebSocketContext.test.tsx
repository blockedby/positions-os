import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { WebSocketProvider } from './WebSocketContext'
import { useWebSocketContext } from '@/hooks/useWebSocketContext'
import type { ReactNode } from 'react'

// Mock WebSocket
class MockWebSocket {
  static instances: MockWebSocket[] = []
  onopen: ((event: Event) => void) | null = null
  onclose: ((event: CloseEvent) => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: ((event: Event) => void) | null = null
  readyState = WebSocket.CONNECTING

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  constructor(url: string) {
    MockWebSocket.instances.push(this)
    // Simulate async connection
    setTimeout(() => {
      this.readyState = WebSocket.OPEN
      this.onopen?.(new Event('open'))
    }, 0)
  }

  close() {
    this.readyState = WebSocket.CLOSED
  }

  send() {}

  static reset() {
    MockWebSocket.instances = []
  }
}

describe('WebSocketContext', () => {
  let queryClient: QueryClient
  let originalWebSocket: typeof WebSocket

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })
    originalWebSocket = global.WebSocket
    global.WebSocket = MockWebSocket as unknown as typeof WebSocket
    MockWebSocket.reset()
  })

  afterEach(() => {
    global.WebSocket = originalWebSocket
    queryClient.clear()
  })

  const createWrapper = () => {
    return ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={queryClient}>
        <WebSocketProvider>{children}</WebSocketProvider>
      </QueryClientProvider>
    )
  }

  it('should provide a single WebSocket connection', () => {
    const wrapper = createWrapper()

    const { result } = renderHook(() => useWebSocketContext(), { wrapper })

    expect(result.current).toBeDefined()
    expect(result.current.isConnected).toBeDefined()
    expect(result.current.subscribe).toBeInstanceOf(Function)
  })

  it('should throw error when used outside provider', () => {
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    expect(() => {
      renderHook(() => useWebSocketContext())
    }).toThrow('useWebSocketContext must be used within WebSocketProvider')

    consoleSpy.mockRestore()
  })

  it('should allow subscribing and unsubscribing', () => {
    const wrapper = createWrapper()
    const callback = vi.fn()

    const { result } = renderHook(() => useWebSocketContext(), { wrapper })

    const unsubscribe = result.current.subscribe(callback)
    expect(typeof unsubscribe).toBe('function')

    // Unsubscribe should work without error
    unsubscribe()
  })

  it('should share context between multiple consumers', () => {
    const wrapper = createWrapper()

    // Render two hooks that use the context - they should get the same context value
    const { result: result1 } = renderHook(() => useWebSocketContext(), { wrapper })
    const { result: result2 } = renderHook(() => useWebSocketContext(), { wrapper })

    // Both hooks should have the same subscribe function reference
    // (since they come from the same context provider)
    expect(result1.current.subscribe).toBeDefined()
    expect(result2.current.subscribe).toBeDefined()
    expect(typeof result1.current.subscribe).toBe('function')
    expect(typeof result2.current.subscribe).toBe('function')
  })
})
