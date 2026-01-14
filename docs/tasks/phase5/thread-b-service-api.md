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
