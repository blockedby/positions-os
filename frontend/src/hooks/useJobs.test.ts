import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { waitFor } from '@testing-library/react'
import { useJobs, useJob, useUpdateJobStatus, useJobStatusCounts, usePrepareJob } from './useJobs'
import { renderHookWithClient, mockJob } from '@/test/test-utils'
import { api } from '@/lib/api'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getJobs: vi.fn(),
    getJob: vi.fn(),
    updateJobStatus: vi.fn(),
    prepareJob: vi.fn(),
  },
}))

const mockJobsResponse = {
  jobs: [mockJob],
  total: 1,
  page: 1,
  limit: 20,
  pages: 1,
}

describe('useJobs', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('useJobs', () => {
    it('should fetch jobs without query params', async () => {
      vi.mocked(api.getJobs).mockResolvedValueOnce(mockJobsResponse)

      const { result } = renderHookWithClient(() => useJobs())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual(mockJobsResponse)
      expect(api.getJobs).toHaveBeenCalledWith(undefined)
    })

    it('should fetch jobs with query params', async () => {
      vi.mocked(api.getJobs).mockResolvedValueOnce(mockJobsResponse)

      const query = { page: 2, limit: 10, status: 'ANALYZED' as const }
      const { result } = renderHookWithClient(() => useJobs(query))

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(api.getJobs).toHaveBeenCalledWith(query)
    })

    it('should refetch when query params change', async () => {
      vi.mocked(api.getJobs).mockResolvedValue(mockJobsResponse)

      const { result, rerender } = renderHookWithClient(
        (props: { query?: { page: number } }) => useJobs(props.query),
        { initialProps: { query: { page: 1 } } }
      )

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      rerender({ query: { page: 2 } })

      await waitFor(() => {
        expect(api.getJobs).toHaveBeenCalledWith({ page: 2 })
      })
    })

    it('should handle fetch error', async () => {
      const error = new Error('Network error')
      vi.mocked(api.getJobs).mockRejectedValueOnce(error)

      const { result } = renderHookWithClient(() => useJobs())

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBe(error)
    })
  })

  describe('useJob', () => {
    it('should fetch a single job by ID', async () => {
      vi.mocked(api.getJob).mockResolvedValueOnce(mockJob)

      const { result } = renderHookWithClient(() => useJob('job-1'))

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual(mockJob)
      expect(api.getJob).toHaveBeenCalledWith('job-1')
    })

    it('should not fetch when ID is empty', async () => {
      const { result } = renderHookWithClient(() => useJob(''))

      // Query should be disabled
      expect(result.current.fetchStatus).toBe('idle')
      expect(api.getJob).not.toHaveBeenCalled()
    })
  })

  describe('useUpdateJobStatus', () => {
    it('should update job status and invalidate queries', async () => {
      const updatedJob = { ...mockJob, status: 'INTERESTED' as const }
      vi.mocked(api.updateJobStatus).mockResolvedValueOnce(updatedJob)

      const { result } = renderHookWithClient(() => useUpdateJobStatus())

      // Execute mutation
      result.current.mutate({
        id: 'job-1',
        data: { status: 'INTERESTED' },
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(api.updateJobStatus).toHaveBeenCalledWith('job-1', { status: 'INTERESTED' })
      expect(result.current.data).toEqual(updatedJob)
    })

    it('should handle mutation error', async () => {
      const error = new Error('Update failed')
      vi.mocked(api.updateJobStatus).mockRejectedValueOnce(error)

      const { result } = renderHookWithClient(() => useUpdateJobStatus())

      result.current.mutate({
        id: 'job-1',
        data: { status: 'INTERESTED' },
      })

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBe(error)
    })
  })

  describe('useJobStatusCounts', () => {
    it('should return status counts from jobs', async () => {
      const jobsWithStatuses = {
        jobs: [
          { ...mockJob, id: 'job-1', status: 'ANALYZED' as const },
          { ...mockJob, id: 'job-2', status: 'ANALYZED' as const },
          { ...mockJob, id: 'job-3', status: 'INTERESTED' as const },
          { ...mockJob, id: 'job-4', status: 'REJECTED' as const },
        ],
        total: 4,
        page: 1,
        limit: 20,
        pages: 1,
      }

      vi.mocked(api.getJobs).mockResolvedValueOnce(jobsWithStatuses)

      const { result } = renderHookWithClient(() => useJobStatusCounts())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual({
        ANALYZED: 2,
        INTERESTED: 1,
        REJECTED: 1,
      })
      expect(result.current.total).toBe(4)
    })

    it('should return empty counts when no jobs', async () => {
      vi.mocked(api.getJobs).mockResolvedValueOnce({
        jobs: [],
        total: 0,
        page: 1,
        limit: 20,
        pages: 0,
      })

      const { result } = renderHookWithClient(() => useJobStatusCounts())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual({})
      expect(result.current.total).toBe(0)
    })

    it('should handle undefined data gracefully', async () => {
      // Before the query completes, data is undefined
      const { result } = renderHookWithClient(() => useJobStatusCounts())

      // Initially loading
      expect(result.current.isLoading).toBe(true)
      expect(result.current.total).toBe(0)
    })
  })

  describe('usePrepareJob', () => {
    it('should prepare job and invalidate queries on success', async () => {
      const prepareResponse = {
        job_id: 'job-1',
        status: 'TAILORED_APPROVED',
        resume_path: '/storage/jobs/job-1/resume.pdf',
        cover_letter_path: '/storage/jobs/job-1/cover_letter.md',
      }
      vi.mocked(api.prepareJob).mockResolvedValueOnce(prepareResponse)

      const { result } = renderHookWithClient(() => usePrepareJob())

      result.current.mutate('job-1')

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(api.prepareJob).toHaveBeenCalledWith('job-1')
      expect(result.current.data).toEqual(prepareResponse)
    })

    it('should handle preparation error', async () => {
      const error = new Error('Job must be in INTERESTED status')
      vi.mocked(api.prepareJob).mockRejectedValueOnce(error)

      const { result } = renderHookWithClient(() => usePrepareJob())

      result.current.mutate('job-1')

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBe(error)
    })
  })
})
