import { useEffect, useState, useCallback } from 'react'
import { useWebSocketContext } from './useWebSocketContext'
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
