# manager_test.go

Unit tests for scrape manager — validates job lifecycle, concurrency, and thread safety.

## Test Fixtures

### MockScraper
In-memory mock implementing `Scraper` interface.
| Field | Purpose |
|-------|---------|
| `Called` | Set to true when `Scrape()` is invoked |
| `Opts` | Stores `ScrapeOptions` passed to `Scrape()` |
| `Delay` | Optional delay before returning (for testing async behavior) |
| `TopicsToReturn` | Topics returned by `ListTopics()` |

## Test Cases

### TestScrapeManager_Start/starts_job_successfully

**Scenario:** Valid options → Job started → Scraper called in goroutine

**Setup:**
- Manager with `MockScraper`

**Steps:**
1. Call `Start()` with channel="test_channel", limit=100
2. Verify returned job has non-nil UUID
3. Verify job.Options.Channel equals "test_channel"
4. Wait 10ms for goroutine to execute
5. Verify `mockScraper.Called` is true
6. Verify scraper received correct channel name

**Validates:**
- Job created with unique ID
- Options stored correctly
- Scraper runs asynchronously in goroutine
- Options passed to scraper correctly

---

### TestScrapeManager_Start/returns_error_when_already_running

**Scenario:** First job starts → Second job rejected with `ErrAlreadyRunning`

**Steps:**
1. Start first job with channel="first"
2. Attempt to start second job with channel="second"
3. Verify error equals `ErrAlreadyRunning`

**Validates:**
- Only one job can run at a time
- Manager state prevents concurrent jobs

---

### TestScrapeManager_Stop/stops_running_job

**Scenario:** Job running → Stop() called → Current() returns nil

**Steps:**
1. Start a job
2. Verify `Current()` returns non-nil
3. Call `Stop()`
4. Wait 10ms for cleanup
5. Verify `Current()` returns nil

**Validates:**
- Stop() cancels running job
- Manager state cleared after stop
- Context cancellation propagates to scraper

---

### TestScrapeManager_Stop/safe_to_call_when_not_running

**Scenario:** Stop() called when no job running → No panic

**Steps:**
1. Create manager with no job
2. Call `Stop()` three times

**Validates:**
- Stop() is idempotent
- No panic on multiple stops when idle

---

### TestScrapeManager_Current/returns_nil_when_not_running

**Scenario:** No job started → Current() returns nil

**Validates:** Idle state returns nil

---

### TestScrapeManager_Current/returns_job_when_running

**Scenario:** Job started → Current() returns same job

**Steps:**
1. Start job, capture returned `job`
2. Call `Current()`
3. Verify current.ID equals job.ID

**Validates:** Current() returns the active job reference

---

### TestScrapeManager_ConcurrentAccess

**Scenario:** 100 goroutines calling Start/Current/Stop → No race conditions

**Setup:**
- Manager with `MockScraper`
- `sync.WaitGroup` for coordination

**Steps:**
1. Spawn 100 goroutines
2. Each calls: Start(), Current(), Stop()
3. Wait for all to complete

**Validates:**
- Thread-safe mutex protection
- No data races
- No deadlocks under concurrent access
