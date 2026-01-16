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
  selectionMode?: boolean
  selectedIds?: Set<string>
  onSelectionChange?: (ids: Set<string>) => void
}

export const JobsTable = ({
  data,
  isLoading,
  selectedJobId,
  onJobClick,
  onPageChange,
  selectionMode = false,
  selectedIds = new Set(),
  onSelectionChange,
}: JobsTableProps) => {
  if (isLoading) {
    return <JobsTableSkeleton />
  }

  if (!data || !data.jobs || data.jobs.length === 0) {
    return (
      <div className="jobs-table-empty">
        <p className="text-muted">No jobs found. Try adjusting your filters or scrape more jobs.</p>
      </div>
    )
  }

  const allSelected = data.jobs.length > 0 &&
    data.jobs.every((job) => selectedIds.has(job.id))

  const handleSelectAll = (checked: boolean) => {
    if (!data?.jobs || !onSelectionChange) return
    if (checked) {
      const newIds = new Set(selectedIds)
      data.jobs.forEach((job) => newIds.add(job.id))
      onSelectionChange(newIds)
    } else {
      const newIds = new Set(selectedIds)
      data.jobs.forEach((job) => newIds.delete(job.id))
      onSelectionChange(newIds)
    }
  }

  const handleRowSelect = (jobId: string, checked: boolean) => {
    if (!onSelectionChange) return
    const newIds = new Set(selectedIds)
    if (checked) {
      newIds.add(jobId)
    } else {
      newIds.delete(jobId)
    }
    onSelectionChange(newIds)
  }

  return (
    <div className="jobs-table-container">
      <table className="jobs-table">
        <thead>
          <tr>
            {selectionMode && (
              <th className="job-col-checkbox">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={(e) => handleSelectAll(e.target.checked)}
                  aria-label="Select all jobs on page"
                />
              </th>
            )}
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
              showCheckbox={selectionMode}
              isChecked={selectedIds.has(job.id)}
              onCheckChange={(checked) => handleRowSelect(job.id, checked)}
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
