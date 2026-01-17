# Fix Remaining 207 Lint Issues Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix all remaining golangci-lint errors after errcheck/bodyclose cleanup

**Architecture:** Configure linter to exclude test files from unused-parameter checks, then systematically fix issues by category - security issues first, then batch fixes for comments, then code-specific fixes.

**Tech Stack:** Go 1.23+, golangci-lint v2

---

## Issue Summary

| Linter | Count | Priority |
|--------|-------|----------|
| gosec G112 (Slowloris) | 1 | **HIGH** |
| revive (package-comments) | 20 | Medium |
| revive (exported) | 40+ | Medium |
| revive (unused-parameter) | 80+ | Low (config fix) |
| nilnil | 11 | Medium |
| gosec (other) | 21 | Low (acceptable) |
| unused | 6 | Medium |
| gocritic | 4 | Low |
| unparam | 2 | Low |
| ineffassign | 1 | Medium |
| revive (code-specific) | 4 | Medium |

---

## Task 1: HIGH PRIORITY - Fix Slowloris Vulnerability (gosec G112)

**Files:**
- Modify: `internal/web/server.go:101`

**Step 1: Write the fix**

The current code creates an http.Server without ReadHeaderTimeout:
```go
s.httpServer = &http.Server{
    Handler: s.router,
}
```

Fix by adding ReadHeaderTimeout:
```go
s.httpServer = &http.Server{
    Handler:           s.router,
    ReadHeaderTimeout: 10 * time.Second,
}
```

**Step 2: Run linter to verify fix**

Run: `golangci-lint run ./internal/web/server.go 2>&1 | grep G112`
Expected: No output (issue fixed)

**Step 3: Commit**

```bash
git add internal/web/server.go
git commit -m "fix(security): add ReadHeaderTimeout to prevent Slowloris attacks"
```

---

## Task 2: Configure Linter - Exclude Test Files from unused-parameter

**Files:**
- Modify: `.golangci.yml`

**Step 1: Add exclusion rule**

Add to `issues.exclude-rules`:
```yaml
    - path: _test\.go
      linters:
        - gosec
        - unparam
        - revive
      text: "unused-parameter"
```

This will exclude ~80 unused-parameter issues in test files where mock implementations don't use all parameters.

**Step 2: Run linter to verify reduction**

Run: `golangci-lint run ./... 2>&1 | grep -c "unused-parameter"`
Expected: Count reduced significantly (from ~80 to ~15)

**Step 3: Commit**

```bash
git add .golangci.yml
git commit -m "chore(lint): exclude test files from unused-parameter checks"
```

---

## Task 3: Fix Code-Specific revive Issues

**Files:**
- Modify: `internal/telegram/persistence_test.go:25`
- Modify: `cmd/tg-auth/main.go:187`
- Modify: `internal/collector/service.go:448`
- Modify: `internal/migrator/migrator.go:12`

### Step 1: Fix var-naming: SessionId → SessionID

File: `internal/telegram/persistence_test.go:25`

Change:
```go
type MockSession struct {
    SessionId int `gorm:"primaryKey"`
    // ...
}
```

To:
```go
type MockSession struct {
    SessionID int `gorm:"primaryKey"`
    // ...
}
```

### Step 2: Fix context-as-argument: ctx should be first parameter

File: `cmd/tg-auth/main.go:187`

Change:
```go
func exportSessionString(memStorage *session.StorageMemory, ctx context.Context) (string, error) {
```

To:
```go
func exportSessionString(ctx context.Context, memStorage *session.StorageMemory) (string, error) {
```

Update all callers of this function.

### Step 3: Fix redefines-builtin-id: min function

File: `internal/collector/service.go:448`

The function `min` redefines Go's built-in. Options:
- **Option A** (recommended): Delete the function and use built-in `min()` (Go 1.21+)
- **Option B**: Rename to `minInt`

Delete the function:
```go
// DELETE THIS:
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}
```

### Step 4: Fix blank-imports comment

File: `internal/migrator/migrator.go:12`

Change:
```go
import (
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
)
```

To:
```go
import (
    // postgres driver for database migrations
    _ "github.com/golang-migrate/migrate/v4/database/postgres"
)
```

### Step 5: Run linter to verify fixes

Run: `golangci-lint run ./internal/telegram/persistence_test.go ./cmd/tg-auth/main.go ./internal/collector/service.go ./internal/migrator/migrator.go 2>&1`
Expected: No var-naming, context-as-argument, redefines-builtin-id, or blank-imports errors

### Step 6: Commit

```bash
git add internal/telegram/persistence_test.go cmd/tg-auth/main.go internal/collector/service.go internal/migrator/migrator.go
git commit -m "fix(lint): fix code-specific revive issues (naming, context order, builtin redefinition)"
```

