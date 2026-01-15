import { useState } from 'react'
import type { JobStatus } from '@/lib/types'
import { Badge, Button, Card, Spinner, type BadgeStatus } from '@/components/ui'
import { useJob, useUpdateJobStatus } from '@/hooks/useJobs'

export interface JobDetailProps {
  jobId: string
  onClose?: () => void
}

const statusToBadge: Record<JobStatus, BadgeStatus> = {
  RAW: 'raw',
  ANALYZED: 'analyzed',
  INTERESTED: 'interested',
  REJECTED: 'rejected',
  TAILORED: 'analyzed',
  SENT: 'interested',
  RESPONDED: 'interested',
}

const statusActions: { status: JobStatus; label: string; variant: 'success' | 'danger' | 'primary' }[] = [
  { status: 'INTERESTED', label: 'Mark Interested', variant: 'success' },
  { status: 'REJECTED', label: 'Reject', variant: 'danger' },
]

export const JobDetail = ({ jobId, onClose }: JobDetailProps) => {
  const { data: job, isLoading, error } = useJob(jobId)
  const updateStatus = useUpdateJobStatus()
  const [expandRaw, setExpandRaw] = useState(false)

  if (isLoading) {
    return (
      <Card className="job-detail">
        <div className="job-detail-loading">
          <Spinner size="lg" />
        </div>
      </Card>
    )
  }

  if (error || !job) {
    return (
      <Card className="job-detail">
        <div className="job-detail-error">
          <p className="text-muted">Failed to load job details</p>
          <Button variant="secondary" onClick={onClose}>
            Close
          </Button>
        </div>
      </Card>
    )
  }

  const data = job.structured_data
  const handleStatusChange = (status: JobStatus) => {
    updateStatus.mutate({ id: job.id, data: { status } })
  }

  return (
    <Card className="job-detail">
      <div className="job-detail-header">
        <div>
          <h3>{data?.title || 'Untitled Job'}</h3>
          <p className="text-muted">{data?.company || 'Unknown Company'}</p>
        </div>
        <Button variant="secondary" size="sm" onClick={onClose} aria-label="Close">
          &times;
        </Button>
      </div>

      <div className="job-detail-status">
        <Badge status={statusToBadge[job.status]}>{job.status}</Badge>
        {data?.is_remote && <span className="remote-badge">Remote</span>}
      </div>

      {(data?.salary_min || data?.salary_max) && (
        <div className="job-detail-section">
          <h4>Salary</h4>
          <p>
            {formatSalary(data.salary_min, data.salary_max, data.currency)}
          </p>
        </div>
      )}

      {data?.technologies && data.technologies.length > 0 && (
        <div className="job-detail-section">
          <h4>Technologies</h4>
          <div className="tech-tags">
            {data.technologies.map((tech) => (
              <span key={tech} className="tech-tag">
                {tech}
              </span>
            ))}
          </div>
        </div>
      )}

      {data?.experience_years && (
        <div className="job-detail-section">
          <h4>Experience Required</h4>
          <p>{data.experience_years}+ years</p>
        </div>
      )}

      {data?.location && (
        <div className="job-detail-section">
          <h4>Location</h4>
          <p>{data.location}</p>
        </div>
      )}

      {data?.contacts && data.contacts.length > 0 && (
        <div className="job-detail-section">
          <h4>Contacts</h4>
          <ul className="contacts-list">
            {data.contacts.map((contact, i) => (
              <li key={i}>{contact}</li>
            ))}
          </ul>
        </div>
      )}

      {data?.description && (
        <div className="job-detail-section">
          <h4>Description</h4>
          <p className="job-description">{data.description}</p>
        </div>
      )}

      <div className="job-detail-section">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => setExpandRaw(!expandRaw)}
        >
          {expandRaw ? 'Hide' : 'Show'} Raw Content
        </Button>
        {expandRaw && (
          <pre className="raw-content">{job.raw_content}</pre>
        )}
      </div>

      <div className="job-detail-actions">
        {statusActions.map(({ status, label, variant }) => (
          <Button
            key={status}
            variant={variant}
            size="sm"
            onClick={() => handleStatusChange(status)}
            loading={updateStatus.isPending}
            disabled={job.status === status}
          >
            {label}
          </Button>
        ))}
      </div>

      <div className="job-detail-meta text-xs text-muted">
        <p>Created: {new Date(job.created_at).toLocaleString()}</p>
        {job.analyzed_at && <p>Analyzed: {new Date(job.analyzed_at).toLocaleString()}</p>}
        {job.source_url && (
          <p>
            Source:{' '}
            <a href={job.source_url} target="_blank" rel="noopener noreferrer">
              View Original
            </a>
          </p>
        )}
      </div>
    </Card>
  )
}

const formatSalary = (
  min?: number | null,
  max?: number | null,
  currency?: string | null
): string => {
  const symbol = currency === 'USD' ? '$' : currency === 'EUR' ? '\u20AC' : '\u20BD'

  if (min && max) {
    return `${symbol}${min.toLocaleString()} - ${symbol}${max.toLocaleString()}`
  }
  if (min) {
    return `From ${symbol}${min.toLocaleString()}`
  }
  if (max) {
    return `Up to ${symbol}${max.toLocaleString()}`
  }
  return '-'
}
