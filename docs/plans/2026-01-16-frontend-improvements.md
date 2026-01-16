# Frontend Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix pagination display, add scrape limit controls to targets, and enable bulk job deletion.

**Architecture:** Three independent features: (1) Backend fix for pagination response, (2) UI enhancements for scrape limits in TargetForm, (3) Full-stack bulk delete with backend endpoint and frontend selection UI.

**Tech Stack:** Go (Fuego API), React + TypeScript, TanStack Query, Pico.css

---

## Summary of Issues Found

| Issue | Root Cause | Fix Location |
|-------|-----------|--------------|
| Pagination missing | Backend `JobsListResponse` lacks `pages` field | `internal/api/types.go`, `internal/api/handlers.go` |
| No scrape limits in UI | `TargetForm` doesn't expose `limit`/`until` fields | `frontend/src/components/settings/TargetForm.tsx` |
| No bulk delete | No DELETE endpoint for jobs; no selection UI | Backend + Frontend |

---

## Task 1: Fix Pagination Response

**Files:**
- Modify: `internal/api/types.go:57-63`
- Modify: `internal/api/handlers.go:67-72`
- Test: `internal/api/handlers_test.go` (if exists, else manual)

**Step 1: Add `Pages` field to JobsListResponse**

Edit `internal/api/types.go`:

```go
// JobsListResponse contains paginated list of jobs.
type JobsListResponse struct {
	Jobs  []JobResponse `json:"jobs" description:"List of jobs"`
	Total int           `json:"total" description:"Total number of matching jobs"`
	Page  int           `json:"page" description:"Current page number"`
	Limit int           `json:"limit" description:"Items per page"`
	Pages int           `json:"pages" description:"Total number of pages"`
}
```

**Step 2: Calculate and return `Pages` in handler**

Edit `internal/api/handlers.go` in `listJobs` function:

```go
// Calculate total pages
pages := (total + limit - 1) / limit
if pages < 1 {
	pages = 1
}

return JobsListResponse{
	Jobs:  JobsFromRepo(jobs),
	Total: total,
	Page:  page,
	Limit: limit,
	Pages: pages,
}, nil
```

**Step 3: Verify pagination works**

Run: `curl "http://localhost:3100/api/v1/jobs?page=1&limit=10" | jq '.pages'`
Expected: Number > 0 (calculated as ceil(total/limit))

**Step 4: Run frontend and verify pagination UI appears**

Run: `cd frontend && bun dev`
Navigate to: `http://localhost:5173/jobs`
Expected: Page selector buttons visible when jobs > limit

**Step 5: Commit**

```bash
git add internal/api/types.go internal/api/handlers.go
git commit -m "$(cat <<'EOF'
fix(api): add pages field to jobs list response

The frontend pagination component requires a `pages` field to display
page selectors. Added calculation: ceil(total/limit).

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add Scrape Limit Controls to Target Form

**Files:**
- Modify: `frontend/src/components/settings/TargetForm.tsx`
- Modify: `frontend/src/components/settings/TargetList.tsx`
- Modify: `frontend/src/pages/Settings.tsx`

### Step 1: Write test for limit/until fields in TargetForm

Create `frontend/src/components/settings/TargetForm.test.tsx`:

```tsx
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TargetForm } from './TargetForm'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
})

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
)