---

## Task 4: Add Package Comments

**Files (20 packages):**
- `cmd/analyzer/main.go`
- `cmd/collector/main.go`
- `cmd/tg-auth/main.go`
- `cmd/tg-topics/main.go`
- `cmd/validate-yaml/main.go`
- `internal/analyzer/consumer.go`
- `internal/api/handlers.go`
- `internal/brain/api.go`
- `internal/collector/handler.go`
- `internal/dispatcher/email_sender.go`
- `internal/llm/client.go`
- `internal/publisher/nats.go`
- `internal/repository/applications.go`
- `internal/telegram/client.go`
- `internal/web/events.go`
- `internal/web/handlers/applications.go`

Also fix malformed package comments:
- `internal/config/config.go` - should be "Package config ..."
- `internal/database/database.go` - should be "Package database ..."
- `internal/logger/logger.go` - should be "Package logger ..."
- `internal/models/target.go` - should be "Package models ..."
- `internal/nats/client.go` - should be "Package nats ..."

### Step 1: Add package comments

Template for each file:
```go
// Package <name> provides <brief description>.
package <name>
```

Examples:
- `cmd/analyzer/main.go`: `// Package main implements the analyzer service that processes jobs through LLM.`
- `internal/api/handlers.go`: `// Package api provides HTTP handlers for the REST API.`
- `internal/brain/api.go`: `// Package brain provides job tailoring services using LLM.`

### Step 2: Run linter to verify

Run: `golangci-lint run ./... 2>&1 | grep "package-comments"`
Expected: No output

### Step 3: Commit

```bash
git add cmd/ internal/
git commit -m "docs(lint): add package comments for all packages"
```

---

## Task 5: Fix Exported Type/Function Comments

**Files with missing exported comments (~40 issues):**

Key patterns to document:
- Types: `// TypeName provides...` or `// TypeName represents...`
- Functions: `// FunctionName does...`
- Methods: `// MethodName does...`
- Constants: Block comment for const blocks

### Step 1: Add comments to exported declarations

Priority files (public API):
- `internal/brain/api.go` - BrainJob → Job, BrainService → Service (fix stuttering)
- `internal/brain/llm.go` - BrainLLM → LLM (fix stuttering)
- `internal/dispatcher/service.go` - DispatcherService → Service (fix stuttering)
- `internal/dispatcher/tracker.go:20` - StatusPending const block
- `internal/models/application.go:13` - DeliveryChannelTGDM const block
- `internal/models/application.go:22` - DeliveryStatusPending const block
- `internal/models/job.go:13` - JobStatusRaw const block
- `internal/models/target.go:14` - TargetTypeTGChannel const block
- `internal/repository/stats.go` - DashboardStats, StatsRepository
- `internal/telegram/manager.go` - Status, Manager, NewManager, GetStatus, GetClient, Stop
- `internal/web/events.go` - Fix comment format for JobUpdatedEvent, JobRowUpdateHTML
- `internal/web/handlers/*.go` - Various handlers and types
- `internal/web/hub.go` - NewHub, Hub.Run

### Step 2: Run linter to verify

Run: `golangci-lint run ./... 2>&1 | grep "exported"`
Expected: No output

### Step 3: Commit

```bash
git add internal/
git commit -m "docs(lint): add comments for exported types and functions"
```

---

## Task 6: Define Sentinel Errors for nilnil Returns

**Files (11 issues):**
- `internal/api/server_test.go:33,63,89`
- `internal/dispatcher/telegram_sender_test.go:47`
- `internal/repository/applications.go:88`
- `internal/repository/jobs.go:194,371`
- `internal/repository/ranges.go:153`
- `internal/repository/targets.go:79,105`
- `internal/web/handlers/applications_test.go:46`

### Step 1: Define sentinel errors

Create or add to `internal/repository/errors.go`:
```go
package repository

import "errors"

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")
```

### Step 2: Update repository methods

Change pattern:
```go
// Before:
if err.Error() == "no rows in result set" {
    return nil, nil
}

// After:
if err.Error() == "no rows in result set" {
    return nil, ErrNotFound
}
```

### Step 3: Update test mocks

For test files returning `nil, nil`, return `nil, repository.ErrNotFound` or define local test sentinel.

### Step 4: Run linter to verify

Run: `golangci-lint run ./... 2>&1 | grep "nilnil"`
Expected: No output

### Step 5: Commit

```bash
git add internal/repository/ internal/api/ internal/dispatcher/ internal/web/handlers/
git commit -m "fix(lint): use sentinel errors instead of (nil, nil) returns"
```

---

## Task 7: Remove Unused Code

