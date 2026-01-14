import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Spinner } from './Spinner'

describe('Spinner', () => {
  it('should render a spinner element', () => {
    render(<Spinner />)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })

  it('should apply spinner class', () => {
    render(<Spinner />)
    expect(screen.getByRole('status')).toHaveClass('spinner')
  })

  it('should apply sm size class', () => {
    render(<Spinner size="sm" />)
    expect(screen.getByRole('status')).toHaveClass('spinner-sm')
  })

  it('should apply md size class by default', () => {
    render(<Spinner />)
    expect(screen.getByRole('status')).toHaveClass('spinner-md')
  })

  it('should apply lg size class', () => {
    render(<Spinner size="lg" />)
    expect(screen.getByRole('status')).toHaveClass('spinner-lg')
  })

  it('should have aria-label for accessibility', () => {
    render(<Spinner />)
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Loading...')
  })

  it('should have custom aria-label when provided', () => {
    render(<Spinner label="Fetching data..." />)
    expect(screen.getByRole('status')).toHaveAttribute('aria-label', 'Fetching data...')
  })

  it('should have visually hidden text for screen readers', () => {
    render(<Spinner />)
    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })
})
