import { type HTMLAttributes } from 'react'

export type SkeletonVariant = 'text' | 'circle' | 'rectangle'

export interface SkeletonProps extends HTMLAttributes<HTMLDivElement> {
  variant?: SkeletonVariant
  width?: string
  height?: string
  animate?: boolean
}

const variantClasses: Record<SkeletonVariant, string> = {
  text: 'skeleton-text',
  circle: 'skeleton-circle',
  rectangle: 'skeleton-rectangle',
}

export function Skeleton({
  variant = 'text',
  width,
  height,
  animate = true,
  className = '',
  style = {},
  ...props
}: SkeletonProps) {
  const classes = [
    'skeleton',
    variantClasses[variant],
    animate ? 'skeleton-animate' : '',
    className,
  ]
    .filter(Boolean)
    .join(' ')

  const inlineStyle: React.CSSProperties = {
    ...style,
    ...(width && { width }),
    ...(height && { height }),
  }

  return (
    <div
      className={classes}
      style={inlineStyle}
      data-testid="skeleton"
      role="presentation"
      aria-hidden="true"
      {...props}
    />
  )
}
