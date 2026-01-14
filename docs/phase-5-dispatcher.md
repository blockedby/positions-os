# Phase 5: Dispatcher — Implementation Plan

## Overview

**Dispatcher** is the service responsible for automatically sending tailored resumes and cover letters to recruiters via Telegram DM and Email.

### Goal

Automate the job application process by sending prepared documents (generated in Phase 4: Brain) to recruiters through multiple channels.

### Current State (Pre-Phase 5)

- ✅ Phase 4: Brain generates tailored resumes and cover letters
- ✅ `job_applications` table exists with delivery tracking columns
- ✅ Telegram client (`internal/telegram/`) supports MTProto
- ✅ WebSocket hub for real-time events
- ✅ NATS infrastructure for async processing

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Web UI                                        │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │  Job Detail → "Send to Telegram" / "Send Email" buttons              │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────┬──────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            REST API                                       │
│  POST /api/v1/applications/{id}/send                                      │
│  Body: { "channel": "TG_DM" | "EMAIL", "recipient": "@username" }        │
└───────────────────────────────────────┬──────────────────────────────────────┘
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Dispatcher Service                                  │
│  ┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐  │
│  │   TG Sender         │  │   Email Sender      │  │  Delivery Tracker   │  │
│  │   (MTProto)         │  │   (SMTP)            │  │  (DB + Events)      │  │
│  └──────────┬──────────┘  └──────────┬──────────┘  └──────────┬──────────┘  │
└─────────────┼───────────────────────┼───────────────────────┼───────────────┘
              │                       │                       │
              ▼                       ▼                       ▼
      ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
      │   Telegram    │       │   SMTP Server │       │   PostgreSQL  │
      │   MTProto     │       │               │       │ job_applic... │
      └───────────────┘       └───────────────┘       └───────────────┘
```

---

## Service Status Monitoring

### Overview

The Dispatcher service exposes its health and operational status through both logs and a REST endpoint. The UI displays real-time status indicators for all dispatcher components.

### Status Indicators

| Component    | Status | Description |
|--------------|--------|-------------|
| Telegram     | READY/UNAUTHORIZED/OFFLINE | Telegram client connectivity |
| Dispatcher   | ENABLED/DISABLED | Master switch for dispatcher |
| Queue        | IDLE/PROCESSING | Background queue status |

### REST Endpoint

```
GET /api/v1/dispatcher/status
→ Returns:
{
  "dispatcher_enabled": true,
  "telegram": {
    "status": "READY",
    "connected_at": "2025-01-14T10:00:00Z",
    "user_id": 12345678
  },
  "queue": {
    "pending": 0,
    "processing": 0,
    "failed_today": 0
  },
  "rate_limit": {
    "available": 10,
    "reset_at": "2025-01-14T10:10:00Z"
  }
}
```

### Logging Strategy

All dispatcher operations emit structured logs:

```
# Service startup
{"level":"info","service":"dispatcher","event":"started","telegram":"READY"}

# Send operation
{"level":"info","service":"dispatcher","event":"send.started","app_id":"uuid","recipient":"@user","channel":"TG_DM"}
{"level":"info","service":"dispatcher","event":"uploading","app_id":"uuid","progress":50}
{"level":"info","service":"dispatcher","event":"send.completed","app_id":"uuid","duration_ms":2500}

# Errors
{"level":"error","service":"dispatcher","event":"send.failed","app_id":"uuid","error":"user not found","retryable":false}
```

### UI Integration

The settings page displays dispatcher status:

```
┌─────────────────────────────────────────────┐
│  Dispatcher Status                          │
│  ┌─────────────────────────────────────┐   │
│  │  Telegram:    ● READY               │   │
│  │  Queue:       ● IDLE (0 pending)    │   │
│  │  Sent today:  5                     │   │
│  │  Failed:      0                     │   │
│  └─────────────────────────────────────┘   │
└─────────────────────────────────────────────┘
```

---

## Database Schema

### Existing Tables (Re-use)

The `job_applications` table already has everything we need:

```sql
CREATE TABLE job_applications (
    id                      UUID PRIMARY KEY,
    job_id                  UUID NOT NULL REFERENCES jobs(id),

    -- generated content (from Brain)
    tailored_resume_md      TEXT,
    cover_letter_md         TEXT,

    -- generated files (from Brain)
    resume_pdf_path         VARCHAR(512),
    cover_letter_pdf_path   VARCHAR(512),

    -- delivery tracking
    delivery_channel        delivery_channel,  -- TG_DM, EMAIL, HH_RESPONSE
    delivery_status         delivery_status,   -- PENDING, SENT, DELIVERED, READ, FAILED
    recipient               VARCHAR(255),      -- @username or email

    -- timestamps
    sent_at                 TIMESTAMPTZ,
    delivered_at            TIMESTAMPTZ,
    read_at                 TIMESTAMPTZ,
    response_received_at    TIMESTAMPTZ,

    recruiter_response      TEXT
);
```

### New Repository Methods Needed

```go
// internal/repository/applications.go

type ApplicationsRepository struct {
    pool *pgxpool.Pool
}

// Create creates a new job application record
func (r *ApplicationsRepository) Create(ctx context.Context, app *JobApplication) error

// GetByJobID returns applications for a job
func (r *ApplicationsRepository) GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*JobApplication, error)

// GetByID returns a single application
func (r *ApplicationsRepository) GetByID(ctx context.Context, id uuid.UUID) (*JobApplication, error)

// UpdateDeliveryStatus updates delivery status
func (r *ApplicationsRepository) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status DeliveryStatus) error

// UpdateRecipient updates recipient info
func (r *ApplicationsRepository) UpdateRecipient(ctx context.Context, id uuid.UUID, recipient string) error

// MarkSent marks application as sent
func (r *ApplicationsRepository) MarkSent(ctx context.Context, id uuid.UUID) error

