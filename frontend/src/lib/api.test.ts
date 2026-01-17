import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api, APIError } from './api'

// ============================================================================
// Test Data
// ============================================================================

const mockJob = {
  id: 'job-1',
  target_id: 'target-1',
  external_id: 'ext-1',
  content_hash: 'hash-1',
  raw_content: 'Job posting content',
  structured_data: {
    title: 'Go Developer',
    company: 'Acme Inc',
    salary_min: 100000,
    salary_max: 150000,
    currency: 'RUB' as const,
    location: 'Moscow',
    is_remote: true,
    language: 'RU' as const,
    technologies: ['Go', 'PostgreSQL'],
    experience_years: 3,
    contacts: ['hr@acme.com'],
  },
  source_url: 'https://t.me/channel/123',
  source_date: '2026-01-15',
  tg_message_id: 123,
  tg_topic_id: null,
  status: 'ANALYZED' as const,
  created_at: '2026-01-15T10:00:00Z',
  updated_at: '2026-01-15T12:00:00Z',
  analyzed_at: '2026-01-15T11:00:00Z',
}

const mockTarget = {
  id: 'target-1',
  name: 'Go Jobs Channel',
  type: 'TG_CHANNEL' as const,
  url: '@go_jobs',
  tg_access_hash: 12345,
  tg_channel_id: 67890,
  metadata: { keywords: ['go', 'golang'] },
  last_scraped_at: '2026-01-15T10:00:00Z',
  last_message_id: 100,
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-15T10:00:00Z',
}

const mockStats = {
  total_jobs: 100,
  analyzed_jobs: 80,
  interested_jobs: 20,
  rejected_jobs: 30,
  today_jobs: 5,
  active_targets: 3,
}

// ============================================================================
// APIError Class Tests
// ============================================================================

describe('APIError', () => {
  it('should create an APIError with message only', () => {
    const error = new APIError('Something went wrong')

    expect(error).toBeInstanceOf(Error)
    expect(error).toBeInstanceOf(APIError)
    expect(error.message).toBe('Something went wrong')
    expect(error.name).toBe('APIError')
    expect(error.status).toBeUndefined()
    expect(error.code).toBeUndefined()
  })

  it('should create an APIError with message and status', () => {
    const error = new APIError('Not found', 404)

    expect(error.message).toBe('Not found')
    expect(error.status).toBe(404)
    expect(error.code).toBeUndefined()
  })

  it('should create an APIError with message, status, and code', () => {
    const error = new APIError('Validation failed', 400, 'VALIDATION_ERROR')

    expect(error.message).toBe('Validation failed')
    expect(error.status).toBe(400)
    expect(error.code).toBe('VALIDATION_ERROR')
  })
})

// ============================================================================
// Response Handling Tests (via API methods)
// ============================================================================

describe('API Response Handling', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('should handle successful JSON response', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve(mockStats),
    } as Response)

    const result = await api.getStats()
    expect(result).toEqual(mockStats)
  })

  it('should handle 204 No Content response', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true,
      status: 204,
      json: () => Promise.reject(new Error('No content')),
    } as Response)

    const result = await api.deleteTarget('target-1')
    expect(result).toBeUndefined()
  })

  it('should throw APIError on error response with JSON body', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 404,
      statusText: 'Not Found',
      json: () => Promise.resolve({ message: 'Target not found', code: 'NOT_FOUND' }),
    } as Response)

    try {
      await api.getTarget('invalid-id')
      expect.fail('Expected APIError to be thrown')
    } catch (error) {
      expect(error).toBeInstanceOf(APIError)
      expect(error).toMatchObject({
        message: 'Target not found',
        status: 404,
        code: 'NOT_FOUND',
      })
    }
  })

  it('should throw APIError with statusText when JSON parsing fails', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
      json: () => Promise.reject(new Error('Invalid JSON')),
    } as Response)

    await expect(api.getStats()).rejects.toMatchObject({
      message: 'Internal Server Error',
      status: 500,
    })
  })

  it('should throw APIError with error field from response', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: false,
      status: 400,
      statusText: 'Bad Request',
      json: () => Promise.resolve({ error: 'Invalid request body' }),
    } as Response)

    await expect(api.createTarget({ name: '', type: 'TG_CHANNEL', url: '' }))
      .rejects.toMatchObject({
        message: 'Invalid request body',
        status: 400,
      })
  })
})

// ============================================================================
// Jobs API Tests
// ============================================================================

