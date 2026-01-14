import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Input } from './Input'

describe('Input', () => {
  it('should render an input element', () => {
    render(<Input />)
    expect(screen.getByRole('textbox')).toBeInTheDocument()
  })

  it('should render with type="text" by default', () => {
    render(<Input />)
    expect(screen.getByRole('textbox')).toHaveAttribute('type', 'text')
  })

  it('should render with type="search" when search variant', () => {
    render(<Input variant="search" />)
    expect(screen.getByRole('searchbox')).toBeInTheDocument()
  })

  it('should apply input class', () => {
    render(<Input />)
    expect(screen.getByRole('textbox')).toHaveClass('input')
  })

  it('should apply variant class', () => {
    render(<Input variant="search" />)
    expect(screen.getByRole('searchbox')).toHaveClass('input-search')
  })

  it('should apply size class', () => {
    render(<Input size="lg" />)
    expect(screen.getByRole('textbox')).toHaveClass('input-lg')
  })

  it('should apply sm size class', () => {
    render(<Input size="sm" />)
    expect(screen.getByRole('textbox')).toHaveClass('input-sm')
  })

  it('should apply custom className', () => {
    render(<Input className="custom-class" />)
    expect(screen.getByRole('textbox')).toHaveClass('custom-class')
  })

  it('should be disabled when disabled prop is true', () => {
    render(<Input disabled />)
    expect(screen.getByRole('textbox')).toBeDisabled()
  })

  it('should have placeholder text', () => {
    render(<Input placeholder="Enter text..." />)
    expect(screen.getByRole('textbox')).toHaveAttribute('placeholder', 'Enter text...')
  })

  it('should call onChange when value changes', async () => {
    const handleChange = vi.fn()
    render(<Input onChange={handleChange} />)

    await userEvent.type(screen.getByRole('textbox'), 'hello')
    expect(handleChange).toHaveBeenCalled()
  })

  it('should have proper ARIA attributes when error', () => {
    render(<Input error aria-describedby="error-msg" />)
    expect(screen.getByRole('textbox')).toHaveAttribute('aria-invalid', 'true')
  })

  it('should render with label', () => {
    render(<Input label="Username" id="username" />)
    expect(screen.getByLabelText('Username')).toBeInTheDocument()
  })

  it('should render helper text', () => {
    render(<Input helperText="Enter your username" id="test" />)
    expect(screen.getByText('Enter your username')).toBeInTheDocument()
  })

  it('should render error message', () => {
    render(<Input error errorMessage="This field is required" id="test" />)
    expect(screen.getByText('This field is required')).toBeInTheDocument()
  })
})