// ListPending returns applications pending delivery
func (r *ApplicationsRepository) ListPending(ctx context.Context, limit int) ([]*JobApplication, error)
```

---

## Delivery Status Verification Flow

### Overview

Telegram **DOES** provide read receipts for direct messages via MTProto updates. Email delivery status requires external services. This flow describes how we track status changes using automatic detection where available.

### Telegram Read Receipt Updates (MTProto)

Telegram sends real-time updates when your messages are read:

| Update Type | Use Case | MTProto Update |
|-------------|----------|----------------|
| Direct Messages (PM) | One-on-one conversations | `updateReadHistoryOutbox` |
| Multiple Messages Read | PM content reads | `updateReadMessagesContents` |
| Channels | Channel message reads | `updateReadChannelOutbox` |

**Key:** `updateReadHistoryOutbox` provides read confirmations for direct messages sent to other users. This is a standard MTProto feature available in 2025 via gotd/td and gotgproto.

### Telegram DM Status Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          TELEGRAM DM STATUS FLOW                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PENDING ──────────────────────────────────────────────────────────────►   │
│    │                                                                        │
│    │  Send started                                                         │
│    ▼                                                                        │
│  SENDING ──────────────────────────────────────────────────────────────►   │
│    │                                                                        │
│    │  Upload + Send successful (no explicit delivery receipt, but send     │
│    │  success = delivered to Telegram server)                              │
│    ▼                                                                        │
│  SENT ────────────────────────────────────────────────────────────────►   │
│    │                                                                        │
│    │  [AUTOMATIC READ DETECTION via updateReadHistoryOutbox]               │
│    │  Telegram client receives update when recipient reads message         │
│    │  Update contains: peer_user_id, max_id, pts                          │
│    │                                                                        │
│    ▼                                                                        │
│  READ (Automatic Detection) ───────────────────────────────────────────►   │
│    │                                                                        │
│    │  User records recruiter response (optional)                           │
│    ▼                                                                        │
│  RESPONDED                                                                  │
│    │                                                                        │
│    │  [Manual override available]                                          │
│    ▼                                                                        │
│  User can manually mark: DELIVERED, READ, RESPONDED                        │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Implementing Read Receipt Detection

The Telegram client already receives `updateReadHistoryOutbox` updates. We need to:

1. **Hook into existing update handler** in `internal/telegram/manager.go`
2. **Map message IDs to application IDs** when sending
3. **Update status automatically** when read update received

```go
// internal/dispatcher/read_tracker.go

type ReadTracker struct {
    repo            *ApplicationsRepository
    tracker         *DeliveryTracker
    messageToApp    map[int64]uuid.UUID  // Telegram msg ID → App ID
    mu              sync.RWMutex
}

// OnMessageRead handles updateReadHistoryOutbox
func (rt *ReadTracker) OnMessageRead(ctx context.Context, peerUserID int64, maxMsgID int64) error {
    rt.mu.RLock()
    appID, found := rt.messageToApp[maxMsgID]
    rt.mu.RUnlock()

    if !found {
        // Not our message (or already processed)
        return nil
    }

    // Automatically mark as READ
    err := rt.tracker.UpdateStatus(ctx, appID, "READ")
    if err != nil {
        return err
    }

    // Clean up mapping
    rt.mu.Lock()
    delete(rt.messageToApp, maxMsgID)
    rt.mu.Unlock()

    return nil
}

// RegisterSentMessage stores mapping for later read detection
func (rt *ReadTracker) RegisterSentMessage(msgID int64, appID uuid.UUID) {
    rt.mu.Lock()
    defer rt.mu.Unlock()
    rt.messageToApp[msgID] = appID
}
```

### Integration with Telegram Manager

```go
// In internal/telegram/manager.go - add update handler

func (m *Manager) HandleUpdate(ctx context.Context, upd tg.UpdateClass) error {
    switch u := upd.(type) {
    case *tg.UpdateReadHistoryOutbox:
        // Peer read our messages up to max_id
        if peer, ok := u.Peer.(*tg.PeerUser); ok {
            return m.readTracker.OnMessageRead(ctx, peer.UserID, u.MaxID)
        }
    case *tg.UpdateReadMessagesContents:
        // Messages marked as read (content viewed)
        // Handle similarly if tracking content views
    }
    return nil
}
```

### Email Status Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           EMAIL STATUS FLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PENDING ──► SENDING ──► SENT                                              │
│                  │                                                          │
│                  │  [NO AUTOMATIC VERIFICATION without paid service]       │
│                  │                                                          │
│                  │  Options:                                                │
│                  │  1. Manual user confirmation                            │
│                  │  2. Future: Add webhook/bounce detection                │
│                  ▼                                                          │
│            (User manually marks as DELIVERED/READ)                          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Status Update API

```
PATCH /api/v1/applications/{id}/delivery
{
  "status": "DELIVERED" | "READ" | "RESPONDED",
  "notes": "Recruiter replied on 2025-01-15"
}
```

### Email Status Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           EMAIL STATUS FLOW                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  PENDING ──► SENDING ──► SENT                                              │
│                  │                                                          │
│                  │  [NO AUTOMATIC VERIFICATION without paid service]       │
│                  │                                                          │
│                  │  Options:                                                │
│                  │  1. Manual user confirmation                            │
│                  │  2. Future: Add webhook/bounce detection                │
│                  ▼                                                          │
│            (User manually marks as DELIVERED/READ)                          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Implementation Notes

1. **Telegram Read Receipts** (Available via MTProto):
   - `updateReadHistoryOutbox`: Fired when peer reads your DMs
   - `updateReadMessagesContents`: Fired when peer reads message contents
   - These are standard MTProto updates, no special API needed
   - Hook into existing update handler in telegram manager

2. **Message ID Mapping**:
   - When sending, store: Telegram msg ID → Application ID
   - Use in-memory map (cleared after read detected)
   - Optional: persist to DB for cross-session tracking

3. **Email Limitation**: Without services like:
   - SendGrid/Mailgun webhooks
   - Gmail API read receipts
   - Return-path headers
   - Email status remains manual until webhook integration

4. **Fallback Manual Updates**:
   - User can still manually mark: DELIVERED, READ, RESPONDED
   - Useful when automatic detection misses edge cases
   - User can record recruiter response text

### Future Enhancements (Phase TBD)

- [ ] Add email webhook integration (SendGrid/Mailgun)
- [ ] Parse recruiter responses from incoming messages
- [ ] Auto-detect "responded" status when recruiter messages back
- [ ] Persist message-to-app mapping to database for cross-session tracking

---

## Components

### 1. Telegram DM Sender

**File:** `internal/dispatcher/telegram_sender.go`

**Responsibilities:**
- Send PDF documents + cover letter text via Telegram DM
- Handle file uploads via MTProto
- Track delivery status
- Handle rate limits and FloodWait

**Key Functions:**

```go
type TelegramSender struct {
    client  *gotgproto.Client
    repo    *ApplicationsRepository
    hub     *web.Hub
    limiter *rate.Limiter
}

// SendApplication sends resume and cover letter via Telegram DM
func (s *TelegramSender) SendApplication(ctx context.Context, appID uuid.UUID, recipient string) error

// UploadAndSend uploads PDF and sends message with text
func (s *TelegramSender) UploadAndSend(ctx context.Context, recipient string, text string, pdfPath string) error

