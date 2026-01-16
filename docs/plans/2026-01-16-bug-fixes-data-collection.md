# Data Collection Bug Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix data collection bugs: missing `source_url` generation and add status transition validation.

**Architecture:** Two targeted fixes: (1) modify collector service to generate Telegram message URLs when creating jobs, (2) add status transition validation in the jobs handler to prevent skipping ANALYZED status.

**Tech Stack:** Go, Chi router, PostgreSQL

---

## Task 1: Fix source_url Generation in Collector

**Files:**
- Modify: `internal/collector/service.go:388-404` (createJob function)
- Create: `internal/collector/service_test.go`

### Step 1: Write the failing test

Create `internal/collector/service_test.go`:

```go
package collector

import (
	"testing"

	"github.com/blockedby/positions-os/internal/telegram"
)

func TestBuildSourceURL(t *testing.T) {
	tests := []struct {
		name       string
		channelURL string
		messageID  int
		want       string
	}{
		{
			name:       "channel with @ prefix",
			channelURL: "@stablegram",
			messageID:  1244,
			want:       "https://t.me/stablegram/1244",
		},
		{
			name:       "channel without @ prefix",
			channelURL: "golang_jobs",
			messageID:  100,
			want:       "https://t.me/golang_jobs/100",
		},
		{
			name:       "channel with https prefix",
			channelURL: "https://t.me/remote_it",
			messageID:  500,
			want:       "https://t.me/remote_it/500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSourceURL(tt.channelURL, tt.messageID)
			if got != tt.want {
				t.Errorf("buildSourceURL() = %s, want %s", got, tt.want)
			}
		})
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/collector/... -run TestBuildSourceURL -v`
Expected: FAIL with "undefined: buildSourceURL"

### Step 3: Write the buildSourceURL helper function

Add to `internal/collector/service.go` (after the `min` helper at line 436):

```go
// buildSourceURL creates a Telegram message URL from channel and message ID
func buildSourceURL(channelURL string, messageID int) string {
	// Extract channel name from various formats
	channel := channelURL

	// Remove @ prefix
	channel = strings.TrimPrefix(channel, "@")

	// Remove https://t.me/ prefix
	channel = strings.TrimPrefix(channel, "https://t.me/")
	channel = strings.TrimPrefix(channel, "http://t.me/")

	return fmt.Sprintf("https://t.me/%s/%d", channel, messageID)
}
```

Add `"strings"` to imports if not present.

### Step 4: Run test to verify it passes

Run: `go test ./internal/collector/... -run TestBuildSourceURL -v`
Expected: PASS

### Step 5: Modify createJob to use buildSourceURL

Modify the `createJob` function signature and implementation in `internal/collector/service.go`:

Change line 388 from:
```go
func (s *Service) createJob(ctx context.Context, targetID uuid.UUID, msg *telegram.Message) error {
```
To:
```go
func (s *Service) createJob(ctx context.Context, targetID uuid.UUID, channelURL string, msg *telegram.Message) error {
```

Modify the job creation (lines 392-399) from:
```go
job := &repository.Job{
	TargetID:    targetID,
	ExternalID:  strconv.FormatInt(msgID, 10),
	RawContent:  msg.Text,
	SourceDate:  &sourceDate,
	TgMessageID: &msgID,
	Status:      "RAW",
}
```
To:
```go
sourceURL := buildSourceURL(channelURL, msg.ID)
job := &repository.Job{
	TargetID:    targetID,
	ExternalID:  strconv.FormatInt(msgID, 10),
	RawContent:  msg.Text,
	SourceURL:   &sourceURL,
	SourceDate:  &sourceDate,
	TgMessageID: &msgID,
	Status:      "RAW",
}
```

### Step 6: Update the Scrape method to pass channelURL

Find line 272 in `internal/collector/service.go`:
```go
if err := s.createJob(ctx, target.ID, &msg); err != nil {
```
Change to:
```go
if err := s.createJob(ctx, target.ID, target.URL, &msg); err != nil {
```

