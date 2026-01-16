import type {
  Job,
  JobsQuery,
  JobsResponse,
  Target,
  CreateTargetRequest,
  UpdateTargetRequest,
  Stats,
  ScrapeRequest,
  ScrapeStatus,
  UpdateJobRequest,
  ApiError,
  AuthStatusResponse,
  StartQRResponse,
  BulkDeleteResponse,
} from './types'

// ============================================================================
// Configuration
// ============================================================================

const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api/v1'
const WS_BASE = import.meta.env.VITE_WS_BASE_URL || `ws://${location.host}/ws`

// ============================================================================
// Error Handling
// ============================================================================

class APIError extends Error implements ApiError {
  status?: number
  code?: string

  constructor(message: string, status?: number, code?: string) {
    super(message)
    this.name = 'APIError'
    this.status = status
    this.code = code
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    let message = 'An error occurred'
    let code: string | undefined

    try {
      const data = await response.json()
      message = data.message || data.error || message
      code = data.code
    } catch {
      message = response.statusText || message
    }

    throw new APIError(message, response.status, code)
  }

  // 204 No Content
  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

// ============================================================================
// API Client
// ============================================================================

const api = {
  // ========================================================================
  // Jobs API
  // ========================================================================

  getJobs(query?: JobsQuery): Promise<JobsResponse> {
    const params = new URLSearchParams()

    if (query) {
      if (query.page) params.set('page', query.page.toString())
      if (query.limit) params.set('limit', query.limit.toString())
      if (query.status) params.set('status', query.status)
      if (query.search) params.set('search', query.search)
      if (query.technologies && query.technologies.length > 0)
        params.set('technologies', query.technologies.join(','))
      if (query.salary_min) params.set('salary_min', query.salary_min.toString())
      if (query.salary_max) params.set('salary_max', query.salary_max.toString())
      if (query.is_remote !== undefined)
        params.set('is_remote', query.is_remote.toString())
      if (query.sort_by) params.set('sort_by', query.sort_by)
      if (query.sort_order) params.set('sort_order', query.sort_order)
    }

    const url = `${API_BASE}/jobs${params.toString() ? `?${params}` : ''}`
    return fetch(url, {
      headers: {
        Accept: 'application/json',
      },
    }).then(handleResponse<JobsResponse>)
  },

  getJob(id: string): Promise<Job> {
    return fetch(`${API_BASE}/jobs/${id}`, {
      headers: {
        Accept: 'application/json',
      },
    }).then(handleResponse<Job>)
  },

  updateJobStatus(id: string, data: UpdateJobRequest): Promise<Job> {
    return fetch(`${API_BASE}/jobs/${id}/status`, {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    }).then(handleResponse<Job>)
  },

  bulkDeleteJobs(ids: string[]): Promise<BulkDeleteResponse> {
    return fetch(`${API_BASE}/jobs`, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ ids }),
    }).then(handleResponse<BulkDeleteResponse>)
  },

  // ========================================================================
  // Targets API
  // ========================================================================

  getTargets(): Promise<Target[]> {
    return fetch(`${API_BASE}/targets`, {
      headers: {
        Accept: 'application/json',
      },
    }).then(handleResponse<Target[]>)
  },

  getTarget(id: string): Promise<Target> {
    return fetch(`${API_BASE}/targets/${id}`, {
      headers: {
        Accept: 'application/json',
      },
    }).then(handleResponse<Target>)
  },

  createTarget(data: CreateTargetRequest): Promise<Target> {
    return fetch(`${API_BASE}/targets`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    }).then(handleResponse<Target>)
  },

  updateTarget(id: string, data: UpdateTargetRequest): Promise<Target> {
    return fetch(`${API_BASE}/targets/${id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    }).then(handleResponse<Target>)
  },

  deleteTarget(id: string): Promise<void> {
    return fetch(`${API_BASE}/targets/${id}`, {
      method: 'DELETE',
    }).then(handleResponse<void>)
  },

  // ========================================================================
  // Stats API
  // ========================================================================

  getStats(): Promise<Stats> {
    return fetch(`${API_BASE}/stats`, {
      headers: {
        Accept: 'application/json',
      },
    }).then(handleResponse<Stats>)
  },

  // ========================================================================
  // Scrape API
  // ========================================================================

  startScrape(data: ScrapeRequest): Promise<void> {
    return fetch(`${API_BASE}/scrape/telegram`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(data),
    }).then(handleResponse<void>)
  },

  stopScrape(): Promise<void> {
    return fetch(`${API_BASE}/scrape/current`, {
      method: 'DELETE',
    }).then(handleResponse<void>)
  },

  getScrapeStatus(): Promise<ScrapeStatus> {
    return fetch(`${API_BASE}/scrape/status`, {
      headers: {
        Accept: 'application/json',
      },
    }).then(handleResponse<ScrapeStatus>)
  },

  // ========================================================================
  // Auth API
  // ========================================================================

  getAuthStatus(): Promise<AuthStatusResponse> {
    return fetch(`${API_BASE}/auth/status`, {
      headers: {
        Accept: 'application/json',
      },
    }).then(handleResponse<AuthStatusResponse>)
  },

  startQR(): Promise<StartQRResponse> {
    return fetch(`${API_BASE}/auth/qr`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    }).then(handleResponse<StartQRResponse>)
  },
}

// ============================================================================
// WebSocket Client
// ============================================================================

interface WSClientOptions {
  onMessage?: (event: MessageEvent) => void
  onOpen?: (event: Event) => void
  onClose?: (event: CloseEvent) => void
  onError?: (event: Event) => void
}

class WSClient {
  public ws: WebSocket | null = null
  private url: string
  private options: WSClientOptions
  private reconnectTimeout: number | null = null
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000

  constructor(options: WSClientOptions = {}) {
    this.url = WS_BASE
    this.options = options
  }

  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return
    }

    this.ws = new WebSocket(this.url)

    this.ws.onopen = (event) => {
      this.reconnectAttempts = 0
      this.options.onOpen?.(event)
    }

    this.ws.onmessage = (event) => {
      this.options.onMessage?.(event)
    }

    this.ws.onclose = (event) => {
      this.options.onClose?.(event)

      // Attempt to reconnect if not closed intentionally
      if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
        this.scheduleReconnect()
      }
    }

    this.ws.onerror = (event) => {
      this.options.onError?.(event)
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimeout) {
      return
    }

    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts)
    this.reconnectTimeout = window.setTimeout(() => {
      this.reconnectTimeout = null
      this.reconnectAttempts++
      this.connect()
    }, delay)
  }

  disconnect(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout)
      this.reconnectTimeout = null
    }

    if (this.ws) {
      this.ws.close(1000, 'Client closing')
      this.ws = null
    }

    this.reconnectAttempts = this.maxReconnectAttempts // Prevent reconnect
  }

  send(data: string): void {
    this.ws?.send(data)
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  get readyState(): number {
    return this.ws?.readyState ?? WebSocket.CLOSED
  }
}

// ============================================================================
// Exports
// ============================================================================

export { api, APIError, WSClient }
export type { WSClientOptions }
