import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ContactLink } from './ContactLink'

describe('ContactLink', () => {
  it('should render email as mailto link', () => {
    render(<ContactLink contact="test@example.com" />)

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'mailto:test@example.com')
    expect(link).toHaveTextContent('test@example.com')
  })

  it('should render telegram handle as tg link', () => {
    render(<ContactLink contact="@username" />)

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'https://t.me/username')
    expect(link).toHaveTextContent('@username')
  })

  it('should render URL as external link', () => {
    render(<ContactLink contact="https://example.com/jobs" />)

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'https://example.com/jobs')
    expect(link).toHaveAttribute('target', '_blank')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('should render phone number as tel link', () => {
    render(<ContactLink contact="+1234567890" />)

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'tel:+1234567890')
  })

  it('should render unknown format as plain text', () => {
    render(<ContactLink contact="some random text" />)

    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(screen.getByText('some random text')).toBeInTheDocument()
  })

  describe('edge cases', () => {
    it('should handle telegram with spaces around @', () => {
      render(<ContactLink contact=" @username " />)

      const link = screen.getByRole('link')
      expect(link).toHaveAttribute('href', 'https://t.me/username')
    })

    it('should detect E.164 formatted phone number', () => {
      render(<ContactLink contact="+14155550123" />)

      const link = screen.getByRole('link')
      expect(link).toHaveAttribute('href', 'tel:+14155550123')
    })

    it('should reject phone number without + prefix', () => {
      render(<ContactLink contact="14155550123" />)

      // Should fall through to unknown (plain text)
      expect(screen.queryByRole('link')).not.toBeInTheDocument()
    })
  })
})
