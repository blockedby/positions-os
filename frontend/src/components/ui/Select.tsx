import { forwardRef, useId } from 'react'
import type { SelectHTMLAttributes } from 'react'

export type SelectSize = 'sm' | 'md' | 'lg'

export interface SelectOption {
  value: string
  label: string
  disabled?: boolean
}

export interface SelectProps extends Omit<SelectHTMLAttributes<HTMLSelectElement>, 'size'> {
  options?: SelectOption[]
  placeholder?: string
  label?: string
  helperText?: string
  errorMessage?: string
  error?: boolean
  size?: SelectSize
}

const sizeClasses: Record<SelectSize, string> = {
  sm: 'select-sm',
  md: '',
  lg: 'select-lg',
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  (
    {
      options = [],
      placeholder,
      label,
      helperText,
      errorMessage,
      error = false,
      size = 'md',
      className = '',
      id,
      ...props
    },
    ref
  ) => {
    const generatedId = useId()
    const selectId = id || generatedId
    const errorId = `${selectId}-error`
    const helperId = `${selectId}-helper`

    const classes = ['select', sizeClasses[size], error ? 'select-error' : '', className]
      .filter(Boolean)
      .join(' ')

    return (
      <div className="select-wrapper">
        {label && (
          <label htmlFor={selectId} className="select-label">
            {label}
          </label>
        )}
        <select
          ref={ref}
          id={selectId}
          className={classes}
          aria-invalid={error}
          aria-describedby={
            errorMessage ? errorId : helperText ? helperId : undefined
          }
          {...props}
        >
          {placeholder && (
            <option value="" disabled>
              {placeholder}
            </option>
          )}
          {options.map((option) => (
            <option
              key={option.value}
              value={option.value}
              disabled={option.disabled}
            >
              {option.label}
            </option>
          ))}
        </select>
        {errorMessage && error && (
          <span id={errorId} className="select-error-message">
            {errorMessage}
          </span>
        )}
        {helperText && !error && (
          <span id={helperId} className="select-helper-text">
            {helperText}
          </span>
        )}
      </div>
    )
  }
)

Select.displayName = 'Select'
