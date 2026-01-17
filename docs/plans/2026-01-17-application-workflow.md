# Interactive Job Application Workflow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Transform "Mark Interested" into a complete interactive workflow with document generation, editing, approval, and sending.

**Architecture:** Test-driven development with bite-sized tasks. Backend handlers wired in collector, brain service generates documents via LLM, frontend provides workflow buttons and review UI. Database stores markdown, PDFs are derived artifacts.

**Tech Stack:** Go 1.23, Chi router, React 19, TanStack Query, Vitest (unit), Playwright (E2E), Pico.css

---

## Phase 0: Add TAILORED_APPROVED Status

### Task 0.1: Add TAILORED_APPROVED to Backend Status Constants

**Files:**
- Modify: `internal/models/job.go:12-20`
- Test: `internal/repository/jobs_test.go`

**Step 1: Write the failing test**

```go
// File: internal/repository/jobs_test.go
// Update TestJob_IsValidStatus test (around line 9)

func TestJob_IsValidStatus(t *testing.T) {
	validStatuses := []string{"RAW", "ANALYZED", "REJECTED", "INTERESTED", "TAILORED", "TAILORED_APPROVED", "SENT", "RESPONDED"}

	for _, status := range validStatuses {
		job := Job{Status: status}
		if !job.IsValidStatus() {
			t.Errorf("status %s should be valid", status)
		}
	}

	invalidJob := Job{Status: "INVALID"}
	if invalidJob.IsValidStatus() {
		t.Error("invalid status should not be valid")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestJob_IsValidStatus ./internal/repository/...`
Expected: FAIL - "TAILORED_APPROVED should be valid"

**Step 3: Write minimal implementation**

```go
// File: internal/models/job.go
// Update constants (lines 12-20)

const (
	JobStatusRaw             JobStatus = "RAW"
	JobStatusAnalyzed        JobStatus = "ANALYZED"
	JobStatusRejected        JobStatus = "REJECTED"
	JobStatusInterested      JobStatus = "INTERESTED"
	JobStatusTailored        JobStatus = "TAILORED"
	JobStatusTailoredApproved JobStatus = "TAILORED_APPROVED"
	JobStatusSent            JobStatus = "SENT"
	JobStatusResponded       JobStatus = "RESPONDED"
)
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestJob_IsValidStatus ./internal/repository/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/models/job.go internal/repository/jobs_test.go
git commit -m "feat: add TAILORED_APPROVED status constant"
```

---

### Task 0.2: Add TAILORED_APPROVED Status Transitions

**Files:**
- Modify: `internal/repository/jobs.go` (CanTransitionTo method)
- Test: `internal/repository/jobs_test.go`

**Step 1: Write the failing tests**

```go
// File: internal/repository/jobs_test.go
// Update TestJob_CanTransitionTo test (around line 60)
// Add these test cases to the tests slice:

// Valid transitions from TAILORED (update existing)
{"TAILORED", "TAILORED_APPROVED", true},  // NEW: can approve after tailoring
{"TAILORED", "SENT", false},               // UPDATE: can't skip approval
{"TAILORED", "REJECTED", true},

// Valid transitions from TAILORED_APPROVED (NEW)
{"TAILORED_APPROVED", "SENT", true},
{"TAILORED_APPROVED", "REJECTED", true},
{"TAILORED_APPROVED", "RAW", true},

// Invalid: can't skip TAILORED_APPROVED
{"TAILORED", "RESPONDED", false},
{"TAILORED_APPROVED", "RESPONDED", false},
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestJob_CanTransitionTo ./internal/repository/...`
Expected: FAIL - TAILORED_APPROVED transitions not implemented

**Step 3: Write minimal implementation**

```go
// File: internal/repository/jobs.go
// Update CanTransitionTo method

case JobStatusTailored:
	return newStatus == JobStatusTailoredApproved ||
		newStatus == JobStatusRejected ||
		newStatus == JobStatusRaw

case JobStatusTailoredApproved:
	return newStatus == JobStatusSent ||
		newStatus == JobStatusRejected ||
		newStatus == JobStatusRaw

case JobStatusSent:
	return newStatus == JobStatusResponded ||
		newStatus == JobStatusRejected ||
		newStatus == JobStatusRaw
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestJob_CanTransitionTo ./internal/repository/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/repository/jobs.go internal/repository/jobs_test.go
git commit -m "feat: add TAILORED_APPROVED status transitions"
```

