import React from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, type RenderHookOptions } from '@testing-library/react'

/**
 * Creates a fresh QueryClient for testing with disabled retries and caching
 */
export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  })
}

/**
 * Creates a wrapper component with QueryClientProvider for testing hooks
 */
export function createWrapper(queryClient?: QueryClient) {
  const client = queryClient ?? createTestQueryClient()

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }
}

/**
 * Renders a hook with QueryClientProvider wrapper
 */
export function renderHookWithClient<TResult, TProps>(
  hook: (props: TProps) => TResult,
  options?: Omit<RenderHookOptions<TProps>, 'wrapper'> & { queryClient?: QueryClient }
) {
  const { queryClient, ...restOptions } = options ?? {}
  const wrapper = createWrapper(queryClient)

  return renderHook(hook, {
    wrapper,
    ...restOptions,
  })
}

/**
 * Mock target data for testing
 */
export const mockTarget = {
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

export const mockInactiveTarget = {
  ...mockTarget,
  id: 'target-2',
  name: 'Inactive Channel',
  url: '@inactive_channel',
  is_active: false,
}

/**
 * Mock job data for testing
 */
export const mockJob = {
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

/**
 * Mock stats data for testing
 */
export const mockStats = {
  total_jobs: 100,
  analyzed_jobs: 80,
  interested_jobs: 20,
  rejected_jobs: 30,
  today_jobs: 5,
  active_targets: 3,
}
