import { useEffect, useRef, useState, useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { WSClient } from '@/lib/api'
import { queryKeys } from '@/lib/types'
import type { WSEvent } from '@/lib/types'

interface WebSocketOptions {
  enabled?: boolean
  onEvent?: (event: WSEvent) => void
}

export function useWebSocket({ enabled = true, onEvent }: WebSocketOptions = {}) {
  const clientRef = useRef<WSClient | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const queryClient = useQueryClient()

  // Store callbacks in refs to avoid triggering reconnections
  const onEventRef = useRef(onEvent)
  const queryClientRef = useRef(queryClient)

  // Update refs on every render (doesn't cause re-renders)
  useEffect(() => {
    onEventRef.current = onEvent
  }, [onEvent])

  useEffect(() => {
    queryClientRef.current = queryClient
  }, [queryClient])

  useEffect(() => {
    if (!enabled) {
      return
    }

    const handleMessage = (event: MessageEvent) => {
      try {
        const wsEvent: WSEvent = JSON.parse(event.data)
        onEventRef.current?.(wsEvent)

        const qc = queryClientRef.current

        // Invalidate queries based on event type
        switch (wsEvent.type) {
          case 'job.new':
          case 'job.updated':
          case 'job.analyzed':
            qc.invalidateQueries({ queryKey: ['jobs'] })
            qc.invalidateQueries({ queryKey: queryKeys.job(wsEvent.job_id) })
            qc.invalidateQueries({ queryKey: queryKeys.stats() })
            break

          case 'target.created':
          case 'target.updated':
            qc.invalidateQueries({ queryKey: queryKeys.targets() })
            qc.invalidateQueries({ queryKey: queryKeys.stats() })
            break

          case 'target.deleted':
            qc.invalidateQueries({ queryKey: queryKeys.targets() })
            qc.invalidateQueries({ queryKey: queryKeys.stats() })
            break

          case 'scrape.started':
          case 'scrape.progress':
          case 'scrape.completed':
            qc.invalidateQueries({ queryKey: ['scrape-status'] })
            qc.invalidateQueries({ queryKey: ['jobs'] })
            qc.invalidateQueries({ queryKey: queryKeys.stats() })
            break

          case 'stats.updated':
            qc.invalidateQueries({ queryKey: queryKeys.stats() })
            break

          case 'tg_qr':
          case 'tg_auth_success':
          case 'error':
            // Auth events are handled by TelegramAuth component via onEvent callback
            // No query invalidation needed here
            break
        }
      } catch (error) {
        console.error('Failed to parse WebSocket event:', error)
      }
    }

    const handleOpen = () => {
      setIsConnected(true)
    }

    const handleClose = () => {
      setIsConnected(false)
    }

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
  }, [enabled]) // Only depend on 'enabled' - callbacks are accessed via refs

  return {
    isConnected,
    client: clientRef.current,
  }
}

// ============================================================================
// Scrape Status Hook
// ============================================================================

export function useScrapeStatus(enabled = true) {
  const [isScraping, setIsScraping] = useState(false)
  const [scrapeTarget, setScrapeTarget] = useState<string>()
  const [scrapeProgress, setScrapeProgress] = useState<{
    processed: number
    total: number
    newJobs: number
  }>()

  // Memoize the event handler to prevent infinite re-renders
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
          total: event.processed, // API doesn't return total
          newJobs: event.new_jobs,
        })
        break

      case 'scrape.completed':
        setIsScraping(false)
        setScrapeTarget(undefined)
        setScrapeProgress(undefined)
        break

      case 'scrape.failed':
      case 'scrape.cancelled':
        setIsScraping(false)
        setScrapeTarget(undefined)
        setScrapeProgress(undefined)
        break
    }
  }, [])

  // Single WebSocket connection with memoized callback
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
