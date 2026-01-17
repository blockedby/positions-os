import { useEffect, useRef } from 'react'
import { useInfiniteJobs } from '@/hooks/useInfiniteJobs'
import { JobRow } from './JobRow'
import { Spinner, Skeleton } from '@/components/ui'
import type { JobsQuery, Job } from '@/lib/types'

export interface InfiniteJobsListProps {
  filters?: Omit<JobsQuery, 'page' | 'limit'>
  onJobSelect?: (job: Job) => void
  selectedJobId?: string
  selectionMode?: boolean
  selectedIds?: Set<string>
  onSelectionChange?: (ids: Set<string>) => void
}

export const InfiniteJobsList = ({
  filters,
  onJobSelect,
  selectedJobId,
  selectionMode = false,
  selectedIds = new Set(),
  onSelectionChange,
}: InfiniteJobsListProps) => {
  const {
    data,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
  } = useInfiniteJobs(filters)

  const loadMoreRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { threshold: 0.1 }
    )

    const current = loadMoreRef.current
    if (current) {
      observer.observe(current)
    }

    return () => {
      if (current) {
        observer.unobserve(current)
      }
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  if (isLoading) {
    return <InfiniteJobsListSkeleton />
  }

  const allJobs = data?.pages.flatMap((page) => page.jobs) ?? []
  const total = data?.pages[0]?.total ?? 0

  if (allJobs.length === 0) {
    return (
      <div className="jobs-table-empty">
        <p className="text-muted">No jobs found. Try adjusting your filters or scrape more jobs.</p>
      </div>
    )
  }

  const allSelected = allJobs.length > 0 &&
    allJobs.every((job) => selectedIds.has(job.id))

  const handleSelectAll = (checked: boolean) => {
    if (!onSelectionChange) return
    if (checked) {
      const newIds = new Set(selectedIds)
      allJobs.forEach((job) => newIds.add(job.id))
      onSelectionChange(newIds)
    } else {
      const newIds = new Set(selectedIds)
      allJobs.forEach((job) => newIds.delete(job.id))
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
    <div className="infinite-jobs-list">
      <div className="jobs-count text-muted text-sm mb-2">
        Showing {allJobs.length} of {total} jobs
      </div>

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
                    aria-label="Select all loaded jobs"
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
            {allJobs.map((job) => (
              <JobRow
                key={job.id}
                job={job}
                onClick={onJobSelect}
                isSelected={selectedJobId === job.id}
                showCheckbox={selectionMode}
                isChecked={selectedIds.has(job.id)}
                onCheckChange={(checked) => handleRowSelect(job.id, checked)}
              />
            ))}
          </tbody>
        </table>
      </div>

      <div ref={loadMoreRef} className="load-more-trigger">
        {isFetchingNextPage && (
          <div className="loading-more">
            <Spinner size="sm" />
            <span>Loading more...</span>
          </div>
        )}
        {!hasNextPage && allJobs.length > 0 && (
          <p className="text-muted text-center text-sm">No more jobs to load</p>
        )}
      </div>
    </div>
  )
}

const InfiniteJobsListSkeleton = () => (
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
