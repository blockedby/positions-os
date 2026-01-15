import { useEffect, useRef, useCallback, useState } from 'react'
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

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      try {
        const wsEvent: WSEvent = JSON.parse(event.data)
        onEvent?.(wsEvent)

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
            queryClient.invalidateQueries({ queryKey: queryKeys.targets() })
            queryClient.invalidateQueries({ queryKey: queryKeys.stats() })
            break

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
    },
    [onEvent, queryClient],
  )

  const handleOpen = useCallback(() => {
    setIsConnected(true)
  }, [])

  const handleClose = useCallback(() => {
    setIsConnected(false)
  }, [])

  useEffect(() => {
    if (!enabled) {
      return
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
  }, [enabled, handleMessage, handleOpen, handleClose])

  return {
    isConnected,
    client: clientRef.current,
  }
}

// ============================================================================
// Scrape Status Hook
// ============================================================================

export function useScrapeStatus(enabled = true) {
  const { isConnected } = useWebSocket({ enabled })
  const [isScraping, setIsScraping] = useState(false)
  const [scrapeTarget, setScrapeTarget] = useState<string>()
  const [scrapeProgress, setScrapeProgress] = useState<{
    processed: number
    total: number
    newJobs: number
  }>()

  // Listen for scrape events
  useWebSocket({
    enabled,
    onEvent: (event: WSEvent) => {
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
    },
  })

  return {
    isScraping,
    target: scrapeTarget,
    progress: scrapeProgress,
    isConnected,
  }
}
