# manager_test.go

Unit tests for Telegram client lifecycle manager.

## Test Environment

Uses in-memory SQLite database for session storage tests.

## Test Cases

### TestManager_StartQR_CallsOnQRCode

**Scenario:** QR factory error → Verifies handler reaches factory

**Expected Results:**
- Error contains "factory reached"
- QR callback not invoked

---

### TestManager_Init_FactoryError_Unauthorized

**Scenario:** Session exists but factory fails → Status Unauthorized

**Setup:**
- Seed sessions table with mock data
- Factory returns error

**Expected Results:**
- Init returns no error (graceful degradation)
- Status = `Unauthorized`

---

### TestManager_GetStatus_Concurrent

**Scenario:** 100 concurrent GetStatus() calls

**Expected Results:**
- No race conditions
- No panics

---

### TestManager_Stop_Graceful

**Scenario:** Stop called on manager

**Expected Results:**
- No panic when stopped
- Safe to call multiple times

---

### TestConvertToGotgprotoSession_RoundTrip

**Scenario:** Session data converted to gotgproto format and back

**Setup:**
- Input: `&session.Data{DC: 2, Addr: "1.2.3.4:443", AuthKey: [...]}`

**Expected Results:**
- `parsed["Version"] == 1`
- `parsed["Data"]["DC"] == 2` (nested structure)
- `parsed["Data"]["Addr"] == "1.2.3.4:443"`

**Validates:**
- Correct JSON wrapping for gotgproto compatibility
- Session data properly nested under "Data" key

## Coverage Summary

| Test | Covers |
|------|--------|
| StartQR_CallsOnQRCode | QR factory invocation |
| Init_FactoryError | Error handling, status fallback |
| GetStatus_Concurrent | Thread safety |
| Stop_Graceful | Safe shutdown |
| ConvertToGotgprotoSession_RoundTrip | Session format validation |