---

### Task 0.3: Create Database Migration for TAILORED_APPROVED

**Files:**
- Create: `migrations/000007_add_tailored_approved_status.up.sql`
- Create: `migrations/000007_add_tailored_approved_status.down.sql`

**Step 1: Write the migration files**

```sql
-- File: migrations/000007_add_tailored_approved_status.up.sql
-- Add TAILORED_APPROVED to job_status enum
ALTER TYPE job_status ADD VALUE IF NOT EXISTS 'TAILORED_APPROVED' AFTER 'TAILORED';
```

```sql
-- File: migrations/000007_add_tailored_approved_status.down.sql
-- Note: PostgreSQL does not support removing enum values
-- This is a no-op migration for rollback safety
-- To truly rollback, would need to recreate the enum type
```

**Step 2: Run migration to verify it applies**

Run: `task collector` (migrations run on startup)
Expected: Logs show `{"version":7,"dirty":false,"message":"migrations complete"}`

**Step 3: Verify in database**

Run: `psql $DATABASE_URL -c "SELECT enum_range(NULL::job_status);"`
Expected: Output includes TAILORED_APPROVED

**Step 4: Commit**

```bash
git add migrations/
git commit -m "feat: add TAILORED_APPROVED database migration"
```

---

## Phase 1: Backend Foundation - Wire Brain Service

### Task 1.1: Add SaveCoverLetter Storage Function

**Files:**
- Modify: `internal/brain/storage.go`
- Test: `internal/brain/storage_test.go`

**Step 1: Write the failing test**

```go
// File: internal/brain/storage_test.go
// Add after TestSaveTailoredResume_SavesFile

func TestSaveCoverLetter_SavesFile(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	jobID := "test-job-456"
	content := "Dear Hiring Manager,\n\nI am excited to apply..."

	// Execute
	err := SaveCoverLetter(tmpDir, jobID, content)

	// Assert
	if err != nil {
		t.Errorf("SaveCoverLetter() error = %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "outputs", jobID, "cover_letter.md")
	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Errorf("saved file not found: %v", err)
	}

	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestSaveCoverLetter ./internal/brain/...`
Expected: FAIL - undefined: SaveCoverLetter

**Step 3: Write minimal implementation**

```go
// File: internal/brain/storage.go
// Add after SaveTailoredResume function

const CoverLetterFilename = "cover_letter.md"

// SaveCoverLetter saves the cover letter for a specific job.
func SaveCoverLetter(storagePath, jobID, content string) error {
	logger.Info("saving cover letter for job: " + jobID)

	outputDir := filepath.Join(storagePath, OutputsDir, jobID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.Error("failed to create output directory", err)
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, CoverLetterFilename)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		logger.Error("failed to save cover letter", err)
		return fmt.Errorf("failed to save cover letter: %w", err)
	}

	logger.Info("cover letter saved successfully")
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestSaveCoverLetter ./internal/brain/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/brain/storage.go internal/brain/storage_test.go
git commit -m "feat: add SaveCoverLetter storage function"
```

---

### Task 1.2: Add StorageDir to Config

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go` (create if not exists)

**Step 1: Write the failing test**

```go
// File: internal/config/config_test.go

package config

import (
	"os"
	"testing"
)

func TestConfig_StorageDirDefault(t *testing.T) {
	// Unset env var to test default
	os.Unsetenv("STORAGE_DIR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.StorageDir != "./storage" {
		t.Errorf("StorageDir = %q, want %q", cfg.StorageDir, "./storage")
	}
}

