import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { JobDetail } from './JobDetail'
import { mockJob } from '@/test/test-utils'
import { api } from '@/lib/api'

// Mock the api module
vi.mock('@/lib/api', () => ({
  api: {
    getJob: vi.fn(),
    updateJobStatus: vi.fn(),
    prepareJob: vi.fn(),
  },
}))

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  })

const renderWithClient = (ui: React.ReactElement) => {
  const queryClient = createTestQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  )
}

describe('JobDetail', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  describe('Prepare Application Button', () => {
    it('should show Prepare Application button when job is INTERESTED', async () => {
      const interestedJob = { ...mockJob, status: 'INTERESTED' as const }
      vi.mocked(api.getJob).mockResolvedValueOnce(interestedJob)

      renderWithClient(<JobDetail jobId="job-1" />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /prepare application/i })).toBeInTheDocument()
      })
    })

    it('should not show Prepare Application button when job is ANALYZED', async () => {
      vi.mocked(api.getJob).mockResolvedValueOnce(mockJob) // status is ANALYZED

      renderWithClient(<JobDetail jobId="job-1" />)

      await waitFor(() => {
        expect(screen.getByText(/Go Developer/i)).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: /prepare application/i })).not.toBeInTheDocument()
    })

    it('should call prepareJob when Prepare Application button is clicked', async () => {
      const interestedJob = { ...mockJob, status: 'INTERESTED' as const }
      vi.mocked(api.getJob).mockResolvedValueOnce(interestedJob)
      vi.mocked(api.prepareJob).mockResolvedValueOnce({
        job_id: 'job-1',
        status: 'TAILORED_APPROVED',
        resume_path: '/storage/jobs/job-1/resume.pdf',
        cover_letter_path: '/storage/jobs/job-1/cover_letter.md',
      })

      renderWithClient(<JobDetail jobId="job-1" />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /prepare application/i })).toBeInTheDocument()
      })

      const prepareButton = screen.getByRole('button', { name: /prepare application/i })
      await userEvent.click(prepareButton)

      await waitFor(() => {
        expect(api.prepareJob).toHaveBeenCalledWith('job-1')
      })
    })

    it('should not show Prepare Application button when job is already TAILORED_APPROVED', async () => {
      const tailoredJob = { ...mockJob, status: 'TAILORED_APPROVED' as const }
      vi.mocked(api.getJob).mockResolvedValueOnce(tailoredJob)

      renderWithClient(<JobDetail jobId="job-1" />)

      await waitFor(() => {
        expect(screen.getByText(/Go Developer/i)).toBeInTheDocument()
      })

      expect(screen.queryByRole('button', { name: /prepare application/i })).not.toBeInTheDocument()
    })
  })

  describe('Status Badge', () => {
    it('should display TAILORED_APPROVED status with correct badge', async () => {
      const tailoredJob = { ...mockJob, status: 'TAILORED_APPROVED' as const }
      vi.mocked(api.getJob).mockResolvedValueOnce(tailoredJob)

      renderWithClient(<JobDetail jobId="job-1" />)

      await waitFor(() => {
        expect(screen.getByText('TAILORED_APPROVED')).toBeInTheDocument()
      })
    })
  })
})
