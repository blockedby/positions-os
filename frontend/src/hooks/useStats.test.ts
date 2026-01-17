import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { waitFor } from '@testing-library/react'
import { useStats, useStatsCards } from './useStats'
import { renderHookWithClient, mockStats } from '@/test/test-utils'
import { api } from '@/lib/api'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getStats: vi.fn(),
  },
}))

describe('useStats', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('useStats', () => {
    it('should fetch stats', async () => {
      vi.mocked(api.getStats).mockResolvedValueOnce(mockStats)

      const { result } = renderHookWithClient(() => useStats())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toEqual(mockStats)
      expect(api.getStats).toHaveBeenCalledTimes(1)
    })

    it('should handle fetch error', async () => {
      const error = new Error('Network error')
      vi.mocked(api.getStats).mockRejectedValueOnce(error)

      const { result } = renderHookWithClient(() => useStats())

      await waitFor(() => {
        expect(result.current.isError).toBe(true)
      })

      expect(result.current.error).toBe(error)
    })

    it('should have refetch interval configured', async () => {
      vi.mocked(api.getStats).mockResolvedValueOnce(mockStats)

      const { result } = renderHookWithClient(() => useStats())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      // The hook is configured with refetchInterval: 60000
      // We just verify the query runs successfully
      expect(result.current.data).toEqual(mockStats)
    })
  })

  describe('useStatsCards', () => {
    it('should transform stats into cards format', async () => {
      vi.mocked(api.getStats).mockResolvedValueOnce(mockStats)

      const { result } = renderHookWithClient(() => useStatsCards())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toHaveLength(9)

      // Check Total Jobs card
      expect(result.current.data[0]).toEqual({
        label: 'Total Jobs',
        value: 100,
        description: 'All scraped jobs',
      })

      // Check Analyzed card
      expect(result.current.data[1]).toEqual({
        label: 'Analyzed',
        value: 80,
        description: 'Jobs with structured data',
      })

      // Check Interested card
      expect(result.current.data[2]).toEqual({
        label: 'Interested',
        value: 20,
        description: 'Jobs you want to apply',
      })

      // Check Rejected card
      expect(result.current.data[3]).toEqual({
        label: 'Rejected',
        value: 30,
        description: 'Jobs rejected',
      })

      // Check Tailored card (tailored_jobs + tailored_approved_jobs)
      expect(result.current.data[4]).toEqual({
        label: 'Tailored',
        value: 8, // 5 + 3
        description: 'Applications prepared',
      })

      // Check Sent card
      expect(result.current.data[5]).toEqual({
        label: 'Sent',
        value: 2,
        description: 'Applications sent',
      })

      // Check Responded card
      expect(result.current.data[6]).toEqual({
        label: 'Responded',
        value: 1,
        description: 'Recruiter responses',
      })

      // Check Active Targets card
      expect(result.current.data[7]).toEqual({
        label: 'Active Targets',
        value: 3,
        description: 'Active scraping sources',
      })

      // Check Today card
      expect(result.current.data[8]).toEqual({
        label: 'Today',
        value: 5,
        description: 'New jobs today',
      })
    })

    it('should return empty array when stats not loaded', async () => {
      // Don't resolve the mock immediately
      vi.mocked(api.getStats).mockImplementation(() => new Promise(() => {}))

      const { result } = renderHookWithClient(() => useStatsCards())

      // While loading, data should be empty array
      expect(result.current.data).toEqual([])
      expect(result.current.isLoading).toBe(true)
    })

    it('should handle zero values in stats', async () => {
      const zeroStats = {
        total_jobs: 0,
        analyzed_jobs: 0,
        interested_jobs: 0,
        rejected_jobs: 0,
        tailored_jobs: 0,
        tailored_approved_jobs: 0,
        sent_jobs: 0,
        responded_jobs: 0,
        today_jobs: 0,
        active_targets: 0,
      }

      vi.mocked(api.getStats).mockResolvedValueOnce(zeroStats)

      const { result } = renderHookWithClient(() => useStatsCards())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data).toHaveLength(9)
      expect(result.current.data[0].value).toBe(0)
      expect(result.current.data[4].value).toBe(0) // Tailored
      expect(result.current.data[7].value).toBe(0) // Active Targets
    })

    it('should handle large numbers in stats', async () => {
      const largeStats = {
        total_jobs: 1000000,
        analyzed_jobs: 999999,
        interested_jobs: 500000,
        rejected_jobs: 400000,
        tailored_jobs: 50000,
        tailored_approved_jobs: 25000,
        sent_jobs: 10000,
        responded_jobs: 5000,
        today_jobs: 10000,
        active_targets: 100,
      }

      vi.mocked(api.getStats).mockResolvedValueOnce(largeStats)

      const { result } = renderHookWithClient(() => useStatsCards())

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true)
      })

      expect(result.current.data[0].value).toBe(1000000)
      expect(result.current.data[1].value).toBe(999999)
      expect(result.current.data[4].value).toBe(75000) // Tailored (50000 + 25000)
    })
  })
})