### Step 7: Run all collector tests

Run: `go test ./internal/collector/... -v`
Expected: All tests PASS

### Step 8: Commit

```bash
git add internal/collector/service.go internal/collector/service_test.go
git commit -m "$(cat <<'EOF'
fix(collector): generate source_url for scraped jobs

Add buildSourceURL helper to create Telegram message URLs from channel
and message ID. Jobs now include source_url field populated as
https://t.me/{channel}/{message_id}.

Fixes #2 from BUG-REPORT.md

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Add Status Transition Validation

**Files:**
- Modify: `internal/repository/jobs.go:46-52` (add transition validation)
- Modify: `internal/repository/jobs_test.go` (add transition tests)
- Modify: `internal/web/handlers/jobs.go:30-64` (use validation in handler)
- Modify: `internal/web/handlers/jobs_test.go` (add handler test)

### Step 1: Write failing tests for valid status transitions

Add to `internal/repository/jobs_test.go`:

```go
// test valid status transitions
func TestJob_CanTransitionTo(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		// Valid transitions from RAW
		{"RAW", "ANALYZED", true},
		{"RAW", "REJECTED", true},

		// Invalid: can't skip ANALYZED to go to INTERESTED
		{"RAW", "INTERESTED", false},
		{"RAW", "TAILORED", false},
		{"RAW", "SENT", false},
		{"RAW", "RESPONDED", false},

		// Valid transitions from ANALYZED
		{"ANALYZED", "INTERESTED", true},
		{"ANALYZED", "REJECTED", true},

		// Valid transitions from INTERESTED
		{"INTERESTED", "TAILORED", true},
		{"INTERESTED", "REJECTED", true},

		// Valid transitions from TAILORED
		{"TAILORED", "SENT", true},
		{"TAILORED", "REJECTED", true},

		// Valid transitions from SENT
		{"SENT", "RESPONDED", true},
		{"SENT", "REJECTED", true},

		// Always allow re-analysis
		{"ANALYZED", "RAW", true},
		{"INTERESTED", "RAW", true},
		{"REJECTED", "RAW", true},
	}

	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			job := Job{Status: tt.from}
			got := job.CanTransitionTo(tt.to)
			if got != tt.want {
				t.Errorf("CanTransitionTo(%s) = %v, want %v", tt.to, got, tt.want)
			}
		})
	}
}
```

### Step 2: Run test to verify it fails

Run: `go test ./internal/repository/... -run TestJob_CanTransitionTo -v`
Expected: FAIL with "job.CanTransitionTo undefined"

### Step 3: Implement CanTransitionTo method

Add to `internal/repository/jobs.go` after `IsNew()` method (around line 58):

```go
// validTransitions defines allowed status transitions
// Key is "from" status, value is set of allowed "to" statuses
var validTransitions = map[string]map[string]bool{
	"RAW": {
		"ANALYZED": true,
		"REJECTED": true,
	},
	"ANALYZED": {
		"INTERESTED": true,
		"REJECTED":   true,
		"RAW":        true, // allow re-analysis
	},
	"INTERESTED": {
		"TAILORED": true,
		"REJECTED": true,
		"RAW":      true,
	},
	"REJECTED": {
		"RAW": true, // allow re-processing
	},
	"TAILORED": {
		"SENT":     true,
		"REJECTED": true,
		"RAW":      true,
	},
	"SENT": {
		"RESPONDED": true,
		"REJECTED":  true,
		"RAW":       true,
	},
	"RESPONDED": {
		"RAW": true,
	},
}

// CanTransitionTo checks if status transition is valid
func (j *Job) CanTransitionTo(newStatus string) bool {
	allowed, ok := validTransitions[j.Status]
	if !ok {
		return false
	}
	return allowed[newStatus]
}
```

### Step 4: Run test to verify it passes

Run: `go test ./internal/repository/... -run TestJob_CanTransitionTo -v`
Expected: PASS

### Step 5: Commit repository changes

```bash
git add internal/repository/jobs.go internal/repository/jobs_test.go
git commit -m "$(cat <<'EOF'
feat(repository): add status transition validation