describe('TargetForm', () => {
  it('renders limit and until fields for Telegram targets', () => {
    render(<TargetForm />, { wrapper })

    expect(screen.getByLabelText(/message limit/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/until date/i)).toBeInTheDocument()
  })

  it('includes limit in create request when provided', async () => {
    const user = userEvent.setup()
    render(<TargetForm />, { wrapper })

    await user.type(screen.getByLabelText(/name/i), 'Test Target')
    await user.type(screen.getByLabelText(/url/i), '@test_channel')
    await user.clear(screen.getByLabelText(/message limit/i))
    await user.type(screen.getByLabelText(/message limit/i), '500')

    // Form should have limit value ready for submission
    expect(screen.getByLabelText(/message limit/i)).toHaveValue(500)
  })
})
```

**Step 2: Run test to verify it fails**

Run: `cd frontend && bun test src/components/settings/TargetForm.test.tsx`
Expected: FAIL - "Unable to find label" for limit/until fields

**Step 3: Add limit and until state to TargetForm**

Edit `frontend/src/components/settings/TargetForm.tsx`:

Add state variables after existing state declarations:

```tsx
const [limit, setLimit] = useState<number | ''>(
  (target?.metadata as TargetMetadata)?.limit || 100
)
const [until, setUntil] = useState(
  (target?.metadata as TargetMetadata)?.until || ''
)
```

Add import at top:

```tsx
import type { Target, TargetType, CreateTargetRequest, UpdateTargetRequest, TargetMetadata } from '@/lib/types'
```

**Step 4: Add form fields for limit and until**

Add after the URL input field in the form JSX:

```tsx
{type.startsWith('TG_') && (
  <>
    <Input
      label="Message Limit"
      type="number"
      placeholder="100"
      value={limit}
      onChange={(e) => setLimit(e.target.value ? parseInt(e.target.value, 10) : '')}
      helperText="Maximum messages to scrape (leave empty for default 100)"
    />

    <Input
      label="Until Date"
      type="date"
      value={until}
      onChange={(e) => setUntil(e.target.value)}
      helperText="Stop scraping at posts older than this date (optional)"
    />
  </>
)}
```

**Step 5: Include metadata in create/update requests**

Modify the `handleSubmit` function to include metadata:

```tsx
const handleSubmit = async (e: React.FormEvent) => {
  e.preventDefault()

  if (!validate()) return

  const metadata: TargetMetadata = {}
  if (limit !== '' && limit > 0) {
    metadata.limit = limit
  }
  if (until) {
    metadata.until = until
  }

  try {
    if (isEditing && target) {
      const data: UpdateTargetRequest = {
        name,
        url,
        is_active: isActive,
        metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
      }
      await updateTarget.mutateAsync({ id: target.id, data })
    } else {
      const data: CreateTargetRequest = {
        name,
        type,
        url,
        is_active: isActive,
        metadata: Object.keys(metadata).length > 0 ? metadata : undefined,
      }
      await createTarget.mutateAsync(data)
    }
    onSuccess?.()
  } catch {
    // Error handled by react-query
  }
}
```

**Step 6: Update useEffect to load metadata when editing**

```tsx
useEffect(() => {
  if (target) {
    setName(target.name)
    setType(target.type)
    setUrl(target.url)
    setIsActive(target.is_active)
    const meta = target.metadata as TargetMetadata
    setLimit(meta?.limit || 100)
    setUntil(meta?.until || '')
  }
}, [target])
```

**Step 7: Run test to verify it passes**

Run: `cd frontend && bun test src/components/settings/TargetForm.test.tsx`
Expected: PASS

**Step 8: Update TargetList to pass limit/until when scraping**

Edit `frontend/src/components/settings/TargetList.tsx`:

Find the `onScrape` call and update it to pass metadata:

```tsx
const handleScrape = (target: Target) => {
  const meta = target.metadata as TargetMetadata
  onScrape?.(target.url, {
    limit: meta?.limit,
    until: meta?.until,
  })
}
```

Update the onScrape prop type if needed and the Button onClick:

```tsx
<Button
  variant="secondary"
  size="small"
  onClick={() => handleScrape(target)}
  disabled={isScrapingThis}
>
```

**Step 9: Update Settings page to use limit/until from scrape call**

Edit `frontend/src/pages/Settings.tsx`:

Update the `handleScrape` function signature and implementation:

```tsx
const handleScrape = async (
  channel: string,
  options?: { limit?: number; until?: string }
) => {
  try {
    await api.startScrape({
      channel,
      limit: options?.limit,
      until: options?.until,
    })
  } catch (error) {
    setError(error instanceof Error ? error.message : 'Failed to start scrape')
  }
}
```

**Step 10: Run tests and verify**

Run: `cd frontend && bun test`
Expected: All tests pass

**Step 11: Manual test**

1. Start backend: `task collector`
2. Start frontend: `cd frontend && bun dev`
3. Navigate to Settings, create a target with limit=50
4. Click Scrape and verify it uses the limit

**Step 12: Commit**

```bash
git add frontend/src/components/settings/TargetForm.tsx \
        frontend/src/components/settings/TargetForm.test.tsx \
        frontend/src/components/settings/TargetList.tsx \
        frontend/src/pages/Settings.tsx
git commit -m "$(cat <<'EOF'
feat(frontend): add scrape limit controls to target form

- Add message limit field (default 100)
- Add until date field for date-based stopping
- Pass limit/until from target metadata when scraping
- Only show fields for Telegram-type targets

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Add Bulk Delete Jobs Feature

### Task 3.1: Backend - Add bulk delete endpoint

**Files:**
- Modify: `internal/api/types.go`
- Modify: `internal/api/handlers.go`
- Modify: `internal/api/server.go`
- Modify: `internal/repository/jobs.go`

