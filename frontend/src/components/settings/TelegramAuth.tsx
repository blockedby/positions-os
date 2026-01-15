/**
 * TelegramAuth Component
 *
 * Implements Telegram QR authentication flow according to:
 * docs/tg-auth-frontend-guidelines.md
 *
 * Features:
 * - QR code display using qrcode.react
 * - 30-second countdown timer
 * - LocalStorage persistence for page reloads
 * - WebSocket event handling (tg_qr, tg_auth_success, error)
 * - Debounced connect button
 */

import { useState, useEffect, useCallback, useRef } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { Card, Button, Spinner } from '@/components/ui'
import { api } from '@/lib/api'
import { QRPersistence } from '@/lib/qr-persistence'
import { useWebSocket } from '@/hooks/useWebSocket'
import type { TelegramStatus, WSEvent } from '@/lib/types'

export interface TelegramAuthProps {
  className?: string
}

type AuthState = 'loading' | 'disconnected' | 'qr_displayed' | 'qr_expired' | 'connected'

export const TelegramAuth = ({ className = '' }: TelegramAuthProps) => {
  const [state, setState] = useState<AuthState>('loading')
  const [, setStatus] = useState<TelegramStatus>('INITIALIZING')
  const [qrUrl, setQrUrl] = useState<string>('')
  const [qrTimestamp, setQrTimestamp] = useState<number>(0)
  const [remainingTime, setRemainingTime] = useState<number>(30)
  const [error, setError] = useState<string>('')
  const [isButtonDisabled, setIsButtonDisabled] = useState(false)
  const lastQRUrlRef = useRef<string>('')
  const timerRef = useRef<number | null>(null)

  // WebSocket event handler
  const handleWSEvent = useCallback((event: WSEvent) => {
    switch (event.type) {
      case 'tg_qr':
        // Skip duplicate QR codes
        if (event.url === lastQRUrlRef.current) {
          return
        }
        lastQRUrlRef.current = event.url
        setQrUrl(event.url)
        setQrTimestamp(Date.now())
        setState('qr_displayed')
        setError('')
        QRPersistence.save(event.url)
        break

      case 'tg_auth_success':
        setState('connected')
        setStatus('READY')
        setQrUrl('')
        setError('')
        QRPersistence.clear()
        if (timerRef.current) {
          clearInterval(timerRef.current)
        }
        break

      case 'error':
        setError(event.message || 'Authentication failed')
        setState('disconnected')
        setQrUrl('')
        QRPersistence.clear()
        if (timerRef.current) {
          clearInterval(timerRef.current)
        }
        break
    }
  }, [])

  // Initialize WebSocket
  useWebSocket({ enabled: true, onEvent: handleWSEvent })

  // Check auth status on mount
  useEffect(() => {
    checkAuthStatus()
  }, [])

  // Countdown timer for QR expiry
  useEffect(() => {
    if (state === 'qr_displayed' && qrTimestamp > 0) {
      // Clear existing timer
      if (timerRef.current) {
        clearInterval(timerRef.current)
      }

      // Update timer every second
      timerRef.current = window.setInterval(() => {
        const remaining = QRPersistence.getRemainingTime(qrTimestamp)
        setRemainingTime(remaining)

        if (remaining <= 0) {
          setState('qr_expired')
          setQrUrl('')
          QRPersistence.clear()
          if (timerRef.current) {
            clearInterval(timerRef.current)
          }
        }
      }, 1000)

      return () => {
        if (timerRef.current) {
          clearInterval(timerRef.current)
        }
      }
    }
  }, [state, qrTimestamp])

  const checkAuthStatus = async () => {
    try {
      const data = await api.getAuthStatus()
      setStatus(data.status)

      if (data.status === 'READY') {
        // Already connected
        setState('connected')
        QRPersistence.clear()
      } else {
        // Not connected - check for saved QR
        const savedQR = QRPersistence.load()
        if (savedQR) {
          // Restore QR with remaining time
          setQrUrl(savedQR.url)
          setQrTimestamp(Date.now() - savedQR.ageSeconds * 1000)
          setState('qr_displayed')
          lastQRUrlRef.current = savedQR.url
        } else {
          setState('disconnected')
        }
      }
    } catch (err) {
      console.error('Failed to check auth status:', err)
      setState('disconnected')
    }
  }

  const startAuth = async () => {
    // Prevent double-click
    if (isButtonDisabled) {
      return
    }

    setIsButtonDisabled(true)
    setState('loading')
    setError('')

    try {
      const response = await api.startQR()

      if (response.error) {
        setError(response.error)
        setState('disconnected')
      } else if (response.status === 'already in progress') {
        // QR flow already running, wait for WebSocket event
        setState('loading')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start authentication')
      setState('disconnected')
    } finally {
      // Re-enable button after 3 seconds (prevent spam)
      setTimeout(() => {
        setIsButtonDisabled(false)
      }, 3000)
    }
  }

  const cancelQR = () => {
    setState('disconnected')
    setQrUrl('')
    setError('')
    QRPersistence.clear()
    if (timerRef.current) {
      clearInterval(timerRef.current)
    }
  }

  return (
    <Card className={`telegram-auth ${className}`}>
      <h3>Telegram Connection</h3>

      {/* CONNECTED STATE */}
      {state === 'connected' && (
        <div className="auth-status auth-connected">
          <span className="status-icon" style={{ fontSize: '24px', color: 'var(--pico-success)' }}>
            ✓
          </span>
          <div className="status-text">
            <p className="status-title">Connected</p>
            <p className="text-xs text-muted">Your Telegram account is connected</p>
          </div>
        </div>
      )}

      {/* DISCONNECTED STATE */}
      {state === 'disconnected' && (
        <div className="auth-idle">
          <p className="text-muted mb-4">
            Connect your Telegram account to scrape channels and groups.
          </p>
          <Button variant="primary" onClick={startAuth} disabled={isButtonDisabled}>
            {isButtonDisabled ? 'Starting...' : 'Connect Telegram'}
          </Button>
        </div>
      )}

      {/* LOADING STATE */}
      {state === 'loading' && (
        <div className="auth-loading" style={{ textAlign: 'center', padding: '2rem 0' }}>
          <Spinner size="lg" />
          <p className="text-muted mt-4">Starting authentication...</p>
        </div>
      )}

      {/* QR DISPLAYED STATE */}
      {state === 'qr_displayed' && qrUrl && (
        <div className="auth-qr">
          <p className="text-muted mb-4">Scan this QR code with Telegram on your phone:</p>

          {/* QR Code */}
          <div
            className="qr-container"
            style={{
              display: 'flex',
              justifyContent: 'center',
              padding: '1rem',
              background: 'white',
              borderRadius: '0.5rem',
              marginBottom: '1rem',
            }}
          >
            <QRCodeSVG value={qrUrl} size={200} level="M" />
          </div>

          {/* Timer */}
          <div className="qr-timer text-center mb-4">
            <p className="text-sm">
              QR code expires in <strong>{remainingTime}</strong> seconds
            </p>
          </div>

          {/* Instructions */}
          <ol className="auth-instructions text-xs text-muted mb-4" style={{ paddingLeft: '1.5rem' }}>
            <li>Open Telegram on your phone</li>
            <li>Go to Settings → Devices → Link Desktop Device</li>
            <li>Scan this QR code</li>
          </ol>

          <Button variant="secondary" size="sm" onClick={cancelQR}>
            Cancel
          </Button>
        </div>
      )}

      {/* QR EXPIRED STATE */}
      {state === 'qr_expired' && (
        <div className="auth-expired">
          <p className="text-muted mb-4">QR code expired. Click to generate a new one.</p>
          <Button variant="primary" onClick={startAuth} disabled={isButtonDisabled}>
            Try Again
          </Button>
        </div>
      )}

      {/* ERROR STATE */}
      {error && state !== 'connected' && (
        <div className="auth-error mt-4">
          <p className="text-danger" style={{ color: 'var(--pico-error)', fontSize: '0.875rem' }}>
            {error}
          </p>
        </div>
      )}

      {/* HELP TEXT */}
      <div className="auth-help text-xs text-muted mt-4" style={{ paddingTop: '1rem', borderTop: '1px solid var(--pico-card-separator-color)' }}>
        <p>
          <strong>Note:</strong> We use your account to read public channels and groups. Your
          credentials are stored securely in the database.
        </p>
      </div>
    </Card>
  )
}
