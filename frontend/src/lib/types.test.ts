import { describe, test, expect } from 'vitest'
import type { JobStatus, Stats } from './types'

describe('JobStatus type', () => {
  test('TAILORED_APPROVED is a valid JobStatus', () => {
    const status: JobStatus = 'TAILORED_APPROVED'
    expect(status).toBe('TAILORED_APPROVED')
  })

  test('all workflow statuses are defined', () => {
    const statuses: JobStatus[] = [
      'RAW',
      'ANALYZED',
      'INTERESTED',
      'REJECTED',
      'TAILORED',
      'TAILORED_APPROVED',
      'SENT',
      'RESPONDED',
    ]
    expect(statuses).toHaveLength(8)
  })
})

describe('Stats type', () => {
  test('Stats has workflow stage counts', () => {
    const stats: Stats = {
      total_jobs: 100,
      analyzed_jobs: 50,
      interested_jobs: 20,
      rejected_jobs: 10,
      tailored_jobs: 5,
      tailored_approved_jobs: 3,
      sent_jobs: 2,
      responded_jobs: 1,
      today_jobs: 5,
      active_targets: 2,
    }

    expect(stats.tailored_jobs).toBe(5)
    expect(stats.tailored_approved_jobs).toBe(3)
    expect(stats.sent_jobs).toBe(2)
    expect(stats.responded_jobs).toBe(1)
  })
})
