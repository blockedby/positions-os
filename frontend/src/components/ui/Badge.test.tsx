import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Badge } from './Badge'

describe('Badge', () => {
  it('should render a badge element', () => {
    render(<Badge>RAW</Badge>)
    expect(screen.getByText('RAW')).toBeInTheDocument()
  })

  it('should apply status-badge class', () => {
    render(<Badge status="raw">RAW</Badge>)
    const badge = screen.getByText('RAW')
    expect(badge).toHaveClass('status-badge')
  })

  it('should apply raw status class', () => {
    render(<Badge status="raw">RAW</Badge>)
    const badge = screen.getByText('RAW')
    expect(badge).toHaveClass('badge-raw')
  })

  it('should apply analyzed status class', () => {
    render(<Badge status="analyzed">ANALYZED</Badge>)
    const badge = screen.getByText('ANALYZED')
    expect(badge).toHaveClass('badge-analyzed')
  })

  it('should apply interested status class', () => {
    render(<Badge status="interested">INTERESTED</Badge>)
    const badge = screen.getByText('INTERESTED')
    expect(badge).toHaveClass('badge-interested')
  })

  it('should apply rejected status class', () => {
    render(<Badge status="rejected">REJECTED</Badge>)
    const badge = screen.getByText('REJECTED')
    expect(badge).toHaveClass('badge-rejected')
  })

  it('should apply paused status class', () => {
    render(<Badge status="paused">PAUSED</Badge>)
    const badge = screen.getByText('PAUSED')
    expect(badge).toHaveClass('badge-paused')
  })

  it('should apply custom className', () => {
    render(<Badge status="raw" className="custom-class">RAW</Badge>)
    const badge = screen.getByText('RAW')
    expect(badge).toHaveClass('custom-class')
  })

  it('should render with default span element', () => {
    render(<Badge status="raw">RAW</Badge>)
    const badge = screen.getByText('RAW')
    expect(badge.tagName).toBe('SPAN')
  })
})
