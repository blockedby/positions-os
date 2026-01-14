# Thread A: Backend Infrastructure

**Developer:** Developer A
**Focus:** Lower-level backend components (database, tracking, integrations)
**Estimated Duration:** ~60% of total work

## Overview

This thread covers foundational backend infrastructure that other components depend on. You will build:
1. Database access layer for job applications
2. Delivery status tracking with WebSocket events
3. Read receipt detection via Telegram updates
4. Email sender stub

---

## Dependencies

**You depend on:** No other threads (you're first!)
**Threads depending on you:** Thread B (Service & API)

**Critical Handoff Points:**
- After **Task 1.5** → Thread B can start Telegram Sender with Repository
- After **Task 2.4** → Thread B can integrate DeliveryTracker
- After **Task 2.5.6** → Thread B can wire up read detection

---

## Stage 1: Repository Layer

### Task 1.1: Create Applications Repository

**File:** `internal/repository/applications.go`

**Description:** Create the repository layer for job applications CRUD operations.

**Pseudo Code:**
```go
TYPE ApplicationsRepository STRUCT
    pool *pgxpool.Pool
    log  *logger.Logger
END

FUNCTION NewApplicationsRepository(pool, log) RETURN *ApplicationsRepository
    RETURN &ApplicationsRepository{pool, log}
END
```

**Acceptance Criteria:**
- [ ] File exists at `internal/repository/applications.go`
- [ ] Struct has pool and logger fields
- [ ] Constructor function compiles

---

### Task 1.2: Implement CRUD Read Operations

**File:** `internal/repository/applications.go`

**Description:** Implement `Create`, `GetByJobID`, `GetByID` methods.

**Pseudo Code:**
```go
FUNCTION Create(ctx, app) RETURN error
    sql := `
        INSERT INTO job_applications (id, job_id, delivery_channel, delivery_status, recipient)
        VALUES ($1, $2, $3, $4, $5)
    `
    _, err := pool.Exec(ctx, sql, app.ID, app.JobID, app.Channel, "PENDING", app.Recipient)
    RETURN err
END

FUNCTION GetByID(ctx, id) RETURN (*JobApplication, error)
    sql := `SELECT * FROM job_applications WHERE id = $1`
    row := pool.QueryRow(ctx, sql, id)
    RETURN scanApplication(row)
END

FUNCTION GetByJobID(ctx, jobID) RETURN ([]*JobApplication, error)
    sql := `SELECT * FROM job_applications WHERE job_id = $1 ORDER BY created_at DESC`
    rows := pool.Query(ctx, sql, jobID)
    RETURN scanAllApplications(rows)
END
```

**Acceptance Criteria:**
- [ ] All methods compile
- [ ] Integration test passes with test database
- [ ] Foreign key constraint validated (job_id must exist)

---

### Task 1.3: Implement Status Update Methods

**File:** `internal/repository/applications.go`

**Description:** Implement `UpdateDeliveryStatus`, `UpdateRecipient`, `MarkSent`.

**Pseudo Code:**
```go
FUNCTION UpdateDeliveryStatus(ctx, id, status) RETURN error
    sql := `UPDATE job_applications SET delivery_status = $1 WHERE id = $2`
    RETURN pool.Exec(ctx, sql, status, id)
END

FUNCTION UpdateRecipient(ctx, id, recipient) RETURN error
    sql := `UPDATE job_applications SET recipient = $1 WHERE id = $2`
    RETURN pool.Exec(ctx, sql, recipient, id)
END

FUNCTION MarkSent(ctx, id) RETURN error
    sql := `UPDATE job_applications SET delivery_status = 'SENT', sent_at = NOW() WHERE id = $1`
    RETURN pool.Exec(ctx, sql, id)
END
```

**Acceptance Criteria:**
- [ ] All methods compile
- [ ] Timestamps are set correctly
- [ ] Invalid UUIDs return proper errors

---

### Task 1.4: Implement ListPending

**File:** `internal/repository/applications.go`

**Description:** Implement `ListPending` for queue processing.

**Pseudo Code:**
```go
FUNCTION ListPending(ctx, limit) RETURN ([]*JobApplication, error)
    sql := `
        SELECT * FROM job_applications
        WHERE delivery_status = 'PENDING'
        ORDER BY created_at ASC
        LIMIT $1
    `
    rows := pool.Query(ctx, sql, limit)
    RETURN scanAllApplications(rows)
END
```

**Acceptance Criteria:**
- [ ] Method compiles
- [ ] Default limit = 10 if not specified
- [ ] Returns oldest applications first (ASC)

---

### Task 1.5: Write Unit Tests

**File:** `internal/repository/applications_test.go`

**Description:** Add comprehensive unit tests for repository.

**Test Cases:**
- [ ] TestCreate_Success
- [ ] TestCreate_InvalidJobID
- [ ] TestGetByID_Found
- [ ] TestGetByID_NotFound
- [ ] TestGetByJobID_Empty
- [ ] TestGetByJobID_Multiple
- [ ] TestUpdateDeliveryStatus_Valid
- [ ] TestMarkSent_TimestampSet
- [ ] TestListPending_Ordering

**Acceptance Criteria:**
- [ ] All tests pass
- [ ] Coverage > 80% for repository layer

**HANDOFF TO THREAD B:** Thread B can now start Telegram Sender using the Repository!

---

## Stage 2: Delivery Tracker

### Task 2.1: Create Tracker File

**File:** `internal/dispatcher/tracker.go`

**Description:** Create tracker with struct and constructor.

**Pseudo Code:**
```go
TYPE DeliveryTracker STRUCT
    repo *ApplicationsRepository
    hub  *web.Hub
    log  *logger.Logger
END

FUNCTION NewDeliveryTracker(repo, hub, log) RETURN *DeliveryTracker
    RETURN &DeliveryTracker{repo, hub, log}
END
```

**Acceptance Criteria:**
- [ ] File exists
- [ ] Struct has required dependencies

---

### Task 2.2: Implement Status Tracking Methods

**File:** `internal/dispatcher/tracker.go`

**Description:** Implement `TrackStart`, `TrackSuccess`, `TrackFailure`.

**Pseudo Code:**
```go
FUNCTION TrackStart(ctx, appID) RETURN error
    // PENDING → SENDING
    IF !ValidateTransition("PENDING", "SENDING") RETURN error
    err := repo.UpdateDeliveryStatus(ctx, appID, "SENDING")
    IF err != nil RETURN err
    BroadcastStatusChanged(appID, "PENDING", "SENDING")
    log.Info("status_changed", "app_id", appID, "to", "SENDING")
    RETURN nil
END

FUNCTION TrackSuccess(ctx, appID) RETURN error
    // SENDING → SENT
    err := repo.MarkSent(ctx, appID)
    IF err != nil RETURN err
    BroadcastStatusChanged(appID, "SENDING", "SENT")
    log.Info("status_changed", "app_id", appID, "to", "SENT")
    RETURN nil
END

FUNCTION TrackFailure(ctx, appID, err) RETURN error
    // any → FAILED
    repo.UpdateDeliveryStatus(ctx, appID, "FAILED")
    BroadcastFailed(appID, err)
    log.Error("send_failed", "app_id", appID, "error", err)
    RETURN nil
END
```

**Acceptance Criteria:**
- [ ] All methods compile
- [ ] Status transitions are validated
- [ ] WebSocket events are emitted

---

### Task 2.3: Implement WebSocket Broadcasting

**File:** `internal/dispatcher/tracker.go`

**Description:** Implement event broadcasting methods.

**Pseudo Code:**
```go
FUNCTION BroadcastStatusChanged(appID, from, to)
    event := Map{
        "type": "dispatcher.status_changed",
        "application_id": appID,
        "previous_status": from,
        "current_status": to,
        "updated_at": NOW().Format(time.RFC3339),
    }
    hub.Broadcast(event)
END

FUNCTION BroadcastProgress(appID, step, progress)
    event := Map{
        "type": "dispatcher.progress",
        "application_id": appID,
        "step": step,
        "progress": progress,
    }
    hub.Broadcast(event)
END

FUNCTION BroadcastFailed(appID, err)
    event := Map{
        "type": "dispatcher.failed",
        "application_id": appID,
        "error": err.Error(),
    }
    hub.Broadcast(event)
END
```

**Acceptance Criteria:**
- [ ] Events match JSON schema
- [ ] Hub receives events

---

### Task 2.4: Write Unit Tests

**File:** `internal/dispatcher/tracker_test.go`

**Description:** Test tracker functionality.

**Test Cases:**
- [ ] TestTrackStart_ValidTransition
- [ ] TestTrackStart_InvalidTransition
- [ ] TestTrackSuccess_AfterStart
- [ ] TestTrackFailure_StoresError
- [ ] TestBroadcast_EventsEmitted

**Acceptance Criteria:**
- [ ] All tests pass
- [ ] Mocked hub and repo

**HANDOFF TO THREAD B:** Thread B can now integrate DeliveryTracker into TelegramSender!

---

## Stage 2.5: Read Receipt Tracker

### Task 2.5.1: Create Read Tracker File

**File:** `internal/dispatcher/read_tracker.go`

**Description:** Create read tracker for Telegram message ID to application ID mapping.

**Pseudo Code:**
```go
TYPE ReadTracker STRUCT
    repo         *ApplicationsRepository
    tracker      *DeliveryTracker
    messageToApp MAP[int64]uuid.UUID  // Telegram msg ID → App ID
    mu           sync.RWMutex
    log          *logger.Logger
END

FUNCTION NewReadTracker(repo, tracker, log) RETURN *ReadTracker
    RETURN &ReadTracker{
        repo: repo,
        tracker: tracker,
        messageToApp: MAKE(map[int64]uuid.UUID),
        log: log,
    }
END
```

**Acceptance Criteria:**
- [ ] File exists
- [ ] Struct has required dependencies
- [ ] Message map initialized empty

---

### Task 2.5.2: Implement Message Registration

**File:** `internal/dispatcher/read_tracker.go`

**Description:** Implement `RegisterSentMessage` for mapping sent messages.

**Pseudo Code:**
```go
FUNCTION RegisterSentMessage(msgID int64, appID uuid.UUID)
    mu.Lock()
    DEFER mu.Unlock()
    messageToApp[msgID] = appID
    log.Debug("registered", "msg_id", msgID, "app_id", appID)
END
```

**Acceptance Criteria:**
- [ ] Method compiles
- [ ] Thread-safe access (mutex used)

---

### Task 2.5.3: Implement Read Update Handler

**File:** `internal/dispatcher/read_tracker.go`

**Description:** Implement `OnMessageRead` for handling `updateReadHistoryOutbox`.

**Pseudo Code:**
```go
FUNCTION OnMessageRead(ctx, peerUserID, maxMsgID) RETURN error
    mu.RLock()
    appID, found := messageToApp[maxMsgID]
    mu.RUnlock()

    IF !found {
        // Not our message, ignore
        RETURN nil
    }

    // Update to READ status
    err := tracker.UpdateStatus(ctx, appID, "READ")
    IF err != nil {
        log.Error("status_update_failed", "app_id", appID, "error", err)
        RETURN err
    }

    // Clean up mapping
    mu.Lock()
    DELETE(messageToApp, maxMsgID)
    mu.Unlock()

    log.Info("auto_read_detected", "app_id", appID, "peer_user_id", peerUserID)
    RETURN nil
END
```

**Acceptance Criteria:**
- [ ] Status updated to READ
- [ ] Mapping removed after processing
- [ ] Event broadcasted

---

### Task 2.5.4: Integrate with Telegram Manager

**File:** `internal/telegram/manager.go`

**Description:** Hook read tracker into Telegram update handler.

**Pseudo Code:**
```go
// Add field to Manager struct
TYPE Manager STRUCT
    // ... existing fields
    readTracker *ReadTracker  // New field
END

// Add to dispatcher initialization
FUNCTION (m *Manager) SetReadTracker(rt *ReadTracker)
    m.mu.Lock()
    DEFER m.mu.Unlock()
    m.readTracker = rt
END

// Handle update in existing dispatcher
FUNCTION (m *Manager) handleUpdate(upd tg.UpdateClass) error
    SWITCH u := upd.(type) {
    CASE *tg.UpdateReadHistoryOutbox:
        IF peer, ok := u.Peer.(*tg.PeerUser); ok {
            IF m.readTracker != nil {
                RETURN m.readTracker.OnMessageRead(context.Background(), peer.UserID, u.MaxID)
            }
        }
    CASE *tg.UpdateReadMessagesContents:
        // Handle content reads if tracking
    }
    RETURN nil
END
```

**Acceptance Criteria:**
- [ ] Read tracker called on read updates
- [ ] Other updates not affected
- [ ] SetReadTracker method available

---

### Task 2.5.5: Update TelegramSender to Register Messages

**File:** `internal/dispatcher/telegram_sender.go`

**Description:** Register sent messages in read tracker after sending.

**Note:** This file will be created by Thread B (Task 3.1). You only need to add the registration call.

**Pseudo Code:**
```go
// In SendApplication method, after successful send:
FUNCTION SendApplication(ctx, appID, recipient) RETURN error
    // ... existing send logic ...

    // After successful send, capture message ID
    result, err := client.API().MessagesSendMedia(ctx, req)
    IF err != nil {
        tracker.TrackFailure(ctx, appID, err)
        RETURN err
    }

    // Register message ID for read detection
    IF result.Updates != nil {
        FOR EACH update IN result.Updates {
            IF msg, ok := update.(*tg.UpdateMessageID); ok {
                readTracker.RegisterSentMessage(msg.ID, appID)
            }
        }
    }

    tracker.TrackSuccess(ctx, appID)
    RETURN nil
END
```

**Acceptance Criteria:**
- [ ] Message ID registered after successful send
- [ ] Registration happens before TrackSuccess
- [ ] Failed send does not register

---

### Task 2.5.6: Write Unit Tests

**File:** `internal/dispatcher/read_tracker_test.go`

**Description:** Test read tracker functionality.

**Test Cases:**
- [ ] TestRegisterSentMessage_StoresMapping
- [ ] TestOnMessageRead_UpdatesStatus
- [ ] TestOnMessageRead_NotFoundReturnsNil
- [ ] TestOnMessageRead_CleansUpMapping
- [ ] TestConcurrentAccess_ThreadSafe

**Acceptance Criteria:**
- [ ] All tests pass
- [ ] Mocked tracker and repo

**HANDOFF TO THREAD B:** Thread B can now complete read detection integration!

---

## Bonus Task: Email Sender Stub

### Task E.1: Create Email Sender Stub

**File:** `internal/dispatcher/email_sender.go`

**Description:** Create stub that returns "not implemented" error.

**Pseudo Code:**
```go
TYPE EmailSender STRUCT{}

FUNCTION NewEmailSender() *EmailSender
    RETURN &EmailSender{}
END

FUNCTION (s *EmailSender) SendApplication(ctx, appID, recipient) error
    RETURN fmt.Errorf("email sender not implemented: use TG_DM channel instead")
END
```

**Acceptance Criteria:**
- [ ] File exists
- [ ] Returns clear error message
- [ ] Future implementation points documented

---

## Completion Checklist

Thread A is complete when:

- [ ] `internal/repository/applications.go` with full CRUD operations
- [ ] `internal/dispatcher/tracker.go` with status tracking and WebSocket events
- [ ] `internal/dispatcher/read_tracker.go` with Telegram read detection
- [ ] `internal/dispatcher/email_sender.go` stub file created
- [ ] All unit tests passing with >80% coverage
- [ ] Integration with Telegram manager complete
- [ ] **HANDOFF:** Notify Thread B that Repository and Tracker are ready

---

## WebSocket Events You Emit

Thread B's UI will listen for these events. Ensure they match exactly:

```json
// Status change
{"type": "dispatcher.status_changed", "application_id": "uuid", "previous_status": "PENDING", "current_status": "SENDING", "updated_at": "2025-01-14T10:00:00Z"}

// Progress update
{"type": "dispatcher.progress", "application_id": "uuid", "step": "uploading", "progress": 50}

// Failure
{"type": "dispatcher.failed", "application_id": "uuid", "error": "user not found"}
```

---

## Coordination Notes

1. **Start first:** Begin with Stage 1 (Repository) immediately
2. **Handoff at Task 1.5:** Let Thread B know they can start Telegram Sender
3. **Handoff at Task 2.4:** Let Thread B know DeliveryTracker is ready
4. **Coordinate at Task 2.5.5:** You'll need to add code to Thread B's telegram_sender.go
5. **Test together:** Do integration testing once both threads are complete