// ResolveUsername resolves @username to InputPeerUser
func (s *TelegramSender) ResolveUsername(ctx context.Context, username string) (*tg.InputPeerUser, error)
```

**Implementation Notes:**
- Use `client.API().MessagesSendMedia()` for file uploads
- File upload via `uploader.NewUploader()`
- Respect Telegram rate limits (1 msg/sec to same user)
- Handle `FLOOD_WAIT` errors automatically

---

### 2. Email Sender

**Status:** STUB ONLY (not implemented in Phase 5)

Email sender exists as a stub that returns "not implemented" error. Implementation deferred until after Telegram DM is stable.

**Reason:**
- Telegram DM is the primary channel for this phase
- Email adds complexity (SMTP, templates, bounce handling)
- Focus on one channel first, add second later

**Stub Implementation:**
```go
// internal/dispatcher/email_sender.go

package dispatcher

import (
    "context"
    "fmt"
    "github.com/google/uuid"
)

// EmailSender stub - not implemented in Phase 5
type EmailSender struct{}

func NewEmailSender() *EmailSender {
    return &EmailSender{}
}

// SendApplication returns "not implemented" error
func (s *EmailSender) SendApplication(ctx context.Context, appID uuid.UUID, recipient string) error {
    return fmt.Errorf("email sender not implemented: use TG_DM channel instead")
}

// Future fields (add when implementing):
// smtpConfig *SMTPConfig
// repo       *ApplicationsRepository
// hub        *web.Hub
// log        *logger.Logger
```

**Future Implementation (Phase 5b):**
- Use `github.com/wneessen/go-mail` (not `net/smtp`)
- Supports: attachments, HTML, proper error handling

---

### 3. Delivery Tracker

**File:** `internal/dispatcher/tracker.go`

**Purpose:** Centralized delivery status tracking with database persistence and WebSocket event broadcasting.

**Responsibilities:**
- Update delivery status in database (atomic transitions)
- Broadcast WebSocket events for UI updates
- Emit structured logs for observability
- Handle delivery failures with error recording
- Prevent invalid status transitions

**Status State Machine:**

```
                    ┌─────────────┐
                    │   PENDING   │  Initial state
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  SENDING    │  Send in progress
                    └──────┬──────┘
                           │
           ┌───────────────┴───────────────┐
           ▼                               ▼
    ┌─────────────┐                 ┌─────────────┐
    │    SENT     │                 │   FAILED    │  Error + retry exhausted
    └──────┬──────┘                 └─────────────┘
           │
           │ (Manual user action)
           ▼
    ┌─────────────┐
    │  DELIVERED  │  User confirmed
    └──────┬──────┘
           │
           │ (Manual user action)
           ▼
    ┌─────────────┐
    │    READ     │  User confirmed read
    └──────┬──────┘
           │
           │ (User records response)
           ▼
    ┌─────────────┐
    │  RESPONDED  │  Final state
    └─────────────┘
```

**Key Functions:**

```go
// internal/dispatcher/tracker.go

type DeliveryTracker struct {
    repo        *ApplicationsRepository
    hub         *web.Hub
    log         *logger.Logger
}

// TrackStart marks delivery as started (PENDING → SENDING)
func (t *DeliveryTracker) TrackStart(ctx context.Context, appID uuid.UUID) error

// TrackSuccess marks delivery as successful (SENDING → SENT)
func (t *DeliveryTracker) TrackSuccess(ctx context.Context, appID uuid.UUID) error

// TrackFailure marks delivery as failed (any → FAILED)
// Stores error message in recruiter_response field
func (t *DeliveryTracker) TrackFailure(ctx context.Context, appID uuid.UUID, err error) error

// TrackProgress reports intermediate progress (during SENDING)
func (t *DeliveryTracker) TrackProgress(ctx context.Context, appID uuid.UUID, step string, progress int) error

// UpdateStatus manually updates status (for user actions)
func (t *DeliveryTracker) UpdateStatus(ctx context.Context, appID uuid.UUID, status DeliveryStatus) error

// GetStatus returns current status of an application
func (t *DeliveryTracker) GetStatus(ctx context.Context, appID uuid.UUID) (DeliveryStatus, error)

// ValidateTransition checks if status transition is valid
func (t *DeliveryTracker) ValidateTransition(from, to DeliveryStatus) bool
```

**WebSocket Events Emitted:**

```go
// Event: dispatcher.status_changed
{
  "type": "dispatcher.status_changed",
  "application_id": "uuid",
  "previous_status": "PENDING",
  "current_status": "SENDING",
  "updated_at": "2025-01-14T10:00:00Z"
}

// Event: dispatcher.progress
{
  "type": "dispatcher.progress",
  "application_id": "uuid",
  "step": "uploading_pdf",
  "progress": 45,
  "message": "Uploading resume.pdf..."
}

// Event: dispatcher.failed
{
  "type": "dispatcher.failed",
  "application_id": "uuid",
  "error": "FLOOD_WAIT: 30 seconds",
  "retryable": true,
  "retry_after_seconds": 30
}
```

**Logging Strategy:**

```go
// TrackStart logs:
{"level":"info","service":"tracker","event":"status_changed","app_id":"uuid","from":"PENDING","to":"SENDING"}

// TrackSuccess logs:
{"level":"info","service":"tracker","event":"status_changed","app_id":"uuid","from":"SENDING","to":"SENT","duration_ms":2500}

// TrackFailure logs:
{"level":"error","service":"tracker","event":"status_changed","app_id":"uuid","from":"SENDING","to":"FAILED","error":"user not found"}

// TrackProgress logs:
{"level":"debug","service":"tracker","event":"progress","app_id":"uuid","step":"uploading","progress":50}
```

**Error Handling:**

| Error Type | Action | Stored in DB |
|------------|--------|--------------|
| User not found | Mark FAILED | Yes (error message) |
| FloodWait | Wait + retry | No (temporary) |
| Network timeout | Retry | No (temporary) |
| File not found | Mark FAILED | Yes (error message) |
| Invalid transition | Log error | No (no change) |

---

### 4. Dispatcher Service

**File:** `internal/dispatcher/service.go`

**Orchestrates sending through different channels:**

```go
type DispatcherService struct {
    tgSender   *TelegramSender
    emailSender *EmailSender
    tracker    *DeliveryTracker
    repo       *ApplicationsRepository
}

// SendApplication dispatches application through specified channel
func (s *DispatcherService) SendApplication(ctx context.Context, req *SendRequest) error

type SendRequest struct {
    JobID       uuid.UUID
    Channel     DeliveryChannel  // TG_DM or EMAIL
    Recipient   string           // @username or email
}

