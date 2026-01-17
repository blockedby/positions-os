import { Card, Spinner } from '@/components/ui'
import { useScrapeStatus } from '@/hooks/useScrapeStatus'

export interface ScrapeStatusProps {
  className?: string
}

export const ScrapeStatus = ({ className }: ScrapeStatusProps) => {
  const { data, isLoading } = useScrapeStatus()

  if (isLoading) {
    return (
      <Card className={className}>
        <div className="scrape-status-loading">
          <Spinner size="sm" />
          <span>Checking status...</span>
        </div>
      </Card>
    )
  }

  const status = data

  return (
    <Card className={className}>
      <div className="scrape-status">
        <h3>Scraping Status</h3>
        {status?.is_scraping ? (
          <div className="scrape-status-active">
            <Spinner size="sm" />
            <div className="scrape-status-info">
              <span className="scrape-target">{status.target}</span>
              <span className="scrape-progress">
                {status.processed} processed, {status.new_jobs} new jobs
              </span>
            </div>
          </div>
        ) : (
          <div className="scrape-status-idle">
            <span className="status-badge status-idle">Idle</span>
            <span className="text-muted">No active scraping</span>
          </div>
        )}
      </div>
    </Card>
  )
}