func TestConfig_StorageDirFromEnv(t *testing.T) {
	os.Setenv("STORAGE_DIR", "/custom/path")
	defer os.Unsetenv("STORAGE_DIR")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.StorageDir != "/custom/path" {
		t.Errorf("StorageDir = %q, want %q", cfg.StorageDir, "/custom/path")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestConfig_StorageDir ./internal/config/...`
Expected: FAIL - cfg.StorageDir undefined

**Step 3: Write minimal implementation**

```go
// File: internal/config/config.go
// Add to Config struct

StorageDir string `env:"STORAGE_DIR" envDefault:"./storage"`
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestConfig_StorageDir ./internal/config/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat: add StorageDir config option"
```

---

### Task 1.3: Add RegisterBrainHandler to Server

**Files:**
- Modify: `internal/web/server.go`
- Test: `internal/web/server_test.go`

**Step 1: Write the failing test**

```go
// File: internal/web/server_test.go
// Add test for brain handler registration

func TestServer_RegisterBrainHandler(t *testing.T) {
	// Create mock handler
	mockHandler := &mockBrainHandler{}

	server := NewServer(/* minimal deps */)
	server.RegisterBrainHandler(mockHandler)

	// Test that routes are registered by making requests
	req := httptest.NewRequest("POST", "/api/v1/jobs/test-id/prepare", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	// Should not be 404 (route exists)
	if w.Code == 404 {
		t.Error("POST /api/v1/jobs/{id}/prepare should be registered")
	}
}

type mockBrainHandler struct{}

func (m *mockBrainHandler) PrepareJob(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *mockBrainHandler) GetDocuments(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (m *mockBrainHandler) DownloadResume(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestServer_RegisterBrainHandler ./internal/web/...`
Expected: FAIL - undefined: RegisterBrainHandler

**Step 3: Write minimal implementation**

```go
// File: internal/web/server.go
// Add after existing Register* methods

func (s *Server) RegisterBrainHandler(handler interface{}) {
	type brainHandler interface {
		PrepareJob(w http.ResponseWriter, r *http.Request)
		GetDocuments(w http.ResponseWriter, r *http.Request)
		DownloadResume(w http.ResponseWriter, r *http.Request)
	}

	if h, ok := handler.(brainHandler); ok {
		s.router.Route("/api/v1/jobs", func(r chi.Router) {
			r.Post("/{id}/prepare", h.PrepareJob)
			r.Get("/{id}/documents", h.GetDocuments)
			r.Get("/{id}/documents/resume.pdf", h.DownloadResume)
		})
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestServer_RegisterBrainHandler ./internal/web/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/web/server.go internal/web/server_test.go
git commit -m "feat: add RegisterBrainHandler to server"
```

---

### Task 1.4: Wire Brain Service in Collector

**Files:**
- Modify: `cmd/collector/main.go`

**Step 1: Identify integration point**

Read `cmd/collector/main.go` to find where to add brain service initialization.

**Step 2: Add imports and initialization**

```go
// File: cmd/collector/main.go
// Add imports
import (
	"github.com/blockedby/positions-os/internal/brain"
)

// Add after statsRepo initialization (around line 94):
storageDir := cfg.StorageDir
if storageDir == "" {
	storageDir = "./storage"
}

brainStorage := &brain.FileStorage{StoragePath: storageDir}
llmClient := llm.NewClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)
pdfRenderer := brain.NewPDFRenderer(storageDir)

brainService := brain.NewService(brainStorage, llmClient, pdfRenderer)
brainService.SetBroadcaster(hub)
brainService.SetStorageDir(storageDir)

brainRepoAdapter := brain.NewJobsRepositoryAdapter(jobsRepo)
prepareService := brain.NewPrepareService(brainService, brainRepoAdapter)
brainHandler := brain.NewHandler(brainRepoAdapter, prepareService)

// Add after RegisterAuthHandler (around line 146):
server.RegisterBrainHandler(brainHandler)
```

**Step 3: Verify it compiles**

Run: `go build ./cmd/collector/...`
Expected: Build succeeds

**Step 4: Run collector and test endpoint**

Run: `task collector`
Test: `curl -X POST http://localhost:3100/api/v1/jobs/test-id/prepare`
Expected: Response (may be 404 for invalid job ID, but not 404 for unknown route)

**Step 5: Commit**

```bash
git add cmd/collector/main.go
git commit -m "feat: wire brain service in collector"
```

---

## Phase 2: Frontend Status Visibility

**Note:** The frontend types.ts already has TAILORED, but the Badge component is missing TAILORED, SENT, and RESPONDED variants. This phase adds all missing workflow statuses to frontend components.

### Task 2.1: Add TAILORED_APPROVED to Frontend Types

**Files:**
- Modify: `frontend/src/lib/types.ts`
- Test: `frontend/src/lib/types.test.ts` (create)

**Step 1: Write the failing test**

```typescript
// File: frontend/src/lib/types.test.ts
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
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bunx vitest --run src/lib/types.test.ts`
Expected: FAIL - Type '"TAILORED_APPROVED"' is not assignable

**Step 3: Write minimal implementation**

```typescript
// File: frontend/src/lib/types.ts
// Update JobStatus type (lines 5-12)

export type JobStatus =
  | 'RAW'
  | 'ANALYZED'
  | 'REJECTED'
  | 'INTERESTED'
  | 'TAILORED'
  | 'TAILORED_APPROVED'
  | 'SENT'
  | 'RESPONDED'
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bunx vitest --run src/lib/types.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/lib/types.ts frontend/src/lib/types.test.ts
git commit -m "feat: add TAILORED_APPROVED to frontend types"
```

---

### Task 2.2: Add Missing Badge Status Variants (TAILORED, TAILORED_APPROVED, SENT, RESPONDED)

**Note:** Badge component currently only has: raw, analyzed, interested, rejected, paused. Missing: tailored, tailored_approved, sent, responded.

**Files:**
- Modify: `frontend/src/components/ui/Badge.tsx`
- Test: `frontend/src/components/ui/Badge.test.tsx` (create)

**Step 1: Write the failing test**

```typescript
// File: frontend/src/components/ui/Badge.test.tsx
import { describe, test, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Badge, type BadgeStatus } from './Badge'

describe('Badge component', () => {
  test('renders tailored status', () => {
    render(<Badge status="tailored">Tailored</Badge>)
    expect(screen.getByText('Tailored')).toHaveClass('badge-tailored')
  })

  test('renders tailored_approved status', () => {
    render(<Badge status="tailored_approved">Ready</Badge>)
    expect(screen.getByText('Ready')).toHaveClass('badge-tailored')
  })

  test('renders sent status', () => {
    render(<Badge status="sent">Sent</Badge>)
    expect(screen.getByText('Sent')).toHaveClass('badge-sent')
  })

  test('renders responded status', () => {
    render(<Badge status="responded">Responded</Badge>)
    expect(screen.getByText('Responded')).toHaveClass('badge-responded')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bunx vitest --run src/components/ui/Badge.test.tsx`
Expected: FAIL - tailored/sent/responded not in BadgeStatus

**Step 3: Write minimal implementation**

```typescript
// File: frontend/src/components/ui/Badge.tsx
import type { HTMLAttributes } from 'react'

export type BadgeStatus =
  | 'raw'
  | 'analyzed'
  | 'interested'
  | 'rejected'
  | 'paused'
  | 'tailored'
  | 'tailored_approved'
  | 'sent'
  | 'responded'

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  status: BadgeStatus
}

const statusClasses: Record<BadgeStatus, string> = {
  raw: 'badge-raw',
  analyzed: 'badge-analyzed',
  interested: 'badge-interested',
  rejected: 'badge-rejected',
  paused: 'badge-paused',
  tailored: 'badge-tailored',
  tailored_approved: 'badge-tailored',  // Same purple as tailored
  sent: 'badge-sent',
  responded: 'badge-responded',
}

export const Badge = ({ status, className = '', children, ...props }: BadgeProps) => {
  const classes = ['status-badge', statusClasses[status], className]
    .filter(Boolean)
    .join(' ')

  return (
    <span className={classes} {...props}>
      {children}
    </span>
  )
}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bunx vitest --run src/components/ui/Badge.test.tsx`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/components/ui/Badge.tsx frontend/src/components/ui/Badge.test.tsx
git commit -m "feat: add tailored/sent/responded badge variants"
```

---

### Task 2.3: Add Badge CSS for New Statuses

**Files:**
- Modify: `frontend/src/index.css` or `frontend/src/styles/badges.css`

**Step 1: Identify badge CSS file**

Run: `grep -r "badge-analyzed" frontend/src/`
Expected: Find the CSS file with badge styles

**Step 2: Add new badge styles**

```css
/* Add to badge styles section */
.badge-tailored {
  background-color: var(--pico-color-purple-600);
  color: white;
}

.badge-sent {
  background-color: var(--pico-color-cyan-600);
  color: white;
}

.badge-responded {
  background-color: var(--pico-color-pink-600);
  color: white;
}
```

**Step 3: Write E2E test for badge rendering**

```typescript
// File: frontend/e2e/application-workflow.spec.ts (add to existing or create)
import { test, expect } from '@playwright/test'

test.describe('Badge Status Rendering', () => {
  test('renders TAILORED badge with correct styling', async ({ page }) => {
    await page.route('**/api/v1/jobs*', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          jobs: [{
            id: 'test-job-1',
            status: 'TAILORED',
            structured_data: { title: 'Go Developer' },
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
          }],
          total: 1, page: 1, limit: 10, pages: 1,
        }),
      })
    })

    await page.goto('/jobs')
    const badge = page.locator('.badge-tailored')
    await expect(badge).toBeVisible()
  })
})
```

**Step 4: Run E2E test**

Run: `task e2e -- --grep "Badge Status Rendering"`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/ frontend/e2e/
git commit -m "feat: add CSS for tailored/sent/responded badges"
```

---

### Task 2.4: Add TAILORED_APPROVED to FilterBar

**Files:**
- Modify: `frontend/src/components/jobs/FilterBar.tsx`
- Test: E2E test will cover this

**Step 1: Locate statusOptions**

Run: `grep -n "statusOptions" frontend/src/components/jobs/FilterBar.tsx`

**Step 2: Add new status option**

```typescript
// File: frontend/src/components/jobs/FilterBar.tsx
// Update statusOptions array

const statusOptions = [
  { value: '', label: 'All Statuses' },
  { value: 'RAW', label: 'Raw' },
  { value: 'ANALYZED', label: 'Analyzed' },
  { value: 'INTERESTED', label: 'Interested' },
  { value: 'REJECTED', label: 'Rejected' },
  { value: 'TAILORED', label: 'Tailored' },
  { value: 'TAILORED_APPROVED', label: 'Ready to Send' },
  { value: 'SENT', label: 'Sent' },
  { value: 'RESPONDED', label: 'Responded' },
]
```

**Step 3: E2E test for FilterBar status options**

```typescript
// File: frontend/e2e/application-workflow.spec.ts (add to existing)
test('FilterBar includes Ready to Send status option', async ({ page }) => {
  await page.route('**/api/v1/jobs*', (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ jobs: [], total: 0, page: 1, limit: 10, pages: 0 }),
    })
  })

  await page.goto('/jobs')
  const statusSelect = page.locator('select').filter({ hasText: /status/i })
  await expect(statusSelect.locator('option[value="TAILORED_APPROVED"]')).toBeAttached()
})
```

**Step 4: Run E2E test**

Run: `task e2e -- --grep "FilterBar includes"`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/components/jobs/FilterBar.tsx frontend/e2e/
git commit -m "feat: add TAILORED_APPROVED to status filter"
```

---

## Phase 3: Prepare Application Button

### Task 3.1: Add prepareJob API Method

**Files:**
- Modify: `frontend/src/lib/api.ts`
- Test: `frontend/src/lib/api.test.ts` (create or modify)

**Step 1: Write the failing test**

```typescript
// File: frontend/src/lib/api.test.ts
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { api } from './api'

describe('API client', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  test('prepareJob calls POST /api/v1/jobs/{id}/prepare', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'started', ws_channel: 'brain.test-id' }),
    })
    global.fetch = mockFetch

    const result = await api.prepareJob('test-id')

    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/v1/jobs/test-id/prepare'),
      expect.objectContaining({ method: 'POST' })
    )
    expect(result.status).toBe('started')
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bunx vitest --run src/lib/api.test.ts`
Expected: FAIL - api.prepareJob is not a function

**Step 3: Write minimal implementation**

```typescript
// File: frontend/src/lib/api.ts
// Add to API class

