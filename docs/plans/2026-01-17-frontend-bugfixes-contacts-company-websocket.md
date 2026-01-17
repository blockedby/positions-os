# Frontend Bug Fixes: Contacts, Company, WebSocket Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix three frontend issues: (1) make contacts clickable with appropriate link types, (2) fix missing company/description by updating LLM prompt, (3) improve jobs page real-time updates with infinite scroll.

**Architecture:**
- P1 issues are independent fixes that can be parallelized
- LLM prompt update is backend-only, affects all future job analysis
- Contact links require smart detection of contact type (email, telegram, URL)
- Infinite scroll replaces pagination for better WebSocket integration

**Tech Stack:** React 19, TypeScript, Go, LLM prompt XML, TanStack Query, WebSocket

**TDD Discipline:** This plan follows Uncle Bob's Three Laws of TDD:
1. You may not write production code unless it is to make a failing unit test pass
2. You may not write more of a unit test than is sufficient to fail (compilation failures count)
3. You may not write more production code than is sufficient to pass the one failing unit test

---

## Priority Summary

| Priority | Issue | Impact | Effort |
|----------|-------|--------|--------|
| **P1** | LLM missing company/description | High - data quality | Medium |
| **P1** | Contacts not clickable | High - usability | Low |
| **P2** | WebSocket + pagination UX | Medium - real-time UX | High |

---

## Task 1: Update LLM Prompt to Extract Company and Description

**Files:**
- Modify: `docs/prompts/job-extraction.xml`

> Note: This task is prompt engineering, not code. TDD does not apply directly, but we verify through manual testing.

### Step 1: Read current prompt and understand the schema

The current prompt extracts:
```json
{
  "title": "...",
  "salary": { "min": 0, "max": 0, "currency": "..." },
  "technologies": [...],
  "contacts": [...],
  "experience_level": "...",
  "employment_type": "..."
}
```

**Missing fields that backend expects:**
- `company` - Company name
- `description` - Full job description
- `location` - City/country
- `is_remote` - boolean
- `experience_years` - numeric (not just level)

### Step 2: Update the prompt schema

Replace the system prompt schema section with:

```xml
Output ONLY valid JSON matching this structure:
{
  "title": "Job Title",
  "company": "Company Name (or null if not found)",
  "description": "Full job description",
  "salary": { "min": 0, "max": 0, "currency": "USD/EUR/RUB/etc" },
  "technologies": ["Go", "PostgreSQL", "Kafka", "Spring", "React"],
  "contacts": ["email@example.com", "@telegram_handle", "https://example.com/jobs"],
  "experience_level": "Junior/Middle/Senior/C-Level",
  "experience_years": 3,
  "employment_type": "Remote/Office/Hybrid",
  "location": "City, Country",
  "is_remote": true
}
```

### Step 3: Update all examples to include new fields

Each example output must now include:
- `company` field
- `description` field
- `location` field
- `is_remote` boolean
- `experience_years` numeric

Example update for `simple_golang`:
```json
{
  "title": "Go разработчик",
  "company": null,
  "description": "Looking for a Go developer with PostgreSQL, Docker, and K8s experience. 3+ years required, remote position.",
  "salary": { "min": 250000, "max": 350000, "currency": "RUB" },
  "technologies": ["Go", "PostgreSQL", "Docker", "K8s"],
  "contacts": ["@recruiter_ivan"],
  "experience_level": "Middle/Senior",
  "experience_years": 3,
  "employment_type": "Remote",
  "location": null,
  "is_remote": true
}
```

### Step 4: Manual verification (ASK USER)

**ASK THE USER:** "Ready to verify the LLM prompt changes. Please run the following commands and confirm the analyzer produces correct JSON output:"

```bash
# Terminal 1: Start infrastructure
task docker-up

# Terminal 2: Start collector
task collector

# Terminal 3: Start analyzer
task analyzer
```

Trigger a scrape via the UI or API, then check analyzer logs for proper JSON extraction with company, description, and location fields.

### Step 5: Commit

```bash
git add docs/prompts/job-extraction.xml
git commit -m "feat(analyzer): update LLM prompt to extract company, description, and location fields

The prompt was missing several fields that the backend data model expects:
- company: Company name from job posting
- description: Full job description text
- location: City/country if mentioned
- is_remote: Boolean flag
- experience_years: Numeric years required"
```

