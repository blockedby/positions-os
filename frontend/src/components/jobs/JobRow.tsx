import type { Job, JobStatus } from '@/lib/types'
import { Badge, type BadgeStatus } from '@/components/ui'

export interface JobRowProps {
  job: Job
  onClick?: (job: Job) => void
  isSelected?: boolean
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

const formatSalary = (job: Job): string => {
  const data = job.structured_data
  if (!data?.salary_min && !data?.salary_max) return '-'

  const currency = data.currency || 'RUB'
  const symbol = currency === 'USD' ? '$' : currency === 'EUR' ? '\u20AC' : '\u20BD'

  if (data.salary_min && data.salary_max) {
    return `${symbol}${formatNumber(data.salary_min)} - ${symbol}${formatNumber(data.salary_max)}`
  }
  if (data.salary_min) {
    return `from ${symbol}${formatNumber(data.salary_min)}`
  }
  if (data.salary_max) {
    return `to ${symbol}${formatNumber(data.salary_max)}`
  }
  return '-'
}

const formatNumber = (num: number): string => {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(0)}K`
  return num.toString()
}

const formatDate = (dateStr: string): string => {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  })
}

export const JobRow = ({ job, onClick, isSelected = false }: JobRowProps) => {
  const title = job.structured_data?.title || 'Untitled'
  const company = job.structured_data?.company || '-'
  const technologies = job.structured_data?.technologies || []

  return (
    <tr
      onClick={() => onClick?.(job)}
      className={`job-row ${isSelected ? 'job-row-selected' : ''}`}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick?.(job)
        }
      }}
    >
      <td className="job-title-cell">
        <div className="job-title">{title}</div>
        <div className="job-company text-muted text-xs">{company}</div>
      </td>
      <td className="job-salary-cell">{formatSalary(job)}</td>
      <td className="job-techs-cell">
        <div className="tech-tags">
          {technologies.slice(0, 3).map((tech) => (
            <span key={tech} className="tech-tag">
              {tech}
            </span>
          ))}
          {technologies.length > 3 && (
            <span className="tech-tag tech-tag-more">+{technologies.length - 3}</span>
          )}
        </div>
      </td>
      <td className="job-status-cell">
        <Badge status={statusToBadge[job.status]}>{job.status}</Badge>
      </td>
      <td className="job-date-cell text-muted">{formatDate(job.created_at)}</td>
    </tr>
  )
}
