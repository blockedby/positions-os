import { HTMLAttributes } from 'react'

export interface CardProps extends HTMLAttributes<HTMLElement> {
  as?: 'article' | 'div' | 'section'
}

export const Card = ({
  as = 'article',
  className = '',
  children,
  ...props
}: CardProps) => {
  const classes = ['card', className].filter(Boolean).join(' ')

  const Tag = as

  return (
    <Tag className={classes} {...props}>
      {children}
    </Tag>
  )
}
