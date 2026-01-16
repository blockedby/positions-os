import { useState } from 'react'
import type { Target, TargetMetadata } from '@/lib/types'
import { Card, Button, Badge, Spinner, type BadgeStatus } from '@/components/ui'
import { useTargets, useDeleteTarget } from '@/hooks/useTargets'
import { TargetForm } from './TargetForm'

export interface ScrapeOptions {
  limit?: number
  until?: string
}

export interface TargetListProps {
  onScrape?: (target: Target, options?: ScrapeOptions) => void
}

const typeToBadge: Record<string, BadgeStatus> = {
  TG_CHANNEL: 'analyzed',
  TG_GROUP: 'analyzed',
  TG_FORUM: 'analyzed',
  HH_SEARCH: 'interested',
  LINKEDIN_SEARCH: 'interested',
}

export const TargetList = ({ onScrape }: TargetListProps) => {
  const { data: targets, isLoading, error } = useTargets()
  const deleteTarget = useDeleteTarget()
  const [editingTarget, setEditingTarget] = useState<Target | null>(null)
  const [showForm, setShowForm] = useState(false)

  if (isLoading) {
    return (
      <Card className="target-list">
        <div className="target-list-loading">
          <Spinner size="lg" />
        </div>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className="target-list">
        <p className="text-muted">Failed to load targets</p>
      </Card>
    )
  }

  const handleDelete = async (id: string) => {
    if (window.confirm('Are you sure you want to delete this target?')) {
      await deleteTarget.mutateAsync(id)
    }
  }

  const handleFormSuccess = () => {
    setShowForm(false)
    setEditingTarget(null)
  }

  const handleScrape = (target: Target) => {
    const meta = target.metadata as TargetMetadata
    onScrape?.(target, {
      limit: meta?.limit,
      until: meta?.until,
    })
  }

  if (showForm || editingTarget) {
    return (
      <TargetForm
        target={editingTarget || undefined}
        onCancel={() => {
          setShowForm(false)
          setEditingTarget(null)
        }}
        onSuccess={handleFormSuccess}
      />
    )
  }

  return (
    <Card className="target-list">
      <div className="target-list-header">
        <h3>Scraping Targets</h3>
        <Button variant="primary" size="sm" onClick={() => setShowForm(true)}>
          Add Target
        </Button>
      </div>

      {!targets || targets.length === 0 ? (
        <p className="text-muted">No targets configured. Add a target to start scraping.</p>
      ) : (
        <ul className="targets">
          {targets.map((target) => (
            <li key={target.id} className="target-item">
              <div className="target-info">
                <div className="target-name-row">
                  <span className="target-name">{target.name}</span>
                  <Badge status={typeToBadge[target.type] || 'paused'}>
                    {formatType(target.type)}
                  </Badge>
                  {!target.is_active && (
                    <Badge status="paused">Paused</Badge>
                  )}
                </div>
                <span className="target-url text-xs text-muted">{target.url}</span>
                {target.last_scraped_at && (
                  <span className="target-last-scraped text-xs text-muted">
                    Last scraped: {new Date(target.last_scraped_at).toLocaleString()}
                  </span>
                )}
              </div>
              <div className="target-actions">
                <Button
                  variant="primary"
                  size="sm"
                  onClick={() => handleScrape(target)}
                  disabled={!target.is_active}
                >
                  Scrape
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setEditingTarget(target)}
                >
                  Edit
                </Button>
                <Button
                  variant="danger"
                  size="sm"
                  onClick={() => handleDelete(target.id)}
                  loading={deleteTarget.isPending}
                >
                  Delete
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </Card>
  )
}

const formatType = (type: string): string => {
  const map: Record<string, string> = {
    TG_CHANNEL: 'Channel',
    TG_GROUP: 'Group',
    TG_FORUM: 'Forum',
    HH_SEARCH: 'HH',
    LINKEDIN_SEARCH: 'LinkedIn',
  }
  return map[type] || type
}