---

## Task 2: Make Contacts Clickable with Smart Link Detection

**Files:**
- Create: `frontend/src/components/jobs/ContactLink.tsx`
- Create: `frontend/src/components/jobs/ContactLink.test.tsx`
- Modify: `frontend/src/components/jobs/JobDetail.tsx`
- Modify: `frontend/src/styles/globals.css`

---

### Cycle 2.1: Email Detection

#### RED: Write failing test for email

```tsx
// frontend/src/components/jobs/ContactLink.test.tsx
import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ContactLink } from './ContactLink'

describe('ContactLink', () => {
  it('should render email as mailto link', () => {
    render(<ContactLink contact="test@example.com" />)

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'mailto:test@example.com')
    expect(link).toHaveTextContent('test@example.com')
  })
})
```

#### Run test to verify RED

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: FAIL - `Cannot find module './ContactLink'`

#### GREEN: Write minimum code to pass

```tsx
// frontend/src/components/jobs/ContactLink.tsx
export interface ContactLinkProps {
  contact: string
}

export const ContactLink = ({ contact }: ContactLinkProps) => {
  return (
    <a href={`mailto:${contact}`}>
      {contact}
    </a>
  )
}
```

#### Run test to verify GREEN

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: PASS

---

### Cycle 2.2: Telegram Detection

#### RED: Write failing test for telegram

```tsx
// Add to ContactLink.test.tsx
it('should render telegram handle as tg link', () => {
  render(<ContactLink contact="@username" />)

  const link = screen.getByRole('link')
  expect(link).toHaveAttribute('href', 'https://t.me/username')
  expect(link).toHaveTextContent('@username')
})
```

#### Run test to verify RED

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: FAIL - `expected 'mailto:@username' to be 'https://t.me/username'`

#### GREEN: Add telegram detection

```tsx
// frontend/src/components/jobs/ContactLink.tsx
export interface ContactLinkProps {
  contact: string
}

export const ContactLink = ({ contact }: ContactLinkProps) => {
  const trimmed = contact.trim()

  // Telegram: starts with @
  if (trimmed.startsWith('@')) {
    return (
      <a href={`https://t.me/${trimmed.slice(1)}`}>
        {contact}
      </a>
    )
  }

  // Default: email
  return (
    <a href={`mailto:${trimmed}`}>
      {contact}
    </a>
  )
}
```

#### Run test to verify GREEN

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: PASS

---

### Cycle 2.3: URL Detection

#### RED: Write failing test for URL

```tsx
// Add to ContactLink.test.tsx
it('should render URL as external link', () => {
  render(<ContactLink contact="https://example.com/jobs" />)

  const link = screen.getByRole('link')
  expect(link).toHaveAttribute('href', 'https://example.com/jobs')
  expect(link).toHaveAttribute('target', '_blank')
  expect(link).toHaveAttribute('rel', 'noopener noreferrer')
})
```

#### Run test to verify RED

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: FAIL - `expected 'mailto:https://example.com/jobs'...`

#### GREEN: Add URL detection

```tsx
// frontend/src/components/jobs/ContactLink.tsx
export interface ContactLinkProps {
  contact: string
}

export const ContactLink = ({ contact }: ContactLinkProps) => {
  const trimmed = contact.trim()

  // URL: starts with http:// or https://
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return (
      <a href={trimmed} target="_blank" rel="noopener noreferrer">
        {contact}
      </a>
    )
  }

  // Telegram: starts with @
  if (trimmed.startsWith('@')) {
    return (
      <a href={`https://t.me/${trimmed.slice(1)}`}>
        {contact}
      </a>
    )
  }

  // Default: email
  return (
    <a href={`mailto:${trimmed}`}>
      {contact}
    </a>
  )
}
```

#### Run test to verify GREEN

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: PASS

---

### Cycle 2.4: Phone Number Detection

#### RED: Write failing test for phone

```tsx
// Add to ContactLink.test.tsx
it('should render phone number as tel link', () => {
  render(<ContactLink contact="+1234567890" />)

  const link = screen.getByRole('link')
  expect(link).toHaveAttribute('href', 'tel:+1234567890')
})
```

#### Run test to verify RED

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: FAIL - `expected 'mailto:+1234567890'...`

#### GREEN: Add phone detection

```tsx
// frontend/src/components/jobs/ContactLink.tsx
export interface ContactLinkProps {
  contact: string
}

