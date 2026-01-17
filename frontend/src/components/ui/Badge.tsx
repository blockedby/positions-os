import type { HTMLAttributes } from 'react'

export type BadgeStatus =
  | 'raw'
  | 'analyzed'
  | 'interested'
  | 'rejected'
  | 'paused'
  | 'tailored'
  | 'tailored_approved'
  | 'sent'
  | 'responded'

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  status: BadgeStatus
}

const statusClasses: Record<BadgeStatus, string> = {
  raw: 'badge-raw',
  analyzed: 'badge-analyzed',
  interested: 'badge-interested',
  rejected: 'badge-rejected',
  paused: 'badge-paused',
  tailored: 'badge-tailored',
  tailored_approved: 'badge-tailored', // Same purple as tailored
  sent: 'badge-sent',
  responded: 'badge-responded',
}

export const Badge = ({ status, className = '', children, ...props }: BadgeProps) => {
  const classes = ['status-badge', statusClasses[status], className]
    .filter(Boolean)
    .join(' ')

  return (
    <span className={classes} {...props}>
      {children}
    </span>
  )
}