// SendViaTelegram sends via Telegram DM
func (s *DispatcherService) SendViaTelegram(ctx context.Context, app *JobApplication, recipient string) error

// SendViaEmail sends via Email
func (s *DispatcherService) SendViaEmail(ctx context.Context, app *JobApplication, recipient string) error
```

---

## REST API Endpoints

```
POST /api/v1/applications
{
  "job_id": "uuid",
  "channel": "TG_DM",
  "recipient": "@recruiter"
}
→ Creates application record, returns { id, status: "PENDING" }

POST /api/v1/applications/{id}/send
{
  "channel": "TG_DM" | "EMAIL",
  "recipient": "@username" | "email@example.com"
}
→ Triggers async sending via NATS

GET /api/v1/applications/{id}
→ Returns application with delivery status

GET /api/v1/applications?job_id={uuid}
→ Returns all applications for a job

PATCH /api/v1/applications/{id}/status
{
  "status": "SENT" | "FAILED"
}
→ Manual status update (for testing/manual override)
```

---

## WebSocket Events

### Dispatcher Events

```json
// Sending started
{ "type": "dispatcher.started", "application_id": "uuid", "channel": "TG_DM", "recipient": "@user" }

// Sending progress
{ "type": "dispatcher.progress", "application_id": "uuid", "step": "uploading", "progress": 50 }

// Sending completed
{ "type": "dispatcher.sent", "application_id": "uuid", "channel": "TG_DM", "sent_at": "2025-01-14T10:00:00Z" }

// Sending failed
{ "type": "dispatcher.failed", "application_id": "uuid", "error": "user not found" }
```

---

## Implementation Order

### Stage 1: Repository Layer

---

#### Task 1.1: Create Applications Repository

**Description:** Create the repository layer for job applications CRUD operations.

**Goal:** Enable database access for `job_applications` table.

**Pseudo Code:**
```
// internal/repository/applications.go
TYPE ApplicationsRepository STRUCT
    pool *pgxpool.Pool
    log  *logger.Logger
END

FUNCTION NewApplicationsRepository(pool, log) RETURN *ApplicationsRepository
    RETURN &ApplicationsRepository{pool, log}
END
```

**Test Cases:**
- [ ] Repository initializes with valid pool
- [ ] Repository nil if pool is nil

**Acceptance Criteria:**
- File exists at `internal/repository/applications.go`
- Struct has pool and logger fields
- Constructor function compiles

---

#### Task 1.2: Implement CRUD Read Operations

**Description:** Implement `Create`, `GetByJobID`, `GetByID` methods.

**Goal:** Enable creating and fetching application records.

**Pseudo Code:**
```
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

**Test Cases:**
- [ ] Create inserts record with PENDING status
- [ ] Create returns error for invalid job_id
- [ ] GetByID returns application or nil not found error
- [ ] GetByJobID returns empty slice for job with no applications
- [ ] GetByJobID returns applications in descending order

**Acceptance Criteria:**
- All methods compile
- Integration test passes with test database
- Foreign key constraint validated (job_id must exist)

---

#### Task 1.3: Implement Status Update Methods

**Description:** Implement `UpdateDeliveryStatus`, `UpdateRecipient`, `MarkSent`.

**Goal:** Enable updating application delivery status and metadata.

**Pseudo Code:**
```
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

**Test Cases:**
- [ ] UpdateDeliveryStatus changes status
- [ ] UpdateDeliveryStatus fails for invalid application_id
- [ ] MarkSent sets status to SENT and populates sent_at
- [ ] MarkSent idempotent (can be called multiple times)

**Acceptance Criteria:**
- All methods compile
- Timestamps are set correctly
- Invalid UUIDs return proper errors

---

#### Task 1.4: Implement ListPending

**Description:** Implement `ListPending` for queue processing.

**Goal:** Fetch applications pending delivery.

**Pseudo Code:**
```
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

**Test Cases:**
- [ ] Returns only PENDING applications
- [ ] Respects limit parameter
- [ ] Returns oldest applications first (ASC)
- [ ] Returns empty slice when no pending applications

**Acceptance Criteria:**
- Method compiles
- Default limit = 10 if not specified

---

#### Task 1.5: Write Unit Tests

**Description:** Add comprehensive unit tests for repository.

**Goal:** Ensure repository works correctly.

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
- All tests pass
- Coverage > 80% for repository layer

---

### Stage 2: Delivery Tracker

---

#### Task 2.1: Create Tracker File

**Description:** Create `internal/dispatcher/tracker.go` with struct.

**Goal:** Define tracker structure.

**Pseudo Code:**
```
// internal/dispatcher/tracker.go
TYPE DeliveryTracker STRUCT
    repo *ApplicationsRepository
    hub  *web.Hub
    log  *logger.Logger
END

FUNCTION NewDeliveryTracker(repo, hub, log) RETURN *DeliveryTracker
    RETURN &DeliveryTracker{repo, hub, log}
END
```

**Test Cases:**
- [ ] Constructor creates valid tracker
- [ ] Nil repo returns error

**Acceptance Criteria:**
- File exists
- Struct has required dependencies

---

#### Task 2.2: Implement Status Tracking Methods

**Description:** Implement `TrackStart`, `TrackSuccess`, `TrackFailure`.

**Goal:** Update status and emit events.

**Pseudo Code:**
```
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

**Test Cases:**
- [ ] TrackStart updates to SENDING
- [ ] TrackStart broadcasts WebSocket event
- [ ] TrackSuccess calls MarkSent
- [ ] TrackFailure stores error message

**Acceptance Criteria:**
- All methods compile
- Status transitions are validated
- WebSocket events are emitted

---

#### Task 2.3: Implement WebSocket Broadcasting

**Description:** Implement event broadcasting methods.

**Goal:** UI receives real-time status updates.

**Pseudo Code:**
```
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

**Test Cases:**
- [ ] BroadcastStatusChanged sends correct event
- [ ] BroadcastProgress includes progress percentage
- [ ] BroadcastFailed includes error message

**Acceptance Criteria:**
- Events match JSON schema
- Hub receives events

---

#### Task 2.4: Write Unit Tests

**Description:** Test tracker functionality.

**Goal:** Verify status tracking works.

**Test Cases:**
- [ ] TestTrackStart_ValidTransition
- [ ] TestTrackStart_InvalidTransition
- [ ] TestTrackSuccess_AfterStart
- [ ] TestTrackFailure_StoresError
- [ ] TestBroadcast_EventsEmitted

**Acceptance Criteria:**
- All tests pass
- Mocked hub and repo

---