export const ContactLink = ({ contact }: ContactLinkProps) => {
  const trimmed = contact.trim()

  // URL: starts with http:// or https://
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return (
      <a href={trimmed} target="_blank" rel="noopener noreferrer">
        {contact}
      </a>
    )
  }

  // Telegram: starts with @
  if (trimmed.startsWith('@')) {
    return (
      <a href={`https://t.me/${trimmed.slice(1)}`}>
        {contact}
      </a>
    )
  }

  // Phone: starts with + followed by digits
  if (/^\+?[\d\s-]{7,}$/.test(trimmed)) {
    return (
      <a href={`tel:${trimmed.replace(/[\s-]/g, '')}`}>
        {contact}
      </a>
    )
  }

  // Default: email
  return (
    <a href={`mailto:${trimmed}`}>
      {contact}
    </a>
  )
}
```

#### Run test to verify GREEN

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: PASS

---

### Cycle 2.5: Unknown Format Fallback

#### RED: Write failing test for unknown format

```tsx
// Add to ContactLink.test.tsx
it('should render unknown format as plain text', () => {
  render(<ContactLink contact="some random text" />)

  expect(screen.queryByRole('link')).not.toBeInTheDocument()
  expect(screen.getByText('some random text')).toBeInTheDocument()
})
```

#### Run test to verify RED

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: FAIL - `expected element to not be in document` (it's rendering as mailto link)

#### GREEN: Add unknown type fallback

```tsx
// frontend/src/components/jobs/ContactLink.tsx
export interface ContactLinkProps {
  contact: string
}

type ContactType = 'email' | 'telegram' | 'url' | 'phone' | 'unknown'

const detectContactType = (contact: string): ContactType => {
  const trimmed = contact.trim()

  // URL: starts with http:// or https://
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return 'url'
  }

  // Telegram: starts with @
  if (trimmed.startsWith('@')) {
    return 'telegram'
  }

  // Phone: starts with + followed by digits
  if (/^\+?[\d\s-]{7,}$/.test(trimmed)) {
    return 'phone'
  }

  // Email: contains @ and . but doesn't start with @
  if (trimmed.includes('@') && !trimmed.startsWith('@') && trimmed.includes('.')) {
    return 'email'
  }

  return 'unknown'
}

export const ContactLink = ({ contact }: ContactLinkProps) => {
  const trimmed = contact.trim()
  const type = detectContactType(contact)

  switch (type) {
    case 'url':
      return (
        <a href={trimmed} target="_blank" rel="noopener noreferrer">
          {contact}
        </a>
      )
    case 'telegram':
      return (
        <a href={`https://t.me/${trimmed.slice(1)}`}>
          {contact}
        </a>
      )
    case 'phone':
      return (
        <a href={`tel:${trimmed.replace(/[\s-]/g, '')}`}>
          {contact}
        </a>
      )
    case 'email':
      return (
        <a href={`mailto:${trimmed}`}>
          {contact}
        </a>
      )
    default:
      return <span>{contact}</span>
  }
}
```

#### Run test to verify GREEN

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: PASS

---

### Cycle 2.6: REFACTOR

All tests are green. Now refactor for cleaner code:

```tsx
// frontend/src/components/jobs/ContactLink.tsx
export interface ContactLinkProps {
  contact: string
}

type ContactType = 'email' | 'telegram' | 'url' | 'phone' | 'unknown'

const detectContactType = (contact: string): ContactType => {
  const trimmed = contact.trim()

  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return 'url'
  }
  if (trimmed.startsWith('@')) {
    return 'telegram'
  }
  if (/^\+?[\d\s-]{7,}$/.test(trimmed)) {
    return 'phone'
  }
  if (trimmed.includes('@') && !trimmed.startsWith('@') && trimmed.includes('.')) {
    return 'email'
  }
  return 'unknown'
}

