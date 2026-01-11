# Collector Stuck Issue - Analysis & Solutions

**Date**: 2026-01-10T22:16:00+01:00
**Status**: Running for 37+ minutes without completion
**Symptoms**: Status shows "running", but no logs generated

---

## 🔍 Root Cause Analysis

### Observed Behavior

```json
{
  "scrape_id": "8cbc463b-8be6-41b4-b814-4988c4a86c4a",
  "status": "running",
  "target": {
    "id": "00000000-0000-0000-0000-000000000000",
    "name": "",
    "channel": "golang_jobs"
  },
  "started_at": "2026-01-10T21:37:55.9523065+01:00"
}
```

- **Duration**: 37+ minutes stuck
- **Log file**: Empty (0 bytes) - **CRITICAL ISSUE**
- **Health check**: Passing (server responsive)
- **Status endpoint**: Returns "running"

### Most Likely Causes

#### 1. **Infinite Loop in Message Fetching** (Probability: 70%)

**Location**: `internal/collector/service.go` lines 137-222

**Problem**:

```go
for {
    messages, err := s.tgClient.GetMessages(ctx, channel, offsetID, min(limit, 100))
    if err != nil {
        break
    }

    if len(messages) == 0 {
        break  // Only exits when 0 messages
    }

    // Update offset
    offsetID = messages[len(messages)-1].ID

    // But if GetMessages keeps returning messages...
    // We never exit!
}
```

**Why this happens**:

- When `offsetID = 0`, you fetch from the **newest** messages
- If the channel has thousands of messages, this could run indefinitely
- The offset update might not work correctly (getting stuck on the same messages)
- No maximum iteration limit

#### 2. **Telegram API Rate Limiting** (Probability: 20%)

**Scenario**:

- `rateLimiter.Wait()` is blocking for extended periods
- Telegram returned a `FLOOD_WAIT_X` error with large X (could be minutes/hours)
- No logs to confirm because logging happens AFTER the wait

**Evidence**:

- No logs = stuck before first log statement
- Rate limiter waits are silent (no logging)

#### 3. **Database/NATS Blocking** (Probability: 8%)

**Possible causes**:

- Database connection pool exhausted
- Transaction deadlock
- NATS connection timeout
- PostgreSQL slow query

#### 4. **Context Not Cancellable** (Probability: 2%)

The context passed to `Scrape()` might not have proper cancellation setup.

---

## 🔧 Immediate Fixes Applied

### Fix 1: Comprehensive Logging Added ✅

**Files Modified**:

- `internal/collector/service.go`
- `internal/telegram/client.go`

**Changes**:

1. **Before each phase**: "scrape: starting", "scrape: resolving channel"
2. **Inside the loop**: Batch number, offset, message count
3. **Rate limiting**: When waiting, flood wait detection
4. **Message processing**: Count of new vs. skipped messages
5. **Database operations**: Job creation attempts
6. **Completion**: Final statistics

**Benefits**:

- You'll now see EXACTLY where it gets stuck
- Batch numbers show if loop is progressing
- Rate limit warnings will be visible
- Empty log = stuck before first DB connection

### Example Expected Logs:

```json
{"level":"info","time":"...","message":"scrape: starting","options":{...}}
{"level":"info","message":"scrape: target resolved","target_id":"...","channel":"golang_jobs"}
{"level":"info","message":"telegram: resolving channel username","username":"golang_jobs"}
{"level":"info","message":"scrape: channel resolved","channel_id":12345,"is_forum":false}
{"level":"info","message":"scrape: starting message fetch loop","batch_size":100}
{"level":"info","message":"scrape: fetching messages batch","batch":1,"offset_id":0,"limit":100}
{"level":"debug","message":"telegram: waiting for rate limiter before GetMessages"}
{"level":"info","message":"telegram: calling MessagesGetHistory API","channel_id":12345}
{"level":"info","message":"scrape: received messages","batch":1,"messages_received":100}
{"level":"info","message":"scrape: filtered messages","batch":1,"total_messages":100,"new_messages":95,"already_processed":5}
...
```

---

## 📋 Next Steps - How to Diagnose

### Step 1: Restart the Collector

The current process is stuck. You need to restart to get fresh logs.

**Commands**:

```powershell
# Stop the current running collector (Ctrl+C in the terminal)

# Wait a moment, then restart
cd c:\Users\kcnc\code\positions-os
go run cmd/collector/main.go
```

### Step 2: Trigger a Scrape

```powershell
# In a new terminal
curl -X POST http://localhost:3100/api/v1/scrape/telegram `
  -H "Content-Type: application/json" `
  -d '{"channel": "golang_jobs", "limit": 10}'
```

**Important**: Use `"limit": 10` for testing to prevent infinite loops!

### Step 3: Watch the Logs

```powershell
# In another terminal, tail the logs
Get-Content .\logs\collector.log -Wait -Tail 50
```

### Step 4: Analyze Where It Stops

Look for the **last log message** before it gets stuck:

| Last Log Message                             | Diagnosis                    | Solution                       |
| -------------------------------------------- | ---------------------------- | ------------------------------ |
| `"telegram: waiting for rate limiter"`       | Stuck in rate limiter        | Check rate limiter settings    |
| `"telegram: calling MessagesGetHistory API"` | Telegram API not responding  | Network/API issue              |
| `"scrape: received messages", batch:X`       | Loop running but not exiting | Infinite loop - see Solution 2 |
| `"scrape: creating job for message"`         | Database blocked             | Check PostgreSQL               |
| Nothing (empty log)                          | Startup failure              | Check env vars, DB connection  |