**Step 1: Add request/response types**

Edit `internal/api/types.go`, add after JobUpdateStatusRequest:

```go
// JobsBulkDeleteRequest contains the request body for bulk deleting jobs.
type JobsBulkDeleteRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=100" description:"Job IDs to delete (max 100)"`
}

// JobsBulkDeleteResponse contains the response after bulk deleting jobs.
type JobsBulkDeleteResponse struct {
	Deleted int `json:"deleted" description:"Number of jobs deleted"`
}
```

**Step 2: Add repository method**

Edit `internal/repository/jobs.go`, add method:

```go
// BulkDelete removes multiple jobs by their IDs.
func (r *JobsRepository) BulkDelete(ctx context.Context, ids []uuid.UUID) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	query := `DELETE FROM jobs WHERE id = ANY($1)`
	result, err := r.db.ExecContext(ctx, query, pq.Array(ids))
	if err != nil {
		return 0, fmt.Errorf("bulk delete jobs: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get affected rows: %w", err)
	}

	return int(affected), nil
}
```

Add import if needed: `"github.com/lib/pq"`

**Step 3: Add handler**

Edit `internal/api/handlers.go`, add after updateJobStatus:

```go
func (s *Server) bulkDeleteJobs(c fuego.ContextWithBody[JobsBulkDeleteRequest]) (JobsBulkDeleteResponse, error) {
	body, err := c.Body()
	if err != nil {
		return JobsBulkDeleteResponse{}, fuego.BadRequestError{Detail: err.Error()}
	}

	if len(body.IDs) == 0 {
		return JobsBulkDeleteResponse{}, fuego.BadRequestError{Detail: "No job IDs provided"}
	}

	if len(body.IDs) > 100 {
		return JobsBulkDeleteResponse{}, fuego.BadRequestError{Detail: "Cannot delete more than 100 jobs at once"}
	}

	deleted, err := s.deps.JobsRepo.BulkDelete(c.Context(), body.IDs)
	if err != nil {
		return JobsBulkDeleteResponse{}, fuego.InternalServerError{Detail: err.Error()}
	}

	// Notify WebSocket clients
	if s.deps.Hub != nil {
		s.deps.Hub.Broadcast(map[string]interface{}{
			"type":    "jobs.deleted",
			"count":   deleted,
			"job_ids": body.IDs,
		})
	}

	return JobsBulkDeleteResponse{Deleted: deleted}, nil
}
```

**Step 4: Register route**

Edit `internal/api/server.go`, find the jobs routes section and add:

```go
fuego.Delete(jobsGroup, "", s.bulkDeleteJobs).
	Summary("Bulk delete jobs").
	Description("Delete multiple jobs by their IDs (max 100)")
```

**Step 5: Test backend endpoint**

Run: `task collector`

Test with curl:
```bash
# First get some job IDs
JOB_IDS=$(curl -s "http://localhost:3100/api/v1/jobs?limit=2" | jq -r '[.jobs[].id] | join("\",\"")')

# Delete them
curl -X DELETE "http://localhost:3100/api/v1/jobs" \
  -H "Content-Type: application/json" \
  -d "{\"ids\":[\"$JOB_IDS\"]}"
```
Expected: `{"deleted": 2}` (or number of jobs deleted)

**Step 6: Commit backend changes**

```bash
git add internal/api/types.go internal/api/handlers.go internal/api/server.go internal/repository/jobs.go
git commit -m "$(cat <<'EOF'
feat(api): add bulk delete endpoint for jobs

- DELETE /api/v1/jobs with {"ids": [...]}
- Max 100 jobs per request for safety
- Broadcasts jobs.deleted WebSocket event

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

### Task 3.2: Frontend - Add selection UI and bulk delete

**Files:**
- Modify: `frontend/src/lib/types.ts`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/hooks/useJobs.ts`
- Modify: `frontend/src/components/jobs/JobsTable.tsx`
- Modify: `frontend/src/components/jobs/JobRow.tsx`
- Modify: `frontend/src/pages/Jobs.tsx`

**Step 1: Add API types and method**

Edit `frontend/src/lib/types.ts`, add:

```ts
export interface BulkDeleteRequest {
  ids: string[]
}

