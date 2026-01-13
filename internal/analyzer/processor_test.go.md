# processor_test.go

Unit tests for job processor — validates LLM job analysis logic in isolation.

## Test Fixtures

### MockJobsRepo
In-memory mock implementing `JobsRepository` interface.
| Field | Purpose |
|-------|---------|
| `Jobs` | Pre-seeded job lookup map |
| `UpdatedData` | Stores data passed to `UpdateStructuredData()` |
| `Err` | Optional error to return from methods |
| `mu` | Mutex for thread safety |

### MockLLMClient
Configurable mock implementing `LLMClient` interface.
| Field | Purpose |
|-------|---------|
| `ExtractFunc` | Function called on `ExtractJobData()`; returns JSON string or error |

## Test Cases

### TestProcessor_ProcessJob/Success

**Scenario:** Valid job → LLM extracts data → Repository updated

**Setup:**
- Job ID with `RawContent: "Go Developer"`
- LLM returns `{"title": "Go Developer"}`
- User prompt template: `"content: {{RAW_CONTENT}}"`

**Steps:**
1. Call `ProcessJob(ctx, jobID)`
2. Verify no error returned
3. Verify repo's `UpdatedData["title"]` equals "Go Developer"

**Validates:**
- `GetByID()` called to fetch job
- User prompt contains raw content
- LLM response parsed as JSON
- `UpdateStructuredData()` called with parsed data

---

### TestProcessor_ProcessJob/InvalidJSON

**Scenario:** LLM returns invalid JSON → Error returned

**Setup:**
- LLM returns `INVALID JSON` (not valid JSON)

**Steps:**
1. Call `ProcessJob(ctx, jobID)`
2. Verify error is non-nil

**Expected:** Error message contains "invalid json received from llm"

**Validates:**
- JSON parsing errors are propagated
- Processor fails gracefully on bad LLM output

---

### TestProcessor_ProcessJob/MarkdownCleanup

**Scenario:** LLM returns JSON wrapped in markdown code blocks → Cleaned successfully

**Setup:**
- LLM returns `"```json\n{\"key\": \"val\"}\n```"`

**Steps:**
1. Call `ProcessJob(ctx, jobID)`
2. Verify no error
3. Verify `UpdatedData["key"]` equals "val"

**Expected:** `cleanJSON()` strips markdown wrappers before parsing

**Validates:**
- `` ```json `` prefix is removed
- `` ``` `` suffix is removed
- Whitespace trimmed
- Remaining JSON parses correctly

---

## Coverage Summary

| Test | Covers |
|------|--------|
| Success | Happy path, prompt building, repo update |
| InvalidJSON | JSON validation error handling |
| MarkdownCleanup | LLM output sanitization (`cleanJSON()`) |
