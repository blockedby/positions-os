// ============================================================================
// Core Types
// ============================================================================

export type JobStatus =
  | 'RAW'
  | 'ANALYZED'
  | 'REJECTED'
  | 'INTERESTED'
  | 'TAILORED'
  | 'SENT'
  | 'RESPONDED'

export type TargetType =
  | 'TG_CHANNEL'
  | 'TG_GROUP'
  | 'TG_FORUM'
  | 'HH_SEARCH'
  | 'LINKEDIN_SEARCH'

export type Currency = 'RUB' | 'USD' | 'EUR' | null
export type Language = 'RU' | 'EN'

// ============================================================================
// Job Types
// ============================================================================

export interface JobData {
  title?: string | null
  description?: string | null
  salary_min?: number | null
  salary_max?: number | null
  currency?: Currency
  location?: string | null
  is_remote: boolean
  language: Language
  technologies: string[]
  experience_years?: number | null
  company?: string | null
  contacts: string[]
}

export interface Job {
  id: string
  target_id: string
  external_id: string
  content_hash: string
  raw_content: string
  structured_data?: JobData | null
  source_url?: string | null
  source_date?: string | null
  tg_message_id?: number | null
  tg_topic_id?: number | null
  status: JobStatus
  created_at: string
  updated_at: string
  analyzed_at?: string | null
}

// ============================================================================
// Target Types
// ============================================================================

export interface TargetMetadata {
  keywords?: string[]
  limit?: number
  include_topics?: boolean
  until?: string // YYYY-MM-DD
  topic_ids?: number[] // For TG_FORUM
}

export interface Target {
  id: string
  name: string
  type: TargetType
  url: string
  tg_access_hash?: number | null
  tg_channel_id?: number | null
  metadata: Record<string, unknown>
  last_scraped_at?: string | null
  last_message_id?: number | null
  is_active: boolean
  created_at: string
  updated_at: string
}

// ============================================================================
// Stats Types
// ============================================================================

export interface Stats {
  total_jobs: number
  analyzed_jobs: number
  interested_jobs: number
  rejected_jobs: number
  today_jobs: number
  active_targets: number
}

export interface StatsCard {
  label: string
  value: number
  description: string
  trend?: number // percentage change
}

// ============================================================================
// API Request/Response Types
// ============================================================================

export interface JobsQuery {
  page?: number
  limit?: number
  status?: JobStatus
  search?: string
  technologies?: string[]
  salary_min?: number
  salary_max?: number
  is_remote?: boolean
  sort_by?: 'created_at' | 'updated_at' | 'salary_max'
  sort_order?: 'asc' | 'desc'
}

export interface JobsResponse {
  jobs: Job[]
  total: number
  page: number
  limit: number
  pages: number
}

export interface ScrapeRequest {
  channel: string
  limit?: number
  until?: string
  topic_ids?: number[]
}

export interface ScrapeStatus {
  is_scraping: boolean
  target?: string
  processed?: number
  total?: number
  new_jobs?: number
}

export interface CreateTargetRequest {
  name: string
  type: TargetType
  url: string
  metadata?: TargetMetadata
  is_active?: boolean
}

export interface UpdateTargetRequest {
  name?: string
  url?: string
  metadata?: TargetMetadata
  is_active?: boolean
}

export interface UpdateJobRequest {
  status?: JobStatus
}

// ============================================================================
// WebSocket Event Types
// ============================================================================

export type WSEventType =
  | 'scrape.started'
  | 'scrape.progress'
  | 'scrape.completed'
  | 'scrape.failed'
  | 'scrape.cancelled'
  | 'job.new'
  | 'job.updated'
  | 'job.analyzed'
  | 'target.created'
  | 'target.updated'
  | 'target.deleted'
  | 'stats.updated'
  | 'tg_qr'
  | 'tg_auth_success'
  | 'error'

export interface BaseWSEvent {
  type: WSEventType
  timestamp: string
}

export interface ScrapeStartedEvent extends BaseWSEvent {
  type: 'scrape.started'
  target: string
  limit: number
}

export interface ScrapeProgressEvent extends BaseWSEvent {
  type: 'scrape.progress'
  target: string
  processed: number
  new_jobs: number
}

export interface ScrapeCompletedEvent extends BaseWSEvent {
  type: 'scrape.completed'
  target: string
  total: number
  new: number
}

export interface ScrapeFailedEvent extends BaseWSEvent {
  type: 'scrape.failed'
  target: string
  error: string
}

export interface ScrapeCancelledEvent extends BaseWSEvent {
  type: 'scrape.cancelled'
  target: string
}

export interface JobNewEvent extends BaseWSEvent {
  type: 'job.new'
  job_id: string
  title?: string
  company?: string
}

export interface JobUpdatedEvent extends BaseWSEvent {
  type: 'job.updated'
  job_id: string
  status: JobStatus
}

export interface JobAnalyzedEvent extends BaseWSEvent {
  type: 'job.analyzed'
  job_id: string
  technologies: string[]
  salary_min?: number | null
  salary_max?: number | null
  company?: string | null
}

export interface TargetCreatedEvent extends BaseWSEvent {
  type: 'target.created'
  target: Target
}

export interface TargetUpdatedEvent extends BaseWSEvent {
  type: 'target.updated'
  target: Target
}

export interface TargetDeletedEvent extends BaseWSEvent {
  type: 'target.deleted'
  target_id: string
}

export interface StatsUpdatedEvent extends BaseWSEvent {
  type: 'stats.updated'
  stats: Stats
}

export interface TgQREvent extends BaseWSEvent {
  type: 'tg_qr'
  url: string
}

export interface TgAuthSuccessEvent extends BaseWSEvent {
  type: 'tg_auth_success'
}

export interface ErrorEvent extends BaseWSEvent {
  type: 'error'
  message: string
}

export type WSEvent =
  | ScrapeStartedEvent
  | ScrapeProgressEvent
  | ScrapeCompletedEvent
  | ScrapeFailedEvent
  | ScrapeCancelledEvent
  | JobNewEvent
  | JobUpdatedEvent
  | JobAnalyzedEvent
  | TargetCreatedEvent
  | TargetUpdatedEvent
  | TargetDeletedEvent
  | StatsUpdatedEvent
  | TgQREvent
  | TgAuthSuccessEvent
  | ErrorEvent

// ============================================================================
// Query Key Types
// ============================================================================

export const queryKeys = {
  jobs: (params?: JobsQuery) => ['jobs', params] as const,
  job: (id: string) => ['job', id] as const,
  targets: () => ['targets'] as const,
  target: (id: string) => ['target', id] as const,
  stats: () => ['stats'] as const,
  scrapeStatus: () => ['scrape-status'] as const,
} as const

// ============================================================================
// Utility Types
// ============================================================================

export type ApiError = {
  message: string
  status?: number
  code?: string
}

export type PaginationParams = {
  page: number
  limit: number
}

// ============================================================================
// Auth Types
// ============================================================================

export type TelegramStatus = 'READY' | 'UNAUTHORIZED' | 'INITIALIZING' | 'ERROR'

export interface AuthStatusResponse {
  status: TelegramStatus
  is_ready: boolean
  qr_in_progress: boolean
}

export interface StartQRResponse {
  status: 'started' | 'already in progress'
  error?: string
}
