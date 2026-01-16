import { useState } from 'react'
import type { Target, ScrapeRequest } from '@/lib/types'
import { api } from '@/lib/api'
import { useWebSocket, useScrapeStatus } from '@/hooks/useWebSocket'
import { TargetList, TelegramAuth } from '@/components/settings'
import type { ScrapeOptions } from '@/components/settings'
import { Card, Button, Spinner } from '@/components/ui'

export default function Settings() {
  const [scrapeError, setScrapeError] = useState<string>('')
  const { isScraping, target: scrapingTarget, progress } = useScrapeStatus()

  // Enable real-time updates
  useWebSocket({ enabled: true })

  const handleScrape = async (target: Target, options?: ScrapeOptions) => {
    setScrapeError('')
    try {
      const request: ScrapeRequest = {
        channel: target.url,
        limit: options?.limit,
        until: options?.until,
      }
      await api.startScrape(request)
    } catch (err) {
      setScrapeError(err instanceof Error ? err.message : 'Failed to start scrape')
    }
  }

  const handleCancelScrape = async () => {
    try {
      await api.stopScrape()
    } catch {
      // Ignore cancel errors
    }
  }

  return (
    <div className="settings-page">
      <h1>Settings</h1>
      <p className="text-muted mb-6">
        Configure scraping targets and application settings.
      </p>

      {/* Scrape Status */}
      {isScraping && (
        <Card className="scrape-status mb-6">
          <div className="scrape-status-content">
            <Spinner size="sm" />
            <div className="scrape-status-info">
              <p className="scrape-status-title">Scraping {scrapingTarget}</p>
              {progress && (
                <p className="text-xs text-muted">
                  Processed: {progress.processed} | New jobs: {progress.newJobs}
                </p>
              )}
            </div>
            <Button variant="danger" size="sm" onClick={handleCancelScrape}>
              Cancel
            </Button>
          </div>
        </Card>
      )}

      {scrapeError && (
        <Card className="scrape-error mb-6">
          <p className="text-danger">{scrapeError}</p>
        </Card>
      )}

      {/* Targets Section */}
      <section className="settings-section mb-6">
        <TargetList onScrape={handleScrape} />
      </section>

      {/* Telegram Auth Section */}
      <section className="settings-section">
        <TelegramAuth />
      </section>
    </div>
  )
}
