# QR Code Blocking Fix Plan - TDD Approach

## Executive Summary

The backend hangs at "Enter Phone Number:" because `gotgproto.ClientTypePhone("")` attempts interactive CLI authentication. This document provides a comprehensive TDD-based solution with pseudo-code and acceptance criteria.

---

## Root Cause Analysis

### 🔴 Problem Statement

```
gotgproto.NewClient(apiID, apiHash, gotgproto.ClientTypePhone(""), opts)
```

When `ClientTypePhone("")` is passed, `gotgproto` interprets this as:

- "No phone provided → prompt the user for CLI input"
- This blocks the goroutine waiting for stdin input that never comes inside Docker

### Key Insight

We need TWO different client creation paths:

1. **Session Restoration Flow**: Use `ClientTypePhone("")` - If a VALID session exists in DB, it restores without prompting
2. **QR Login Flow**: Use `gotd/td` directly (not gotgproto's `NewClient`) - Creates raw client, then calls QR auth

---

## Architecture Fix

### Option A: Two-Phase Client Initialization (Recommended)

```
┌──────────────────────────────────────────────────────────────┐
│                      Manager.Init()                            │
│  Check DB → Has Session? → YES → DefaultClient (gotgproto)     │
│                         → NO  → Status=UNAUTHORIZED (no client)│
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                      Manager.StartQR()                         │
│  Create RAW td/telegram client (not gotgproto)                 │
│  → client.QR().Auth() with update dispatcher                   │
│  → On success: save session to DB                              │
│  → Reinitialize Manager with saved session                     │
└──────────────────────────────────────────────────────────────┘
```

### Option B: Custom AuthConversator (Alternative)

```
// gotgproto allows a custom AuthConversator that never blocks
type NonBlockingAuth struct{}
func (a *NonBlockingAuth) AskPhoneNumber() (string, error) { return "", ErrNeedQR }
func (a *NonBlockingAuth) AskCode() (string, error) { return "", ErrNeedQR }
// ... etc
```

We'll implement **Option A** as it's cleaner and separates concerns.

---

## TDD Implementation Plan

### Phase 1: RED - Write Failing Tests

#### Test 1.1: `TestQRClientFactory_ReturnsRawClient`

```go
// File: internal/telegram/qr_client_test.go

func TestQRClientFactory_ReturnsRawClient(t *testing.T) {
    // Arrange
    cfg := &config.Config{TGApiID: 12345, TGApiHash: "hash"}

    // Act: Create QR-specific client factory result
    rawClient, dispatcher, memStorage, err := NewQRClient(cfg)

    // Assert
    require.NoError(t, err)
    require.NotNil(t, rawClient, "raw td/telegram client should be created")
    require.NotNil(t, dispatcher, "update dispatcher should be provided")
    require.NotNil(t, memStorage, "memory storage should be provided for session capture")
}
```

**AC**: Test FAILS because `NewQRClient` doesn't exist

#### Test 1.2: `TestQRClientFactory_DoesNotBlock`

```go
func TestQRClientFactory_DoesNotBlock(t *testing.T) {
    cfg := &config.Config{TGApiID: 12345, TGApiHash: "hash"}

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    done := make(chan struct{})
    go func() {
        _, _, _, _ = NewQRClient(cfg)
        close(done)
    }()

    select {
    case <-done:
        // Success: function returned without blocking
    case <-ctx.Done():
        t.Fatal("NewQRClient blocked for >2 seconds - likely waiting for CLI input")
    }
}
```

**AC**: Test FAILS (times out) with current implementation that uses `gotgproto.ClientTypePhone("")`

#### Test 1.3: `TestManager_StartQR_UsesQRClient`

```go
func TestManager_StartQR_UsesQRClient(t *testing.T) {
    // Setup
    db := setupTestDB(t)
    cfg := &config.Config{TGApiID: 12345, TGApiHash: "hash"}
    m := NewManager(cfg, db)

    qrClientCalled := false
    m.SetQRClientFactory(func(cfg *config.Config) (*QRClientBundle, error) {
        qrClientCalled = true
        return nil, errors.New("mock: stop after factory")
    })

    // Act
    _ = m.StartQR(context.Background(), func(url string) {})

    // Assert
    assert.True(t, qrClientCalled, "StartQR should use QRClientFactory, not regular ClientFactory")
}
```

**AC**: Test FAILS because `SetQRClientFactory` doesn't exist

#### Test 1.4: `TestManager_StartQR_SavesSessionToDB`

```go
func TestManager_StartQR_SavesSessionToDB_AfterSuccess(t *testing.T) {
    // This is an integration boundary test
    // Mocks the QR auth success and verifies session persistence

    db := setupTestDB(t)
    cfg := &config.Config{...}
    m := NewManager(cfg, db)

    // Mock: QR auth returns success with fake session data
    m.SetQRClientFactory(func(cfg *config.Config) (*QRClientBundle, error) {
        return &QRClientBundle{
            MockAuthSuccess: true,
            SessionData: []byte(`{"DC":2,"AuthKey":"dGVzdA=="}`),
        }, nil
    })

    // Act
    err := m.StartQR(context.Background(), func(url string) {})

    // Assert
    require.NoError(t, err)

    var count int64
    db.Table("sessions").Count(&count)
    assert.Greater(t, count, int64(0), "session should be saved after QR success")
}
```

**AC**: Test FAILS because session saving after QR auth isn't implemented

---

### Phase 2: GREEN - Implement Minimal Code

#### Step 2.1: Create `internal/telegram/qr_client.go`

```go
package telegram

import (
    "github.com/gotd/td/session"
    "github.com/gotd/td/telegram"
    "github.com/gotd/td/tg"
)

// QRClientBundle contains all components needed for QR authentication
type QRClientBundle struct {
    Client     *telegram.Client
    Dispatcher *tg.UpdateDispatcher
    Storage    *session.StorageMemory
}

// NewQRClient creates a raw td/telegram client suitable for QR authentication.
// Unlike gotgproto's NewClient, this does NOT attempt interactive CLI auth.
func NewQRClient(cfg *config.Config) (*QRClientBundle, error) {
    memStorage := &session.StorageMemory{}
    dispatcher := tg.NewUpdateDispatcher()

    client := telegram.NewClient(cfg.TGApiID, cfg.TGApiHash, telegram.Options{
        SessionStorage: memStorage,
        UpdateHandler:  dispatcher,
    })

    return &QRClientBundle{
        Client:     client,
        Dispatcher: dispatcher,
        Storage:    memStorage,
    }, nil
}
```

#### Step 2.2: Add QR Factory to Manager

```go
// In manager.go

type QRClientFactory func(cfg *config.Config) (*QRClientBundle, error)

type Manager struct {
    // ... existing fields ...
    qrClientFactory QRClientFactory
}

func NewManager(...) *Manager {
    return &Manager{
        // ... existing ...
        qrClientFactory: NewQRClient, // default
    }
}

func (m *Manager) SetQRClientFactory(f QRClientFactory) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.qrClientFactory = f
}
```

#### Step 2.3: Refactor `StartQR` to use QR Client

```go
func (m *Manager) StartQR(ctx context.Context, onQRCode func(url string)) error {
    m.mu.Lock()
    if m.status == StatusReady {
        m.mu.Unlock()
        return fmt.Errorf("already logged in")
    }
    m.mu.Unlock()

    m.log.Info().Msg("telegram: creating QR-specific client")

    // Use the QR client factory (raw td/telegram, not gotgproto)
    bundle, err := m.qrClientFactory(m.cfg)
    if err != nil {
        return fmt.Errorf("create QR client: %w", err)
    }

    var authErr error
    var sessionData *session.Data

    // Run the client connection
    err = bundle.Client.Run(ctx, func(ctx context.Context) error {
        qr := bundle.Client.QR()
        loggedIn := qrlogin.OnLoginToken(bundle.Dispatcher)

        _, authErr = qr.Auth(ctx, loggedIn, func(ctx context.Context, token qrlogin.Token) error {
            m.log.Info().Str("url", token.URL()).Msg("telegram: QR token generated")
            onQRCode(token.URL())
            return nil
        })

        if authErr != nil {
            return authErr
        }

        // On success, capture session
        loader := session.Loader{Storage: bundle.Storage}
        sessionData, authErr = loader.Load(ctx)
        return authErr
    })

    if err != nil || authErr != nil {
        return fmt.Errorf("QR auth flow: %w", errors.Join(err, authErr))
    }

    // Save session to database
    if err := m.saveSessionToDB(sessionData); err != nil {
        return fmt.Errorf("save session: %w", err)
    }

    // Reinitialize with saved session
    return m.Init(ctx)
}

func (m *Manager) saveSessionToDB(data *session.Data) error {
    // Serialize session.Data to JSON
    dataJSON, err := json.Marshal(data)
    if err != nil {
        return err
    }

    // Create gotgproto-compatible session record
    sess := storage.Session{
        Version: storage.LatestVersion,
        Data:    dataJSON,
    }

    // Save to database
    return m.db.Save(&sess).Error
}
```

---

### Phase 3: REFACTOR - Clean Up

#### Refactor 3.1: Extract Session Conversion Logic

```go
// internal/telegram/session_converter.go

// ConvertToGotgprotoSession converts gotd session.Data to gotgproto storage.Session
func ConvertToGotgprotoSession(data *session.Data) (*storage.Session, error) {
    dataJSON, err := json.Marshal(data)
    if err != nil {
        return nil, fmt.Errorf("marshal session: %w", err)
    }
    return &storage.Session{
        Version: storage.LatestVersion,
        Data:    dataJSON,
    }, nil
}
```

#### Refactor 3.2: Add Graceful Shutdown for QR Client

```go
// Ensure QR client stops properly on context cancel
defer func() {
    if bundle.Client != nil {
        // Client stops automatically when Run returns
    }
}()
```

#### Refactor 3.3: Improve Error Messages

```go
const (
    ErrQRClientCreation = "failed to create QR authentication client"
    ErrQRAuthFlow       = "QR authentication flow failed"
    ErrSessionSave      = "failed to persist session after QR authentication"
)
```

---

## Acceptance Criteria (AC) Summary

### AC-1: Non-Blocking Client Creation ✅

| Criteria                                | Verification           |
| --------------------------------------- | ---------------------- |
| `NewQRClient()` returns in <2 seconds   | Unit test with timeout |
| No "Enter Phone Number:" prompt in logs | Docker log inspection  |
| Client is ready for `.QR().Auth()` call | Integration test       |

### AC-2: QR Token Broadcast ✅

| Criteria                              | Verification            |
| ------------------------------------- | ----------------------- |
| Token URL sent to `onQRCode` callback | Unit test mock          |
| WebSocket receives `tg_qr_code` event | E2E test with WS client |
| QR code visible at `/settings` page   | Manual browser test     |

### AC-3: Session Persistence ✅

| Criteria                              | Verification        |
| ------------------------------------- | ------------------- |
| `sessions` table populated after auth | DB query assertion  |
| Container restart preserves auth      | Integration test    |
| No re-auth required after restart     | Manual verification |

### AC-4: Status Updates ✅

| Criteria                               | Verification            |
| -------------------------------------- | ----------------------- |
| Status → `READY` after successful auth | Unit test on Manager    |
| Frontend shows "Connected"             | Browser UI verification |
| `/api/v1/scrape/status` returns ready  | HTTP test               |

---

## File Changes Summary

| File                                          | Change                                         |
| --------------------------------------------- | ---------------------------------------------- |
| `internal/telegram/qr_client.go`              | **NEW** - Raw td/telegram client for QR        |
| `internal/telegram/qr_client_test.go`         | **NEW** - Unit tests for QR client             |
| `internal/telegram/session_converter.go`      | **NEW** - Session format conversion            |
| `internal/telegram/session_converter_test.go` | **NEW** - Conversion tests                     |
| `internal/telegram/manager.go`                | **MODIFY** - Add QR factory, refactor StartQR  |
| `internal/telegram/factory.go`                | **MODIFY** - Keep for session restoration only |
| `internal/telegram/qr_test.go`                | **MODIFY** - Update tests for new architecture |

---

## Execution Order

```
1. [RED]    Write TestQRClientFactory_DoesNotBlock → FAILS
2. [RED]    Write TestQRClientFactory_ReturnsRawClient → FAILS
3. [GREEN]  Implement NewQRClient in qr_client.go
4. [GREEN]  Run tests → TestQRClientFactory_* PASS

5. [RED]    Write TestManager_StartQR_UsesQRClient → FAILS
6. [GREEN]  Add QRClientFactory to Manager
7. [GREEN]  Run tests → PASS

8. [RED]    Write TestManager_StartQR_SavesSessionToDB → FAILS
9. [GREEN]  Implement saveSessionToDB
10.[GREEN]  Run tests → PASS

11.[REFACTOR] Extract session converter
12.[REFACTOR] Add comprehensive error handling
13.[REFACTOR] Add logging improvements

14.[E2E]    Build Docker → Deploy → Test QR flow in browser
```

---

## Risk Mitigation

### Risk 1: gotd/td client incompatibility with gotgproto storage

**Mitigation**: Session converter explicitly transforms `session.Data` → `storage.Session`

### Risk 2: DC Migration during QR auth

**Mitigation**: Use `client.QR()` (with auto-migration) not `qrlogin.NewQR()`

### Risk 3: Auth key invalidation on retry

**Mitigation**: Keep the DROP TABLE logic but only trigger on specific 401 errors

---

## Quick Win Test Command

```bash
# Run just the QR-related tests during development
go test -v ./internal/telegram/... -run "QR"
```
