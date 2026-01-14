import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Skeleton } from './Skeleton'

describe('Skeleton', () => {
  it('should render a skeleton element', () => {
    render(<Skeleton />)
    expect(screen.getByTestId('skeleton')).toBeInTheDocument()
  })

  it('should apply skeleton class', () => {
    render(<Skeleton />)
    expect(screen.getByTestId('skeleton')).toHaveClass('skeleton')
  })

  it('should apply variant class', () => {
    render(<Skeleton variant="circle" />)
    expect(screen.getByTestId('skeleton')).toHaveClass('skeleton-circle')
  })

  it('should apply text variant by default', () => {
    render(<Skeleton />)
    expect(screen.getByTestId('skeleton')).toHaveClass('skeleton-text')
  })

  it('should apply rectangle variant', () => {
    render(<Skeleton variant="rectangle" />)
    expect(screen.getByTestId('skeleton')).toHaveClass('skeleton-rectangle')
  })

  it('should apply custom width', () => {
    render(<Skeleton width="100px" />)
    expect(screen.getByTestId('skeleton')).toHaveStyle({ width: '100px' })
  })

  it('should apply custom height', () => {
    render(<Skeleton height="50px" />)
    expect(screen.getByTestId('skeleton')).toHaveStyle({ height: '50px' })
  })

  it('should apply custom className', () => {
    render(<Skeleton className="custom-class" />)
    expect(screen.getByTestId('skeleton')).toHaveClass('custom-class')
  })

  it('should be animating by default', () => {
    render(<Skeleton />)
    expect(screen.getByTestId('skeleton')).toHaveClass('skeleton-animate')
  })

  it('should not animate when animate is false', () => {
    render(<Skeleton animate={false} />)
    expect(screen.getByTestId('skeleton')).not.toHaveClass('skeleton-animate')
  })
})
