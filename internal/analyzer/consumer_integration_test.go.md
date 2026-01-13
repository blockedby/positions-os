# consumer_integration_test.go

Integration tests for NATS consumer — validates end-to-end message flow from NATS to job processing.

## Test Environment

**Prerequisites:**
- `INTEGRATION_TEST=1` environment variable set
- `NATS_URL` environment variable pointing to running NATS server
- 10-second timeout for entire test

**Test Setup:**
1. Connect to NATS using real client
2. Ensure `jobs` stream exists with subject `jobs.new`
3. Create mock `JobsRepository` with pre-seeded job data
4. Create mock `LLMClient` that returns structured JSON
5. Initialize `Processor` with mocks
6. Start `Consumer` subscribing to `jobs.new`

## Test Cases

### TestConsumer_Integration

**Scenario:** Publish job event → Consumer receives → Processor processes → Repository updated

**Steps:**
1. Seed mock repo with a RAW job (ID: `jobID`, RawContent: "Go Developer")
2. Start consumer listening on `jobs.new`
3. Publish `{job_id: jobID}` event to NATS
4. Poll repository every 100ms for updates
5. Verify `UpdatedData["title"]` equals "Go Developer"

**Expected Results:**
- Message successfully consumed from `jobs.new`
- `Processor.ProcessJob()` called with correct JobID
- Mock LLM returns `{"title": "Go Developer"}`
- Repository receives `UpdateStructuredData()` call
- Test completes within 10-second timeout

**Mock Behavior:**
| Mock | Behavior |
|------|----------|
| `MockJobsRepo` | Returns pre-seeded job by ID; stores updates in `UpdatedData` map |
| `MockLLMClient` | Returns hardcoded JSON `{"title": "Go Developer"}` |

**Failure Modes:**
- Timeout if message not processed within 10s
- Error if NATS connection fails
- Error if stream creation fails
- Error if publish fails