async prepareJob(jobId: string): Promise<{ status: string; ws_channel: string }> {
  const response = await fetch(`${this.baseURL}/jobs/${jobId}/prepare`, {
    method: 'POST',
  })
  if (!response.ok) throw new Error('Failed to prepare job')
  return response.json()
}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bunx vitest --run src/lib/api.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/lib/api.ts frontend/src/lib/api.test.ts
git commit -m "feat: add prepareJob API method"
```

---

### Task 3.2: Add usePrepareJob Hook

**Files:**
- Modify: `frontend/src/hooks/useJobs.ts`
- Test: `frontend/src/hooks/useJobs.test.ts` (create)

**Step 1: Write the failing test**

```typescript
// File: frontend/src/hooks/useJobs.test.ts
import { describe, test, expect } from 'vitest'
import { renderHook } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { usePrepareJob } from './useJobs'
import type { ReactNode } from 'react'

const wrapper = ({ children }: { children: ReactNode }) => {
  const queryClient = new QueryClient()
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}

describe('usePrepareJob hook', () => {
  test('returns mutation function', () => {
    const { result } = renderHook(() => usePrepareJob(), { wrapper })
    expect(result.current.mutateAsync).toBeDefined()
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bunx vitest --run src/hooks/useJobs.test.ts`
Expected: FAIL - usePrepareJob is not exported

**Step 3: Write minimal implementation**

```typescript
// File: frontend/src/hooks/useJobs.ts
// Add after existing hooks

export function usePrepareJob() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (jobId: string) => api.prepareJob(jobId),
    onSuccess: (data, jobId) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.job(jobId) })
      queryClient.invalidateQueries({ queryKey: queryKeys.jobs() })
      console.log('Job preparation started:', data.ws_channel)
    },
  })
}
```

**Step 4: Run test to verify it passes**

Run: `cd frontend && bunx vitest --run src/hooks/useJobs.test.ts`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/hooks/useJobs.ts frontend/src/hooks/useJobs.test.ts
git commit -m "feat: add usePrepareJob React Query hook"
```

