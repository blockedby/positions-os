import { type HTMLAttributes } from 'react'

export type SpinnerSize = 'sm' | 'md' | 'lg'

export interface SpinnerProps extends HTMLAttributes<HTMLSpanElement> {
  size?: SpinnerSize
  label?: string
}

const sizeClasses: Record<SpinnerSize, string> = {
  sm: 'spinner-sm',
  md: 'spinner-md',
  lg: 'spinner-lg',
}

export function Spinner({
  size = 'md',
  label = 'Loading...',
  className = '',
  ...props
}: SpinnerProps) {
  const classes = ['spinner', sizeClasses[size], className]
    .filter(Boolean)
    .join(' ')

  return (
    <span
      className={classes}
      role="status"
      aria-label={label}
      aria-live="polite"
      {...props}
    >
      <span className="sr-only">{label}</span>
    </span>
  )
}
