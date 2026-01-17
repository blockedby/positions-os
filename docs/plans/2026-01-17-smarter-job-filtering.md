# Smarter Job Filtering Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix the scraping filter to check if jobs actually exist in the database, not just if message IDs are within the parsed range. This allows re-scraping messages that were previously skipped (empty) but whose IDs are still tracked.

**Architecture:** Add a new method to JobsRepository that returns existing message IDs for a target. Modify the filter to combine range-based filtering with actual job existence checks. Messages are considered "new" if they're outside the range OR if they're inside the range but no job exists for them.

**Tech Stack:** Go 1.23+, PostgreSQL, pgx/v5

---

## Background

**Current Behavior:**
- `parsed_ranges` table stores min/max message IDs that have been "seen"
- `FilterNew()` marks any message ID within [min, max] as "already processed"
- Problem: Empty/skipped messages get their IDs tracked but no job is created
- Result: Re-scraping never processes those messages again

**Example from logs:**
- Range: [1911, 1933] (23 message IDs)
- Actual jobs: only 6 (IDs: 1928-1933)
- Messages 1911-1927 are filtered as "processed" but have no jobs

**Solution:**
1. Query jobs table to get existing message IDs for the target
2. A message is "new" if: outside range OR (inside range AND no job exists)

---

### Task 1: Add GetExistingMessageIDs to JobsRepository

**Files:**
- Modify: `internal/repository/jobs.go`
- Test: `internal/repository/jobs_test.go`

**Step 1: Write the failing test**

Add to `internal/repository/jobs_test.go`:

```go
func TestJob_ExternalIDConversion(t *testing.T) {
	// Test that external_id (string) correctly represents tg_message_id (int64)
	job := &Job{
		ExternalID:  "1234",
		TgMessageID: ptr(int64(1234)),
	}

	// external_id should be parseable to int64 matching tg_message_id
	parsed, err := strconv.ParseInt(job.ExternalID, 10, 64)
	if err != nil {
		t.Fatalf("ExternalID not parseable: %v", err)
	}
	if parsed != *job.TgMessageID {
		t.Errorf("ExternalID %d != TgMessageID %d", parsed, *job.TgMessageID)
	}
}

func ptr[T any](v T) *T {
	return &v
}
```

**Step 2: Run test to verify it passes (this confirms our assumption)**

Run: `go test -v -run TestJob_ExternalIDConversion ./internal/repository/...`
Expected: PASS (external_id is string version of message ID)

**Step 3: Write the failing test for GetExistingMessageIDs**

Add to `internal/repository/jobs_test.go`:

```go
func TestJobsRepository_GetExistingMessageIDs(t *testing.T) {
	// This is a unit test for the SQL query structure
	// The actual DB test would be in jobs_db_test.go

	// For now, we test the interface exists
	var _ interface {
		GetExistingMessageIDs(ctx context.Context, targetID uuid.UUID) ([]int64, error)
	} = (*JobsRepository)(nil)
}
```

**Step 4: Run test to verify it fails**

Run: `go test -v -run TestJobsRepository_GetExistingMessageIDs ./internal/repository/...`
Expected: FAIL - method doesn't exist yet

**Step 5: Implement GetExistingMessageIDs**

Add to `internal/repository/jobs.go` after the `BulkDelete` method:

```go
// GetExistingMessageIDs returns all tg_message_ids for jobs belonging to a target
// Used to check which messages already have jobs created (vs just being in parsed range)
func (r *JobsRepository) GetExistingMessageIDs(ctx context.Context, targetID uuid.UUID) ([]int64, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT tg_message_id
		FROM jobs
		WHERE target_id = $1 AND tg_message_id IS NOT NULL
	`, targetID)
	if err != nil {
		return nil, fmt.Errorf("get existing message ids: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan message id: %w", err)
		}
		ids = append(ids, id)
	}

	if ids == nil {
		return []int64{}, nil
	}
	return ids, nil
}
```

**Step 6: Run test to verify it passes**

Run: `go test -v -run TestJobsRepository_GetExistingMessageIDs ./internal/repository/...`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/repository/jobs.go internal/repository/jobs_test.go
git commit -m "feat(repository): add GetExistingMessageIDs for smarter filtering

Returns all tg_message_ids that have jobs for a target.
Used to distinguish messages that were truly processed vs just seen."
```

---

### Task 2: Create SmartMessageFilter

**Files:**
- Modify: `internal/repository/ranges.go`
- Test: `internal/repository/ranges_test.go`

**Step 1: Write the failing test**

Add to `internal/repository/ranges_test.go`:

```go
func TestSmartMessageFilter_FilterNew(t *testing.T) {
	tests := []struct {
		name          string
		minParsed     int64
		maxParsed     int64
		existingJobs  []int64
		inputIDs      []int64
		expectedIDs   []int64
	}{
		{
			name:          "all new when no parsed range",
			minParsed:     0,
			maxParsed:     0,
			existingJobs:  []int64{},
			inputIDs:      []int64{100, 101, 102},
			expectedIDs:   []int64{100, 101, 102},
		},
		{
			name:          "filters only messages with existing jobs",
			minParsed:     100,
			maxParsed:     110,
			existingJobs:  []int64{105, 106, 107}, // only these have jobs
			inputIDs:      []int64{103, 105, 106, 107, 108},
			expectedIDs:   []int64{103, 108}, // 103 and 108 have no jobs
		},
		{
			name:          "messages outside range are always new",
			minParsed:     100,
			maxParsed:     110,
			existingJobs:  []int64{105},
			inputIDs:      []int64{99, 105, 111},
			expectedIDs:   []int64{99, 111}, // outside range = new
		},
		{
			name:          "empty existing jobs means all in-range are new",
			minParsed:     100,
			maxParsed:     110,
			existingJobs:  []int64{},
			inputIDs:      []int64{103, 105, 107},
			expectedIDs:   []int64{103, 105, 107},
		},
		{
			name:          "real scenario - gaps in job creation",
			minParsed:     1911,
			maxParsed:     1933,
			existingJobs:  []int64{1928, 1929, 1930, 1931, 1932, 1933},
			inputIDs:      []int64{1920, 1925, 1928, 1930, 1933, 1934},
			expectedIDs:   []int64{1920, 1925, 1934}, // 1920, 1925 no jobs; 1934 outside
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewSmartMessageFilter(tt.minParsed, tt.maxParsed, tt.existingJobs)
			result := filter.FilterNew(tt.inputIDs)

			if len(tt.expectedIDs) == 0 && len(result) == 0 {
				return
			}

			if len(result) != len(tt.expectedIDs) {
				t.Errorf("FilterNew() returned %d items, want %d\ngot: %v\nwant: %v",
					len(result), len(tt.expectedIDs), result, tt.expectedIDs)
				return
			}

			for i, id := range result {
				if id != tt.expectedIDs[i] {
					t.Errorf("FilterNew()[%d] = %d, want %d", i, id, tt.expectedIDs[i])
				}
			}
		})
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestSmartMessageFilter_FilterNew ./internal/repository/...`
Expected: FAIL - SmartMessageFilter doesn't exist

**Step 3: Implement SmartMessageFilter**

Add to `internal/repository/ranges.go` after `MessageIDFilter`:

```go
// SmartMessageFilter filters messages based on both range AND existing jobs
// A message is "new" if:
// 1. It's outside the parsed range [min, max], OR
// 2. It's inside the range but no job exists for it
type SmartMessageFilter struct {
	minParsed    int64
	maxParsed    int64
	existingJobs map[int64]bool
}

// NewSmartMessageFilter creates a filter with range and existing job IDs
func NewSmartMessageFilter(minParsed, maxParsed int64, existingJobIDs []int64) *SmartMessageFilter {
	jobSet := make(map[int64]bool, len(existingJobIDs))
	for _, id := range existingJobIDs {
		jobSet[id] = true
	}
	return &SmartMessageFilter{
		minParsed:    minParsed,
		maxParsed:    maxParsed,
		existingJobs: jobSet,
	}
}

// FilterNew returns message IDs that should be processed
// Messages are new if outside range OR if inside range but no job exists
func (f *SmartMessageFilter) FilterNew(messageIDs []int64) []int64 {
	if len(messageIDs) == 0 {
		return []int64{}
	}

	// If no range exists, all messages are new
	if f.minParsed == 0 && f.maxParsed == 0 {
		return messageIDs
	}

	var newIDs []int64
	for _, id := range messageIDs {
		// Outside range = definitely new
		if id < f.minParsed || id > f.maxParsed {
			newIDs = append(newIDs, id)
			continue
		}
		// Inside range but no job exists = also new
		if !f.existingJobs[id] {
			newIDs = append(newIDs, id)
		}
	}

	if newIDs == nil {
		return []int64{}
	}
	return newIDs
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestSmartMessageFilter_FilterNew ./internal/repository/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/repository/ranges.go internal/repository/ranges_test.go
git commit -m "feat(repository): add SmartMessageFilter for job-aware filtering

New filter checks both parsed range AND existing jobs.
Messages inside range but without jobs are considered new.
Fixes issue where empty/skipped messages blocked re-processing."
```

---

### Task 3: Add NewSmartFilter to RangesRepository

**Files:**
- Modify: `internal/repository/ranges.go`

**Step 1: Write the interface test**

Add to `internal/repository/ranges_test.go`:

```go
func TestRangesRepository_NewSmartFilter_Interface(t *testing.T) {
	// Verify the method signature exists
	type smartFilterCreator interface {
		NewSmartFilter(ctx context.Context, targetID uuid.UUID, existingJobIDs []int64) (*SmartMessageFilter, error)
	}
	var _ smartFilterCreator = (*RangesRepository)(nil)
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestRangesRepository_NewSmartFilter_Interface ./internal/repository/...`
Expected: FAIL - method doesn't exist

**Step 3: Implement NewSmartFilter**

Add to `internal/repository/ranges.go` after `NewFilter`:

```go
// NewSmartFilter creates a smart message filter that checks both range AND job existence
func (r *RangesRepository) NewSmartFilter(ctx context.Context, targetID uuid.UUID, existingJobIDs []int64) (*SmartMessageFilter, error) {
	pr, err := r.GetRange(ctx, targetID)
	if err != nil {
		return nil, err
	}
	if pr == nil {
		return NewSmartMessageFilter(0, 0, existingJobIDs), nil
	}
	return NewSmartMessageFilter(pr.MinMsgID, pr.MaxMsgID, existingJobIDs), nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v -run TestRangesRepository_NewSmartFilter_Interface ./internal/repository/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/repository/ranges.go internal/repository/ranges_test.go
git commit -m "feat(repository): add NewSmartFilter factory method

Creates SmartMessageFilter from parsed range + existing job IDs."
```

---

### Task 4: Update Collector Service to Use Smart Filtering

**Files:**
- Modify: `internal/collector/service.go`

**Step 1: Update the Service struct to include jobs repository reference**

The service already has `jobs *repository.JobsRepository` - verify this exists.

**Step 2: Modify Scrape method to use smart filter**

Find lines 137-143 in `internal/collector/service.go`:

```go
// get message filter for deduplication
s.log.Debug().Msg("scrape: creating message filter")
filter, err := s.ranges.NewFilter(ctx, target.ID)
if err != nil {
	s.log.Error().Err(err).Msg("scrape: failed to create filter")
	return nil, fmt.Errorf("create filter: %w", err)
}
```

Replace with:

```go
// get existing job message IDs for smart filtering
s.log.Debug().Msg("scrape: fetching existing job message IDs")
existingJobIDs, err := s.jobs.GetExistingMessageIDs(ctx, target.ID)
if err != nil {
	s.log.Error().Err(err).Msg("scrape: failed to get existing job IDs")
	return nil, fmt.Errorf("get existing job IDs: %w", err)
}
s.log.Debug().Int("existing_jobs", len(existingJobIDs)).Msg("scrape: found existing jobs")

// get smart message filter for deduplication
s.log.Debug().Msg("scrape: creating smart message filter")
filter, err := s.ranges.NewSmartFilter(ctx, target.ID, existingJobIDs)
if err != nil {
	s.log.Error().Err(err).Msg("scrape: failed to create filter")
	return nil, fmt.Errorf("create filter: %w", err)
}
```

**Step 3: Update the filter usage (line 222)**

The existing code calls `filter.FilterNew(msgIDs)` - this works because both `MessageIDFilter` and `SmartMessageFilter` have the same method signature. No change needed.

**Step 4: Run existing tests**

Run: `go test -v ./internal/collector/...`
Expected: PASS (no breaking changes)

**Step 5: Commit**

```bash
git add internal/collector/service.go
git commit -m "feat(collector): use smart filtering for message deduplication

Now checks if jobs actually exist, not just if message IDs are in range.
Allows re-processing of messages that were previously skipped (empty)."
```

---

### Task 5: Run Integration Test

**Step 1: Reset the parsed range to test the fix**

```bash
docker exec jhos-postgres psql -U jhos -d jhos -c "DELETE FROM parsed_ranges;"
```

**Step 2: Rebuild and restart the collector**

```bash
docker compose up -d --build collector
```

**Step 3: Scrape and verify more jobs are created**

Trigger a scrape via the UI or API and check:
- Logs should show "existing_jobs" count
- More messages should be processed as "new"

**Step 4: Verify with database query**

```bash
docker exec jhos-postgres psql -U jhos -d jhos -c "SELECT COUNT(*) FROM jobs;"
```

Expected: More jobs than before (depending on channel content)

---

### Task 6: Run All Tests

**Step 1: Run Go tests**

```bash
task test
```

Expected: All tests pass

**Step 2: Run linter**

```bash
task lint
```

Expected: No new lint errors

**Step 3: Final commit if needed**

```bash
git add -A
git commit -m "test: verify smarter job filtering works end-to-end"
```

---

## Verification Checklist

- [ ] `go test ./internal/repository/...` passes
- [ ] `go test ./internal/collector/...` passes
- [ ] `task lint` passes
- [ ] Scraping creates jobs for previously-skipped message IDs
- [ ] Logs show "existing_jobs" count during scrape
- [ ] Re-scraping same channel processes messages that had no jobs