---

### Task 3.3: Add Prepare Application Button to JobDetail

**Files:**
- Modify: `frontend/src/components/jobs/JobDetail.tsx`
- Test: E2E test (next task)

**Step 1: Locate action buttons section**

Run: `grep -n "Mark Interested\|status.*button" frontend/src/components/jobs/JobDetail.tsx`

**Step 2: Add Prepare Application button**

```typescript
// File: frontend/src/components/jobs/JobDetail.tsx
// Add import
import { usePrepareJob } from '@/hooks/useJobs'

// In component, add hook
const prepareApplication = usePrepareJob()

// Add button after INTERESTED status check
{job.status === 'INTERESTED' && (
  <Button
    variant="primary"
    size="sm"
    onClick={() => prepareApplication.mutateAsync(job.id)}
    loading={prepareApplication.isPending}
  >
    Prepare Application
  </Button>
)}
```

**Step 3: Commit (E2E test in Task 3.4 will verify)**

```bash
git add frontend/src/components/jobs/JobDetail.tsx
git commit -m "feat: add Prepare Application button for INTERESTED jobs"
```

---

### Task 3.4: E2E Test for Prepare Application Flow

**Files:**
- Create: `frontend/e2e/application-workflow.spec.ts`

**Step 1: Write the E2E test**