export interface BulkDeleteResponse {
  deleted: number
}
```

Edit `frontend/src/lib/api.ts`, add method:

```ts
bulkDeleteJobs(ids: string[]): Promise<BulkDeleteResponse> {
  return fetch(`${API_BASE}/jobs`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ ids }),
  }).then(handleResponse<BulkDeleteResponse>)
},
```

**Step 2: Add mutation hook**

Edit `frontend/src/hooks/useJobs.ts`, add:

```ts
export const useBulkDeleteJobs = () => {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (ids: string[]) => api.bulkDeleteJobs(ids),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
      queryClient.invalidateQueries({ queryKey: ['stats'] })
    },
  })
}
```

Add `useQueryClient` to imports if not present.

**Step 3: Update JobRow to support selection**

Edit `frontend/src/components/jobs/JobRow.tsx`:

Add props for selection:

```tsx
export interface JobRowProps {
  job: Job
  onClick?: (job: Job) => void
  isSelected?: boolean
  isChecked?: boolean
  onCheckChange?: (checked: boolean) => void
  showCheckbox?: boolean
}
```

Add checkbox to the row:

```tsx
export const JobRow = ({
  job,
  onClick,
  isSelected,
  isChecked,
  onCheckChange,
  showCheckbox,
}: JobRowProps) => {
  // ... existing code ...

  return (
    <tr
      className={`job-row ${isSelected ? 'selected' : ''}`}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role="button"
    >
      {showCheckbox && (
        <td className="job-row-checkbox" onClick={(e) => e.stopPropagation()}>
          <input
            type="checkbox"
            checked={isChecked}
            onChange={(e) => onCheckChange?.(e.target.checked)}
            aria-label={`Select ${title}`}
          />
        </td>
      )}
      {/* ... rest of existing columns ... */}
    </tr>
  )
}
```

**Step 4: Update JobsTable for bulk selection**

Edit `frontend/src/components/jobs/JobsTable.tsx`:

Update props:

```tsx
export interface JobsTableProps {
  data?: JobsResponse
  isLoading?: boolean
  selectedJobId?: string
  onJobClick?: (job: Job) => void
  onPageChange?: (page: number) => void
  // New props for selection
  selectionMode?: boolean
  selectedIds?: Set<string>
  onSelectionChange?: (ids: Set<string>) => void
}
```

Update component:

```tsx
export const JobsTable = ({
  data,
  isLoading,
  selectedJobId,
  onJobClick,
  onPageChange,
  selectionMode = false,
  selectedIds = new Set(),
  onSelectionChange,
}: JobsTableProps) => {
  // ... loading/empty checks ...

  const allSelected = data?.jobs.length > 0 &&
    data.jobs.every((job) => selectedIds.has(job.id))

  const handleSelectAll = (checked: boolean) => {
    if (!data?.jobs || !onSelectionChange) return
    if (checked) {
      const newIds = new Set(selectedIds)
      data.jobs.forEach((job) => newIds.add(job.id))
      onSelectionChange(newIds)
    } else {
      const newIds = new Set(selectedIds)
      data.jobs.forEach((job) => newIds.delete(job.id))
      onSelectionChange(newIds)
    }
  }

  const handleRowSelect = (jobId: string, checked: boolean) => {
    if (!onSelectionChange) return
    const newIds = new Set(selectedIds)
    if (checked) {
      newIds.add(jobId)
    } else {
      newIds.delete(jobId)
    }
    onSelectionChange(newIds)
  }

  return (
    <div className="jobs-table-container">
      <table className="jobs-table">
        <thead>
          <tr>
            {selectionMode && (
              <th className="job-col-checkbox">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={(e) => handleSelectAll(e.target.checked)}
                  aria-label="Select all jobs on page"
                />
              </th>
            )}
            <th>Job</th>
            <th>Salary</th>
            <th>Technologies</th>
            <th>Status</th>
            <th>Date</th>
          </tr>
        </thead>
        <tbody>
          {data.jobs.map((job) => (
            <JobRow
              key={job.id}
              job={job}
              onClick={onJobClick}
              isSelected={selectedJobId === job.id}
              showCheckbox={selectionMode}
              isChecked={selectedIds.has(job.id)}
              onCheckChange={(checked) => handleRowSelect(job.id, checked)}
            />
          ))}
        </tbody>
      </table>

      {data.pages > 1 && (
        <Pagination
          currentPage={data.page}
          totalPages={data.pages}
          totalItems={data.total}
          onPageChange={onPageChange}
        />
      )}
    </div>
  )
}
```

**Step 5: Update Jobs page with selection mode and delete action**

Edit `frontend/src/pages/Jobs.tsx`:

Add state and imports:

```tsx
import { useJobs, useUpdateJobStatus, useBulkDeleteJobs } from '@/hooks/useJobs'

