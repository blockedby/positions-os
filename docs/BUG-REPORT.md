# Bug Report: Data Collection Issues

**Date:** 2026-01-16
**Database:** 197 jobs scraped from @stablegram

---

## Critical Issues

### 1. Analyzer Not Working - LLM API Key Missing

**Severity:** CRITICAL
**Impact:** 0 jobs analyzed, all 196 stuck in RAW status

**Error:**
```
llm extraction: llm completion: error, status code: 401,
status: 401 Unauthorized, message: Authorization Token Missing
```

**Cause:** `LLM_API_KEY` environment variable not set in analyzer container.

**Current env:**
```
LLM_BASE_URL=https://api.z.ai/api/coding/paas/v4
LLM_MODEL=glm-4.6
LLM_TEMPERATURE=0.1
LLM_API_KEY=  # MISSING!
```

**Fix:** Add `LLM_API_KEY` to `.env` or `docker-compose.yml`

---

## Data Quality Issues

### 2. `source_url` Always NULL

**Severity:** HIGH
**Affected:** 197/197 jobs (100%)

| Field | Expected | Actual |
|-------|----------|--------|
| `source_url` | `https://t.me/stablegram/1244` | `NULL` |

**Root cause:** Collector doesn't build Telegram message URL.

**Location:** `internal/collector/scraper.go` - job creation logic

**Fix:** Generate URL as `https://t.me/{channel}/{message_id}`

---

### 3. `tg_topic_id` Always NULL

**Severity:** LOW (for non-forum channels)
**Affected:** 197/197 jobs

For @stablegram this is expected - it's a channel, not a forum.
But for forum targets this field should be populated.

**Verify:** Check if forums properly populate `tg_topic_id`

---

### 4. `structured_data` Empty for INTERESTED Jobs

**Severity:** MEDIUM
**Affected:** 1 INTERESTED job has `structured_data = {}`

A job was marked INTERESTED manually but never analyzed.
Status changed without LLM processing.

**Expected flow:** RAW → ANALYZED → INTERESTED
**Actual:** RAW → INTERESTED (skipped analysis)

---

## Summary Table

| Field | Status | Count | Notes |
|-------|--------|-------|-------|
| `raw_content` | OK | 197/197 | |
| `content_hash` | OK | 197/197 | |
| `tg_message_id` | OK | 197/197 | |
| `external_id` | OK | 197/197 | |
| `source_date` | OK | 197/197 | |
| `source_url` | MISSING | 0/197 | Not generated |
| `tg_topic_id` | NULL | 0/197 | Expected for channels |
| `structured_data` | EMPTY | 0/197 | LLM not working |
| `status` | STUCK | 196 RAW | Needs LLM |

---

## Action Items

1. **[CRITICAL]** Set `LLM_API_KEY` in environment
2. **[HIGH]** Fix collector to generate `source_url`
3. **[LOW]** Verify forum scraping populates `tg_topic_id`
4. **[LOW]** Consider adding validation - can't change status to INTERESTED if not ANALYZED

---

## Database Stats

```sql
-- Jobs by status
RAW:        196
INTERESTED:   1

-- Scraping targets
@stablegram: active, last_scraped: 2026-01-16 21:25:11
             messages: 1040-1244 (205 total scraped)

-- Parsed ranges
min_msg_id: 1040
max_msg_id: 1244
```