```typescript
// File: frontend/e2e/application-workflow.spec.ts
import { test, expect } from '@playwright/test'

test.describe('Application Workflow', () => {
  test.beforeEach(async ({ page }) => {
    // Mock jobs API
    await page.route('**/api/v1/jobs*', (route) => {
      if (route.request().method() === 'GET') {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            jobs: [{
              id: 'test-job-1',
              status: 'INTERESTED',
              structured_data: { title: 'Go Developer' },
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            }],
            total: 1,
            page: 1,
            limit: 10,
            pages: 1,
          }),
        })
      }
    })

    // Mock prepare endpoint
    await page.route('**/api/v1/jobs/*/prepare', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'started', ws_channel: 'brain.test-job-1' }),
      })
    })
  })

  test('shows Prepare Application button for INTERESTED jobs', async ({ page }) => {
    await page.goto('/jobs')
    await page.click('text=Go Developer')

    await expect(page.getByRole('button', { name: /prepare application/i })).toBeVisible()
  })

  test('clicking Prepare Application triggers API call', async ({ page }) => {
    let prepareCalled = false
    await page.route('**/api/v1/jobs/*/prepare', (route) => {
      prepareCalled = true
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'started', ws_channel: 'brain.test-job-1' }),
      })
    })

    await page.goto('/jobs')
    await page.click('text=Go Developer')
    await page.click('button:has-text("Prepare Application")')

    await page.waitForTimeout(500)
    expect(prepareCalled).toBe(true)
  })
})
```

