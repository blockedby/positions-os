# Plan: Fix 50 Pre-existing Lint Errors

## Overview
Fix all golangci-lint errors after upgrading to v2 configuration. Errors fall into two categories:
- **bodyclose** (5): HTTP response bodies not closed in tests
- **errcheck** (45): Unchecked error return values

## Tasks

### Task 1: golangci.yml v2 config fix (DONE)
- Move `gci` from linters to formatters section
- Change `default: fast` to `default: none`
- Remove deprecated `relative-path-mode`

### Task 2: Fix bodyclose in test files (5 errors)
Files:
- `internal/web/handlers/jobs_test.go` (2 errors, lines 297, 309)
- `internal/web/server_test.go` (3 errors, lines 28, 115, 132)

Fix: Close response bodies with `defer resp.Body.Close()` after checking error.

### Task 3: Fix errcheck in cmd/ (2 errors)
Files:
- `cmd/analyzer/main.go:30` - `logger.Init`
- `cmd/collector/main.go:192` - `server.Stop`

Fix: Check and log errors appropriately.

### Task 4: Fix errcheck in internal/collector (2 errors)
Files:
- `internal/collector/handler.go:182` - `json.Encode`
- `internal/collector/manager_test.go:180` - `manager.Start`

### Task 5: Fix errcheck in internal/dispatcher and telegram (2 errors)
Files:
- `internal/dispatcher/telegram_sender.go:289` - `fmt.Sscanf`
- `internal/telegram/client.go:345` - `fmt.Sscanf`

### Task 6: Fix errcheck in internal/llm, migrator, nats (5 errors)
Files:
- `internal/llm/prompts_test.go:15` - `os.RemoveAll`
- `internal/migrator/migrator.go` (2 errors, lines 62, 92) - `migrator.Close`
- `internal/nats/client.go` (2 errors, lines 76, 79) - `msg.Nak`, `msg.Ack`

### Task 7: Fix errcheck in internal/repository tests (4 errors)
File: `internal/repository/applications_test.go` (lines 55, 56, 165, 166)
- `pool.Exec` calls

### Task 8: Fix errcheck in internal/web/handlers (17 errors)
Files:
- `applications.go` (5 errors)
- `applications_test.go` (1 error)
- `auth.go` (4 errors)
- `dispatcher.go` (1 error)
- `jobs.go` (2 errors)
- `jobs_test.go` (2 errors)
- `stats.go` (1 error)
- `targets.go` (1 error)

### Task 9: Fix errcheck in internal/web/hub.go (12 errors)
Lines: 119, 124, 127, 135, 140, 141, 148, 164, 167, 168
Various WebSocket operations.

### Task 10: Fix errcheck in internal/web/server (2 errors)
Files:
- `server.go:85` - `w.Write`
- `server_test.go:23` - `srv.Start`

### Task 11: Fix internal/telegram/persistence_test.go (1 error)
Line 37: `db.AutoMigrate`

## Verification
After each task:
```bash
task lint 2>&1 | grep -c "^internal\|^cmd" || echo "0 errors"
```

Final verification:
```bash
task lint && task test
```
