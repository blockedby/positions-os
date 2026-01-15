import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Select } from './Select'

describe('Select', () => {
  const options = [
    { value: 'us', label: 'United States' },
    { value: 'uk', label: 'United Kingdom' },
    { value: 'ca', label: 'Canada' },
  ]

  it('should render a select element', () => {
    render(<Select options={options} />)
    expect(screen.getByRole('combobox')).toBeInTheDocument()
  })

  it('should render options', () => {
    const { container } = render(<Select options={options} />)
    expect(screen.getByRole('combobox')).toBeInTheDocument()
    expect(container.querySelectorAll('option')).toHaveLength(3)
  })

  it('should apply select class', () => {
    render(<Select options={options} />)
    expect(screen.getByRole('combobox')).toHaveClass('select')
  })

  it('should apply size class', () => {
    render(<Select options={options} size="lg" />)
    expect(screen.getByRole('combobox')).toHaveClass('select-lg')
  })

  it('should apply sm size class', () => {
    render(<Select options={options} size="sm" />)
    expect(screen.getByRole('combobox')).toHaveClass('select-sm')
  })

  it('should apply custom className', () => {
    render(<Select options={options} className="custom-class" />)
    expect(screen.getByRole('combobox')).toHaveClass('custom-class')
  })

  it('should be disabled when disabled prop is true', () => {
    render(<Select options={options} disabled />)
    expect(screen.getByRole('combobox')).toBeDisabled()
  })

  it('should have placeholder option when provided', () => {
    render(<Select options={options} placeholder="Select a country" />)
    expect(screen.getByText('Select a country')).toBeInTheDocument()
  })

  it('should call onChange when value changes', async () => {
    const handleChange = vi.fn()
    render(<Select options={options} onChange={handleChange} />)

    const select = screen.getByRole('combobox')
    await userEvent.selectOptions(select, 'uk')
    expect(handleChange).toHaveBeenCalled()
  })

  it('should render with label', () => {
    render(<Select options={options} label="Country" id="country" />)
    expect(screen.getByLabelText('Country')).toBeInTheDocument()
  })

  it('should render helper text', () => {
    render(<Select options={options} helperText="Choose your country" id="test" />)
    expect(screen.getByText('Choose your country')).toBeInTheDocument()
  })

  it('should render error message', () => {
    render(<Select options={options} error errorMessage="This field is required" id="test" />)
    expect(screen.getByText('This field is required')).toBeInTheDocument()
  })

  it('should have proper ARIA attributes when error', () => {
    render(<Select options={options} error aria-describedby="error-msg" id="test" />)
    expect(screen.getByRole('combobox')).toHaveAttribute('aria-invalid', 'true')
  })
})
