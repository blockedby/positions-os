import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { FilterBar } from './FilterBar'

describe('FilterBar', () => {
  describe('Technology Filter', () => {
    it('should render technology input field', () => {
      const onFilter = vi.fn()
      render(<FilterBar onFilter={onFilter} />)

      expect(screen.getByPlaceholderText(/technologies/i)).toBeInTheDocument()
    })

    it('should include technologies in filter when applied', async () => {
      const onFilter = vi.fn()
      render(<FilterBar onFilter={onFilter} />)

      const techInput = screen.getByPlaceholderText(/technologies/i)
      await userEvent.type(techInput, 'go, react')

      const applyButton = screen.getByRole('button', { name: /apply/i })
      await userEvent.click(applyButton)

      expect(onFilter).toHaveBeenCalledWith(
        expect.objectContaining({
          technologies: ['go', 'react'],
        })
      )
    })

    it('should clear technologies when Clear is clicked', async () => {
      const onFilter = vi.fn()
      render(<FilterBar onFilter={onFilter} />)

      const techInput = screen.getByPlaceholderText(/technologies/i)
      await userEvent.type(techInput, 'go, react')

      const clearButton = screen.getByRole('button', { name: /clear/i })
      await userEvent.click(clearButton)

      expect(techInput).toHaveValue('')
      expect(onFilter).toHaveBeenCalledWith({})
    })
  })

  describe('Salary Range Filter', () => {
    it('should render salary min and max inputs', () => {
      const onFilter = vi.fn()
      render(<FilterBar onFilter={onFilter} />)

      expect(screen.getByPlaceholderText(/min salary/i)).toBeInTheDocument()
      expect(screen.getByPlaceholderText(/max salary/i)).toBeInTheDocument()
    })

    it('should include salary range in filter when applied', async () => {
      const onFilter = vi.fn()
      render(<FilterBar onFilter={onFilter} />)

      const minInput = screen.getByPlaceholderText(/min salary/i)
      const maxInput = screen.getByPlaceholderText(/max salary/i)

      await userEvent.type(minInput, '100000')
      await userEvent.type(maxInput, '200000')

      const applyButton = screen.getByRole('button', { name: /apply/i })
      await userEvent.click(applyButton)

      expect(onFilter).toHaveBeenCalledWith(
        expect.objectContaining({
          salary_min: 100000,
          salary_max: 200000,
        })
      )
    })

    it('should allow filtering with only min salary', async () => {
      const onFilter = vi.fn()
      render(<FilterBar onFilter={onFilter} />)

      const minInput = screen.getByPlaceholderText(/min salary/i)
      await userEvent.type(minInput, '150000')

      const applyButton = screen.getByRole('button', { name: /apply/i })
      await userEvent.click(applyButton)

      expect(onFilter).toHaveBeenCalledWith(
        expect.objectContaining({
          salary_min: 150000,
        })
      )
      // salary_max should not be in the call (undefined is not included in objectContaining)
    })

    it('should clear salary inputs when Clear is clicked', async () => {
      const onFilter = vi.fn()
      render(<FilterBar onFilter={onFilter} />)

      const minInput = screen.getByPlaceholderText(/min salary/i)
      const maxInput = screen.getByPlaceholderText(/max salary/i)

      await userEvent.type(minInput, '100000')
      await userEvent.type(maxInput, '200000')

      const clearButton = screen.getByRole('button', { name: /clear/i })
      await userEvent.click(clearButton)

      expect(minInput).toHaveValue(null) // number inputs return null when empty
      expect(maxInput).toHaveValue(null)
      expect(onFilter).toHaveBeenCalledWith({})
    })
  })
})