// Inside component:
const [selectionMode, setSelectionMode] = useState(false)
const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
const bulkDelete = useBulkDeleteJobs()
```

Add handlers:

```tsx
const handleBulkDelete = async () => {
  if (selectedIds.size === 0) return

  const confirmed = window.confirm(
    `Delete ${selectedIds.size} job(s)? This cannot be undone.`
  )
  if (!confirmed) return

  try {
    await bulkDelete.mutateAsync(Array.from(selectedIds))
    setSelectedIds(new Set())
    setSelectionMode(false)
  } catch (error) {
    console.error('Failed to delete jobs:', error)
  }
}

const toggleSelectionMode = () => {
  setSelectionMode(!selectionMode)
  if (selectionMode) {
    setSelectedIds(new Set())
  }
}
```

Add UI controls above table:

```tsx
<div className="jobs-actions">
  <Button
    variant={selectionMode ? 'primary' : 'secondary'}
    size="small"
    onClick={toggleSelectionMode}
  >
    {selectionMode ? 'Cancel' : 'Select'}
  </Button>

  {selectionMode && selectedIds.size > 0 && (
    <Button
      variant="danger"
      size="small"
      onClick={handleBulkDelete}
      loading={bulkDelete.isPending}
    >
      Delete ({selectedIds.size})
    </Button>
  )}
</div>
```

Pass selection props to JobsTable:

```tsx
<JobsTable
  data={data}
  isLoading={isLoading}
  selectedJobId={selectedJobId}
  onJobClick={handleJobClick}
  onPageChange={handlePageChange}
  selectionMode={selectionMode}
  selectedIds={selectedIds}
  onSelectionChange={setSelectedIds}
/>
```

**Step 6: Add danger button variant**

Edit `frontend/src/components/ui/Button.tsx`:

If not already present, add danger variant:

```tsx
export type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'danger'

// In className logic:
const variantClasses = {
  primary: '',
  secondary: 'secondary',
  outline: 'outline',
  danger: 'contrast',  // Pico.css danger style
}
```

**Step 7: Add CSS for checkbox column and actions bar**

Edit `frontend/src/styles/globals.css`, add:

```css
.jobs-actions {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.job-col-checkbox {
  width: 40px;
}

.job-row-checkbox {
  width: 40px;
  text-align: center;
}

.job-row-checkbox input[type="checkbox"] {
  margin: 0;
}
```

**Step 8: Run tests**

Run: `cd frontend && bun test`
Expected: All tests pass

**Step 9: Manual integration test**

1. Start backend: `task collector`
2. Start frontend: `cd frontend && bun dev`
3. Navigate to Jobs page
4. Click "Select" button
5. Check some jobs
6. Click "Delete (N)" button
7. Confirm deletion
8. Verify jobs are removed from list

**Step 10: Commit frontend changes**

```bash
git add frontend/src/lib/types.ts \
        frontend/src/lib/api.ts \
        frontend/src/hooks/useJobs.ts \
        frontend/src/components/jobs/JobsTable.tsx \
        frontend/src/components/jobs/JobRow.tsx \
        frontend/src/components/ui/Button.tsx \
        frontend/src/pages/Jobs.tsx \
        frontend/src/styles/globals.css
git commit -m "$(cat <<'EOF'
feat(frontend): add bulk delete for jobs

- Add selection mode toggle on Jobs page
- Add checkboxes to job rows
- Add "Select all" in table header
- Add bulk delete button with confirmation
- Integrates with new DELETE /api/v1/jobs endpoint

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Run Full Test Suite

**Step 1: Run backend tests**

Run: `task test`
Expected: All Go tests pass

**Step 2: Run frontend unit tests**

Run: `cd frontend && bun test`
Expected: All Vitest tests pass

**Step 3: Run E2E tests**

Run: `task e2e`
Expected: All Playwright tests pass

**Step 4: Final commit if any fixes needed**

```bash
git add -A
git commit -m "$(cat <<'EOF'
test: fix any test failures from frontend improvements

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Verification Checklist

- [ ] Pagination shows page selectors when jobs > limit
- [ ] Target form has limit and until fields for Telegram targets
- [ ] Limit/until values saved to target metadata
- [ ] Scraping uses target's limit/until values
- [ ] Jobs page has "Select" toggle button
- [ ] Checkboxes appear in selection mode
- [ ] "Select all" checkbox works
- [ ] Bulk delete removes selected jobs
- [ ] All tests pass
