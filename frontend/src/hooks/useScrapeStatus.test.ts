import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { waitFor } from '@testing-library/react'
import { useScrapeStatus } from './useScrapeStatus'
import { renderHookWithClient } from '@/test/test-utils'
import { api } from '@/lib/api'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getScrapeStatus: vi.fn(),
  },
}))

describe('useScrapeStatus', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  it('should fetch scrape status when not scraping', async () => {
    vi.mocked(api.getScrapeStatus).mockResolvedValueOnce({
      is_scraping: false,
    })

    const { result } = renderHookWithClient(() => useScrapeStatus())

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data).toEqual({ is_scraping: false })
    expect(api.getScrapeStatus).toHaveBeenCalledTimes(1)
  })

  it('should fetch scrape status when actively scraping', async () => {
    vi.mocked(api.getScrapeStatus).mockResolvedValueOnce({
      is_scraping: true,
      target: '@golang_jobs',
      processed: 50,
      new_jobs: 12,
    })

    const { result } = renderHookWithClient(() => useScrapeStatus())

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data).toEqual({
      is_scraping: true,
      target: '@golang_jobs',
      processed: 50,
      new_jobs: 12,
    })
  })

  it('should handle fetch error', async () => {
    const error = new Error('Network error')
    vi.mocked(api.getScrapeStatus).mockRejectedValueOnce(error)

    const { result } = renderHookWithClient(() => useScrapeStatus())

    await waitFor(() => {
      expect(result.current.isError).toBe(true)
    })

    expect(result.current.error).toBe(error)
  })
})
