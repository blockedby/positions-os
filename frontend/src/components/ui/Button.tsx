import { forwardRef, ButtonHTMLAttributes, AnchorHTMLAttributes } from 'react'

export type ButtonVariant = 'primary' | 'secondary' | 'success' | 'danger'
export type ButtonSize = 'sm' | 'md' | 'lg'

type ButtonAsButton = {
  as?: 'button'
  href?: never
} & Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'size'>

type ButtonAsAnchor = {
  as: 'a'
  href?: string
} & Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'size'>

type BaseButtonProps = {
  variant?: ButtonVariant
  size?: ButtonSize
  loading?: boolean
}

export type ButtonProps = BaseButtonProps & (ButtonAsButton | ButtonAsAnchor)

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'btn-primary',
  secondary: 'btn-secondary',
  success: 'btn-success',
  danger: 'btn-danger',
}

const sizeClasses: Record<ButtonSize, string> = {
  sm: 'btn-sm',
  md: 'btn-md',
  lg: 'btn-lg',
}

export const Button = forwardRef<HTMLButtonElement | HTMLAnchorElement, ButtonProps>(
  (props, ref) => {
    const {
      variant = 'primary',
      size = 'md',
      loading = false,
      className = '',
      children,
      as = 'button',
      ...restProps
    } = props

    const classes = [
      'btn',
      variantClasses[variant],
      sizeClasses[size],
      loading ? 'btn-loading' : '',
      className,
    ]
      .filter(Boolean)
      .join(' ')

    if (as === 'a') {
      const anchorProps = restProps as Omit<AnchorHTMLAttributes<HTMLAnchorElement>, 'size'>
      const isDisabled = loading
      return (
        <a
          className={classes}
          aria-disabled={isDisabled}
          href={isDisabled ? undefined : anchorProps.href}
          role="button"
          {...anchorProps}
          ref={ref as React.Ref<HTMLAnchorElement>}
        >
          {children}
        </a>
      )
    }

    const buttonProps = restProps as Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'size'>
    const isDisabled = buttonProps.disabled || loading

    return (
      <button
        ref={ref as React.Ref<HTMLButtonElement>}
        className={classes}
        disabled={isDisabled}
        {...buttonProps}
      >
        {children}
      </button>
    )
  }
)

Button.displayName = 'Button'