describe('Jobs API', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('getJobs', () => {
    const mockJobsResponse = {
      jobs: [mockJob],
      total: 1,
      page: 1,
      limit: 20,
      pages: 1,
    }

    it('should fetch jobs without query params', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      const result = await api.getJobs()

      expect(result).toEqual(mockJobsResponse)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs',
        expect.objectContaining({
          headers: { Accept: 'application/json' },
        })
      )
    })

    it('should fetch jobs with pagination params', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      await api.getJobs({ page: 2, limit: 10 })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs?page=2&limit=10',
        expect.any(Object)
      )
    })

    it('should fetch jobs with status filter', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      await api.getJobs({ status: 'ANALYZED' })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs?status=ANALYZED',
        expect.any(Object)
      )
    })

    it('should fetch jobs with search query', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      await api.getJobs({ search: 'golang' })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs?search=golang',
        expect.any(Object)
      )
    })

    it('should fetch jobs with technologies filter', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      await api.getJobs({ technologies: ['Go', 'PostgreSQL'] })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs?technologies=Go%2CPostgreSQL',
        expect.any(Object)
      )
    })

    it('should fetch jobs with salary range', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      await api.getJobs({ salary_min: 100000, salary_max: 200000 })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs?salary_min=100000&salary_max=200000',
        expect.any(Object)
      )
    })

    it('should fetch jobs with is_remote filter', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      await api.getJobs({ is_remote: true })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs?is_remote=true',
        expect.any(Object)
      )
    })

    it('should fetch jobs with sorting params', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      await api.getJobs({ sort_by: 'created_at', sort_order: 'desc' })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs?sort_by=created_at&sort_order=desc',
        expect.any(Object)
      )
    })

    it('should not include empty technologies array', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJobsResponse),
      } as Response)

      await api.getJobs({ technologies: [] })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs',
        expect.any(Object)
      )
    })
  })

  describe('getJob', () => {
    it('should fetch a single job by ID', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockJob),
      } as Response)

      const result = await api.getJob('job-1')

      expect(result).toEqual(mockJob)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs/job-1',
        expect.objectContaining({
          headers: { Accept: 'application/json' },
        })
      )
    })
  })

  describe('updateJobStatus', () => {
    it('should update job status', async () => {
      const updatedJob = { ...mockJob, status: 'INTERESTED' as const }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(updatedJob),
      } as Response)

      const result = await api.updateJobStatus('job-1', { status: 'INTERESTED' })

      expect(result).toEqual(updatedJob)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs/job-1/status',
        expect.objectContaining({
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ status: 'INTERESTED' }),
        })
      )
    })
  })
})

// ============================================================================
// Prepare Job API Tests
// ============================================================================

describe('Prepare Job API', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('prepareJob', () => {
    it('should prepare job for application', async () => {
      const prepareResponse = {
        job_id: 'job-1',
        status: 'TAILORED_APPROVED',
        resume_path: '/storage/jobs/job-1/resume.pdf',
        cover_letter_path: '/storage/jobs/job-1/cover_letter.md',
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(prepareResponse),
      } as Response)

      const result = await api.prepareJob('job-1')

      expect(result).toEqual(prepareResponse)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/jobs/job-1/prepare',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        })
      )
    })

    it('should throw APIError when job not in INTERESTED status', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        json: () => Promise.resolve({ message: 'Job must be in INTERESTED status' }),
      } as Response)

      await expect(api.prepareJob('job-1')).rejects.toMatchObject({
        message: 'Job must be in INTERESTED status',
        status: 400,
      })
    })

    it('should throw APIError when job not found', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        json: () => Promise.resolve({ message: 'Job not found' }),
      } as Response)

      await expect(api.prepareJob('invalid-id')).rejects.toMatchObject({
        message: 'Job not found',
        status: 404,
      })
    })
  })
})

// ============================================================================
// Targets API Tests
// ============================================================================

describe('Targets API', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('getTargets', () => {
    it('should fetch all targets', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve([mockTarget]),
      } as Response)

      const result = await api.getTargets()

      expect(result).toEqual([mockTarget])
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/targets',
        expect.objectContaining({
          headers: { Accept: 'application/json' },
        })
      )
    })

    it('should return empty array when no targets exist', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve([]),
      } as Response)

      const result = await api.getTargets()

      expect(result).toEqual([])
    })
  })

  describe('getTarget', () => {
    it('should fetch a single target by ID', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockTarget),
      } as Response)

      const result = await api.getTarget('target-1')

      expect(result).toEqual(mockTarget)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/targets/target-1',
        expect.any(Object)
      )
    })
  })

  describe('createTarget', () => {
    it('should create a new target', async () => {
      const createRequest = {
        name: 'New Channel',
        type: 'TG_CHANNEL' as const,
        url: '@new_channel',
        metadata: { keywords: ['react'] },
        is_active: true,
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: () => Promise.resolve({ ...mockTarget, ...createRequest }),
      } as Response)

      const result = await api.createTarget(createRequest)

      expect(result.name).toBe('New Channel')
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/targets',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(createRequest),
        })
      )
    })

    it('should create target with minimal required fields', async () => {
      const createRequest = {
        name: 'Minimal Target',
        type: 'TG_CHANNEL' as const,
        url: '@minimal',
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 201,
        json: () => Promise.resolve({ ...mockTarget, ...createRequest }),
      } as Response)

      await api.createTarget(createRequest)

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/targets',
        expect.objectContaining({
          body: JSON.stringify(createRequest),
        })
      )
    })
  })

  describe('updateTarget', () => {
    it('should update an existing target', async () => {
      const updateRequest = {
        name: 'Updated Channel',
        is_active: false,
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ ...mockTarget, ...updateRequest }),
      } as Response)

      const result = await api.updateTarget('target-1', updateRequest)

      expect(result.name).toBe('Updated Channel')
      expect(result.is_active).toBe(false)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/targets/target-1',
        expect.objectContaining({
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(updateRequest),
        })
      )
    })

    it('should update target metadata', async () => {
      const updateRequest = {
        metadata: { keywords: ['go', 'backend'], limit: 50 },
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ ...mockTarget, metadata: updateRequest.metadata }),
      } as Response)

      const result = await api.updateTarget('target-1', updateRequest)

      expect(result.metadata).toEqual(updateRequest.metadata)
    })
  })

  describe('deleteTarget', () => {
    it('should delete a target', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 204,
        json: () => Promise.reject(new Error('No content')),
      } as Response)

      const result = await api.deleteTarget('target-1')

      expect(result).toBeUndefined()
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/targets/target-1',
        expect.objectContaining({
          method: 'DELETE',
        })
      )
    })
  })
})