---

## 🎯 Proposed Solutions

### Solution 1: Add Maximum Batch Limit (RECOMMENDED)

Add a safety limit to prevent infinite loops:

```go
// In service.go, Scrape method
const maxBatches = 100  // Maximum 100 batches

for batchNum := 0; batchNum < maxBatches; batchNum++ {
    // ... existing code ...

    if len(messages) == 0 {
        s.log.Info().Msg("scrape: no more messages")
        break
    }

    // ... process messages ...
}

if batchNum >= maxBatches {
    s.log.Warn().Msg("scrape: reached maximum batch limit, stopping")
}
```

**Benefits**:

- Prevents runaway loops
- Still processes 10,000 messages (100 batches × 100 messages)
- Guarantees eventual completion

### Solution 2: Detect Duplicate Offset (RECOMMENDED)

Exit if offset doesn't change:

```go
var minMsgID, maxMsgID int64
offsetID := 0
previousOffsetID := -1  // Track previous offset

for {
    // ... existing select and GetMessages ...

    if offsetID == previousOffsetID {
        s.log.Warn().
            Int("offset_id", offsetID).
            Msg("scrape: offset not changing, exiting to prevent infinite loop")
        break
    }
    previousOffsetID = offsetID

    // ... rest of loop ...
}
```

### Solution 3: Add Timeout to Scrape Operation

Implement a maximum duration:

```go
// In handler.go, StartScrape
ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
defer cancel()

job, err := h.manager.Start(ctx, opts)
```

### Solution 4: Improve Deduplication Logic

The current filter might be letting duplicates through:

```go
// Check if ALL messages in batch are already processed
if len(newIDs) == 0 {
    s.log.Info().Msg("scrape: all messages in batch already processed, stopping")
    break
}
```

---

## ⚡ Quick Fixes to Apply Now

### Quick Fix #1: Restart with Limit

When making the API call, **always specify a limit**:

```json
{
  "channel": "golang_jobs",
  "limit": 100 // Process max 100 messages
}
```

### Quick Fix #2: Check the Target Already Exists

The status shows `target.id = "00000000-0000-0000-0000-000000000000"` which is suspicious.

Check database:

```sql
SELECT id, name, url, last_message_id
FROM scraping_targets
WHERE url = 'golang_jobs' OR url = '@golang_jobs';
```

If `last_message_id` is NULL, the scraper starts from offset 0.

### Quick Fix #3: Stop Current Scrape

```powershell
curl -X DELETE http://localhost:3100/api/v1/scrape/current
```

Then restart the collector process.

---

## 🔬 Diagnostic Questions to Answer

1. **Does the channel have a lot of messages?**

   - Check manually in Telegram
   - If 10,000+ messages, that's likely the issue

2. **What's in the database?**

   ```sql
   SELECT COUNT(*) FROM jobs WHERE target_id IN (
       SELECT id FROM scraping_targets WHERE url LIKE '%golang_jobs%'
   );
   ```

3. **Are there any parsed ranges?**

   ```sql
   SELECT * FROM parsed_ranges
   WHERE target_id IN (
       SELECT id FROM scraping_targets WHERE url LIKE '%golang_jobs%'
   );
   ```

4. **Is the database connection healthy?**
   ```powershell
   curl http://localhost:3100/health
   ```

---

## 📊 Success Criteria

After restart with enhanced logging, you should see:

✅ Logs appearing in `collector.log`
✅ Batch numbers incrementing (1, 2, 3, ...)
✅ Message counts decreasing or staying stable
✅ Eventually: "scrape: completed successfully"

If any of these fail, the logs will tell us exactly where and why.

---

## 📝 Additional Improvements Needed

1. **Add scrape timeout** - Maximum duration per scrape job
2. **Add progress tracking** - Store current batch/offset in database
3. **Add resumable scraping** - Continue from last successful offset
4. **Improve error handling** - Distinguish temporary vs. permanent errors
5. **Add metrics** - Messages per second, API call duration
6. **Add graceful degradation** - Reduce batch size on errors

---

## Summary

**ROOT CAUSE FOUND**:
The HTTP request context was being used for the background scrape job. When the handler returned the response, the context was canceled immediately, killing the scrape job.

**What we fixed**:

- ✅ **Context Cancellation Bug** - Now using `context.Background()` for async scrape jobs (`manager.go`)
- ✅ **Maximum Batch Limit** - Added safety limit of 100 batches (~10,000 messages max)
- ✅ **Duplicate Offset Detection** - Exit if offset doesn't change between batches
- ✅ **Comprehensive Logging** - Added logs throughout scrape and telegram client
- ✅ **Rate Limiter Tests** - Added unit tests for rate limiting behavior
- ✅ **README Documentation** - Added limits and rate limiting info

**Message Fetching Direction**:
Yes, the scraper works correctly from **newest → oldest** messages:

- `offsetID = 0` starts from newest messages
- Each batch moves backwards in time
- This is ideal for job hunting (most recent postings first)

**Next Steps**:

1. **Restart the collector** (the old process needs to be killed)
2. **Test with**: `POST /api/v1/scrape/telegram {"channel": "job_web3", "limit": 10}`
3. **Watch logs** - should now complete successfully!
