import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Card } from './Card'

describe('Card', () => {
  it('should render a card element', () => {
    const { container } = render(<Card>Card content</Card>)
    expect(screen.getByText('Card content')).toBeInTheDocument()
    expect(container.querySelector('.card')).toBeInTheDocument()
  })

  it('should apply card class', () => {
    const { container } = render(<Card>Card content</Card>)
    const card = container.querySelector('.card')
    expect(card).toHaveClass('card')
  })

  it('should render children', () => {
    render(
      <Card>
        <h2>Title</h2>
        <p>Description</p>
      </Card>
    )
    expect(screen.getByText('Title')).toBeInTheDocument()
    expect(screen.getByText('Description')).toBeInTheDocument()
  })

  it('should apply custom className', () => {
    const { container } = render(<Card className="custom-class">Card content</Card>)
    const card = container.querySelector('.card')
    expect(card).toHaveClass('custom-class')
  })

  it('should render with default article element', () => {
    const { container } = render(<Card>Card content</Card>)
    const card = container.querySelector('.card')
    expect(card?.tagName).toBe('ARTICLE')
  })

  it('should forward HTML attributes', () => {
    const { container } = render(<Card data-testid="test-card">Card content</Card>)
    const card = container.querySelector('.card')
    expect(card).toHaveAttribute('data-testid', 'test-card')
  })

  it('should render as div when as="div"', () => {
    const { container } = render(<Card as="div">Card content</Card>)
    const card = container.querySelector('.card')
    expect(card?.tagName).toBe('DIV')
  })

  it('should render as section when as="section"', () => {
    const { container } = render(<Card as="section">Card content</Card>)
    const card = container.querySelector('.card')
    expect(card?.tagName).toBe('SECTION')
  })
})
