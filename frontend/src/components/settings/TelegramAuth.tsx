import { useState, useEffect } from 'react'
import { Card, Button, Spinner } from '@/components/ui'

export interface TelegramAuthProps {
  className?: string
}

type AuthState = 'idle' | 'loading' | 'qr' | 'code' | 'connected' | 'error'

export const TelegramAuth = ({ className = '' }: TelegramAuthProps) => {
  const [state, setState] = useState<AuthState>('idle')
  const [qrCode, setQrCode] = useState<string>('')
  const [error, setError] = useState<string>('')

  // Check connection status on mount
  useEffect(() => {
    checkConnection()
  }, [])

  const checkConnection = async () => {
    try {
      const response = await fetch('/api/v1/telegram/status')
      if (response.ok) {
        const data = await response.json()
        setState(data.connected ? 'connected' : 'idle')
      }
    } catch {
      // API might not exist yet
      setState('idle')
    }
  }

  const startAuth = async () => {
    setState('loading')
    setError('')

    try {
      const response = await fetch('/api/v1/telegram/auth/start', {
        method: 'POST',
      })

      if (!response.ok) {
        throw new Error('Failed to start authentication')
      }

      const data = await response.json()
      if (data.qr_code) {
        setQrCode(data.qr_code)
        setState('qr')
        pollAuthStatus()
      } else if (data.need_code) {
        setState('code')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed')
      setState('error')
    }
  }

  const pollAuthStatus = () => {
    const interval = setInterval(async () => {
      try {
        const response = await fetch('/api/v1/telegram/auth/status')
        if (response.ok) {
          const data = await response.json()
          if (data.connected) {
            setState('connected')
            clearInterval(interval)
          } else if (data.expired) {
            setState('idle')
            clearInterval(interval)
          }
        }
      } catch {
        // Continue polling
      }
    }, 2000)

    // Stop polling after 5 minutes
    setTimeout(() => clearInterval(interval), 5 * 60 * 1000)
  }

  return (
    <Card className={`telegram-auth ${className}`}>
      <h3>Telegram Connection</h3>

      {state === 'connected' && (
        <div className="auth-status auth-connected">
          <span className="status-icon">&#x2713;</span>
          <div className="status-text">
            <p className="status-title">Connected</p>
            <p className="text-xs text-muted">Your Telegram account is connected</p>
          </div>
        </div>
      )}

      {state === 'idle' && (
        <div className="auth-idle">
          <p className="text-muted mb-4">
            Connect your Telegram account to scrape channels and groups.
          </p>
          <Button variant="primary" onClick={startAuth}>
            Connect Telegram
          </Button>
        </div>
      )}

      {state === 'loading' && (
        <div className="auth-loading">
          <Spinner size="lg" />
          <p className="text-muted mt-4">Starting authentication...</p>
        </div>
      )}

      {state === 'qr' && qrCode && (
        <div className="auth-qr">
          <p className="text-muted mb-4">
            Scan this QR code with Telegram on your phone:
          </p>
          <div className="qr-container">
            <img src={qrCode} alt="Telegram QR Code" className="qr-image" />
          </div>
          <ol className="auth-instructions text-xs text-muted mt-4">
            <li>Open Telegram on your phone</li>
            <li>Go to Settings &gt; Devices &gt; Link Desktop Device</li>
            <li>Scan this QR code</li>
          </ol>
          <Button variant="secondary" size="sm" onClick={() => setState('idle')} className="mt-4">
            Cancel
          </Button>
        </div>
      )}

      {state === 'error' && (
        <div className="auth-error">
          <p className="text-danger mb-4">{error || 'Authentication failed'}</p>
          <Button variant="primary" onClick={startAuth}>
            Try Again
          </Button>
        </div>
      )}

      <div className="auth-help text-xs text-muted mt-4">
        <p>
          <strong>Note:</strong> We use your account to read public channels and groups.
          Your credentials are never stored on our servers.
        </p>
      </div>
    </Card>
  )
}