// ============================================================================
// Stats API Tests
// ============================================================================

describe('Stats API', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('getStats', () => {
    it('should fetch stats', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(mockStats),
      } as Response)

      const result = await api.getStats()

      expect(result).toEqual(mockStats)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/stats',
        expect.objectContaining({
          headers: { Accept: 'application/json' },
        })
      )
    })
  })
})

// ============================================================================
// Scrape API Tests
// ============================================================================

describe('Scrape API', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('startScrape', () => {
    it('should start a scrape with required params', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(undefined),
      } as Response)

      await api.startScrape({ channel: '@go_jobs' })

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/scrape/telegram',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ channel: '@go_jobs' }),
        })
      )
    })

    it('should start a scrape with all params', async () => {
      const scrapeRequest = {
        channel: '@go_jobs',
        limit: 100,
        until: '2026-01-01',
        topic_ids: [1, 2, 3],
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(undefined),
      } as Response)

      await api.startScrape(scrapeRequest)

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/scrape/telegram',
        expect.objectContaining({
          body: JSON.stringify(scrapeRequest),
        })
      )
    })
  })

  describe('stopScrape', () => {
    it('should stop the current scrape', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 204,
        json: () => Promise.reject(new Error('No content')),
      } as Response)

      await api.stopScrape()

      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/scrape/current',
        expect.objectContaining({
          method: 'DELETE',
        })
      )
    })
  })

  describe('getScrapeStatus', () => {
    it('should get scrape status when not scraping', async () => {
      const statusResponse = {
        is_scraping: false,
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(statusResponse),
      } as Response)

      const result = await api.getScrapeStatus()

      expect(result).toEqual(statusResponse)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/scrape/status',
        expect.any(Object)
      )
    })

    it('should get scrape status when scraping', async () => {
      const statusResponse = {
        is_scraping: true,
        target: '@go_jobs',
        processed: 50,
        total: 100,
        new_jobs: 10,
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(statusResponse),
      } as Response)

      const result = await api.getScrapeStatus()

      expect(result).toEqual(statusResponse)
    })
  })
})

// ============================================================================
// Auth API Tests
// ============================================================================

describe('Auth API', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  describe('getAuthStatus', () => {
    it('should get auth status when ready', async () => {
      const statusResponse = {
        status: 'READY',
        is_ready: true,
        qr_in_progress: false,
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(statusResponse),
      } as Response)

      const result = await api.getAuthStatus()

      expect(result).toEqual(statusResponse)
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/auth/status',
        expect.any(Object)
      )
    })

    it('should get auth status when unauthorized', async () => {
      const statusResponse = {
        status: 'UNAUTHORIZED',
        is_ready: false,
        qr_in_progress: false,
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(statusResponse),
      } as Response)

      const result = await api.getAuthStatus()

      expect(result.status).toBe('UNAUTHORIZED')
      expect(result.is_ready).toBe(false)
    })
  })

  describe('startQR', () => {
    it('should start QR code auth', async () => {
      const response = {
        status: 'started',
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(response),
      } as Response)

      const result = await api.startQR()

      expect(result.status).toBe('started')
      expect(fetch).toHaveBeenCalledWith(
        '/api/v1/auth/qr',
        expect.objectContaining({
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        })
      )
    })

    it('should handle QR already in progress', async () => {
      const response = {
        status: 'already in progress',
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve(response),
      } as Response)

      const result = await api.startQR()

      expect(result.status).toBe('already in progress')
    })
  })
})