### Stage 2.5: Read Receipt Tracker

---

#### Task 2.5.1: Create Read Tracker File

**Description:** Create `internal/dispatcher/read_tracker.go`.

**Goal:** Track Telegram message IDs to application IDs for read detection.

**Pseudo Code:**
```
// internal/dispatcher/read_tracker.go
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

**Test Cases:**
- [ ] Constructor creates valid tracker
- [ ] Message map initialized empty

**Acceptance Criteria:**
- File exists
- Struct has required dependencies

---

#### Task 2.5.2: Implement Message Registration

**Description:** Implement `RegisterSentMessage` for mapping sent messages.

**Goal:** Store Telegram message ID → Application ID mapping.

**Pseudo Code:**
```
FUNCTION RegisterSentMessage(msgID int64, appID uuid.UUID)
    mu.Lock()
    DEFER mu.Unlock()
    messageToApp[msgID] = appID
    log.Debug("registered", "msg_id", msgID, "app_id", appID)
END
```

**Test Cases:**
- [ ] Message ID mapped to application ID
- [ ] Duplicate message ID overwrites
- [ ] Thread-safe (concurrent calls)

**Acceptance Criteria:**
- Method compiles
- Thread-safe access

---

#### Task 2.5.3: Implement Read Update Handler

**Description:** Implement `OnMessageRead` for handling `updateReadHistoryOutbox`.

**Goal:** Auto-update status to READ when Telegram sends update.

**Pseudo Code:**
```
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

**Test Cases:**
- [ ] Found message updates status to READ
- [ ] Not found message returns nil
- [ ] Mapping cleaned up after processing
- [ ] WebSocket event emitted

**Acceptance Criteria:**
- Status updated to READ
- Mapping removed
- Event broadcasted

---

#### Task 2.5.4: Integrate with Telegram Manager

**Description:** Hook read tracker into Telegram update handler.

**Goal:** Receive `updateReadHistoryOutbox` updates.

**Pseudo Code:**
```
// In internal/telegram/manager.go
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

**Test Cases:**
- [ ] UpdateReadHistoryOutbox triggers OnMessageRead
- [ ] UpdateReadMessagesContents handled (optional)
- [ ] Non-matching updates ignored

**Acceptance Criteria:**
- Read tracker called on read updates
- Other updates not affected

---

#### Task 2.5.5: Update TelegramSender to Register Messages

**Description:** Register sent messages in read tracker after sending.

**Goal:** Enable read detection for sent applications.

**Pseudo Code:**
```
// In telegram_sender.go SendApplication method
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

**Test Cases:**
- [ ] Message ID registered after successful send
- [ ] Registration happens before TrackSuccess
- [ ] Failed send does not register

**Acceptance Criteria:**
- All sent messages registered
- Read tracking works end-to-end

---

#### Task 2.5.6: Write Unit Tests

**Description:** Test read tracker functionality.

**Goal:** Verify read detection works.

**Test Cases:**
- [ ] TestRegisterSentMessage_StoresMapping
- [ ] TestOnMessageRead_UpdatesStatus
- [ ] TestOnMessageRead_NotFoundReturnsNil
- [ ] TestOnMessageRead_CleansUpMapping
- [ ] TestConcurrentAccess_ThreadSafe

**Acceptance Criteria:**
- All tests pass
- Mocked tracker and repo

---

### Stage 3: Telegram Sender

---

#### Task 3.1: Create Telegram Sender File

**Description:** Create `internal/dispatcher/telegram_sender.go`.

**Goal:** Define sender structure.

**Pseudo Code:**
```
TYPE TelegramSender STRUCT
    client  *gotgproto.Client
    tracker  *DeliveryTracker
    repo     *ApplicationsRepository
    limiter  *rate.Limiter  // 1 per 10 seconds
    log      *logger.Logger
END

FUNCTION NewTelegramSender(client, tracker, repo, log) RETURN *TelegramSender
    RETURN &TelegramSender{
        client: client,
        tracker: tracker,
        repo: repo,
        limiter: rate.NewLimiter(rate.Every(10 * time.Second), 1),
        log: log,
    }
END
```

**Test Cases:**
- [ ] Constructor creates valid sender
- [ ] Rate limiter configured for 1 per 10 sec

**Acceptance Criteria:**
- File exists
- Limiter rate = 1 per 10 seconds

---

#### Task 3.2: Implement Username Resolution

**Description:** Implement `ResolveUsername` method.

**Goal:** Convert @username to Telegram InputPeerUser.

**Pseudo Code:**
```
FUNCTION ResolveUsername(ctx, username) RETURN (*tg.InputPeerUser, error)
    // Strip @ if present
    IF username[0] == '@' {
        username = username[1:]
    }

    // Use contacts.ResolveUsername API
    result, err := client.API().ContactsResolveUsername(ctx, username)
    IF err != nil RETURN nil, err

    RETURN &tg.InputPeerUser{
        UserID:   result.Users()[0].ID(),
        AccessHash: result.Users()[0].AccessHash(),
    }, nil
END
```

**Test Cases:**
- [ ] Resolves valid username
- [ ] Handles @ prefix
- [ ] Returns error for non-existent user
- [ ] Returns error for empty username

**Acceptance Criteria:**
- Method compiles
- Handles FLOOD_WAIT error

---

#### Task 3.3: Implement Upload and Send

**Description:** Implement `UploadAndSend` for file + message.

**Goal:** Send PDF with cover letter text.

**Pseudo Code:**
```
FUNCTION UploadAndSend(ctx, recipient, text, pdfPath) RETURN error
    peer, err := ResolveUsername(ctx, recipient)
    IF err != nil RETURN err

    // Wait for rate limiter
    limiter.Wait(ctx)

    // Upload PDF
    file := tg.InputFileLocal{Path: pdfPath}
    uploaded, err := uploader.NewUploader(client).Upload(ctx, file)
    IF err != nil RETURN err

    // Send media with caption
    media := &tg.InputMediaDocument{
        ID: uploaded.InputDocumentFile,
        Caption: text,
    }

    _, err = client.API().MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
        Peer: peer,
        Media: media,
    })
    RETURN err
END
```

**Test Cases:**
- [ ] Uploads PDF successfully
- [ ] Sends cover letter as caption
- [ ] Respects rate limit
- [ ] Returns error for missing file

**Acceptance Criteria:**
- File upload works
- Rate limit enforced (1 per 10 sec)

---

#### Task 3.4: Implement SendApplication Orchestration

**Description:** Orchestrate full send flow.

**Goal:** Send resume + cover letter via Telegram DM.

