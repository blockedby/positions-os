import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ScrapeStatus } from './ScrapeStatus'

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

// Mock the hook
vi.mock('@/hooks/useScrapeStatus', () => ({
  useScrapeStatus: vi.fn(),
}))

import { useScrapeStatus } from '@/hooks/useScrapeStatus'

describe('ScrapeStatus', () => {
  it('should show idle state when not scraping', () => {
    vi.mocked(useScrapeStatus).mockReturnValue({
      data: { is_scraping: false },
      isLoading: false,
      error: null,
    } as ReturnType<typeof useScrapeStatus>)

    render(<ScrapeStatus />, { wrapper: createWrapper() })

    expect(screen.getByText(/idle/i)).toBeInTheDocument()
  })

  it('should show progress when actively scraping', () => {
    vi.mocked(useScrapeStatus).mockReturnValue({
      data: {
        is_scraping: true,
        target: '@golang_jobs',
        processed: 50,
        new_jobs: 12,
      },
      isLoading: false,
      error: null,
    } as ReturnType<typeof useScrapeStatus>)

    render(<ScrapeStatus />, { wrapper: createWrapper() })

    expect(screen.getByText('@golang_jobs')).toBeInTheDocument()
    expect(screen.getByText(/50 processed/i)).toBeInTheDocument()
    expect(screen.getByText(/12 new jobs/i)).toBeInTheDocument()
  })

  it('should show loading state while fetching status', () => {
    vi.mocked(useScrapeStatus).mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as ReturnType<typeof useScrapeStatus>)

    render(<ScrapeStatus />, { wrapper: createWrapper() })

    expect(screen.getByText(/checking status/i)).toBeInTheDocument()
  })
})
