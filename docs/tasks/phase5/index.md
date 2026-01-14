# Phase 5: Dispatcher - Task Split for 2 Developers

This directory contains parallel task tracks for implementing Phase 5: Dispatcher.

## Quick Reference

| Thread | Developer | Focus | Tasks | Est. Duration |
|--------|-----------|-------|-------|--------------|
| **Thread A** | Developer A | Backend Infrastructure | Repository, Tracker, Read Detection | ~60% |
| **Thread B** | Developer B | Service & API | Sender, Service, REST, UI | ~40% |

---

## Task Files

- **[Thread A: Backend Infrastructure](thread-a-backend-infrastructure.md)** - Database layer, status tracking, read receipts
- **[Thread B: Service & API](thread-b-service-api.md)** - Telegram sender, orchestrator, REST API, UI

---

## Parallel Execution Strategy

```
Timeline:
─────────────────────────────────────────────────────────────────────────►

Thread A:  [1. Repo]──[2. Tracker]──[2.5 ReadTracker]──[Email Stub]
              │            │                │
              ▼            ▼                ▼
Thread B:     [3. Sender]──[5. Service]──[6. API]──[7. UI]──[8. Status]
              │            │            │
              └────────────┴────────────┘
              Integration Points

Legend:
[1. Repo]        = Tasks 1.1-1.5 (Repository Layer)
[2. Tracker]     = Tasks 2.1-2.4 (Delivery Tracker)
[2.5 ReadTracker]= Tasks 2.5.1-2.5.6 (Read Receipt Tracker)
[3. Sender]      = Tasks 3.1-3.6 (Telegram Sender)
[5. Service]     = Tasks 5.1-5.4 (Dispatcher Service)
[6. API]         = Tasks 6.1-6.6 (REST API)
[7. UI]          = Tasks 7.1-7.5 (UI Integration)
[8. Status]      = Tasks 8.1-8.2 (Service Status API)
```

---

## Critical Handoffs

### Handoff 1: Repository Complete
**When:** Thread A completes Task 1.5
**Action:** Thread A notifies Thread B
**Thread B can:** Start integrating real Repository into Telegram Sender

### Handoff 2: Delivery Tracker Complete
**When:** Thread A completes Task 2.4
**Action:** Thread A notifies Thread B
**Thread B can:** Start integrating real DeliveryTracker into Telegram Sender

### Handoff 3: Read Tracker Integration
**When:** Thread A reaches Task 2.5.5, Thread B at Task 3.4
**Action:** Thread A adds registration code to Thread B's telegram_sender.go
**Both:** Coordinate the message ID registration after successful send

---

## Task Overview

### Thread A Tasks

| Stage | Tasks | Description |
|-------|-------|-------------|
| 1 | 1.1-1.5 | Repository Layer (Create, Get, Update, ListPending) |
| 2 | 2.1-2.4 | Delivery Tracker (Status tracking, WebSocket events) |
| 2.5 | 2.5.1-2.5.6 | Read Receipt Tracker (Telegram read detection) |
| - | E.1 | Email Sender Stub |

### Thread B Tasks

| Stage | Tasks | Description |
|-------|-------|-------------|
| 3 | 3.1-3.6 | Telegram Sender (Upload, send, rate limiting) |
| 4 | 5.1-5.4 | Dispatcher Service (Orchestration, routing) |
| 5 | 6.1-6.6 | REST API (CRUD, send endpoint, status) |
| 6 | 7.1-7.5 | UI Integration (Buttons, modals, progress) |
| 7 | 8.1-8.2 | Service Status API (Health, monitoring) |

---

## Files Created by Each Thread

### Thread A Creates
```
internal/repository/
├── applications.go         # Repository implementation
└── applications_test.go    # Repository tests

internal/dispatcher/
├── tracker.go              # Delivery tracking
├── tracker_test.go         # Tracker tests
├── read_tracker.go         # Read receipt detection
├── read_tracker_test.go    # Read tracker tests
└── email_sender.go         # Email sender stub

internal/telegram/
└── manager.go              # MODIFIED: Add read tracker hook
```

### Thread B Creates
```
internal/dispatcher/
├── telegram_sender.go      # Telegram DM sender
├── telegram_sender_test.go # Sender tests
├── service.go              # Dispatcher service
├── service_test.go         # Service tests
└── types.go                # Shared types

internal/web/handlers/
├── applications.go         # REST handlers
├── applications_test.go    # API tests
└── dispatcher.go           # Status endpoint handler

internal/web/
└── server.go               # MODIFIED: Register routes

static/partials/
└── job_applications.html   # UI components

static/pages/
└── settings.html           # MODIFIED: Add status display
```

---

## Starting the Work

### Developer A (Thread A)
1. Start immediately with Task 1.1 (Repository)
2. Work through Stage 1 sequentially
3. Notify Developer B at Task 1.5 completion
4. Continue to Stage 2 (Delivery Tracker)
5. Coordinate at Task 2.5.5 for read tracker integration

### Developer B (Thread B)
1. Start with Task 3.1 using **interface mocks** for Repository/Tracker
2. Implement Telegram Sender with stubs initially
3. Work on UI (Tasks 7.1-7.2) with mock data while waiting
4. Replace stubs with real implementations after Thread A handoffs
5. Complete integration tests after both threads done

---

## Integration Testing

**After both threads complete:**

1. **End-to-End Flow Test**
   - Create application via API
   - Send via Telegram DM
   - Verify status updates in UI
   - Confirm read detection works

2. **WebSocket Events Test**
   - Subscribe to WebSocket
   - Trigger send operation
   - Verify all events received

3. **Error Handling Test**
   - Invalid username
   - Missing PDF file
   - FloodWait scenario

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
- [ ] Automatic read detection works via `updateReadHistoryOutbox`
- [ ] Manual status updates work (DELIVERED, READ, RESPONDED)
- [ ] Email sender stub returns "not implemented"

---

## Questions?

- Refer to the full implementation plan: `../../phase-5-dispatcher.md`
- Coordinate via shared status board or daily sync
- Raise blockers immediately to avoid delays
