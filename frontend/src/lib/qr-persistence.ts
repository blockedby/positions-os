/**
 * QR Code Persistence Module
 *
 * Handles localStorage persistence for QR codes during page reloads.
 * Based on the reference implementation in docs/tg-auth-frontend-guidelines.md
 */

const STORAGE_KEY_URL = 'tg_qr_url'
const STORAGE_KEY_TIMESTAMP = 'tg_qr_timestamp'
const QR_EXPIRY_SECONDS = 30

export interface SavedQR {
  url: string
  ageSeconds: number
}

export const QRPersistence = {
  /**
   * Save QR URL to localStorage with current timestamp
   */
  save(url: string): void {
    try {
      localStorage.setItem(STORAGE_KEY_URL, url)
      localStorage.setItem(STORAGE_KEY_TIMESTAMP, Date.now().toString())
    } catch (error) {
      console.error('Failed to save QR to localStorage:', error)
    }
  },

  /**
   * Load QR from localStorage if it hasn't expired
   * Returns null if no QR is saved or if it has expired
   */
  load(): SavedQR | null {
    try {
      const url = localStorage.getItem(STORAGE_KEY_URL)
      const timestamp = localStorage.getItem(STORAGE_KEY_TIMESTAMP)

      if (!url || !timestamp) {
        return null
      }

      const ageSeconds = (Date.now() - parseInt(timestamp, 10)) / 1000

      if (ageSeconds > QR_EXPIRY_SECONDS) {
        this.clear()
        return null
      }

      return {
        url,
        ageSeconds: Math.floor(ageSeconds),
      }
    } catch (error) {
      console.error('Failed to load QR from localStorage:', error)
      return null
    }
  },

  /**
   * Clear saved QR data from localStorage
   */
  clear(): void {
    try {
      localStorage.removeItem(STORAGE_KEY_URL)
      localStorage.removeItem(STORAGE_KEY_TIMESTAMP)
    } catch (error) {
      console.error('Failed to clear QR from localStorage:', error)
    }
  },

  /**
   * Get remaining time in seconds for a saved QR
   */
  getRemainingTime(timestamp: number): number {
    const elapsed = (Date.now() - timestamp) / 1000
    return Math.max(0, QR_EXPIRY_SECONDS - Math.floor(elapsed))
  },

  /**
   * QR expiry duration in seconds
   */
  get expirySeconds(): number {
    return QR_EXPIRY_SECONDS
  },
}
