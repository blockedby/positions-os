# Plan: Fix 50 Pre-existing Lint Errors

## Status: COMPLETED

## Overview
Fix all golangci-lint errcheck and bodyclose errors after upgrading to v2 configuration.

**Results:**
- **bodyclose** (5): ✅ All fixed
- **errcheck** (45+): ✅ All fixed

## Completed Tasks

### Task 1: golangci.yml v2 config fix ✅
- Move `gci` from linters to formatters section
- Change `default: fast` to `default: none`
- Remove deprecated `relative-path-mode`

### Task 2: Fix bodyclose in test files ✅
Files:
- `internal/web/handlers/jobs_test.go`
- `internal/web/server_test.go`

### Task 3: Fix errcheck in cmd/ ✅
Files:
- `cmd/analyzer/main.go:30` - `logger.Init`
- `cmd/collector/main.go:192` - `server.Stop`

### Task 4: Fix errcheck in internal/collector ✅
Files:
- `internal/collector/handler.go` - `json.Encode`
- `internal/collector/manager_test.go` - `manager.Start`

### Task 5: Fix errcheck in internal/dispatcher and telegram ✅
Files:
- `internal/dispatcher/telegram_sender.go` - `fmt.Sscanf`
- `internal/telegram/client.go` - `fmt.Sscanf`

### Task 6: Fix errcheck in internal/llm, migrator, nats ✅
Files:
- `internal/llm/prompts_test.go` - `os.RemoveAll`
- `internal/migrator/migrator.go` - `migrator.Close`
- `internal/nats/client.go` - `msg.Nak`, `msg.Ack`

### Task 7: Fix errcheck in internal/repository tests ✅
File: `internal/repository/applications_test.go`

### Task 8: Fix errcheck in internal/web/handlers ✅
Files: applications.go, applications_test.go, auth.go, dispatcher.go, jobs.go, jobs_test.go, stats.go, targets.go

### Task 9: Fix errcheck in internal/web/hub.go ✅
Various WebSocket operations.

### Task 10: Fix errcheck in internal/web/server ✅
Files: server.go, server_test.go

### Task 11: Fix internal/telegram/persistence_test.go ✅

### Task 12: Fix gci import formatting ✅
Auto-applied gci formatter to 35 files.

### Task 13: Fix integration test ✅
- `tests/integration/collector_test.go` - `logger.Init`

## Commits Made
1. `docs: add plan for fixing 50 lint errors`
2. `fix(lint): update golangci.yml to v2 format`
3. `fix(lint): close HTTP response bodies in tests`
4. `fix(lint): check error returns in cmd/ packages`
5. `fix(lint): handle errors in internal/collector`
6. `fix(lint): explicitly ignore fmt.Sscanf errors`
7. `fix(lint): handle errors in llm, migrator, nats`
8. `fix(lint): handle errors in repository and telegram tests`
9. `fix(lint): handle json.Encode errors in web handlers`
10. `fix(lint): handle errors in web/hub.go`
11. `fix(lint): handle errors in web/server`
12. `fix(lint): complete errcheck fixes in test files`
13. `style: fix import ordering with gci`

## Remaining Issues (Out of Scope)
The following linter categories have issues but were not part of the original errcheck/bodyclose scope:
- gosec: Security warnings (file permissions, potential file inclusion)
- revive: Code style (package comments, exported types)
- nilnil: Return (nil, nil) patterns
- gocritic: Exit-after-defer warnings
- unparam: Unused parameters
- unused: Unused code

These can be addressed in a follow-up PR if desired.