const getHref = (contact: string, type: ContactType): string | null => {
  const trimmed = contact.trim()

  switch (type) {
    case 'email':
      return `mailto:${trimmed}`
    case 'telegram':
      return `https://t.me/${trimmed.slice(1)}`
    case 'url':
      return trimmed
    case 'phone':
      return `tel:${trimmed.replace(/[\s-]/g, '')}`
    default:
      return null
  }
}

export const ContactLink = ({ contact }: ContactLinkProps) => {
  const type = detectContactType(contact)
  const href = getHref(contact, type)

  if (!href) {
    return <span>{contact}</span>
  }

  const isExternal = type === 'url'

  return (
    <a
      href={href}
      target={isExternal ? '_blank' : undefined}
      rel={isExternal ? 'noopener noreferrer' : undefined}
    >
      {contact}
    </a>
  )
}
```

#### Run tests to verify still GREEN

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: PASS

---

### Cycle 2.7: Edge Case - Telegram with spaces

#### RED: Write failing test

```tsx
// Add to ContactLink.test.tsx
describe('ContactLink edge cases', () => {
  it('should handle telegram with spaces around @', () => {
    render(<ContactLink contact=" @username " />)

    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', 'https://t.me/username')
  })
})
```

#### Run test

```bash
cd frontend && bun test ContactLink.test.tsx
```

Expected: Should PASS (already trimming in detectContactType)

---

### Step 2.8: Integrate into JobDetail

Update JobDetail.tsx to use ContactLink:

```tsx
// In frontend/src/components/jobs/JobDetail.tsx
// Add import at top:
import { ContactLink } from './ContactLink'

// Replace lines 112-121 with:
{data?.contacts && data.contacts.length > 0 && (
  <div className="job-detail-section">
    <h4>Contacts</h4>
    <ul className="contacts-list">
      {data.contacts.map((contact, i) => (
        <li key={i}>
          <ContactLink contact={contact} />
        </li>
      ))}
    </ul>
  </div>
)}
```

### Step 2.9: Add CSS

```css
/* Add to frontend/src/styles/globals.css in .contacts-list section */
.contacts-list a {
  color: var(--pico-primary);
  text-decoration: none;
}

.contacts-list a:hover {
  text-decoration: underline;
}
```

### Step 2.10: Run full test suite

```bash
cd frontend && bun test
```

Expected: All PASS

### Step 2.11: Commit

```bash
git add frontend/src/components/jobs/ContactLink.tsx \
        frontend/src/components/jobs/ContactLink.test.tsx \
        frontend/src/components/jobs/JobDetail.tsx \
        frontend/src/styles/globals.css
git commit -m "feat(frontend): make contacts clickable with smart link detection

Contacts are now rendered as clickable links based on detected type:
- Email addresses: mailto: links
- Telegram handles (@user): t.me links
- URLs: external links (new tab)
- Phone numbers: tel: links
- Unknown: plain text fallback"
```

---

## Task 3: Add Infinite Scroll to Jobs Page

**Files:**
- Create: `frontend/src/hooks/useInfiniteJobs.ts`
- Create: `frontend/src/hooks/useInfiniteJobs.test.ts`
- Create: `frontend/src/components/jobs/InfiniteJobsList.tsx`
- Modify: `frontend/src/contexts/WebSocketContext.tsx`
- Modify: `frontend/src/pages/Jobs.tsx`
- Modify: `frontend/src/styles/globals.css`

---

### Cycle 3.1: useInfiniteJobs Hook - First Page

#### RED: Write failing test

```tsx
// frontend/src/hooks/useInfiniteJobs.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import { useInfiniteJobs } from './useInfiniteJobs'

vi.mock('@/lib/api', () => ({
  api: {
    getJobs: vi.fn(),
  },
}))

import { api } from '@/lib/api'

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  )
}

