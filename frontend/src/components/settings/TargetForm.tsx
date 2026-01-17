import { useState } from 'react'
import type { Target, TargetType, CreateTargetRequest, UpdateTargetRequest, TargetMetadata } from '@/lib/types'
import { Input, Select, Button, Card } from '@/components/ui'
import { useCreateTarget, useUpdateTarget } from '@/hooks/useTargets'

export interface TargetFormProps {
  target?: Target
  onCancel?: () => void
  onSuccess?: () => void
}

const targetTypeOptions = [
  { value: 'TG_CHANNEL', label: 'Telegram Channel' },
  { value: 'TG_GROUP', label: 'Telegram Group' },
  { value: 'TG_FORUM', label: 'Telegram Forum' },
  { value: 'HH_SEARCH', label: 'HeadHunter Search' },
  { value: 'LINKEDIN_SEARCH', label: 'LinkedIn Search' },
]

export const TargetForm = ({ target, onCancel, onSuccess }: TargetFormProps) => {
  const isEditing = !!target
  const createTarget = useCreateTarget()
  const updateTarget = useUpdateTarget()

  const [name, setName] = useState(target?.name || '')
  const [type, setType] = useState<TargetType>(target?.type || 'TG_CHANNEL')
  const [url, setUrl] = useState(target?.url || '')
  const [isActive, setIsActive] = useState(target?.is_active ?? true)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [limit, setLimit] = useState<number | ''>(
    (target?.metadata as TargetMetadata)?.limit || 100
  )
  const [until, setUntil] = useState(
    (target?.metadata as TargetMetadata)?.until || ''
  )

  // Note: Parent component should use key={target?.id} to reset form when editing different target

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {}

    if (!name.trim()) {
      newErrors.name = 'Name is required'
    }

    if (!url.trim()) {
      newErrors.url = 'URL is required'
    } else if (type.startsWith('TG_') && !url.startsWith('@') && !url.includes('t.me')) {
      newErrors.url = 'Telegram URL should start with @ or include t.me'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validate()) return

    const metadata: TargetMetadata = {}
    if (limit !== '' && limit > 0) {
      metadata.limit = limit
    }
    if (until) {
      metadata.until = until
    }

    try {
      if (isEditing && target) {
        const data: UpdateTargetRequest = {
          name,
          url,
          is_active: isActive,
          metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
        }
        await updateTarget.mutateAsync({ id: target.id, data })
      } else {
        const data: CreateTargetRequest = {
          name,
          type,
          url,
          is_active: isActive,
          metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
        }
        await createTarget.mutateAsync(data)
      }
      onSuccess?.()
    } catch {
      // Error handled by react-query
    }
  }

  const isPending = createTarget.isPending || updateTarget.isPending

  return (
    <Card className="target-form">
      <h3>{isEditing ? 'Edit Target' : 'Add Target'}</h3>
      <form onSubmit={handleSubmit}>
        <Input
          label="Name"
          placeholder="e.g., Go Jobs"
          value={name}
          onChange={(e) => setName(e.target.value)}
          error={!!errors.name}
          errorMessage={errors.name}
        />

        {!isEditing && (
          <Select
            label="Type"
            options={targetTypeOptions}
            value={type}
            onChange={(e) => setType(e.target.value as TargetType)}
          />
        )}

        <Input
          label="URL"
          placeholder="e.g., @golang_jobs or https://t.me/golang_jobs"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          error={!!errors.url}
          errorMessage={errors.url}
          helperText={type.startsWith('TG_') ? 'Use @channel_name or full t.me URL' : undefined}
        />

        {type.startsWith('TG_') && (
          <>
            <Input
              label="Message Limit"
              type="number"
              min={1}
              placeholder="100"
              value={limit}
              onChange={(e) => setLimit(e.target.value ? parseInt(e.target.value, 10) : '')}
              helperText="Maximum messages to scrape (leave empty for default 100)"
            />

            <Input
              label="Until Date"
              type="date"
              value={until}
              onChange={(e) => setUntil(e.target.value)}
              helperText="Stop scraping at posts older than this date (optional)"
            />
          </>
        )}

        <div className="form-checkbox">
          <input
            type="checkbox"
            id="isActive"
            checked={isActive}
            onChange={(e) => setIsActive(e.target.checked)}
          />
          <label htmlFor="isActive">Active</label>
        </div>

        <div className="form-actions">
          <Button type="button" variant="secondary" onClick={onCancel} disabled={isPending}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={isPending}>
            {isEditing ? 'Save' : 'Create'}
          </Button>
        </div>
      </form>
    </Card>
  )
}