**Step 2: Run E2E test to verify it fails**

Run: `task e2e -- --grep "Application Workflow"`
Expected: FAIL - button not found (until implementation complete)

**Step 3: Verify test passes after implementation**

Run: `task e2e -- --grep "Application Workflow"`
Expected: PASS

**Step 4: Commit**

```bash
git add frontend/e2e/application-workflow.spec.ts
git commit -m "test: add E2E tests for application workflow"
```

---

## Phase 4: Dashboard Stats Enhancement

### Task 4.1: Add Tailored/Sent/Responded to Backend Stats

**Files:**
- Modify: `internal/repository/stats.go`
- Test: `internal/repository/stats_test.go` (create if not exists)

**Step 1: Write the failing test**

```go
// File: internal/repository/stats_test.go
package repository

import (
	"testing"
)

func TestDashboardStats_HasWorkflowCounts(t *testing.T) {
	stats := DashboardStats{}

	// These fields should exist
	_ = stats.TailoredJobs
	_ = stats.SentJobs
	_ = stats.RespondedJobs
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestDashboardStats ./internal/repository/...`
Expected: FAIL - unknown field TailoredJobs

**Step 3: Write minimal implementation**

```go
// File: internal/repository/stats.go
// Update DashboardStats struct

type DashboardStats struct {
	TotalJobs      int `json:"total_jobs"`
	AnalyzedJobs   int `json:"analyzed_jobs"`
	InterestedJobs int `json:"interested_jobs"`
	RejectedJobs   int `json:"rejected_jobs"`
	TailoredJobs   int `json:"tailored_jobs"`
	SentJobs       int `json:"sent_jobs"`
	RespondedJobs  int `json:"responded_jobs"`
	TodayJobs      int `json:"today_jobs"`
	ActiveTargets  int `json:"active_targets"`
}
```

**Step 4: Update GetStats query**

```sql
SELECT
    COUNT(*) as total,
    COUNT(CASE WHEN status = 'ANALYZED' THEN 1 END) as analyzed,
    COUNT(CASE WHEN status = 'INTERESTED' THEN 1 END) as interested,
    COUNT(CASE WHEN status = 'REJECTED' THEN 1 END) as rejected,
    COUNT(CASE WHEN status = 'TAILORED' OR status = 'TAILORED_APPROVED' THEN 1 END) as tailored,
    COUNT(CASE WHEN status = 'SENT' THEN 1 END) as sent,
    COUNT(CASE WHEN status = 'RESPONDED' THEN 1 END) as responded,
    COUNT(CASE WHEN created_at >= CURRENT_DATE THEN 1 END) as today
FROM jobs
```

**Step 5: Run test to verify it passes**

Run: `go test -v -run TestDashboardStats ./internal/repository/...`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/repository/stats.go internal/repository/stats_test.go
git commit -m "feat: add tailored/sent/responded to dashboard stats"
```

---

### Task 4.2: Update Frontend Stats Types

**Files:**
- Modify: `frontend/src/lib/types.ts`
- Modify: `frontend/src/hooks/useStats.ts`

**Step 1: Update Stats interface**

```typescript
// File: frontend/src/lib/types.ts
// Update Stats interface

export interface Stats {
  total_jobs: number
  analyzed_jobs: number
  interested_jobs: number
  rejected_jobs: number
  tailored_jobs: number      // NEW
  sent_jobs: number          // NEW
  responded_jobs: number     // NEW
  today_jobs: number
  active_targets: number
}
```

**Step 2: Update useStats to include new stats**

```typescript
// File: frontend/src/hooks/useStats.ts
// Update stats cards array