describe('useInfiniteJobs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should fetch first page of jobs', async () => {
    vi.mocked(api.getJobs).mockResolvedValue({
      jobs: [{ id: '1', status: 'RAW' }],
      total: 50,
      page: 1,
      limit: 20,
    })

    const { result } = renderHook(() => useInfiniteJobs(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true)
    })

    expect(result.current.data?.pages).toHaveLength(1)
  })
})
```

#### Run test to verify RED

```bash
cd frontend && bun test useInfiniteJobs.test.ts
```

Expected: FAIL - `Cannot find module './useInfiniteJobs'`

#### GREEN: Write minimum code

```tsx
// frontend/src/hooks/useInfiniteJobs.ts
import { useInfiniteQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

export function useInfiniteJobs() {
  return useInfiniteQuery({
    queryKey: ['jobs', 'infinite'],
    queryFn: async ({ pageParam = 1 }) => {
      return api.getJobs({ page: pageParam, limit: 20 })
    },
    initialPageParam: 1,
    getNextPageParam: () => undefined,
  })
}
```

#### Run test to verify GREEN

```bash
cd frontend && bun test useInfiniteJobs.test.ts
```

Expected: PASS

---

### Cycle 3.2: hasNextPage Detection

#### RED: Write failing test

```tsx
// Add to useInfiniteJobs.test.ts
it('should have hasNextPage when more pages exist', async () => {
  vi.mocked(api.getJobs).mockResolvedValue({
    jobs: [{ id: '1', status: 'RAW' }],
    total: 50,
    page: 1,
    limit: 20,
  })

  const { result } = renderHook(() => useInfiniteJobs(), {
    wrapper: createWrapper(),
  })

  await waitFor(() => {
    expect(result.current.isSuccess).toBe(true)
  })

  expect(result.current.hasNextPage).toBe(true)
})
```

#### Run test to verify RED

```bash
cd frontend && bun test useInfiniteJobs.test.ts
```

Expected: FAIL - `expected false to be true` (getNextPageParam returns undefined)

#### GREEN: Implement pagination logic

```tsx
// frontend/src/hooks/useInfiniteJobs.ts
import { useInfiniteQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'

const PAGE_SIZE = 20

export function useInfiniteJobs() {
  return useInfiniteQuery({
    queryKey: ['jobs', 'infinite'],
    queryFn: async ({ pageParam = 1 }) => {
      return api.getJobs({ page: pageParam, limit: PAGE_SIZE })
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      const totalPages = Math.ceil(lastPage.total / PAGE_SIZE)
      if (lastPage.page < totalPages) {
        return lastPage.page + 1
      }
      return undefined
    },
  })
}
```

#### Run test to verify GREEN

```bash
cd frontend && bun test useInfiniteJobs.test.ts
```

Expected: PASS

---

### Cycle 3.3: fetchNextPage

#### RED: Write failing test

```tsx
// Add to useInfiniteJobs.test.ts
it('should fetch next page when requested', async () => {
  vi.mocked(api.getJobs)
    .mockResolvedValueOnce({
      jobs: [{ id: '1', status: 'RAW' }],
      total: 50,
      page: 1,
      limit: 20,
    })
    .mockResolvedValueOnce({
      jobs: [{ id: '2', status: 'ANALYZED' }],
      total: 50,
      page: 2,
      limit: 20,
    })

  const { result } = renderHook(() => useInfiniteJobs(), {
    wrapper: createWrapper(),
  })

  await waitFor(() => {
    expect(result.current.isSuccess).toBe(true)
  })

  await result.current.fetchNextPage()

  await waitFor(() => {
    expect(result.current.data?.pages).toHaveLength(2)
  })
})
```

#### Run test

```bash
cd frontend && bun test useInfiniteJobs.test.ts
```

Expected: Should PASS (fetchNextPage comes from useInfiniteQuery)

---

### Cycle 3.4: Filter Support

#### RED: Write failing test

```tsx
// Add to useInfiniteJobs.test.ts
it('should pass filters to API', async () => {
  vi.mocked(api.getJobs).mockResolvedValue({
    jobs: [],
    total: 0,
    page: 1,
    limit: 20,
  })

  renderHook(() => useInfiniteJobs({ status: 'ANALYZED' }), {
    wrapper: createWrapper(),
  })

  await waitFor(() => {
    expect(api.getJobs).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'ANALYZED' })
    )
  })
})
```

#### Run test to verify RED

```bash
cd frontend && bun test useInfiniteJobs.test.ts
```

Expected: FAIL - `expected undefined to equal 'ANALYZED'`

#### GREEN: Add filters support

```tsx
// frontend/src/hooks/useInfiniteJobs.ts
import { useInfiniteQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { JobsQuery } from '@/lib/types'

const PAGE_SIZE = 20

export function useInfiniteJobs(filters?: Omit<JobsQuery, 'page' | 'limit'>) {
  return useInfiniteQuery({
    queryKey: ['jobs', 'infinite', filters],
    queryFn: async ({ pageParam = 1 }) => {
      return api.getJobs({
        ...filters,
        page: pageParam,
        limit: PAGE_SIZE,
      })
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      const totalPages = Math.ceil(lastPage.total / PAGE_SIZE)
      if (lastPage.page < totalPages) {
        return lastPage.page + 1
      }
      return undefined
    },
  })
}
```

#### Run test to verify GREEN

```bash
cd frontend && bun test useInfiniteJobs.test.ts
```

Expected: PASS

---

### Step 3.5: Create InfiniteJobsList Component

No TDD cycle needed for UI wiring. Create the component:

```tsx
// frontend/src/components/jobs/InfiniteJobsList.tsx
import { useEffect, useRef } from 'react'
import { useInfiniteJobs } from '@/hooks/useInfiniteJobs'
import { JobsTable } from './JobsTable'
import { Spinner } from '@/components/ui'
import type { JobsQuery, Job } from '@/lib/types'

export interface InfiniteJobsListProps {
  filters?: Omit<JobsQuery, 'page' | 'limit'>
  onJobSelect?: (job: Job) => void
  selectedJobId?: string
}

export const InfiniteJobsList = ({
  filters,
  onJobSelect,
  selectedJobId,
}: InfiniteJobsListProps) => {
  const {
    data,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
  } = useInfiniteJobs(filters)

  const loadMoreRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          fetchNextPage()
        }
      },
      { threshold: 0.1 }
    )

    const current = loadMoreRef.current
    if (current) {
      observer.observe(current)
    }

    return () => {
      if (current) {
        observer.unobserve(current)
      }
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage])

  if (isLoading) {
    return (
      <div className="jobs-loading">
        <Spinner size="lg" />
      </div>
    )
  }

  const allJobs = data?.pages.flatMap((page) => page.jobs) ?? []
  const total = data?.pages[0]?.total ?? 0

  return (
    <div className="infinite-jobs-list">
      <div className="jobs-count text-muted mb-2">
        Showing {allJobs.length} of {total} jobs
      </div>

      <JobsTable
        jobs={allJobs}
        onRowClick={onJobSelect}
        selectedJobId={selectedJobId}
      />

      <div ref={loadMoreRef} className="load-more-trigger">
        {isFetchingNextPage && (
          <div className="loading-more">
            <Spinner size="sm" />
            <span>Loading more...</span>
          </div>
        )}
        {!hasNextPage && allJobs.length > 0 && (
          <p className="text-muted text-center">No more jobs to load</p>
        )}
      </div>
    </div>
  )
}
```

---

### Step 3.6: Update WebSocket Context

```tsx
// In frontend/src/contexts/WebSocketContext.tsx
// Update the job event handlers to also invalidate infinite query:

