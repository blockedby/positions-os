import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import { useInfiniteJobs } from './useInfiniteJobs'
import type { Job } from '@/lib/types'

vi.mock('@/lib/api', () => ({
  api: {
    getJobs: vi.fn(),
  },
}))

import { api } from '@/lib/api'

const mockJob = (overrides: Partial<Job> = {}): Job => ({
  id: '1',
  target_id: 'target-1',
  external_id: 'ext-1',
  content_hash: 'hash-1',
  raw_content: 'content',
  status: 'RAW',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  ...overrides,
})

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children)
}

describe('useInfiniteJobs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should fetch first page of jobs', async () => {
    vi.mocked(api.getJobs).mockResolvedValue({
      jobs: [mockJob()],
      total: 50,
      page: 1,
      limit: 20,
      pages: 3,
    })

    const { result } = renderHook(() => useInfiniteJobs(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.pages).toHaveLength(1)
  })

  it('should have hasNextPage when more pages exist', async () => {
    vi.mocked(api.getJobs).mockResolvedValue({
      jobs: [mockJob()],
      total: 50,
      page: 1,
      limit: 20,
      pages: 3,
    })

    const { result } = renderHook(() => useInfiniteJobs(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.hasNextPage).toBe(true)
  })

  it('should fetch next page when requested', async () => {
    vi.mocked(api.getJobs)
      .mockResolvedValueOnce({
        jobs: [mockJob({ id: '1' })],
        total: 50,
        page: 1,
        limit: 20,
        pages: 3,
      })
      .mockResolvedValueOnce({
        jobs: [mockJob({ id: '2', status: 'ANALYZED' })],
        total: 50,
        page: 2,
        limit: 20,
        pages: 3,
      })

    const { result } = renderHook(() => useInfiniteJobs(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    // Trigger fetch and wait for it to complete
    result.current.fetchNextPage()

    await waitFor(() => {
      expect(result.current.data?.pages).toHaveLength(2)
    }, { timeout: 3000 })
  })

  it('should pass filters to API', async () => {
    vi.mocked(api.getJobs).mockResolvedValue({
      jobs: [],
      total: 0,
      page: 1,
      limit: 20,
      pages: 0,
    })

    renderHook(() => useInfiniteJobs({ status: 'ANALYZED' }), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(api.getJobs).toHaveBeenCalledWith(
        expect.objectContaining({ status: 'ANALYZED' })
      )
    })
  })
})
