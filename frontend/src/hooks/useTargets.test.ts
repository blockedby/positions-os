import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { waitFor } from '@testing-library/react'
import {
  useTargets,
  useTarget,
  useCreateTarget,
  useUpdateTarget,
  useDeleteTarget,
  useActiveTargets,
} from './useTargets'
import {
  renderHookWithClient,
  mockTarget,
  mockInactiveTarget,
} from '@/test/test-utils'
import { api } from '@/lib/api'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getTargets: vi.fn(),
    getTarget: vi.fn(),
    createTarget: vi.fn(),
    updateTarget: vi.fn(),
    deleteTarget: vi.fn(),
  },
}))

describe('useTargets', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('useTargets', () => {
    it('should fetch all targets', async () => {
      vi.mocked(api.getTargets).mockResolvedValueOnce([mockTarget])

      const { result } = renderHookWithClient(() => useTargets())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual([mockTarget])
      expect(api.getTargets).toHaveBeenCalledTimes(1)
    })

    it('should handle empty targets list', async () => {
      vi.mocked(api.getTargets).mockResolvedValueOnce([])

      const { result } = renderHookWithClient(() => useTargets())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual([])
    })

    it('should handle fetch error', async () => {
      const error = new Error('Network error')
      vi.mocked(api.getTargets).mockRejectedValueOnce(error)

      const { result } = renderHookWithClient(() => useTargets())

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBe(error)
    })
  })

  describe('useTarget', () => {
    it('should fetch a single target by ID', async () => {
      vi.mocked(api.getTarget).mockResolvedValueOnce(mockTarget)

      const { result } = renderHookWithClient(() => useTarget('target-1'))

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual(mockTarget)
      expect(api.getTarget).toHaveBeenCalledWith('target-1')
    })

    it('should not fetch when ID is empty', async () => {
      const { result } = renderHookWithClient(() => useTarget(''))

      // Query should be disabled
      expect(result.current.fetchStatus).toBe('idle')
      expect(api.getTarget).not.toHaveBeenCalled()
    })
  })

  describe('useCreateTarget', () => {
    it('should create a target and return created data', async () => {
      const newTarget = { ...mockTarget, id: 'new-target', name: 'New Channel' }
      vi.mocked(api.createTarget).mockResolvedValueOnce(newTarget)

      const { result } = renderHookWithClient(() => useCreateTarget())

      // Execute mutation
      result.current.mutate({
        name: 'New Channel',
        type: 'TG_CHANNEL',
        url: '@new_channel',
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(api.createTarget).toHaveBeenCalledWith({
        name: 'New Channel',
        type: 'TG_CHANNEL',
        url: '@new_channel',
      })
      expect(result.current.data).toEqual(newTarget)
    })
  })

  describe('useUpdateTarget', () => {
    it('should update a target and return updated data', async () => {
      const updatedTarget = { ...mockTarget, name: 'Updated Name' }
      vi.mocked(api.updateTarget).mockResolvedValueOnce(updatedTarget)

      const { result } = renderHookWithClient(() => useUpdateTarget())

      // Execute mutation
      result.current.mutate({
        id: 'target-1',
        data: { name: 'Updated Name' },
      })

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(api.updateTarget).toHaveBeenCalledWith('target-1', { name: 'Updated Name' })
      expect(result.current.data).toEqual(updatedTarget)
    })
  })

  describe('useDeleteTarget', () => {
    it('should delete a target successfully', async () => {
      vi.mocked(api.deleteTarget).mockResolvedValueOnce(undefined)

      const { result } = renderHookWithClient(() => useDeleteTarget())

      // Execute mutation
      result.current.mutate('target-1')

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(api.deleteTarget).toHaveBeenCalledWith('target-1')
    })
  })

  describe('useActiveTargets', () => {
    it('should filter and return only active targets', async () => {
      vi.mocked(api.getTargets).mockResolvedValueOnce([mockTarget, mockInactiveTarget])

      const { result } = renderHookWithClient(() => useActiveTargets())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual([mockTarget])
      expect(result.current.total).toBe(2)
      expect(result.current.activeCount).toBe(1)
    })

    it('should return empty array when no active targets', async () => {
      vi.mocked(api.getTargets).mockResolvedValueOnce([mockInactiveTarget])

      const { result } = renderHookWithClient(() => useActiveTargets())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual([])
      expect(result.current.total).toBe(1)
      expect(result.current.activeCount).toBe(0)
    })

    it('should handle empty targets list', async () => {
      vi.mocked(api.getTargets).mockResolvedValueOnce([])

      const { result } = renderHookWithClient(() => useActiveTargets())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual([])
      expect(result.current.total).toBe(0)
      expect(result.current.activeCount).toBe(0)
    })
  })
})