**Files (6 issues):**
- `internal/analyzer/processor_test.go:164` - func `contains` is unused
- `internal/dispatcher/telegram_sender_test.go:108-120` - type `mockJobApplication` and methods unused

### Step 1: Delete unused code

Delete the unused functions and types.

### Step 2: Run linter to verify

Run: `golangci-lint run ./... 2>&1 | grep "unused"`
Expected: No output

### Step 3: Commit

```bash
git add internal/analyzer/processor_test.go internal/dispatcher/telegram_sender_test.go
git commit -m "fix(lint): remove unused code in test files"
```

---

## Task 8: Fix ineffassign Issue

**Files:**
- `internal/repository/jobs.go:273`

### Step 1: Fix ineffectual assignment

The line `argID++` at the end of a function is ineffectual. Either:
- Remove the line if argID is not used after
- Or fix the logic if it should be used

### Step 2: Run linter to verify

Run: `golangci-lint run ./internal/repository/jobs.go 2>&1 | grep "ineffassign"`
Expected: No output

### Step 3: Commit

```bash
git add internal/repository/jobs.go
git commit -m "fix(lint): remove ineffectual assignment in jobs repository"
```

---

## Task 9: Fix gocritic exitAfterDefer (Optional)

**Files (4 issues):**
- `cmd/analyzer/main.go:71`
- `cmd/collector/main.go:73`
- `cmd/tg-auth/main.go:177`
- `cmd/tg-topics/main.go:67`

These are warnings about `log.Fatal` or `os.Exit` after defer. Options:
- **Option A**: Restructure to return error from main
- **Option B**: Suppress with `//nolint:gocritic` (acceptable for CLI entry points)

### Step 1: Evaluate and fix or suppress

For CLI apps, this pattern is common and often acceptable. Add suppression comment:
```go
//nolint:gocritic // log.Fatal in main is acceptable pattern
log.Fatal().Err(err).Msg("failed to connect")
```

### Step 2: Run linter to verify

Run: `golangci-lint run ./cmd/... 2>&1 | grep "exitAfterDefer"`
Expected: No output (or significantly reduced)

### Step 3: Commit

```bash
git add cmd/
git commit -m "fix(lint): address gocritic exitAfterDefer in CLI entry points"
```

---

## Task 10: Fix unparam Issues (Optional)

**Files (2 issues):**
- `internal/brain/prompts.go:52` - parseXMLPrompt result 2 (error) is always nil
- `internal/brain/prompts.go:59` - parseXMLPromptWithTemplates result 2 (error) is always nil

### Step 1: Evaluate and fix

If the error return is genuinely always nil, either:
- Remove the error return from the function signature
- Or add actual error handling if it should return errors

### Step 2: Run linter to verify

Run: `golangci-lint run ./internal/brain/prompts.go 2>&1 | grep "unparam"`
Expected: No output

### Step 3: Commit

```bash
git add internal/brain/prompts.go
git commit -m "fix(lint): fix always-nil error returns in prompts"
```

---

## Task 11: Fix Remaining unused-parameter in Non-Test Code

**Files (after config exclusion, ~15 remaining):**
- `internal/api/handlers.go` - fuego context params (interface requirement)
- `internal/api/server.go:238,261` - ctx, req params
- `internal/collector/handler.go` - http.Request params
- `internal/collector/manager.go:61` - ctx param
- `internal/dispatcher/email_sender.go` - ctx params
- `internal/llm/client.go:48` - rawContent param
- `internal/nats/client.go:20` - ctx param
- `internal/telegram/factory.go:15` - ctx param
- `internal/web/handlers/auth.go` - http.Request params
- `internal/web/handlers/dispatcher.go:45` - http.Request param
- `internal/web/hub.go:30` - http.Request param
- `internal/web/server.go:33,82` - repo, http.Request params

### Step 1: Rename unused params to _

For params required by interface but unused:
```go
// Before:
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {

// After:
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
```

### Step 2: Run linter to verify

Run: `golangci-lint run ./... 2>&1 | grep "unused-parameter"`
Expected: No output

### Step 3: Commit

```bash
git add internal/
git commit -m "fix(lint): rename unused parameters to underscore"
```

---

## Out of Scope (Acceptable Issues)

The following gosec issues are acceptable and don't need fixing:
- **G304 (file inclusion via variable)**: File paths come from config/CLI args, not user input
- **G301/G302/G306 (file permissions)**: 0755/0644 are standard for non-sensitive files

These can be suppressed globally in `.golangci.yml` if desired:
```yaml
issues:
  exclude-rules:
    - linters:
        - gosec
      text: "G304|G301|G302|G306"
```

---

## Verification

After completing all tasks:

```bash
golangci-lint run ./...
```

Expected: No errors (or only the acceptable gosec warnings if not suppressed)
