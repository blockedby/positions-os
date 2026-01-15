import { Component } from 'react'
import type { ReactNode } from 'react'

export interface ErrorBoundaryProps {
  children: ReactNode
  fallback?: ReactNode
  onError?: (error: Error, errorInfo: React.ErrorInfo) => void
}

export interface ErrorBoundaryState {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    this.props.onError?.(error, errorInfo)
  }

  handleReset = () => {
    this.setState({ hasError: false, error: undefined })
  }

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback
      }

      return (
        <div role="alert" className="card error-boundary">
          <h3>Something went wrong</h3>
          <p className="text-muted mb-4">
            An error occurred while rendering this component.
          </p>
          {this.state.error && (
            <details className="mb-4">
              <summary className="text-sm">Error details</summary>
              <pre className="text-xs mt-2">{this.state.error.message}</pre>
            </details>
          )}
          <button onClick={this.handleReset} className="btn btn-secondary btn-sm">
            Try again
          </button>
        </div>
      )
    }

    return this.props.children
  }
}
