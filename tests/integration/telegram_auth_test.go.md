# telegram_auth_test.go

Integration tests for Telegram authentication — validates session persistence and status flow.

## Test Environment

**Prerequisites:**
- `INTEGRATION_TEST=1` environment variable set
- No network calls (factory mocked)

**Test Setup:**
1. Create in-memory SQLite database
2. Create sessions table: `CREATE TABLE sessions (version integer primary key, data blob)`
3. Initialize Manager with test config
4. Mock client factory to avoid real Telegram connections

## Test Cases

### TestTelegramAuth_EmptyDB_StatusUnauthorized

**Scenario:** Fresh database → No session → Unauthorized status

**Steps:**
1. Create empty in-memory DB
2. Create sessions table (empty)
3. Call `m.Init(ctx)`
4. Check status

**Expected Results:**
- Init returns no error
- Status = `Unauthorized`
- No connection attempt made

---

### TestTelegramAuth_SessionInDB_StatusReady

**Scenario:** Valid session in DB → Ready status

**Setup:**
- Seed session in gotgproto format: `{"Version":1,"Data":{"DC":2,"AuthKey":"dGVzdA=="}}`
- Mock factory returns `&gotgproto.Client{}`

**Steps:**
1. Seed sessions table
2. Call `m.Init(ctx)`
3. Check status

**Expected Results:**
- Init returns no error
- Status = `Ready`

---

### TestTelegramAuth_InvalidSession_FallbackUnauthorized

**Scenario:** Corrupted session → Factory fails → Unauthorized

**Setup:**
- Seed invalid JSON: `invalid-json-garbage`
- Mock factory returns error

**Expected Results:**
- Init returns no error (graceful)
- Status = `Unauthorized`

---

### TestTelegramAuth_SessionPersistence_Restart

**Scenario:** Save session → "Restart" (new Manager) → Session restored

**Setup:**
- Shared in-memory DB with cache enabled
- Mock factory returns `&gotgproto.Client{}`

**Steps:**
1. Create first Manager, Init → Status = `Unauthorized`
2. Save session directly to DB: `{"Version":1,"Data":{"DC":2,...}}`
3. Create second Manager ("restart")
4. Mock factory to return success
5. Call `m2.Init(ctx)`

**Expected Results:**
- First Manager: Status = `Unauthorized` (empty DB)
- Second Manager: Status = `Ready` (session persisted)
- Session survives "restart"

**Validates:**
- Session correctly saved to database
- Session correctly loaded on restart
- Manager properly detects existing session

## Failure Modes

- Timeout if context expires before completion
- Error if database operations fail
- Graceful fallback if factory fails

## Coverage Summary

| Test | Covers |
|------|--------|
| EmptyDB_StatusUnauthorized | Initial state without session |
| SessionInDB_StatusReady | Session load from database |
| InvalidSession_FallbackUnauthorized | Error handling |
| SessionPersistence_Restart | Save/load lifecycle |
