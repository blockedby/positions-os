import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Button } from './Button'

describe('Button', () => {
  it('should render a button element', () => {
    render(<Button>Click me</Button>)
    expect(screen.getByRole('button')).toBeInTheDocument()
  })

  it('should render children text', () => {
    render(<Button>Click me</Button>)
    expect(screen.getByRole('button')).toHaveTextContent('Click me')
  })

  it('should apply primary variant class', () => {
    render(<Button variant="primary">Primary</Button>)
    expect(screen.getByRole('button')).toHaveClass('btn-primary')
  })

  it('should apply secondary variant class', () => {
    render(<Button variant="secondary">Secondary</Button>)
    expect(screen.getByRole('button')).toHaveClass('btn-secondary')
  })

  it('should apply success variant class', () => {
    render(<Button variant="success">Success</Button>)
    expect(screen.getByRole('button')).toHaveClass('btn-success')
  })

  it('should apply danger variant class', () => {
    render(<Button variant="danger">Danger</Button>)
    expect(screen.getByRole('button')).toHaveClass('btn-danger')
  })

  it('should apply small size class', () => {
    render(<Button size="sm">Small</Button>)
    expect(screen.getByRole('button')).toHaveClass('btn-sm')
  })

  it('should apply medium size class by default', () => {
    render(<Button>Default</Button>)
    expect(screen.getByRole('button')).toHaveClass('btn-md')
  })

  it('should apply large size class', () => {
    render(<Button size="lg">Large</Button>)
    expect(screen.getByRole('button')).toHaveClass('btn-lg')
  })

  it('should show loading state', () => {
    render(<Button loading>Loading</Button>)
    expect(screen.getByRole('button')).toHaveClass('btn-loading')
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('should be disabled when disabled prop is true', () => {
    render(<Button disabled>Disabled</Button>)
    expect(screen.getByRole('button')).toBeDisabled()
  })

  it('should not call onClick when disabled', async () => {
    const handleClick = vi.fn()
    render(<Button onClick={handleClick} disabled>Click me</Button>)
    await userEvent.click(screen.getByRole('button'))
    expect(handleClick).not.toHaveBeenCalled()
  })

  it('should call onClick when clicked', async () => {
    const handleClick = vi.fn()
    render(<Button onClick={handleClick}>Click me</Button>)
    await userEvent.click(screen.getByRole('button'))
    expect(handleClick).toHaveBeenCalledTimes(1)
  })

  it('should render as a link when using as="a" prop', () => {
    render(<Button as="a" href="https://example.com">Link</Button>)
    const link = screen.getByRole('button')
    expect(link.tagName).toBe('A')
    expect(link).toHaveAttribute('href', 'https://example.com')
  })

  it('should have proper ARIA label when provided', () => {
    render(<Button aria-label="Close dialog">×</Button>)
    expect(screen.getByRole('button')).toHaveAttribute('aria-label', 'Close dialog')
  })
})
