import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/types'
import type { CreateTargetRequest, UpdateTargetRequest } from '@/lib/types'

// ============================================================================
// Queries
// ============================================================================

export function useTargets() {
  return useQuery({
    queryKey: queryKeys.targets(),
    queryFn: () => api.getTargets(),
  })
}

export function useTarget(id: string) {
  return useQuery({
    queryKey: queryKeys.target(id),
    queryFn: () => api.getTarget(id),
    enabled: !!id,
  })
}

// ============================================================================
// Mutations
// ============================================================================

export function useCreateTarget() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CreateTargetRequest) => api.createTarget(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.targets() })
      queryClient.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

export function useUpdateTarget() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateTargetRequest }) =>
      api.updateTarget(id, data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.targets() })
      queryClient.invalidateQueries({ queryKey: queryKeys.target(variables.id) })
      queryClient.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

export function useDeleteTarget() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => api.deleteTarget(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.targets() })
      queryClient.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}

// ============================================================================
// Selectors
// ============================================================================

export function useActiveTargets() {
  const { data, ...rest } = useTargets()

  const activeTargets = data?.filter((t) => t.is_active) ?? []

  return {
    ...rest,
    data: activeTargets,
    total: data?.length ?? 0,
    activeCount: activeTargets.length,
  }
}
