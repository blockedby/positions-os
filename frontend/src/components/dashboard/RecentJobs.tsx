import { Link } from 'react-router-dom'
import type { Job, JobStatus } from '@/lib/types'
import { Card, Badge, Skeleton, type BadgeStatus } from '@/components/ui'
import { useJobs } from '@/hooks/useJobs'

export interface RecentJobsProps {
  limit?: number
  className?: string
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

export const RecentJobs = ({ limit = 5, className = '' }: RecentJobsProps) => {
  const { data, isLoading, error } = useJobs({
    limit,
    sort_by: 'created_at',
    sort_order: 'desc',
  })

  if (isLoading) {
    return <RecentJobsSkeleton limit={limit} className={className} />
  }

  if (error) {
    return (
      <Card className={`recent-jobs ${className}`}>
        <p className="text-muted">Failed to load recent jobs</p>
      </Card>
    )
  }

  const jobs = data?.jobs || []

  return (
    <Card className={`recent-jobs ${className}`}>
      <div className="recent-jobs-header">
        <h3>Recent Jobs</h3>
        <Link to="/jobs" className="text-xs">
          View all
        </Link>
      </div>

      {jobs.length === 0 ? (
        <p className="text-muted">No jobs yet. Start by scraping some channels.</p>
      ) : (
        <ul className="recent-jobs-list">
          {jobs.map((job) => (
            <RecentJobItem key={job.id} job={job} />
          ))}
        </ul>
      )}
    </Card>
  )
}

const RecentJobItem = ({ job }: { job: Job }) => {
  const title = job.structured_data?.title || 'Untitled'
  const company = job.structured_data?.company || '-'

  return (
    <li className="recent-job-item">
      <Link to={`/jobs?id=${job.id}`} className="recent-job-link">
        <div className="recent-job-info">
          <span className="recent-job-title">{title}</span>
          <span className="recent-job-company text-xs text-muted">{company}</span>
        </div>
        <Badge status={statusToBadge[job.status]}>{job.status}</Badge>
      </Link>
    </li>
  )
}

const RecentJobsSkeleton = ({
  limit,
  className = '',
}: {
  limit: number
  className?: string
}) => (
  <Card className={`recent-jobs ${className}`}>
    <div className="recent-jobs-header">
      <Skeleton variant="text" className="w-24 h-5" />
      <Skeleton variant="text" className="w-12 h-3" />
    </div>
    <ul className="recent-jobs-list">
      {Array.from({ length: limit }).map((_, i) => (
        <li key={i} className="recent-job-item">
          <div className="recent-job-link">
            <div className="recent-job-info">
              <Skeleton variant="text" className="w-40 h-4" />
              <Skeleton variant="text" className="w-24 h-3 mt-1" />
            </div>
            <Skeleton variant="text" className="w-16 h-5" />
          </div>
        </li>
      ))}
    </ul>
  </Card>
)