**Pseudo Code:**
```
FUNCTION SendApplication(ctx, appID, recipient) RETURN error
    app, err := repo.GetByID(ctx, appID)
    IF err != nil RETURN err

    // Track start
    tracker.TrackStart(ctx, appID)

    // Send resume PDF
    err = UploadAndSend(ctx, recipient, app.CoverLetterMD, app.ResumePDFPath)
    IF err != nil {
        tracker.TrackFailure(ctx, appID, err)
        RETURN err
    }

    // Track success
    tracker.TrackSuccess(ctx, appID)
    RETURN nil
END
```

**Test Cases:**
- [ ] Successful send flow
- [ ] Failure tracked on error
- [ ] Missing PDF results in FAILED

**Acceptance Criteria:**
- Full flow works end-to-end
- Status transitions correctly

---

#### Task 3.5: Add Rate Limiting

**Description:** Ensure rate limiter prevents spam.

**Goal:** Max 1 message per 10 seconds.

**Pseudo Code:**
```
// Already in Task 3.1
limiter: rate.NewLimiter(rate.Every(10 * time.Second), 1)

// In UploadAndSend:
limiter.Wait(ctx)  // Blocks until 10sec elapsed since last send
```

**Test Cases:**
- [ ] Two rapid sends wait 10 seconds
- [ ] One send completes immediately
- [ ] Rate limit logged

**Acceptance Criteria:**
- Cannot send faster than 1 per 10 sec
- Wait time logged

---

#### Task 3.6: Write Integration Tests

**Description:** Test with real Telegram session.

**Goal:** Verify actual Telegram sending works.

**Test Cases:**
- [ ] Send to self succeeds
- [ ] Invalid username fails
- [ ] File upload succeeds

**Acceptance Criteria:**
- Tests pass with valid session
- Skipped if no session available

---

### Stage 4: Dispatcher Service

> **Note:** Email Sender is a stub that returns "not implemented" error

---

#### Task 5.1: Create Dispatcher Service File

**Description:** Create `internal/dispatcher/service.go`.

**Goal:** Main orchestrator for sending.

**Pseudo Code:**
```
TYPE DispatcherService STRUCT
    tgSender *TelegramSender
    tracker   *DeliveryTracker
    repo      *ApplicationsRepository
    log       *logger.Logger
}

FUNCTION NewDispatcherService(tgSender, tracker, repo, log) RETURN *DispatcherService
    RETURN &DispatcherService{tgSender, tracker, repo, log}
END
```

**Test Cases:**
- [ ] Constructor creates valid service

**Acceptance Criteria:**
- File exists
- Struct has required fields

---

#### Task 5.2: Implement SendApplication Routing

**Description:** Route to correct sender based on channel.

**Goal:** Dispatch to Telegram or fail for unsupported.

**Pseudo Code:**
```
FUNCTION SendApplication(ctx, req) RETURN error
    // Validate request
    IF req.Channel == "TG_DM" {
        RETURN s.SendViaTelegram(ctx, req.JobID, req.Recipient)
    }
    IF req.Channel == "EMAIL" {
        RETURN errors.New("email sender not implemented: use TG_DM channel instead")
    }
    RETURN errors.New("unsupported channel")
END
```

**Test Cases:**
- [ ] TG_DM routes to telegram sender
- [ ] EMAIL returns "not implemented" error
- [ ] Invalid channel returns error

**Acceptance Criteria:**
- Only TG_DM works in this phase
- Clear error for EMAIL

---

#### Task 5.3: Implement SendViaTelegram

**Description:** Telegram send orchestration.

**Goal:** Create application and send.

**Pseudo Code:**
```
FUNCTION SendViaTelegram(ctx, jobID, recipient) RETURN error
    // Create application record
    app := &JobApplication{
        ID:        uuid.New(),
        JobID:     jobID,
        Channel:   "TG_DM",
        Recipient: recipient,
        Status:    "PENDING",
    }
    err := repo.Create(ctx, app)
    IF err != nil RETURN err

    // Send via Telegram
    RETURN tgSender.SendApplication(ctx, app.ID, recipient)
END
```

**Test Cases:**
- [ ] Creates application record
- [ ] Calls telegram sender
- [ ] Returns error if create fails

**Acceptance Criteria:**
- Application created before send
- Send uses application ID

---

#### Task 5.4: Write Unit Tests

**Description:** Test dispatcher service.

**Goal:** Verify routing works.

**Test Cases:**
- [ ] TestSendApplication_TGDM_Success
- [ ] TestSendApplication_EMAIL_NotImplemented
- [ ] TestSendApplication_InvalidChannel
- [ ] TestSendViaTelegram_CreatesApplication

**Acceptance Criteria:**
- All tests pass
- Mocked dependencies

---

### Stage 5: REST API

---

#### Task 6.1: Create Applications Handler

**Description:** Create `internal/web/handlers/applications.go`.

**Goal:** HTTP handlers for applications.

**Pseudo Code:**
```
TYPE ApplicationsHandler STRUCT {
    dispatcher *DispatcherService
    repo       *ApplicationsRepository
}

FUNCTION NewApplicationsHandler(dispatcher, repo) RETURN *ApplicationsHandler
    RETURN &ApplicationsHandler{dispatcher, repo}
END
```

**Test Cases:**
- [ ] Handler constructs

**Acceptance Criteria:**
- File exists
- Struct has dependencies

---

#### Task 6.2: Implement CRUD Handlers

**Description:** Create, GetByID, GetByJobID endpoints.

**Goal:** REST API for applications.

**Pseudo Code:**
```
FUNCTION Create(w, r)
    var req struct {
        JobID     uuid.UUID `json:"job_id"`
        Channel   string    `json:"channel"`
        Recipient string    `json:"recipient"`
    }
    decodeJSON(r.Body, &req)

    app := &JobApplication{
        ID: uuid.New(),
        JobID: req.JobID,
        Channel: req.Channel,
        Recipient: req.Recipient,
        Status: "PENDING",
    }
    repo.Create(r.Context(), app)
    writeJSON(w, 201, app)
END

FUNCTION GetByID(w, r)
    id := chi.URLParam(r, "id")
    app, err := repo.GetByID(r.Context(), id)
    IF err != nil { writeError(w, 404); RETURN }
    writeJSON(w, 200, app)
END

FUNCTION GetByJobID(w, r)
    jobID := r.URL.Query().Get("job_id")
    apps, err := repo.GetByJobID(r.Context(), jobID)
    writeJSON(w, 200, apps)
END
```

**Test Cases:**
- [ ] Create returns 201 with application
- [ ] GetByID returns 404 for not found
- [ ] GetByJobID returns array