Add CanTransitionTo method to validate job status changes.
Prevents invalid transitions like RAW -> INTERESTED (must go through ANALYZED).
Always allows returning to RAW for re-processing.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

### Step 6: Write failing test for handler validation

Add to `internal/web/handlers/jobs_test.go`:

```go
func TestJobsHandler_UpdateStatus_InvalidTransition(t *testing.T) {
	mockRepo := mocks.NewMockJobsRepository(t)
	handler := NewJobsHandler(mockRepo, nil)

	// Job is in RAW status
	jobID := uuid.New()
	mockRepo.EXPECT().GetByID(mock.Anything, jobID).Return(&repository.Job{
		ID:     jobID,
		Status: "RAW",
	}, nil)

	// Try to transition RAW -> INTERESTED (invalid, must go through ANALYZED)
	body := `{"status": "INTERESTED"}`
	req := httptest.NewRequest("PATCH", "/api/v1/jobs/"+jobID.String()+"/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", jobID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	handler.UpdateStatus(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("UpdateStatus() status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rr.Body.String(), "Invalid status transition") {
		t.Errorf("UpdateStatus() body = %s, want 'Invalid status transition'", rr.Body.String())
	}
}
```

Add imports if needed: `"strings"`, `"context"`, `"net/http/httptest"`.

### Step 7: Run test to verify it fails

Run: `go test ./internal/web/handlers/... -run TestJobsHandler_UpdateStatus_InvalidTransition -v`
Expected: FAIL (handler doesn't check transitions)

### Step 8: Update handler to validate transitions

Modify `internal/web/handlers/jobs.go` `UpdateStatus` function. After the status validation (line 51) add:

```go
// Get current job to check transition validity
job, err := h.repo.GetByID(r.Context(), id)
if err != nil {
	http.Error(w, err.Error(), http.StatusInternalServerError)
	return
}
if job == nil {
	http.NotFound(w, r)
	return
}

// Validate status transition
if !job.CanTransitionTo(payload.Status) {
	http.Error(w, "Invalid status transition", http.StatusBadRequest)
	return
}
```

The handler now needs GetByID in the interface. Check if JobsRepository interface already has it.

### Step 9: Run test to verify it passes

Run: `go test ./internal/web/handlers/... -run TestJobsHandler_UpdateStatus -v`
Expected: All UpdateStatus tests PASS

### Step 10: Run full test suite

Run: `go test ./... -v`
Expected: All tests PASS

### Step 11: Commit handler changes

```bash
git add internal/web/handlers/jobs.go internal/web/handlers/jobs_test.go
git commit -m "$(cat <<'EOF'
fix(handler): validate status transitions before update

UpdateStatus handler now checks if the requested status transition is
valid before applying. Returns 400 Bad Request for invalid transitions
like RAW -> INTERESTED.

Fixes #4 from BUG-REPORT.md

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Configuration Fix (Manual Step)

**Files:**
- Modify: `.env` (add LLM_API_KEY)

This is a configuration-only task. The user needs to:

1. Add `LLM_API_KEY` to `.env`:
```bash
LLM_API_KEY=your-api-key-here
```

2. Restart the analyzer service:
```bash
task docker-down && task docker-up
```

This addresses Bug #1 (CRITICAL) from the bug report.

---

## Summary

| Bug | Severity | Fix | Task |
|-----|----------|-----|------|
| #1 LLM API Key Missing | CRITICAL | Add to `.env` | Task 3 (manual) |
| #2 source_url NULL | HIGH | Generate URL in collector | Task 1 |
| #3 tg_topic_id NULL | LOW | Already handled in code (channels don't have topics) | N/A |
| #4 Status skip ANALYZED | LOW | Add transition validation | Task 2 |

**Total code tasks:** 2
**Estimated test coverage:** 100% (all new code has tests)