{
  label: 'Tailored',
  value: data.tailored_jobs,
  description: 'Ready to send',
},
{
  label: 'Sent',
  value: data.sent_jobs,
  description: 'Applications sent',
},
{
  label: 'Responded',
  value: data.responded_jobs,
  description: 'Recruiter responses',
},
```

**Step 3: E2E test for dashboard stats**

```typescript
// File: frontend/e2e/application-workflow.spec.ts (add to existing)
test('Dashboard shows tailored/sent/responded stats', async ({ page }) => {
  await page.route('**/api/v1/stats', (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        total_jobs: 100,
        analyzed_jobs: 80,
        interested_jobs: 20,
        rejected_jobs: 10,
        tailored_jobs: 5,
        sent_jobs: 3,
        responded_jobs: 1,
        today_jobs: 5,
        active_targets: 2,
      }),
    })
  })

  await page.route('**/api/v1/jobs*', (route) => {
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ jobs: [], total: 0, page: 1, limit: 8, pages: 0 }),
    })
  })

  await page.goto('/')
  await expect(page.getByText('Tailored')).toBeVisible()
  await expect(page.getByText('5').first()).toBeVisible()  // tailored count
  await expect(page.getByText('Sent')).toBeVisible()
  await expect(page.getByText('Responded')).toBeVisible()
})
```

**Step 4: Run E2E test**

Run: `task e2e -- --grep "Dashboard shows tailored"`
Expected: PASS

**Step 5: Commit**

```bash
git add frontend/src/lib/types.ts frontend/src/hooks/useStats.ts frontend/e2e/
git commit -m "feat: add tailored/sent/responded stats to frontend"
```

---

## Phase 5: Document Review and Approval (Deferred)

**Note:** The following tasks implement the ApplicationReview component, DocumentEditor, and approval flow. They follow the same TDD pattern but are organized as a separate phase:

- Task 5.1: Add Update Method to Applications Repository
- Task 5.2: Add Document Update Handler (PATCH /api/v1/jobs/{id}/documents/resume)
- Task 5.3: Create DocumentEditor Component
- Task 5.4: Create ApplicationReview Component
- Task 5.5: Integrate Review Flow in JobDetail
- Task 5.6: E2E Test for Review and Approval Flow

---

## Phase 6: Application Sending (Deferred)

**Note:** The following tasks implement backend message sending. They require additional infrastructure:

- Task 6.1: Add SendApplication Handler (POST /api/v1/jobs/{id}/send)
- Task 6.2: Create TelegramSender Service
- Task 6.3: Create EmailSender Service
- Task 6.4: Wire Senders in Collector
- Task 6.5: Add SMTP Config
- Task 6.6: Create SendApplicationModal Component
- Task 6.7: Integrate Send Modal in JobDetail
- Task 6.8: E2E Test for Send Flow

---

## Regression Testing

**Run after completing each phase to catch regressions early.**

### Backend Regression

```bash
# Run all Go tests
task test

# Run linter
task lint
```

Expected: All tests pass, no lint errors.

### Frontend Regression

```bash
cd frontend

# Unit tests
bunx vitest --run

# Lint
bun lint

# Type check
bunx tsc --noEmit
```

Expected: All tests pass, no lint/type errors.

### E2E Regression

```bash
# Run full E2E suite with isolated containers
task e2e-docker
```

Expected: All existing E2E tests pass (targets, api, websocket, filters).

### Fix Any Regressions Before Proceeding

If any test fails:
1. Identify the breaking change
2. Fix the issue (update test or fix implementation)
3. Re-run the full regression suite
4. Only proceed to next phase when all tests pass

---

## Verification Checklist

After completing all tasks:

- [ ] `task test` passes (Go tests)
- [ ] `cd frontend && bunx vitest --run` passes (frontend unit tests)
- [ ] `task e2e-docker` passes (E2E tests)
- [ ] `task lint` passes (Go linting)
- [ ] `cd frontend && bun lint` passes (ESLint)
- [ ] All new statuses visible in UI
- [ ] Prepare Application button appears for INTERESTED jobs
- [ ] Dashboard shows tailored/sent/responded counts

---

## Execution Handoff

Plan complete and saved to `~/.claude/plans/kind-napping-puppy.md`.

**Two execution options:**

1. **Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

2. **Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