**Acceptance Criteria:**
- All endpoints work
- JSON format correct

---

#### Task 6.3: Implement Send Endpoint

**Description:** POST `/api/v1/applications/{id}/send`.

**Goal:** Trigger sending.

**Pseudo Code:**
```
FUNCTION Send(w, r)
    id := chi.URLParam(r, "id")
    var req struct {
        Channel   string `json:"channel"`
        Recipient string `json:"recipient"`
    }
    decodeJSON(r.Body, &req)

    // Get application
    app, err := repo.GetByID(r.Context(), id)
    IF err != nil { writeError(w, 404); RETURN }

    // Send
    go dispatcher.SendApplication(r.Context(), &SendRequest{
        JobID:     app.JobID,
        Channel:   req.Channel,
        Recipient: req.Recipient,
    })

    writeJSON(w, 202, map[string]string{"status": "sending"})
END
```

**Test Cases:**
- [ ] Returns 202 Accepted
- [ ] Triggers background send
- [ ] Returns 404 for invalid ID

**Acceptance Criteria:**
- Async send initiated
- Immediate response

---

#### Task 6.4: Register Routes

**Description:** Add routes to server.

**Goal:** Expose endpoints.

**Pseudo Code:**
```
// In internal/web/server.go
FUNCTION RegisterApplicationsHandler(handler)
    router.Route("/api/v1/applications", func(r chi.Router) {
        r.Post("/", handler.Create)
        r.Get("/{id}", handler.GetByID)
        r.Get("/", handler.GetByJobID)  // Query param: job_id
        r.Post("/{id}/send", handler.Send)
        r.Patch("/{id}/delivery", handler.UpdateDelivery)
    })
END
```

**Test Cases:**
- [ ] All routes accessible
- [ ] Route parameters parsed

**Acceptance Criteria:**
- Routes registered
- Server compiles

---

#### Task 6.5: Write Integration Tests

**Description:** Test full API flow.

**Goal:** End-to-end verification.

**Test Cases:**
- [ ] POST /api/v1/applications creates record
- [ ] POST /api/v1/applications/{id}/send triggers send
- [ ] GET /api/v1/applications/{id} returns status

**Acceptance Criteria:**
- Integration tests pass
- Test database used

---

### Stage 6: UI Integration

---

#### Task 7.1: Add Send Buttons

**Description:** Add "Send to Telegram" button to job detail.

**Goal:** User can trigger send from UI.

**Pseudo Code:**
```
<!-- In job detail template -->
<button
    class="btn btn-primary"
    hx-post="/api/v1/applications/{job_id}/send"
    hx-confirm="Enter recruiter's @username:"
    hx-vals='{"channel": "TG_DM", "recipient": "REPLACE_WITH_USERNAME"}'>
    Send to Telegram
</button>
```

**Test Cases:**
- [ ] Button visible on job detail
- [ ] Click prompts for username
- [ ] Send triggered after confirmation

**Acceptance Criteria:**
- Button renders
- HTMX request sent

---

#### Task 7.2: Add Recipient Input Modal

**Description:** Modal for entering recipient.

**Goal:** Clean input UX.

**Pseudo Code:**
```
<!-- Modal for recipient input -->
<div id="send-modal" class="modal">
    <div class="modal-content">
        <h3>Send Application</h3>
        <label>Recipient (@username or email)</label>
        <input type="text" id="recipient-input" placeholder="@recruiter">
        <button onclick="sendApplication()">Send</button>
    </div>
</div>

<script>
FUNCTION sendApplication()
    recipient = document.getElementById("recipient-input").value
    fetch("/api/v1/applications/{job_id}/send", {
        method: "POST",
        body: JSON.stringify({channel: "TG_DM", recipient: recipient})
    })
END
</script>
```

**Test Cases:**
- [ ] Modal opens on button click
- [ ] Recipient input validated
- [ ] Send triggered on submit

**Acceptance Criteria:**
- Modal works
- Input required

---

#### Task 7.3: Show Delivery Progress

**Description:** Display real-time progress via WebSocket.

**Goal:** User sees send progress.

**Pseudo Code:**
```
<!-- Listen for dispatcher events -->
<script>
ws.addEventListener("message", (event) => {
    data = JSON.parse(event.data)

    IF data.type == "dispatcher.status_changed" {
        updateStatusBadge(data.application_id, data.current_status)
    }

    IF data.type == "dispatcher.progress" {
        updateProgressBar(data.application_id, data.progress)
    }

    IF data.type == "dispatcher.failed" {
        showError(data.application_id, data.error)
    }
})
</script>

<!-- Status badge in UI -->
<span class="status-badge status-{status}">{status}</span>
```

**Test Cases:**
- [ ] Status updates on WebSocket event
- [ ] Progress bar updates
- [ ] Error shown on failure

**Acceptance Criteria:**
- Real-time updates work
- UI reflects status changes

---

#### Task 7.4: Display Application History

**Description:** Show applications for a job.

**Goal:** User sees send history.

**Pseudo Code:**
```
<!-- Applications list in job detail -->
<div class="applications-list">
    <h3>Applications</h3>
    <table>
        <tr>
            <th>Date</th>
            <th>Channel</th>
            <th>Recipient</th>
            <th>Status</th>
        </tr>
        <!-- FOR EACH application IN applications -->
        <tr>
            <td>{app.created_at}</td>
            <td>{app.delivery_channel}</td>
            <td>{app.recipient}</td>
            <td><span class="status-badge status-{app.delivery_status}">{app.delivery_status}</span></td>
        </tr>
    </table>
</div>
```

**Test Cases:**
- [ ] List shows all applications
- [ ] Empty state shown when none
- [ ] Status badges colored correctly

**Acceptance Criteria:**
- History visible
- Sorted by date

---

#### Task 7.5: Add Status Update UI

**Description:** Allow manual status updates.

**Goal:** User can mark as delivered/read.

**Pseudo Code:**
```
<!-- Status update buttons -->
<div class="status-actions">
    <button onclick="updateStatus('DELIVERED')">Mark Delivered</button>
    <button onclick="updateStatus('READ')">Mark Read</button>
    <button onclick="updateStatus('RESPONDED')">Mark Responded</button>
</div>

<script>
FUNCTION updateStatus(status)
    fetch(`/api/v1/applications/{id}/delivery`, {
        method: "PATCH",
        body: JSON.stringify({status: status})
    })
END
</script>
```

**Test Cases:**
- [ ] Buttons update status
- [ ] WebSocket event emitted
- [ ] UI updates after change

**Acceptance Criteria:**
- Manual updates work
- Buttons only shown for SENT applications

