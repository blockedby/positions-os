import type { Job, JobsResponse } from '@/lib/types'
import { JobRow } from './JobRow'
import { Pagination } from './Pagination'
import { Skeleton } from '@/components/ui'

export interface JobsTableProps {
  data?: JobsResponse
  isLoading?: boolean
  selectedJobId?: string
  onJobClick?: (job: Job) => void
  onPageChange?: (page: number) => void
}

export const JobsTable = ({
  data,
  isLoading,
  selectedJobId,
  onJobClick,
  onPageChange,
}: JobsTableProps) => {
  if (isLoading) {
    return <JobsTableSkeleton />
  }

  if (!data || data.jobs.length === 0) {
    return (
      <div className="jobs-table-empty">
        <p className="text-muted">No jobs found. Try adjusting your filters or scrape more jobs.</p>
      </div>
    )
  }

  return (
    <div className="jobs-table-container">
      <table className="jobs-table">
        <thead>
          <tr>
            <th>Job</th>
            <th>Salary</th>
            <th>Technologies</th>
            <th>Status</th>
            <th>Date</th>
          </tr>
        </thead>
        <tbody>
          {data.jobs.map((job) => (
            <JobRow
              key={job.id}
              job={job}
              onClick={onJobClick}
              isSelected={selectedJobId === job.id}
            />
          ))}
        </tbody>
      </table>

      {data.pages > 1 && (
        <Pagination
          currentPage={data.page}
          totalPages={data.pages}
          totalItems={data.total}
          onPageChange={onPageChange}
        />
      )}
    </div>
  )
}

const JobsTableSkeleton = () => (
  <div className="jobs-table-container">
    <table className="jobs-table">
      <thead>
        <tr>
          <th>Job</th>
          <th>Salary</th>
          <th>Technologies</th>
          <th>Status</th>
          <th>Date</th>
        </tr>
      </thead>
      <tbody>
        {Array.from({ length: 10 }).map((_, i) => (
          <tr key={i}>
            <td>
              <Skeleton variant="text" className="w-48" />
              <Skeleton variant="text" className="w-24 mt-1" />
            </td>
            <td>
              <Skeleton variant="text" className="w-24" />
            </td>
            <td>
              <div className="flex gap-1">
                <Skeleton variant="text" className="w-12" />
                <Skeleton variant="text" className="w-12" />
              </div>
            </td>
            <td>
              <Skeleton variant="text" className="w-20" />
            </td>
            <td>
              <Skeleton variant="text" className="w-16" />
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  </div>
)
