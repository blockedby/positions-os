import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TargetForm } from './TargetForm'
import { mockTarget } from '../../test/test-utils'

// Mock the hooks
vi.mock('../../hooks/useTargets', () => ({
  useCreateTarget: vi.fn(),
  useUpdateTarget: vi.fn(),
}))

import { useCreateTarget, useUpdateTarget } from '../../hooks/useTargets'

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

describe('TargetForm', () => {
  const mockOnCancel = vi.fn()
  const mockOnSuccess = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()

    vi.mocked(useCreateTarget).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn().mockResolvedValue(mockTarget),
      isPending: false,
    } as unknown as ReturnType<typeof useCreateTarget>)

    vi.mocked(useUpdateTarget).mockReturnValue({
      mutate: vi.fn(),
      mutateAsync: vi.fn().mockResolvedValue(mockTarget),
      isPending: false,
    } as unknown as ReturnType<typeof useUpdateTarget>)
  })

  describe('Create Mode', () => {
    it('should render empty form for creating new target', () => {
      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      expect(screen.getByText(/add target/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /create/i })).toBeInTheDocument()
    })

    it('should have empty input fields initially', () => {
      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      const nameInput = screen.getByLabelText(/name/i) as HTMLInputElement
      const urlInput = screen.getByLabelText(/url/i) as HTMLInputElement

      expect(nameInput.value).toBe('')
      expect(urlInput.value).toBe('')
    })

    it('should call createTarget mutation on submit', async () => {
      const createMutateAsync = vi.fn().mockResolvedValue(mockTarget)
      vi.mocked(useCreateTarget).mockReturnValue({
        mutate: vi.fn(),
        mutateAsync: createMutateAsync,
        isPending: false,
      } as unknown as ReturnType<typeof useCreateTarget>)

      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      await userEvent.type(screen.getByLabelText(/name/i), 'New Channel')
      await userEvent.type(screen.getByLabelText(/url/i), '@new_channel')

      const submitButton = screen.getByRole('button', { name: /create/i })
      await userEvent.click(submitButton)

      await waitFor(() => {
        expect(createMutateAsync).toHaveBeenCalledWith(
          expect.objectContaining({
            name: 'New Channel',
            url: '@new_channel',
          })
        )
      })
    })
  })

  describe('Edit Mode', () => {
    it('should render form with target data for editing', () => {
      render(
        <TargetForm target={mockTarget} onCancel={mockOnCancel} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      )

      expect(screen.getByText(/edit target/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument()
    })

    it('should populate fields with target data', () => {
      render(
        <TargetForm target={mockTarget} onCancel={mockOnCancel} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      )

      const nameInput = screen.getByLabelText(/name/i) as HTMLInputElement
      const urlInput = screen.getByLabelText(/url/i) as HTMLInputElement

      expect(nameInput.value).toBe('Go Jobs Channel')
      expect(urlInput.value).toBe('@go_jobs')
    })

    it('should call updateTarget mutation on submit', async () => {
      const updateMutateAsync = vi.fn().mockResolvedValue(mockTarget)
      vi.mocked(useUpdateTarget).mockReturnValue({
        mutate: vi.fn(),
        mutateAsync: updateMutateAsync,
        isPending: false,
      } as unknown as ReturnType<typeof useUpdateTarget>)

      render(
        <TargetForm target={mockTarget} onCancel={mockOnCancel} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      )

      const nameInput = screen.getByLabelText(/name/i)
      await userEvent.clear(nameInput)
      await userEvent.type(nameInput, 'Updated Channel')

      const submitButton = screen.getByRole('button', { name: /save/i })
      await userEvent.click(submitButton)

      await waitFor(() => {
        expect(updateMutateAsync).toHaveBeenCalledWith(
          expect.objectContaining({
            id: 'target-1',
          })
        )
      })
    })
  })

  describe('Validation', () => {
    it('should show validation error when name is empty', async () => {
      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      // Fill URL but not name
      await userEvent.type(screen.getByLabelText(/url/i), '@test_channel')

      const submitButton = screen.getByRole('button', { name: /create/i })
      await userEvent.click(submitButton)

      await waitFor(() => {
        expect(screen.getByText(/name is required/i)).toBeInTheDocument()
      })
    })

    it('should show validation error when URL is empty', async () => {
      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      // Fill name but not URL
      await userEvent.type(screen.getByLabelText(/name/i), 'Test Channel')

      const submitButton = screen.getByRole('button', { name: /create/i })
      await userEvent.click(submitButton)

      await waitFor(() => {
        expect(screen.getByText(/url is required/i)).toBeInTheDocument()
      })
    })
  })

  describe('Form Actions', () => {
    it('should call onCancel when cancel button clicked', async () => {
      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      await userEvent.click(cancelButton)

      expect(mockOnCancel).toHaveBeenCalled()
    })

    it('should call onSuccess after successful creation', async () => {
      const createMutateAsync = vi.fn().mockResolvedValue(mockTarget)
      vi.mocked(useCreateTarget).mockReturnValue({
        mutate: vi.fn(),
        mutateAsync: createMutateAsync,
        isPending: false,
      } as unknown as ReturnType<typeof useCreateTarget>)

      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      await userEvent.type(screen.getByLabelText(/name/i), 'New Channel')
      await userEvent.type(screen.getByLabelText(/url/i), '@new_channel')

      const submitButton = screen.getByRole('button', { name: /create/i })
      await userEvent.click(submitButton)

      await waitFor(() => {
        expect(mockOnSuccess).toHaveBeenCalled()
      })
    })
  })

  describe('Loading State', () => {
    it('should disable submit button while creating', () => {
      vi.mocked(useCreateTarget).mockReturnValue({
        mutate: vi.fn(),
        mutateAsync: vi.fn(),
        isPending: true,
      } as unknown as ReturnType<typeof useCreateTarget>)

      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      // Button should be disabled and have loading class when isPending
      const submitButton = screen.getByRole('button', { name: /create/i })
      expect(submitButton).toBeDisabled()
      expect(submitButton).toHaveClass('btn-loading')
    })

    it('should disable submit button while updating', () => {
      vi.mocked(useUpdateTarget).mockReturnValue({
        mutate: vi.fn(),
        mutateAsync: vi.fn(),
        isPending: true,
      } as unknown as ReturnType<typeof useUpdateTarget>)

      render(
        <TargetForm target={mockTarget} onCancel={mockOnCancel} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      )

      const submitButton = screen.getByRole('button', { name: /save/i })
      expect(submitButton).toBeDisabled()
      expect(submitButton).toHaveClass('btn-loading')
    })
  })

  describe('Type Selection', () => {
    it('should have type selector with TG_CHANNEL option in create mode', () => {
      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      const typeSelect = screen.getByLabelText(/type/i) as HTMLSelectElement
      expect(typeSelect).toBeInTheDocument()
    })

    it('should not show type selector in edit mode', () => {
      render(
        <TargetForm target={mockTarget} onCancel={mockOnCancel} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      )

      // Type selector should not be present in edit mode
      expect(screen.queryByLabelText(/type/i)).not.toBeInTheDocument()
    })
  })

  describe('Active Toggle', () => {
    it('should have active checkbox checked by default in create mode', () => {
      render(<TargetForm onCancel={mockOnCancel} onSuccess={mockOnSuccess} />, {
        wrapper: createWrapper(),
      })

      const activeCheckbox = document.querySelector('#isActive') as HTMLInputElement
      expect(activeCheckbox.checked).toBe(true)
    })

    it('should reflect target active status in edit mode', () => {
      const inactiveTarget = { ...mockTarget, is_active: false }

      render(
        <TargetForm target={inactiveTarget} onCancel={mockOnCancel} onSuccess={mockOnSuccess} />,
        { wrapper: createWrapper() }
      )

      const activeCheckbox = document.querySelector('#isActive') as HTMLInputElement
      expect(activeCheckbox.checked).toBe(false)
    })
  })
})