case 'job.new':
case 'job.updated':
case 'job.analyzed':
  queryClient.invalidateQueries({ queryKey: ['jobs'] })
  queryClient.invalidateQueries({ queryKey: ['jobs', 'infinite'] })
  queryClient.invalidateQueries({ queryKey: queryKeys.job(wsEvent.job_id) })
  queryClient.invalidateQueries({ queryKey: queryKeys.stats() })
  break
```

---

### Step 3.7: Update Jobs Page

```tsx
// frontend/src/pages/Jobs.tsx
import { useState } from 'react'
import { useWebSocket } from '@/hooks/useWebSocket'
import { InfiniteJobsList } from '@/components/jobs/InfiniteJobsList'
import { FilterBar } from '@/components/jobs/FilterBar'
import { JobDetail } from '@/components/jobs/JobDetail'
import type { JobsQuery, Job } from '@/lib/types'

export default function Jobs() {
  useWebSocket({ enabled: true })

  const [filters, setFilters] = useState<Omit<JobsQuery, 'page' | 'limit'>>({
    sort_by: 'created_at',
    sort_order: 'desc',
  })
  const [selectedJob, setSelectedJob] = useState<Job | null>(null)

  const handleFilter = (newFilters: Partial<JobsQuery>) => {
    const { page, limit, ...rest } = newFilters
    setFilters((prev) => ({ ...prev, ...rest }))
  }

  return (
    <div className="jobs-page">
      <div className="jobs-header">
        <h1>Jobs</h1>
        <p className="text-muted">Browse and manage job postings</p>
      </div>

      <FilterBar onFilter={handleFilter} />

      <div className="jobs-content">
        <div className="jobs-list-container">
          <InfiniteJobsList
            filters={filters}
            onJobSelect={setSelectedJob}
            selectedJobId={selectedJob?.id}
          />
        </div>

        {selectedJob && (
          <div className="job-detail-container">
            <JobDetail
              jobId={selectedJob.id}
              onClose={() => setSelectedJob(null)}
            />
          </div>
        )}
      </div>
    </div>
  )
}
```

---

### Step 3.8: Add CSS

```css
/* Add to frontend/src/styles/globals.css */
.infinite-jobs-list {
  display: flex;
  flex-direction: column;
}

