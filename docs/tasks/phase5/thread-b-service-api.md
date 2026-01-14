# Thread B: Service & API

**Developer:** Developer B
**Focus:** Higher-level service orchestration, REST API, and UI
**Estimated Duration:** ~40% of total work

## Overview

This thread covers user-facing components and service orchestration. You will build:
1. Telegram DM sender (file upload + messaging)
2. Dispatcher service (main orchestrator)
3. REST API endpoints
4. UI integration (buttons, modals, progress display)
5. Service status monitoring

---

## Dependencies

**You depend on:** Thread A (Backend Infrastructure)
**Threads depending on you:** None (you're last!)

**Critical Dependencies from Thread A:**
- **Tasks 1.1-1.5** (Repository) → Required for Telegram Sender
- **Tasks 2.1-2.4** (DeliveryTracker) → Required for Telegram Sender
- **Tasks 2.5.1-2.5.6** (ReadTracker) → Required for read detection integration

**Strategy while Thread A works:**
- Start with Telegram Sender using **interface mocks** for Repository/Tracker
- Swap in real implementations once Thread A hands off
- Begin UI work early (can mock API responses)

---

## Stage 3: Telegram Sender

### Task 3.1: Create Telegram Sender File

**File:** `internal/dispatcher/telegram_sender.go`

**Description:** Create sender structure with rate limiter.

**Pseudo Code:**
```go
TYPE TelegramSender STRUCT
    client  *gotgproto.Client
    tracker  *DeliveryTracker       // From Thread A, Task 2.x
    repo     *ApplicationsRepository // From Thread A, Task 1.x
    readTracker *ReadTracker         // From Thread A, Task 2.5.x
    limiter  *rate.Limiter  // 1 per 10 seconds
    log      *logger.Logger
END

FUNCTION NewTelegramSender(client, tracker, repo, readTracker, log) RETURN *TelegramSender
    RETURN &TelegramSender{
        client: client,
        tracker: tracker,
        repo: repo,
        readTracker: readTracker,
        limiter: rate.NewLimiter(rate.Every(10 * time.Second), 1),
        log: log,
    }
END
```

**Acceptance Criteria:**
- [ ] File exists
- [ ] Limiter rate = 1 per 10 seconds
- [ ] All dependencies injected (use interfaces initially if needed)

---

### Task 3.2: Implement Username Resolution

**File:** `internal/dispatcher/telegram_sender.go`

**Description:** Implement `ResolveUsername` method.

**Pseudo Code:**
```go
FUNCTION (s *TelegramSender) ResolveUsername(ctx, username) RETURN (*tg.InputPeerUser, error)
    // Strip @ if present
    IF username[0] == '@' {
        username = username[1:]
    }

    // Use contacts.ResolveUsername API
    result, err := s.client.API().ContactsResolveUsername(ctx, username)
    IF err != nil RETURN nil, err

    RETURN &tg.InputPeerUser{
        UserID:   result.Users()[0].ID(),
        AccessHash: result.Users()[0].AccessHash(),
    }, nil
END
```

**Acceptance Criteria:**
- [ ] Method compiles
- [ ] Handles @ prefix
- [ ] Returns error for non-existent user
- [ ] Handles FLOOD_WAIT error

---

### Task 3.3: Implement Upload and Send

**File:** `internal/dispatcher/telegram_sender.go`

**Description:** Implement `UploadAndSend` for file + message.

**Pseudo Code:**
```go
FUNCTION (s *TelegramSender) UploadAndSend(ctx, recipient, text, pdfPath) RETURN error
    peer, err := s.ResolveUsername(ctx, recipient)
    IF err != nil RETURN err

    // Wait for rate limiter
    err = s.limiter.Wait(ctx)
    IF err != nil RETURN err

    // Upload PDF
    file := tg.InputFileLocal{Path: pdfPath}
    uploaded, err := uploader.NewUploader(s.client).Upload(ctx, file)
    IF err != nil RETURN err

    // Send media with caption
    media := &tg.InputMediaDocument{
        ID: uploaded.InputDocumentFile,
        Caption: text,
    }

    _, err = s.client.API().MessagesSendMedia(ctx, &tg.MessagesSendMediaRequest{
        Peer: peer,
        Media: media,
    })
    RETURN err
END
```

**Acceptance Criteria:**
- [ ] File upload works
- [ ] Cover letter sent as caption
- [ ] Rate limit enforced (1 per 10 sec)

---

### Task 3.4: Implement SendApplication Orchestration

**File:** `internal/dispatcher/telegram_sender.go`

**Description:** Orchestrate full send flow with tracking.

**Pseudo Code:**
```go
FUNCTION (s *TelegramSender) SendApplication(ctx, appID, recipient) RETURN error
    app, err := s.repo.GetByID(ctx, appID)
    IF err != nil RETURN err

    // Track start
    err = s.tracker.TrackStart(ctx, appID)
    IF err != nil RETURN err

    // Send resume PDF
    err = s.UploadAndSend(ctx, recipient, app.CoverLetterMD, app.ResumePDFPath)
    IF err != nil {
        s.tracker.TrackFailure(ctx, appID, err)
        RETURN err
    }

    // Register message for read detection (Thread A will add this)
    // TODO: Coordinate with Thread A at Task 2.5.5

    // Track success
    err = s.tracker.TrackSuccess(ctx, appID)
    IF err != nil RETURN err

    RETURN nil
END
```

**Acceptance Criteria:**
- [ ] Full flow works end-to-end
- [ ] Status transitions correctly (PENDING → SENDING → SENT)
- [ ] Errors tracked properly

---

### Task 3.5: Add FloodWait Handling

**File:** `internal/dispatcher/telegram_sender.go`

**Description:** Handle Telegram FloodWait errors automatically.

**Pseudo Code:**
```go
FUNCTION (s *TelegramSender) handleFloodWait(ctx context.Context, err error) error {
    IF FloodWaitError, ok := err.(*tg.ErrorFloodWait); ok {
        waitSeconds := FloodWaitError.Seconds
        s.log.Warn("flood_wait", "wait_seconds", waitSeconds)

        SELECT {
        CASE <-ctx.Done():
            RETURN ctx.Err()
        CASE <-time.After(time.Duration(waitSeconds) * time.Second):
            // Retry after wait
            RETURN nil
        }
    }
    RETURN err
}

// Use in UploadAndSend:
_, err = s.client.API().MessagesSendMedia(ctx, req)
IF err != nil {
    IF fwErr := s.handleFloodWait(ctx, err); fwErr == nil {
        // Retry once after FloodWait
        _, err = s.client.API().MessagesSendMedia(ctx, req)
    }
    RETURN err
}
```

**Acceptance Criteria:**
- [ ] FloodWait errors trigger wait
- [ ] Automatic retry after wait
- [ ] Cancellation respected

---

### Task 3.6: Write Integration Tests

**File:** `internal/dispatcher/telegram_sender_test.go`

**Description:** Test with real Telegram session.

**Test Cases:**
- [ ] Send to self succeeds
- [ ] Invalid username fails
- [ ] File upload succeeds
- [ ] Rate limit enforced

**Acceptance Criteria:**
- [ ] Tests pass with valid session
- [ ] Skipped if no session available

---

## Stage 4: Dispatcher Service

### Task 5.1: Create Dispatcher Service File

**File:** `internal/dispatcher/service.go`

**Description:** Main orchestrator for sending.

**Pseudo Code:**
```go
TYPE DispatcherService STRUCT
    tgSender    *TelegramSender
    emailSender *EmailSender
    tracker     *DeliveryTracker
    repo        *ApplicationsRepository
    log         *logger.Logger
}

FUNCTION NewDispatcherService(tgSender, emailSender, tracker, repo, log) RETURN *DispatcherService
    RETURN &DispatcherService{tgSender, emailSender, tracker, repo, log}
END
```

**Acceptance Criteria:**
- [ ] File exists
- [ ] Struct has required fields

---

### Task 5.2: Implement SendApplication Routing

**File:** `internal/dispatcher/service.go`

**Description:** Route to correct sender based on channel.

**Pseudo Code:**
```go
FUNCTION (s *DispatcherService) SendApplication(ctx, req) RETURN error
    // Validate request
    IF req.Channel == "TG_DM" {
        RETURN s.SendViaTelegram(ctx, req.JobID, req.Recipient)
    }
    IF req.Channel == "EMAIL" {
        RETURN s.emailSender.SendApplication(ctx, req.JobID, req.Recipient)
    }
    RETURN errors.New("unsupported channel")
END
```

**Acceptance Criteria:**
- [ ] TG_DM routes to telegram sender
- [ ] EMAIL returns "not implemented" error (from Thread A stub)
- [ ] Invalid channel returns error

---

### Task 5.3: Implement SendViaTelegram

**File:** `internal/dispatcher/service.go`

**Description:** Telegram send orchestration.

**Pseudo Code:**
```go
FUNCTION (s *DispatcherService) SendViaTelegram(ctx, jobID, recipient) RETURN error
    // Create application record
    app := &JobApplication{
        ID:        uuid.New(),
        JobID:     jobID,
        Channel:   "TG_DM",
        Recipient: recipient,
        Status:    "PENDING",
    }
    err := s.repo.Create(ctx, app)
    IF err != nil RETURN err

    // Send via Telegram
    RETURN s.tgSender.SendApplication(ctx, app.ID, recipient)
END
```

**Acceptance Criteria:**
- [ ] Application created before send
- [ ] Send uses application ID
- [ ] Returns error if create fails

---

### Task 5.4: Write Unit Tests

**File:** `internal/dispatcher/service_test.go`

**Description:** Test dispatcher service.

**Test Cases:**
- [ ] TestSendApplication_TGDM_Success
- [ ] TestSendApplication_EMAIL_NotImplemented
- [ ] TestSendApplication_InvalidChannel
- [ ] TestSendViaTelegram_CreatesApplication

**Acceptance Criteria:**
- [ ] All tests pass
- [ ] Mocked dependencies

---

## Stage 5: REST API

### Task 6.1: Create Applications Handler

**File:** `internal/web/handlers/applications.go`

**Description:** HTTP handlers for applications.

**Pseudo Code:**
```go
TYPE ApplicationsHandler STRUCT {
    dispatcher *DispatcherService
    repo       *ApplicationsRepository
}

FUNCTION NewApplicationsHandler(dispatcher, repo) RETURN *ApplicationsHandler
    RETURN &ApplicationsHandler{dispatcher, repo}
END
```

**Acceptance Criteria:**
- [ ] File exists
- [ ] Struct has dependencies

---

### Task 6.2: Implement CRUD Handlers

**File:** `internal/web/handlers/applications.go`

**Description:** Create, GetByID, GetByJobID endpoints.

**Pseudo Code:**
```go
FUNCTION (h *ApplicationsHandler) Create(w, r)
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
    h.repo.Create(r.Context(), app)
    writeJSON(w, 201, app)
END

FUNCTION (h *ApplicationsHandler) GetByID(w, r)
    id := chi.URLParam(r, "id")
    app, err := h.repo.GetByID(r.Context(), id)
    IF err != nil { writeError(w, 404); RETURN }
    writeJSON(w, 200, app)
END

FUNCTION (h *ApplicationsHandler) GetByJobID(w, r)
    jobID := r.URL.Query().Get("job_id")
    apps, err := h.repo.GetByJobID(r.Context(), jobID)
    writeJSON(w, 200, apps)
END
```

**Acceptance Criteria:**
- [ ] All endpoints work
- [ ] JSON format correct
- [ ] Error handling proper

---

### Task 6.3: Implement Send Endpoint

**File:** `internal/web/handlers/applications.go`

**Description:** POST `/api/v1/applications/{id}/send`.

**Pseudo Code:**
```go
FUNCTION (h *ApplicationsHandler) Send(w, r)
    id := chi.URLParam(r, "id")
    var req struct {
        Channel   string `json:"channel"`
        Recipient string `json:"recipient"`
    }
    decodeJSON(r.Body, &req)

    // Get application
    app, err := h.repo.GetByID(r.Context(), id)
    IF err != nil { writeError(w, 404); RETURN }

    // Send asynchronously
    go h.dispatcher.SendApplication(r.Context(), &SendRequest{
        JobID:     app.JobID,
        Channel:   req.Channel,
        Recipient: req.Recipient,
    })

    writeJSON(w, 202, map[string]string{"status": "sending"})
END
```

**Acceptance Criteria:**
- [ ] Returns 202 Accepted
- [ ] Triggers background send
- [ ] Returns 404 for invalid ID

---

### Task 6.4: Implement UpdateDelivery Endpoint

**File:** `internal/web/handlers/applications.go`

**Description:** PATCH `/api/v1/applications/{id}/delivery` for manual status updates.

**Pseudo Code:**
```go
FUNCTION (h *ApplicationsHandler) UpdateDelivery(w, r)
    id := chi.URLParam(r, "id")
    var req struct {
        Status string `json:"status"`  // DELIVERED, READ, RESPONDED
        Notes  string `json:"notes"`
    }
    decodeJSON(r.Body, &req)

    // Update status
    err := h.repo.UpdateDeliveryStatus(r.Context(), id, req.Status)
    IF err != nil { writeError(w, 500); RETURN }

    // Store notes if provided
    IF req.Notes != "" {
        // TODO: Store in recruiter_response field
    }

    writeJSON(w, 200, map[string]string{"status": "updated"})
END
```

**Acceptance Criteria:**
- [ ] Manual status updates work
- [ ] Notes stored
- [ ] WebSocket event emitted

---

### Task 6.5: Register Routes

**File:** `internal/web/server.go`

**Description:** Add routes to server.

**Pseudo Code:**
```go
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

**Acceptance Criteria:**
- [ ] All routes accessible
- [ ] Route parameters parsed
- [ ] Server compiles

---

### Task 6.6: Write Integration Tests

**File:** `internal/web/handlers/applications_test.go`

**Description:** Test full API flow.

**Test Cases:**
- [ ] POST /api/v1/applications creates record
- [ ] POST /api/v1/applications/{id}/send triggers send
- [ ] GET /api/v1/applications/{id} returns status
- [ ] PATCH /api/v1/applications/{id}/delivery updates status

**Acceptance Criteria:**
- [ ] Integration tests pass
- [ ] Test database used

---

## Stage 6: UI Integration

### Task 7.1: Add Send Buttons

**File:** `static/partials/job_applications.html` (new)

**Description:** Add "Send to Telegram" button to job detail.

**Pseudo Code:**
```html
<!-- In job detail template -->
<div class="application-actions">
    <button
        class="btn btn-primary"
        onclick="openSendModal('TG_DM')">
        Send to Telegram
    </button>
    <button
        class="btn btn-secondary"
        onclick="openSendModal('EMAIL')"
        disabled
        title="Email not implemented yet">
        Send Email (Coming Soon)
    </button>
</div>
```

**Acceptance Criteria:**
- [ ] Button visible on job detail
- [ ] Email button disabled with tooltip
- [ ] Opens modal on click

---

### Task 7.2: Add Recipient Input Modal

**File:** `static/partials/job_applications.html`

**Description:** Modal for entering recipient.

**Pseudo Code:**
```html
<!-- Modal for recipient input -->
<dialog id="send-modal" class="modal">
    <div class="modal-content">
        <h3>Send Application</h3>
        <form onsubmit="sendApplication(event)">
            <label>Channel</label>
            <select id="send-channel" disabled>
                <option value="TG_DM">Telegram DM</option>
                <option value="EMAIL">Email (Not Implemented)</option>
            </select>

            <label>Recipient</label>
            <input
                type="text"
                id="recipient-input"
                placeholder="@recruiter"
                required>

            <div class="modal-actions">
                <button type="button" onclick="closeSendModal()">Cancel</button>
                <button type="submit" class="btn btn-primary">Send</button>
            </div>
        </form>
    </div>
</dialog>

<script>
FUNCTION openSendModal(channel)
    document.getElementById("send-modal").showModal()
    document.getElementById("send-channel").value = channel
END

FUNCTION closeSendModal()
    document.getElementById("send-modal").close()
END

FUNCTION sendApplication(event)
    event.preventDefault()
    recipient = document.getElementById("recipient-input").value
    channel = document.getElementById("send-channel").value

    fetch("/api/v1/applications/{job_id}/send", {
        method: "POST",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({channel: channel, recipient: recipient})
    })
    .then(r => r.json())
    .then(data => {
        closeSendModal()
        showStatus("Sending...")
    })
    .catch(err => showError(err))
END
</script>
```

**Acceptance Criteria:**
- [ ] Modal opens on button click
- [ ] Recipient input validated
- [ ] Send triggered on submit
- [ ] Channel shows correct option

---

### Task 7.3: Show Delivery Progress

**File:** `static/partials/job_applications.html`

**Description:** Display real-time progress via WebSocket.

**Pseudo Code:**
```html
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
<span class="status-badge status-{status}" id="status-{app_id}">{status}</span>

<!-- Progress bar (hidden by default) -->
<div class="progress-bar" id="progress-{app_id}" style="display: none;">
    <div class="progress-fill" style="width: 0%"></div>
    <span class="progress-text">Uploading...</span>
</div>
```

**Acceptance Criteria:**
- [ ] Status updates on WebSocket event
- [ ] Progress bar updates
- [ ] Error shown on failure
- [ ] Events match Thread A's tracker output

---

### Task 7.4: Display Application History

**File:** `static/partials/job_applications.html`

**Description:** Show applications for a job.

**Pseudo Code:**
```html
<!-- Applications list in job detail -->
<div class="applications-list">
    <h3>Applications</h3>
    <table>
        <thead>
            <tr>
                <th>Date</th>
                <th>Channel</th>
                <th>Recipient</th>
                <th>Status</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            <!-- FOR EACH application IN applications -->
            <tr>
                <td>{app.created_at | date}</td>
                <td>{app.delivery_channel}</td>
                <td>{app.recipient}</td>
                <td>
                    <span class="status-badge status-{app.delivery_status}">
                        {app.delivery_status}
                    </span>
                </td>
                <td>
                    <button onclick="updateStatus('{app.id}', 'DELIVERED')">
                        Mark Delivered
                    </button>
                    <button onclick="updateStatus('{app.id}', 'READ')">
                        Mark Read
                    </button>
                </td>
            </tr>
        </tbody>
    </table>
</div>
```

**Acceptance Criteria:**
- [ ] List shows all applications
- [ ] Empty state shown when none
- [ ] Status badges colored correctly
- [ ] Sorted by date (newest first)

---

### Task 7.5: Add Status Update UI

**File:** `static/partials/job_applications.html`

**Description:** Allow manual status updates.

**Pseudo Code:**
```html
<!-- Status update buttons (inline or dropdown) -->
<div class="status-actions">
    <select onchange="updateStatus('{app.id}', this.value)">
        <option value="">Update Status...</option>
        <option value="DELIVERED">Mark Delivered</option>
        <option value="READ">Mark Read</option>
        <option value="RESPONDED">Mark Responded</option>
    </select>
</div>

<script>
FUNCTION updateStatus(appID, status)
    IF !status RETURN

    fetch(`/api/v1/applications/${appID}/delivery`, {
        method: "PATCH",
        headers: {"Content-Type": "application/json"},
        body: JSON.stringify({status: status})
    })
    .then(r => r.json())
    .then(data => {
        showNotification("Status updated to " + status)
    })
    .catch(err => showError(err))
END
</script>
```

**Acceptance Criteria:**
- [ ] Buttons/dropdown update status
- [ ] WebSocket event emitted
- [ ] UI updates after change
- [ ] Only shown for SENT applications

---

## Stage 7: Service Status API

### Task 8.1: Implement Status Endpoint

**File:** `internal/web/handlers/dispatcher.go`

**Description:** GET `/api/v1/dispatcher/status`.

**Pseudo Code:**
```go
FUNCTION GetStatus(w, r)
    tgStatus := telegramManager.GetStatus()
    pendingCount, _ := repo.ListPending(r.Context(), 100)
    sendingCount, _ := repo.ListByStatus(r.Context(), "SENDING")

    status := map[string]interface{}{
        "dispatcher_enabled": true,
        "telegram": map[string]string{
            "status": string(tgStatus),
        },
        "queue": map[string]interface{}{
            "pending":    len(pendingCount),
            "processing": len(sendingCount),
        },
        "rate_limit": map[string]interface{}{
            "available": 1,  // TODO: from rate limiter
            "reset_at":   time.Now().Add(10 * time.Second).Format(time.RFC3339),
        },
    }
    writeJSON(w, 200, status)
END
```

**Acceptance Criteria:**
- [ ] Returns 200 with status
- [ ] Telegram status reflects actual state
- [ ] Queue counts accurate

---

### Task 8.2: Add Status to Settings UI

**File:** `static/pages/settings.html`

**Description:** Display status on settings page.

**Pseudo Code:**
```html
<!-- In settings page -->
<div class="dispatcher-status">
    <h3>Dispatcher Status</h3>

    <div class="status-card">
        <div class="status-item">
            <span>Telegram:</span>
            <span class="status-badge status-{telegram_status}">{telegram_status}</span>
        </div>
        <div class="status-item">
            <span>Queue:</span>
            <span>{pending} pending, {processing} processing</span>
        </div>
        <div class="status-item">
            <span>Rate Limit:</span>
            <span>{available} available (resets at {reset_at})</span>
        </div>
    </div>
</div>

<script>
// Poll status every 30 seconds
setInterval(() => {
    fetch("/api/v1/dispatcher/status")
        .then(r => r.json())
        .then(data => {
            updateStatusUI(data)
        })
}, 30000)

FUNCTION updateStatusUI(data)
    // Update each status element
END
</script>
```

**Acceptance Criteria:**
- [ ] Status displayed correctly
- [ ] Auto-refreshes every 30 seconds
- [ ] Visual indicators (READY = green, etc.)

---

## Completion Checklist

Thread B is complete when:

- [ ] `internal/dispatcher/telegram_sender.go` with full send flow
- [ ] `internal/dispatcher/service.go` orchestrator
- [ ] `internal/web/handlers/applications.go` REST handlers
- [ ] `internal/web/handlers/dispatcher.go` status handler
- [ ] `static/partials/job_applications.html` UI components
- [ ] Routes registered in server.go
- [ ] All tests passing
- [ ] UI shows real-time progress via WebSocket
- [ ] Service status visible in settings page
- [ ] **INTEGRATION TEST:** Test full flow with Thread A components

---

## WebSocket Events You Listen For

Ensure your UI handles these events from Thread A's tracker:

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

1. **Wait for Task 1.5 from Thread A** before integrating real Repository
2. **Wait for Task 2.4 from Thread A** before integrating real DeliveryTracker
3. **Wait for Task 2.5.5 coordination** - Thread A will add read tracking to your telegram_sender.go
4. **Start UI work early** - can use mock data while waiting for Thread A
5. **Final integration test** with Thread A to verify end-to-end flow

---

## Parallel Work Strategy

**While Waiting for Thread A:**
1. Create telegram_sender.go with interface mocks
2. Implement service.go with stub dependencies
3. Build UI components with mock data
4. Write test cases

**After Thread A Handoffs:**
1. Replace mocks with real Repository
2. Integrate real DeliveryTracker
3. Coordinate ReadTracker integration
4. Run full integration tests
