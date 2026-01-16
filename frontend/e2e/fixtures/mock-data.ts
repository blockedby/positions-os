// Mock data for E2E tests

export const mockTargets = {
  channel: {
    id: '550e8400-e29b-41d4-a716-446655440001',
    name: 'Go Jobs Channel',
    type: 'TG_CHANNEL',
    url: '@golang_jobs',
    is_active: true,
    metadata: {},
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    last_scraped_at: null,
  },
  forum: {
    id: '550e8400-e29b-41d4-a716-446655440002',
    name: 'Rust Forum',
    type: 'TG_FORUM',
    url: '@rust_jobs',
    is_active: true,
    metadata: { topic_ids: [1, 42, 100] },
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    last_scraped_at: '2025-01-10T12:00:00Z',
  },
  inactive: {
    id: '550e8400-e29b-41d4-a716-446655440003',
    name: 'Paused Target',
    type: 'TG_CHANNEL',
    url: '@paused_channel',
    is_active: false,
    metadata: {},
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-05T00:00:00Z',
    last_scraped_at: null,
  },
}

export const mockJobs = {
  raw: {
    id: '660e8400-e29b-41d4-a716-446655440001',
    external_id: 'tg_12345',
    source_channel: '@golang_jobs',
    status: 'RAW',
    raw_content: 'Looking for Go developer with 3+ years experience...',
    structured_data: null,
    created_at: '2025-01-15T10:00:00Z',
    updated_at: '2025-01-15T10:00:00Z',
  },
  analyzed: {
    id: '660e8400-e29b-41d4-a716-446655440002',
    external_id: 'tg_12346',
    source_channel: '@golang_jobs',
    status: 'ANALYZED',
    raw_content: 'Senior Backend Engineer needed. Remote OK. 150-250k RUB.',
    structured_data: {
      title: 'Senior Backend Engineer',
      description: 'Senior Backend Engineer needed',
      salary_min: 150000,
      salary_max: 250000,
      currency: 'RUB',
      location: 'Remote',
      is_remote: true,
      language: 'RU',
      technologies: ['Go', 'PostgreSQL', 'Docker'],
      experience_years: 5,
      company: null,
      contacts: ['@hr_contact'],
    },
    created_at: '2025-01-15T09:00:00Z',
    updated_at: '2025-01-15T09:30:00Z',
  },
  interested: {
    id: '660e8400-e29b-41d4-a716-446655440003',
    external_id: 'tg_12347',
    source_channel: '@rust_jobs',
    status: 'INTERESTED',
    raw_content: 'Rust developer for fintech startup...',
    structured_data: {
      title: 'Rust Developer',
      salary_min: 200000,
      salary_max: 350000,
      currency: 'RUB',
      is_remote: true,
      technologies: ['Rust', 'Tokio', 'PostgreSQL'],
    },
    created_at: '2025-01-14T08:00:00Z',
    updated_at: '2025-01-14T12:00:00Z',
  },
}

export const mockStats = {
  total_jobs: 156,
  by_status: {
    RAW: 45,
    ANALYZED: 78,
    INTERESTED: 23,
    REJECTED: 10,
  },
  by_source: {
    '@golang_jobs': 89,
    '@rust_jobs': 67,
  },
  active_targets: 2,
  total_targets: 3,
}

// Helper to generate a new target for creation tests
export function generateNewTarget(overrides?: Partial<(typeof mockTargets)['channel']>) {
  return {
    name: 'Test Channel',
    type: 'TG_CHANNEL',
    url: '@test_channel',
    is_active: true,
    ...overrides,
  }
}
