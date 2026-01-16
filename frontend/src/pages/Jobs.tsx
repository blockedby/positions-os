import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import type { Job, JobsQuery } from '@/lib/types'
import { useJobs, useBulkDeleteJobs } from '@/hooks/useJobs'
import { useWebSocket } from '@/hooks/useWebSocket'
import { FilterBar, JobsTable, JobDetail } from '@/components/jobs'
import { Button } from '@/components/ui'

export default function Jobs() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [filters, setFilters] = useState<JobsQuery>({
    page: 1,
    limit: 20,
    sort_by: 'created_at',
    sort_order: 'desc',
  })
  const [selectedJobId, setSelectedJobId] = useState<string | undefined>(
    searchParams.get('id') || undefined
  )
  const [selectionMode, setSelectionMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())

  const { data, isLoading } = useJobs(filters)
  const bulkDelete = useBulkDeleteJobs()

  // Enable real-time updates
  useWebSocket({ enabled: true })

  // Sync URL params with selected job
  useEffect(() => {
    const urlJobId = searchParams.get('id')
    if (urlJobId !== selectedJobId) {
      setSelectedJobId(urlJobId || undefined)
    }
  }, [searchParams, selectedJobId])

  const handleJobClick = (job: Job) => {
    setSelectedJobId(job.id)
    setSearchParams({ id: job.id })
  }

  const handleCloseDetail = () => {
    setSelectedJobId(undefined)
    setSearchParams({})
  }

  const handleFilter = (newFilters: JobsQuery) => {
    setFilters({ ...filters, ...newFilters, page: 1 })
  }

  const handlePageChange = (page: number) => {
    setFilters({ ...filters, page })
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
        <span className="text-muted text-xs">
          {data?.total ? `${data.total} jobs` : ''}
        </span>
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
          <JobsTable
            data={data}
            isLoading={isLoading}
            selectedJobId={selectedJobId}
            onJobClick={handleJobClick}
            onPageChange={handlePageChange}
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
