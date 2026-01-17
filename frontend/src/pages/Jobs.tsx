import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import type { Job, JobsQuery } from '@/lib/types'
import { useBulkDeleteJobs } from '@/hooks/useJobs'
import { useWebSocket } from '@/hooks/useWebSocket'
import { FilterBar, InfiniteJobsList, JobDetail } from '@/components/jobs'
import { Button } from '@/components/ui'

export default function Jobs() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [filters, setFilters] = useState<Omit<JobsQuery, 'page' | 'limit'>>({
    sort_by: 'created_at',
    sort_order: 'desc',
  })
  // Derive selectedJobId directly from URL - no useState needed
  const selectedJobId = searchParams.get('id') || undefined
  const [selectionMode, setSelectionMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  const bulkDelete = useBulkDeleteJobs()

  // Enable real-time updates
  useWebSocket({ enabled: true })

  const handleJobClick = (job: Job) => {
    setSearchParams({ id: job.id })
  }

  const handleCloseDetail = () => {
    setSearchParams({})
  }

  const handleFilter = (newFilters: JobsQuery) => {
    // Omit page/limit as they're handled by infinite scroll
    const { status, sort_by, sort_order, search, technologies, salary_min, salary_max, is_remote } =
      newFilters
    setFilters((prev) => ({
      ...prev,
      status,
      sort_by,
      sort_order,
      search,
      technologies,
      salary_min,
      salary_max,
      is_remote,
    }))
  }

  const handleBulkDelete = async () => {
    if (selectedIds.size === 0) return

    const confirmed = window.confirm(
      `Delete ${selectedIds.size} job(s)? This cannot be undone.`
    )
    if (!confirmed) return

    try {
      await bulkDelete.mutateAsync(Array.from(selectedIds))
      setSelectedIds(new Set())
      setSelectionMode(false)
    } catch (error) {
      console.error('Failed to delete jobs:', error)
    }
  }

  const toggleSelectionMode = () => {
    setSelectionMode(!selectionMode)
    if (selectionMode) {
      setSelectedIds(new Set())
    }
  }

  return (
    <div className="jobs-page">
      <div className="jobs-header">
        <h1>Jobs</h1>
      </div>

      <FilterBar onFilter={handleFilter} />

      <div className="jobs-actions">
        <Button
          variant={selectionMode ? 'primary' : 'secondary'}
          size="sm"
          onClick={toggleSelectionMode}
        >
          {selectionMode ? 'Cancel' : 'Select'}
        </Button>

        {selectionMode && selectedIds.size > 0 && (
          <Button
            variant="danger"
            size="sm"
            onClick={handleBulkDelete}
            loading={bulkDelete.isPending}
          >
            Delete ({selectedIds.size})
          </Button>
        )}
      </div>

      <div className="jobs-content">
        <div className={`jobs-list ${selectedJobId ? 'jobs-list-with-detail' : ''}`}>
          <InfiniteJobsList
            filters={filters}
            onJobSelect={handleJobClick}
            selectedJobId={selectedJobId}
            selectionMode={selectionMode}
            selectedIds={selectedIds}
            onSelectionChange={setSelectedIds}
          />
        </div>

        {selectedJobId && (
          <div className="jobs-detail">
            <JobDetail jobId={selectedJobId} onClose={handleCloseDetail} />
          </div>
        )}
      </div>
    </div>
  )
}
