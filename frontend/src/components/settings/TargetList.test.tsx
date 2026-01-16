import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TargetList } from './TargetList'
import { mockTarget, mockInactiveTarget } from '../../test/test-utils'

// Mock the hooks
vi.mock('../../hooks/useTargets', () => ({
  useTargets: vi.fn(),
  useDeleteTarget: vi.fn(),
  useCreateTarget: vi.fn(),
  useUpdateTarget: vi.fn(),
}))

import { useTargets, useDeleteTarget, useCreateTarget, useUpdateTarget } from '../../hooks/useTargets'

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  }
}

describe('TargetList', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    // Default mock implementations
    vi.mocked(useDeleteTarget).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useDeleteTarget>)

    vi.mocked(useCreateTarget).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useCreateTarget>)

    vi.mocked(useUpdateTarget).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn(),
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateTarget>)
  })

  describe('Loading State', () => {
    it('should show loading spinner while fetching targets', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: undefined,
        isLoading: true,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      // The component uses Spinner component in loading state
      expect(document.querySelector('.target-list-loading')).toBeInTheDocument()
    })
  })

  describe('Error State', () => {
    it('should show error message when fetch fails', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: undefined,
        isLoading: false,
        error: new Error('Failed to fetch targets'),
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      expect(screen.getByText(/failed to load targets/i)).toBeInTheDocument()
    })
  })

  describe('Empty State', () => {
    it('should show empty state message when no targets exist', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      expect(screen.getByText(/no targets configured/i)).toBeInTheDocument()
    })
  })

  describe('Target Display', () => {
    it('should display list of targets with names', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [mockTarget, mockInactiveTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      expect(screen.getByText('Go Jobs Channel')).toBeInTheDocument()
      expect(screen.getByText('Inactive Channel')).toBeInTheDocument()
    })

    it('should display target URLs', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [mockTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      expect(screen.getByText('@go_jobs')).toBeInTheDocument()
    })

    it('should display target type badge', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [mockTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      // TargetList formats TG_CHANNEL as "Channel"
      expect(screen.getByText('Channel')).toBeInTheDocument()
    })

    it('should show paused badge for inactive targets', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [mockInactiveTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      expect(screen.getByText('Paused')).toBeInTheDocument()
    })

    it('should display multiple targets as list items', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [mockTarget, mockInactiveTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      const listItems = document.querySelectorAll('.target-item')
      expect(listItems.length).toBe(2)
    })
  })

  describe('Target Actions', () => {
    it('should call delete mutation when delete button clicked and confirmed', async () => {
      const deleteMutateAsync = vi.fn().mockResolvedValue(undefined)
      vi.mocked(useTargets).mockReturnValue({
        data: [mockTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      vi.mocked(useDeleteTarget).mockReturnValue({
        mutate: vi.fn(),
        mutateAsync: deleteMutateAsync,
        isPending: false,
      } as unknown as ReturnType<typeof useDeleteTarget>)

      // Mock window.confirm
      vi.spyOn(window, 'confirm').mockReturnValue(true)

      render(<TargetList />, { wrapper: createWrapper() })

      const deleteButton = screen.getByRole('button', { name: /delete/i })
      await userEvent.click(deleteButton)

      expect(deleteMutateAsync).toHaveBeenCalledWith('target-1')
    })

    it('should not delete if user cancels confirmation', async () => {
      const deleteMutateAsync = vi.fn()
      vi.mocked(useTargets).mockReturnValue({
        data: [mockTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      vi.mocked(useDeleteTarget).mockReturnValue({
        mutate: vi.fn(),
        mutateAsync: deleteMutateAsync,
        isPending: false,
      } as unknown as ReturnType<typeof useDeleteTarget>)

      // Mock window.confirm to return false
      vi.spyOn(window, 'confirm').mockReturnValue(false)

      render(<TargetList />, { wrapper: createWrapper() })

      const deleteButton = screen.getByRole('button', { name: /delete/i })
      await userEvent.click(deleteButton)

      expect(deleteMutateAsync).not.toHaveBeenCalled()
    })

    it('should open edit form when edit button clicked', async () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [mockTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      const editButton = screen.getByRole('button', { name: /edit/i })
      await userEvent.click(editButton)

      await waitFor(() => {
        expect(screen.getByText(/edit target/i)).toBeInTheDocument()
      })
    })

    it('should call onScrape callback when scrape button clicked', async () => {
      const onScrape = vi.fn()
      vi.mocked(useTargets).mockReturnValue({
        data: [mockTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList onScrape={onScrape} />, { wrapper: createWrapper() })

      const scrapeButton = screen.getByRole('button', { name: /scrape/i })
      await userEvent.click(scrapeButton)

      expect(onScrape).toHaveBeenCalledWith(mockTarget)
    })

    it('should disable scrape button for inactive targets', () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [mockInactiveTarget],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      const scrapeButton = screen.getByRole('button', { name: /scrape/i })
      expect(scrapeButton).toBeDisabled()
    })
  })

  describe('Add Target', () => {
    it('should show add target form when add button clicked', async () => {
      vi.mocked(useTargets).mockReturnValue({
        data: [],
        isLoading: false,
        error: null,
      } as unknown as ReturnType<typeof useTargets>)

      render(<TargetList />, { wrapper: createWrapper() })

      const addButton = screen.getByRole('button', { name: /add target/i })
      await userEvent.click(addButton)

      await waitFor(() => {
        expect(screen.getByText(/add target/i)).toBeInTheDocument()
      })
    })
  })
})
