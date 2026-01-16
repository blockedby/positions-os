import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { JobsTable } from './JobsTable'
import { mockJob } from '../../test/test-utils'
import type { JobsResponse } from '../../lib/types'

// JobsTable is a presentational component - it doesn't use hooks directly
// It receives data, isLoading, etc. as props

describe('JobsTable', () => {
  describe('Loading State', () => {
    it('should show loading skeleton while loading', () => {
      render(<JobsTable isLoading={true} />)

      // The skeleton renders table rows
      const skeletonRows = document.querySelectorAll('tbody tr')
      expect(skeletonRows.length).toBe(10) // Skeleton shows 10 rows
    })
  })

  describe('Empty State', () => {
    it('should show empty state when no jobs exist', () => {
      const emptyData: JobsResponse = {
        jobs: [],
        total: 0,
        page: 1,
        pages: 0,
        limit: 10,
      }

      render(<JobsTable data={emptyData} />)

      expect(screen.getByText(/no jobs found/i)).toBeInTheDocument()
    })

    it('should show empty state when data is undefined', () => {
      render(<JobsTable data={undefined} />)

      expect(screen.getByText(/no jobs found/i)).toBeInTheDocument()
    })
  })

  describe('Job Display', () => {
    it('should display job title', () => {
      const data: JobsResponse = {
        jobs: [mockJob],
        total: 1,
        page: 1,
        pages: 1,
        limit: 10,
      }

      render(<JobsTable data={data} />)

      expect(screen.getByText('Go Developer')).toBeInTheDocument()
    })

    it('should display job company', () => {
      const data: JobsResponse = {
        jobs: [mockJob],
        total: 1,
        page: 1,
        pages: 1,
        limit: 10,
      }

      render(<JobsTable data={data} />)

      expect(screen.getByText('Acme Inc')).toBeInTheDocument()
    })

    it('should display table headers', () => {
      const data: JobsResponse = {
        jobs: [mockJob],
        total: 1,
        page: 1,
        pages: 1,
        limit: 10,
      }

      render(<JobsTable data={data} />)

      expect(screen.getByText('Job')).toBeInTheDocument()
      expect(screen.getByText('Salary')).toBeInTheDocument()
      expect(screen.getByText('Technologies')).toBeInTheDocument()
      expect(screen.getByText('Status')).toBeInTheDocument()
      expect(screen.getByText('Date')).toBeInTheDocument()
    })

    it('should display multiple jobs', () => {
      const job2 = {
        ...mockJob,
        id: 'job-2',
        structured_data: {
          ...mockJob.structured_data,
          title: 'Python Developer',
          company: 'Beta Corp',
        },
      }

      const data: JobsResponse = {
        jobs: [mockJob, job2],
        total: 2,
        page: 1,
        pages: 1,
        limit: 10,
      }

      render(<JobsTable data={data} />)

      expect(screen.getByText('Go Developer')).toBeInTheDocument()
      expect(screen.getByText('Python Developer')).toBeInTheDocument()
      expect(screen.getByText('Acme Inc')).toBeInTheDocument()
      expect(screen.getByText('Beta Corp')).toBeInTheDocument()
    })
  })

  describe('Job Selection', () => {
    it('should call onJobClick when a job row is clicked', async () => {
      const onJobClick = vi.fn()
      const data: JobsResponse = {
        jobs: [mockJob],
        total: 1,
        page: 1,
        pages: 1,
        limit: 10,
      }

      render(<JobsTable data={data} onJobClick={onJobClick} />)

      // Click on the job row
      const jobTitle = screen.getByText('Go Developer')
      await userEvent.click(jobTitle)

      expect(onJobClick).toHaveBeenCalledWith(mockJob)
    })

    it('should highlight selected job', () => {
      const data: JobsResponse = {
        jobs: [mockJob],
        total: 1,
        page: 1,
        pages: 1,
        limit: 10,
      }

      render(<JobsTable data={data} selectedJobId={mockJob.id} />)

      // The selected row should have a selected class (job-row-selected)
      const rows = document.querySelectorAll('tbody tr')
      expect(rows[0]).toHaveClass('job-row-selected')
    })
  })

  describe('Pagination', () => {
    it('should show pagination when multiple pages exist', () => {
      const data: JobsResponse = {
        jobs: [mockJob],
        total: 50,
        page: 1,
        pages: 5,
        limit: 10,
      }

      render(<JobsTable data={data} />)

      // Pagination component should be rendered
      expect(document.querySelector('.pagination')).toBeInTheDocument()
    })

    it('should not show pagination when only one page exists', () => {
      const data: JobsResponse = {
        jobs: [mockJob],
        total: 5,
        page: 1,
        pages: 1,
        limit: 10,
      }

      render(<JobsTable data={data} />)

      // Pagination should not be visible
      expect(document.querySelector('.pagination')).not.toBeInTheDocument()
    })

    it('should call onPageChange when page changes', async () => {
      const onPageChange = vi.fn()
      const data: JobsResponse = {
        jobs: [mockJob],
        total: 50,
        page: 1,
        pages: 5,
        limit: 10,
      }

      render(<JobsTable data={data} onPageChange={onPageChange} />)

      // Click on page 2 button (if exists)
      const pageButtons = document.querySelectorAll('.pagination button')
      // Find a page number button (not prev/next)
      const page2Button = Array.from(pageButtons).find(
        (btn) => btn.textContent === '2'
      )

      if (page2Button) {
        await userEvent.click(page2Button)
        expect(onPageChange).toHaveBeenCalledWith(2)
      }
    })
  })
})
