# handler_test.go

HTTP handler tests — validates request/response handling using httptest.

## Test Cases

### TestHandler_Health

**Endpoint:** GET /health

**Expected:** HTTP 200 with `{"status": "ok", "time": "..."}`

---

### TestHandler_StartScrape/returns_400_on_empty_request

**Request:** POST /api/v1/scrape/telegram with `{}`

**Expected:** HTTP 400, error message

**Validates:** Missing `channel` field triggers validation error

---

### TestHandler_StartScrape/returns_400_on_invalid_json

**Request:** POST /api/v1/scrape/telegram with `not json`

**Expected:** HTTP 400, "invalid json"

**Validates:** Malformed JSON rejected

---

### TestHandler_StartScrape/returns_400_on_negative_limit

**Request:** `{"channel": "@test", "limit": -1}`

**Expected:** HTTP 400, `ErrInvalidLimit`

**Validates:** Negative limit validation propagates to HTTP response

---

### TestHandler_StartScrape/returns_200_on_valid_request

**Request:** `{"channel": "@test_channel"}`

**Expected:**
- HTTP 200
- Response JSON with `scrape_id`, `status: "running"`, `started_at`, `target`

**Validates:** Successful scrape start returns proper response structure

---

### TestHandler_StartScrape/returns_409_when_already_running

**Setup:** MockScraper with 100ms delay (keeps job running)

**Steps:**
1. Start first job → expect 200
2. Start second job → expect 409 Conflict

**Validates:** Concurrent job prevention returns correct HTTP status

---

### TestHandler_StopScrape/returns_200_even_when_not_running

**Request:** DELETE /api/v1/scrape/current (no job running)

**Expected:** HTTP 200

**Validates:** Stop is idempotent at HTTP level

---

### TestHandler_StopScrape/stops_running_job

**Steps:**
1. Start job with POST /api/v1/scrape/telegram
2. Call DELETE /api/v1/scrape/current
3. Verify HTTP 200
4. Verify `manager.Current()` returns nil

**Validates:** Stop endpoint actually stops the job

---

### TestHandler_Status/returns_no_job_when_not_running

**Request:** GET /api/v1/scrape/status (idle)

**Expected:** HTTP 200, `{"status": "idle"}`

**Validates:** Status endpoint works when no job running

---

### TestHandler_ListForumTopics/returns_topics_with_correct_json_keys

**Setup:** MockScraper returns `[{ID: 1, Title: "General"}]`

**Request:** GET /api/v1/tools/telegram/topics?channel=@test

**Expected:**
- HTTP 200
- JSON array with `"id"` and `"title"` keys (lowercase)

**Validates:** JSON field naming matches frontend expectations (camelCase)

## Coverage Summary

| Endpoint | Status | Body | Validation |
|----------|--------|------|------------|
| GET /health | ✅ | ✅ | — |
| POST /api/v1/scrape/telegram | ✅ | ✅ | ✅ |
| DELETE /api/v1/scrape/current | ✅ | ✅ | — |
| GET /api/v1/scrape/status | ✅ | ✅ | — |
| GET /api/v1/tools/telegram/topics | ✅ | ✅ | ✅ |
