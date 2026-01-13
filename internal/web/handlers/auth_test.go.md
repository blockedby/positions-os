# auth_test.go

Unit tests for Telegram authentication handler.

## Test Environment / Test Fixtures

### MockTelegramClient
Mock implementation of `TelegramClient` interface.
| Method | Behavior |
|--------|----------|
| `StartQR()` | Returns error or calls `onQRCode` callback before returning |
| `GetStatus()` | Returns mocked `telegram.Status` |
| `IsQRInProgress()` | Returns mocked bool |
| `CancelQR()` | Records call (void return) |

### MockHub
Mock for WebSocket hub broadcasting.
| Method | Behavior |
|--------|----------|
| `Broadcast()` | Appends message to captured slice |

## Test Cases

### TestAuthHandler_StartQR_Success
**Scenario:** Valid request → QR flow starts → Success broadcast

**Setup:**
- Mock status: `StatusUnauthorized`
- Mock in-progress: `false`

**Expected Results:**
- HTTP 200 status
- Response body: `{"status":"started"}`
- `StartQR()` called
- `Broadcast()` called with QR code and success message

---

### TestAuthHandler_StartQR_AlreadyLoggedIn
**Scenario:** User already authenticated → Returns 400

**Setup:**
- Mock status: `StatusReady`

**Expected Results:**
- HTTP 400 status
- Response body: `{"error":"already logged in"}`
- `StartQR()` NOT called

---

### TestAuthHandler_StartQR_AlreadyInProgress
**Scenario:** QR flow already running → Returns 202

**Setup:**
- Mock status: `StatusUnauthorized`
- Mock in-progress: `true`

**Expected Results:**
- HTTP 202 Accepted
- Response body: `{"status":"already in progress"}`
- `StartQR()` NOT called

---

### TestAuthHandler_StartQR_BroadcastsQRCode
**Scenario:** QR callback triggered → WebSocket broadcast

**Setup:**
- Mock hub that captures broadcasts
- Mock `StartQR()` that calls `onQRCode("http://t.me/auth/test")`

**Expected Results:**
- Message 1: `{"type":"tg_qr","url":"http://t.me/auth/test"}`
- Message 2: `{"type":"tg_auth_success"}`

---

### TestAuthHandler_StartQR_BroadcastsError
**Scenario:** QR flow fails → Error broadcast

**Setup:**
- Mock `StartQR()` returns `errors.New("auth failed")`

**Expected Results:**
- HTTP 200 (error handled async)
- Broadcast: `{"type":"error","message":"auth failed"}`

---

### TestAuthHandler_StartQR_ConcurrentCalls
**Scenario:** Multiple simultaneous requests → All handled safely

**Setup:**
- Launch 10 goroutines calling handler concurrently

**Expected Results:**
- All return 200 OK
- No race conditions or panics

## Coverage Summary

| Test | Covers |
|------|--------|
| Success | Happy path, QR callback, success broadcast |
| AlreadyLoggedIn | Status check before starting QR |
| AlreadyInProgress | In-progress flag check |
| BroadcastsQRCode | WebSocket integration |
| BroadcastsError | Error handling and broadcasting |
| ConcurrentCalls | Thread safety |
