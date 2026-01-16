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
