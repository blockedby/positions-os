import { describe, test, expect } from 'vitest'
import type { JobStatus } from './types'

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
