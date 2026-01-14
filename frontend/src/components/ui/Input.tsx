import { forwardRef, InputHTMLAttributes, useId } from 'react'

export type InputVariant = 'text' | 'search' | 'email' | 'password' | 'number'
export type InputSize = 'sm' | 'md' | 'lg'

export interface InputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'size'> {
  variant?: InputVariant
  size?: InputSize
  label?: string
  helperText?: string
  errorMessage?: string
  error?: boolean
}

const variantClasses: Record<InputVariant, string> = {
  text: '',
  search: 'input-search',
  email: '',
  password: '',
  number: '',
}

const sizeClasses: Record<InputSize, string> = {
  sm: 'input-sm',
  md: '',
  lg: 'input-lg',
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  (
    {
      variant = 'text',
      size = 'md',
      label,
      helperText,
      errorMessage,
      error = false,
      className = '',
      id,
      type,
      ...props
    },
    ref
  ) => {
    const generatedId = useId()
    const inputId = id || generatedId
    const errorId = `${inputId}-error`
    const helperId = `${inputId}-helper`

    const classes = [
      'input',
      variantClasses[variant],
      sizeClasses[size],
      error ? 'input-error' : '',
      className,
    ]
      .filter(Boolean)
      .join(' ')

    const inputType = type || (variant === 'search' ? 'search' : variant)

    return (
      <div className="input-wrapper">
        {label && (
          <label htmlFor={inputId} className="input-label">
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          type={inputType}
          className={classes}
          aria-invalid={error}
          aria-describedby={
            errorMessage ? errorId : helperText ? helperId : undefined
          }
          {...props}
        />
        {errorMessage && error && (
          <span id={errorId} className="input-error-message">
            {errorMessage}
          </span>
        )}
        {helperText && !error && (
          <span id={helperId} className="input-helper-text">
            {helperText}
          </span>
        )}
      </div>
    )
  }
)

Input.displayName = 'Input'