---

### Stage 7: Service Status API

---

#### Task 8.1: Implement Status Endpoint

**Description:** GET `/api/v1/dispatcher/status`.

**Goal:** Expose service health.

**Pseudo Code:**
```
FUNCTION GetStatus(w, r)
    tgStatus := telegramManager.GetStatus()
    status := map[string]interface{}{
        "dispatcher_enabled": true,
        "telegram": map[string]string{
            "status": string(tgStatus),
        },
        "queue": map[string]int{
            "pending": getCount("PENDING"),
            "processing": getCount("SENDING"),
        },
    }
    writeJSON(w, 200, status)
END
```

**Test Cases:**
- [ ] Returns 200 with status
- [ ] Telegram status reflects actual state

**Acceptance Criteria:**
- Endpoint works
- JSON structure matches schema

---

#### Task 8.2: Add Status to Settings UI

**Description:** Display status on settings page.

**Goal:** User sees service status.

**Pseudo Code:**
```
<!-- In settings page -->
<div class="dispatcher-status">
    <h3>Dispatcher Status</h3>
    <div class="status-item">
        <span>Telegram:</span>
        <span class="status-badge status-{telegram_status}">{telegram_status}</span>
    </div>
    <div class="status-item">
        <span>Queue:</span>
        <span>{pending} pending, {processing} processing</span>
    </div>
</div>

<script>
// Poll status every 30 seconds
setInterval(() => {
    fetch("/api/v1/dispatcher/status")
        .then(r => r.json())
        .then(updateStatusUI)
}, 30000)
</script>
```

**Test Cases:**
- [ ] Status displayed correctly
- [ ] Auto-refreshes

**Acceptance Criteria:**
- Status visible
- Updates automatically

---

### Removed: NATS Integration

> **Decision:** NATS not needed for this phase. Sending is synchronous from REST API.
> Future: Consider NATS for background queue processing if needed.

---

## Configuration

### Environment Variables

```env
# Dispatcher
DISPATCHER_ENABLED=true          # Master switch for dispatcher

# Telegram (already exists)
TG_API_ID=
TG_API_HASH=

# Rate Limiting
DISPATCHER_RATE_PER_10SEC=1      # Messages per 10 seconds (default: 1)
```

### Deferred (stub only):
- SMTP_* variables (email integration - add when implementing email_sender.go)

---

## Error Handling

| Error                     | Action                     | Retry    |
|---------------------------|----------------------------|----------|
| User not found (TG)       | Mark FAILED, notify user   | No       |
| FloodWait (TG)            | Wait, retry                | Yes (3x) |
| Invalid email             | Mark FAILED, notify user   | No       |
| SMTP timeout              | Retry                      | Yes (2x)  |
| File not found            | Mark FAILED                | No       |
| Network error             | Retry with backoff        | Yes (3x)  |

---

## Security Considerations

1. **Telegram Anti-Spam**
   - **Rate limit: 1 message per 10 seconds maximum**
   - Implemented via `rate.Limiter` in TelegramSender
   - Random delays NOT needed (rate limiter is sufficient)
   - Don't send identical messages (use tailored cover letters)

2. **Email** (Future - stub only in Phase 5)
   - Stub returns "not implemented" error
   - When implementing: use `github.com/wneessen/go-mail`
   - Add app-specific passwords, rate limiting, bounce handling

3. **Rate Limiting Implementation**
   ```
   limiter: rate.NewLimiter(rate.Every(10 * time.Second), 1)
   ```
   - Blocks until 10 seconds elapsed since last send
   - Automatic - no manual delays needed
   - Wait time logged for observability

---

## Testing Strategy

### Unit Tests
- Mock Telegram client
- Mock SMTP client
- Mock repository
- Test status transitions

### Integration Tests
- Test with real Telegram account (dedicated test account)
- Test with test email (Mailtrap or similar)
- Test WebSocket events

### Manual Testing
- Send to own Telegram account
- Send to own email
- Verify document attachments
- Check status updates in UI

---

## Dependencies

```go
// Existing (already in project)
github.com/celestix/gotgproto    // Telegram MTProto client
github.com/go-chi/chi/v5         // HTTP router
golang.org/x/time/rate           // Rate limiter (in stdlib)

// No new dependencies required for Phase 5 (email sender is stub only)
```

### Deferred (when implementing email):
- `github.com/wneessen/go-mail` for SMTP (not `net/smtp`)

---

## Files to Create

```
internal/dispatcher/
├── service.go              # Main dispatcher service
├── telegram_sender.go      # Telegram DM sender
├── email_sender.go         # Email sender (STUB - returns "not implemented")
├── tracker.go              # Delivery tracking
├── read_tracker.go         # Read receipt detection (NEW)
├── types.go                # Shared types
└── service_test.go         # Tests

internal/repository/
├── applications.go         # Applications repository
└── applications_test.go    # Tests

internal/web/handlers/
├── applications.go         # REST handlers
├── applications_test.go    # Tests
└── dispatcher.go           # Status endpoint handler

static/partials/
└── job_applications.html   # UI component
```

---

## Open Questions (Answered)

1. **NATS or Direct?** ✅ **Direct**
   - Sending is synchronous from REST API (goroutine for background)
   - No NATS queue needed for this phase
   - Future: Consider NATS for background queue processing

2. **Queue Management?** ✅ **Simple List**
   - `ListPending()` for fetching pending applications
   - Manual trigger via UI
   - Future: Background worker if needed

3. **Retry Strategy?** ✅ **Manual**
   - Failed sends are NOT auto-retried
   - User can retry via UI
   - FloodWait errors do auto-wait (built into TelegramSender)

4. **Message Templates?** ✅ **Cover Letter from Brain**
   - Use `cover_letter_md` from Phase 4
   - No additional templating needed
   - Future: Custom intro templates

---

## Success Criteria

Phase 5 is complete when:

- [ ] User can send resume via Telegram DM to recruiter
- [ ] Delivery status is tracked in database
- [ ] WebSocket events show real-time progress
- [ ] UI shows application history per job
- [ ] Rate limiting prevents spam (1 per 10 seconds)
- [ ] Errors are handled gracefully with user feedback
- [ ] Service status visible in UI and logs
- [ ] **Automatic read detection works** via `updateReadHistoryOutbox`
- [ ] Manual status updates work (DELIVERED, READ, RESPONDED) as fallback
- [ ] **Email sender stub returns "not implemented"** (full implementation deferred)

### Deferred (not in Phase 5):
- Email sending (stub only - use `github.com/wneessen/go-mail` when implementing)
- NATS integration (not needed - direct REST API calls)

---