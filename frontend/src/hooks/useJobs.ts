import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/types'
import type { JobsQuery, UpdateJobRequest } from '@/lib/types'

// ============================================================================
// Queries
// ============================================================================

export function useJobs(query?: JobsQuery) {
  return useQuery({
    queryKey: queryKeys.jobs(query),
    queryFn: () => api.getJobs(query),
  })
}

export function useJob(id: string) {
  return useQuery({
    queryKey: queryKeys.job(id),
    queryFn: () => api.getJob(id),
    enabled: !!id,
  })
}

// ============================================================================
// Mutations
// ============================================================================

export function useUpdateJobStatus() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateJobRequest }) =>
      api.updateJobStatus(id, data),
    onSuccess: (updatedJob) => {
      // Invalidate job detail
      queryClient.invalidateQueries({ queryKey: queryKeys.job(updatedJob.id) })

      // Invalidate jobs list
      queryClient.invalidateQueries({ queryKey: ['jobs'] })

      // Invalidate stats
      queryClient.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

export function useBulkDeleteJobs() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (ids: string[]) => api.bulkDeleteJobs(ids),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
      queryClient.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

// ============================================================================
// Selectors
// ============================================================================

export function useJobStatusCounts() {
  const { data, ...rest } = useJobs()

  const counts = data?.jobs.reduce(
    (acc, job) => {
      acc[job.status] = (acc[job.status] || 0) + 1
      return acc
    },
    {} as Record<string, number>,
  )

  return {
    ...rest,
    data: counts,
    total: data?.total ?? 0,
  }
}