.load-more-trigger {
  padding: 1rem;
  text-align: center;
  min-height: 60px;
}

.loading-more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

.jobs-count {
  font-size: 0.875rem;
}
```

---

### Step 3.9: Run full test suite

```bash
cd frontend && bun test
```

Expected: All PASS

---

### Step 3.10: Commit

```bash
git add frontend/src/hooks/useInfiniteJobs.ts \
        frontend/src/hooks/useInfiniteJobs.test.ts \
        frontend/src/components/jobs/InfiniteJobsList.tsx \
        frontend/src/contexts/WebSocketContext.tsx \
        frontend/src/pages/Jobs.tsx \
        frontend/src/styles/globals.css
git commit -m "feat(frontend): replace pagination with infinite scroll on Jobs page

Infinite scroll provides better UX with real-time WebSocket updates:
- Jobs load automatically as user scrolls down
- New jobs from WebSocket invalidate the infinite query
- Intersection observer triggers next page fetch
- Shows count of loaded vs total jobs"
```

---

## Task 4: E2E Tests and Manual Verification

**ASK THE USER:** "All code changes complete. Ready for final verification. Please run the following steps:"

### Step 1: Run unit tests

```bash
cd frontend && bun test
```

Expected: All PASS

### Step 2: Run lint

```bash
cd frontend && bun lint
```

Expected: No errors

### Step 3: Build frontend

```bash
cd frontend && bun build
```

Expected: Build succeeds

### Step 4: Run E2E tests

```bash
task e2e-docker
```

Expected: All pass or documented known issues

### Step 5: Manual verification checklist

**ASK THE USER to verify each item:**

1. **Contacts clickable:**
   - [ ] Email opens mail client (mailto:)
   - [ ] Telegram @handle opens t.me link
   - [ ] URLs open in new tab
   - [ ] Phone numbers open phone dialer

2. **Company name (requires re-analyzing jobs):**
   - [ ] Trigger new scrape after prompt update
   - [ ] New jobs show company name if present in posting
   - [ ] Shows "Unknown Company" only when truly unknown

3. **Infinite scroll:**
   - [ ] Jobs load on scroll down
   - [ ] Loading indicator appears while fetching
   - [ ] "No more jobs" shows at end of list
   - [ ] WebSocket updates refresh visible jobs

### Step 6: Fix any issues found

If issues found during verification, apply fixes following TDD:
1. Write failing test for the bug
2. Fix with minimum code
3. Verify test passes

---

## Summary

| Task | Description | TDD Cycles | Tests |
|------|-------------|------------|-------|
| 1 | LLM prompt for company/description | N/A (prompt) | Manual |
| 2 | Clickable contacts | 7 cycles | 8 tests |
| 3 | Infinite scroll | 4 cycles | 4 tests |
| 4 | E2E verification | N/A | E2E suite |

**Total new tests:** 12 unit tests + E2E

**Dependencies:**
- Task 1 is independent (backend prompt)
- Task 2 is independent (frontend component)
- Task 3 depends on existing JobsTable component
- Task 4 depends on all previous tasks
