# Telegram Authentication - Extended Test Plan

## Overview

This document extends the initial test plan to cover the full Telegram QR login flow, including MTProto-level mechanics like Data Center (DC) migration and 2FA handling. These tests are designed to catch subtle integration bugs like dispatcher mismatch or session capture failures.

---

## 🏗️ Technical Flow Breakdown

### 1. The QR Lifecycle

1. **Token Export**: Client calls `auth.exportLoginToken`.
2. **Token Display**: Backend sends `tg://login?token=...` to Frontend via WebSocket.
3. **User Action**: User scans with official Telegram app and clicks "Allow".
4. **Update Reception**: (CRITICAL) MTProto server sends `updateLoginToken`. The client's **UpdateDispatcher** must be listening and active.
5. **Confirmation**: Client makes second `auth.exportLoginToken` call to finalize auth.
6. **DC Migration** (If needed): Server returns `auth.loginTokenMigrateTo`. Client must reconnect to the new DC and call `auth.importLoginToken`.
7. **2FA Check** (If needed): Backend must detect `PASSWORD_HASH_INVALID` and request 2FA.

### 2. Failure Surface Analysis

| Failure Type             | Symptom                                              | Likely Cause                                                                         |
| ------------------------ | ---------------------------------------------------- | ------------------------------------------------------------------------------------ |
| **Update Mismatch**      | QR scanned on phone, but backend hangs until timeout | Client is using a copy or different instance of Dispatcher                           |
| **DC Migration Failure** | Error "MTProto error: MIGRATE_X"                     | Client options don't support auto-migration                                          |
| **Session Loss**         | Auth succeeds, but restart requires re-scan          | Session storage not flushed or correctly marshaled                                   |
| **Token Expiry**         | Phone says "Token Expired" quickly                   | Client not calling `exportLoginToken` frequently enough or missing confirmation call |

---

## 🧪 Detailed Test Cases

### 1. Phase 1: Bundle Integrity (Bug Hunter)

**Goal**: Catch the "missing updates" bug where the client and bundle don't share the same dispatcher.

| Test ID   | Case Name                                 | Description                                                                                                                          |
| --------- | ----------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| **TH-01** | `TestQRClient_DispatcherPointerIntegrity` | **(BUG DISCOVERED)** Verifies that `bundle.Dispatcher` is the EXACT same pointer instance as the one passed to `telegram.NewClient`. |
| **TH-02** | `TestQRClient_StorageIntegrity`           | Verifies that `bundle.Storage` is shared between the bundle and the client.                                                          |

### 2. Phase 2: Success Path Simulation

**Goal**: Verify the transition from "Scanned" to "Authenticated".

| Test ID   | Case Name                              | Description                                                                                          |
| --------- | -------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| **TS-01** | `TestQRFlow_AuthSuccess`               | Simulates an `updateLoginToken` being pushed to the dispatcher and verifies `qr.Auth()` return.      |
| **TS-02** | `TestQRFlow_SessionCaptured`           | Verifies that after `qr.Auth()` returns, the memory storage contains a non-empty auth key and DC ID. |
| **TS-03** | `TestSessionConverter_JSONConsistency` | Verifies that `ConvertToGotgprotoSession` produces JSON matching `gotgproto`'s expectations.         |

### 3. Phase 3: Edge Cases & Resilience

**Goal**: Handle migration and account-specific settings.

| Test ID   | Case Name                      | Description                                                                                         |
| --------- | ------------------------------ | --------------------------------------------------------------------------------------------------- |
| **TE-01** | `TestQRFlow_MigrationRequired` | Mocks a DC migration response and verifies the client successfully handles the jump.                |
| **TE-02** | `TestQRFlow_2FA_Detection`     | Verifies that the manager detects when 2FA is required (even if not fully supporting entry yet).    |
| **TE-03** | `TestQRFlow_TokenRefreshLoop`  | Verifies that the `onToken` callback is called multiple times if the user doesn't scan immediately. |

---

## 🛠️ Mock Implementation Sample (for `internal/telegram/qr_flow_test.go`)

```go
func TestQRFlow_UpdateLoopIntegrity(t *testing.T) {
    bundle, _ := NewQRClient(cfg)

    // Register a test handler
    called := false
    bundle.Dispatcher.OnLoginToken(func(ctx context.Context, u *tg.UpdateLoginToken) error {
        called = true
        return nil
    })

    // Trigger an update on the dispatcher
    bundle.Dispatcher.Handle(ctx, []tg.UpdateClass{&tg.UpdateLoginToken{}})

    // Assert
    assert.True(t, called, "Dispatcher didn't trigger the registered handler")
}
```

---

## 🏁 Acceptance Criteria Matrix

| Requirement                     | Verification Test                        |
| ------------------------------- | ---------------------------------------- |
| Client receives MTProto updates | `TestQRFlow_UpdateLoopIntegrity`         |
| Session persists as valid JSON  | `TestConvertToGotgprotoSession_Success`  |
| Manager re-inits after QR       | `TestManager_StartQR_ReinitAfterSuccess` |
| Multiple token attempts succeed | `TestQRFlow_TokenRefreshLoop`            |
