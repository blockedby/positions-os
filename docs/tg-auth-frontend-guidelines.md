# Telegram Authentication Frontend Guidelines

## Overview

This document describes the complete Telegram QR authentication flow as implemented in the Go+HTMX version of Positions OS. Use this as a reference when implementing authentication in any frontend (React, Vue, etc.).

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Session String Lifecycle](#2-session-string-lifecycle)
3. [QR Authentication Flow](#3-qr-authentication-flow)
4. [API Endpoints](#4-api-endpoints)
5. [WebSocket Events](#5-websocket-events)
6. [Frontend Implementation (Reference)](#6-frontend-implementation-reference)
7. [State Machine](#7-state-machine)
8. [Common Pitfalls](#8-common-pitfalls)
9. [Testing Checklist](#9-testing-checklist)
10. [Session Invalidation Detection (Implementation Plan)](#10-session-invalidation-detection-implementation-plan)

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                           FRONTEND                                   │
│  ┌─────────────┐    ┌──────────────┐    ┌────────────────────────┐ │
│  │  Settings   │    │  WebSocket   │    │   QR Code Display      │ │
│  │    Page     │───▶│   Client     │───▶│   (qrcodejs lib)       │ │
│  └─────────────┘    └──────────────┘    └────────────────────────┘ │
│         │                  ▲                                        │
│         │                  │ events (tg_qr, tg_auth_success)       │
│         ▼                  │                                        │
│  ┌─────────────┐    ┌──────────────┐                               │
│  │ HTTP POST   │    │   Hub        │                               │
│  │ /api/v1/    │    │  Broadcast   │                               │
│  │ auth/qr     │    │              │                               │
│  └─────────────┘    └──────────────┘                               │
└────────────┼───────────────▲────────────────────────────────────────┘
             │               │
             ▼               │
┌────────────────────────────┼────────────────────────────────────────┐
│                      BACKEND                                         │
│  ┌─────────────┐    ┌──────────────┐    ┌────────────────────────┐ │
│  │ AuthHandler │───▶│   Manager    │───▶│   QR Client (gotd)     │ │
│  │ StartQR()   │    │   StartQR()  │    │   auth.exportLoginToken│ │
│  └─────────────┘    └──────────────┘    └────────────────────────┘ │
│                            │                        │               │
│                            │                        ▼               │
│                            │            ┌────────────────────────┐ │
│                            ▼            │   Telegram MTProto     │ │
│                     ┌──────────────┐    │   (QR Login API)       │ │
│                     │  PostgreSQL  │    └────────────────────────┘ │
│                     │  sessions    │                               │
│                     │  table       │                               │
│                     └──────────────┘                               │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Components

| Component | Location | Role |
|-----------|----------|------|
| AuthHandler | `internal/web/handlers/auth.go` | HTTP endpoints for auth status & QR initiation |
| Manager | `internal/telegram/manager.go` | Manages client lifecycle, QR flow coordination |
| QRClient | `internal/telegram/qr_client.go` | Raw gotd client for QR authentication |
| Hub | `internal/web/hub.go` | WebSocket broadcast to all connected clients |
| SessionConverter | `internal/telegram/session_converter.go` | Converts gotd session to gotgproto format |

---

## 2. Session String Lifecycle

### Session States

```
┌─────────────────┐
│  NO SESSION     │  (fresh install, database empty)
│  Status:        │
│  UNAUTHORIZED   │
└────────┬────────┘
         │
         │ User clicks "Connect Telegram"
         │ POST /api/v1/auth/qr
         ▼
┌─────────────────┐
│  QR DISPLAYED   │  (waiting for user to scan)
│  Status:        │
│  UNAUTHORIZED   │
│  qr_in_progress:│
│  true           │
└────────┬────────┘
         │
         │ User scans QR with Telegram app
         │ Backend receives updateLoginToken
         ▼
┌─────────────────┐
│  AUTHENTICATED  │  (session saved to DB)
│  Status:        │
│  READY          │
└────────┬────────┘
         │
         │ App restart / container restart
         │ Manager.Init() loads session from DB
         ▼
┌─────────────────┐
│  SESSION        │  (persistent across restarts)
│  RESTORED       │
│  Status: READY  │
└─────────────────┘
```

### Session Storage

Sessions are stored in PostgreSQL in the `sessions` table:

```sql
CREATE TABLE sessions (
    version INTEGER PRIMARY KEY,  -- Always 1 (latest version)
    data    BYTEA                 -- JSON-serialized session.Data
);
```

The `data` column contains:

```json
{
  "Config": {...},
  "DC": 2,
  "Addr": "149.154.167.50:443",
  "AuthKey": "<base64 encoded 256 bytes>",
  "AuthKeyID": "<base64 encoded 8 bytes>",
  "Salt": 1234567890
}
```

### Session Conversion

When saving a session from QR auth:

1. gotd returns raw `session.Data` from `session.Loader`
2. Backend serializes to JSON
3. Wraps in `storage.Session{Version: 1, Data: jsonBytes}`
4. Saves to DB via GORM

When restoring session on startup:

1. `gotgproto.NewClient` reads from `sessions` table via `SqlSession`
2. Automatically deserializes and authenticates
3. No user interaction required

---

## 3. QR Authentication Flow

### Sequence Diagram

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│ Frontend │     │  Hub/WS  │     │ Handler  │     │ Manager  │     │ Telegram │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                │                │
     │ POST /auth/qr  │                │                │                │
     │───────────────────────────────▶│                │                │
     │                │                │                │                │
     │                │                │ StartQR()     │                │
     │                │                │──────────────▶│                │
     │                │                │                │                │
     │                │                │                │ exportLoginToken
     │                │                │                │───────────────▶│
     │                │                │                │                │
     │                │                │                │◀───────────────│
     │                │                │                │   token.URL()  │
     │                │                │                │                │
     │                │  Broadcast     │◀───────────────│                │
     │                │  {type:"tg_qr"}│                │                │
     │                │◀───────────────│                │                │
     │                │                │                │                │
     │◀───────────────│                │                │                │
     │  WS: tg_qr     │                │                │                │
     │                │                │                │                │
     │ Display QR     │                │                │                │
     │ (30s timer)    │                │                │                │
     │                │                │                │                │
     │                │                │                │ User scans QR  │
     │                │                │                │◀───────────────│
     │                │                │                │ updateLoginToken
     │                │                │                │                │
     │                │                │                │ Auth success   │
     │                │                │                │◀───────────────│
     │                │                │                │                │
     │                │                │◀───────────────│ saveSessionToDB
     │                │                │                │ Manager.Init() │
     │                │                │                │                │
     │                │  Broadcast     │◀───────────────│                │
     │                │  {type:        │                │                │
     │                │  "tg_auth_     │                │                │
     │                │   success"}    │                │                │
     │                │◀───────────────│                │                │
     │◀───────────────│                │                │                │
     │  WS: success   │                │                │                │
     │                │                │                │                │
     │ Hide QR        │                │                │                │
     │ Show Connected │                │                │                │
```

### QR Token Lifecycle

1. **Generation**: Backend calls `auth.exportLoginToken` via gotd
2. **Validity**: ~30 seconds (Telegram-imposed limit)
3. **Refresh**: If not scanned, new token generated automatically
4. **Format**: `tg://login?token=<base64url encoded token>`

---

## 4. API Endpoints

### GET /api/v1/auth/status

Returns current Telegram connection status.

**Response:**

```json
{
  "status": "READY" | "UNAUTHORIZED" | "INITIALIZING" | "ERROR",
  "is_ready": true | false,
  "qr_in_progress": true | false
}
```

### POST /api/v1/auth/qr

Initiates QR login flow. QR codes are delivered via WebSocket, not in HTTP response.

**Responses:**

| Status | Body | Meaning |
|--------|------|---------|
| 200 | `{"status": "started"}` | QR flow initiated, watch WebSocket for QR |
| 202 | `{"status": "already in progress"}` | QR flow already running |
| 400 | `{"error": "already logged in"}` | Already authenticated |

### GET /api/v1/scrape/status

Alternative endpoint that includes Telegram status (for backwards compatibility).

**Response:**

```json
{
  "telegram_status": "READY" | "UNAUTHORIZED" | ...,
  "scraping_active": false,
  ...
}
```

---

## 5. WebSocket Events

### Connection

```javascript
const protocol = location.protocol === "https:" ? "wss:" : "ws:";
const ws = new WebSocket(`${protocol}//${location.host}/ws`);
```

### Event Types

#### tg_qr - QR Code Generated

```json
{
  "type": "tg_qr",
  "url": "tg://login?token=abc123..."
}
```

Frontend should:
1. Display QR code using the URL
2. Start 30-second countdown timer
3. Save URL to localStorage for page reload recovery

#### tg_auth_success - Authentication Successful

```json
{
  "type": "tg_auth_success"
}
```

Frontend should:
1. Hide QR code
2. Clear countdown timer
3. Clear localStorage
4. Update status to "Connected"
5. Hide "Connect" button

#### error - Authentication Failed

```json
{
  "type": "error",
  "message": "QR auth flow failed: timeout"
}
```

Frontend should:
1. Hide QR code
2. Clear timer and localStorage
3. Show error message
4. Re-enable "Connect" button

### WebSocket Message Handler (Reference)

```javascript
conn.onmessage = function(evt) {
  try {
    const msg = JSON.parse(evt.data);

    if (msg.type === "tg_qr") {
      // Dispatch custom event for auth component
      document.dispatchEvent(new CustomEvent("tg_qr", { detail: msg }));
    } else if (msg.type === "tg_auth_success") {
      document.dispatchEvent(new Event("tg_auth_success"));
    } else if (msg.type === "error") {
      document.dispatchEvent(new CustomEvent("tg_auth_error", { detail: msg }));
    }
  } catch (e) {
    console.error("WS parse error:", e);
  }
};
```

---

## 6. Frontend Implementation (Reference)

### Go+HTMX Version

Located in `internal/web/templates/pages/settings.html`.

#### Key Features

1. **QR Code Display**: Uses `qrcodejs` library
2. **30-Second Timer**: Visual countdown for QR expiry
3. **LocalStorage Persistence**: Survives page reloads during auth
4. **Debounced Button**: Prevents double-click spam

#### LocalStorage Keys

| Key | Value | Purpose |
|-----|-------|---------|
| `tg_qr_url` | QR URL string | Recover QR after page reload |
| `tg_qr_timestamp` | Unix timestamp | Calculate remaining validity |

#### QR Persistence Module

```javascript
const QR_PERSISTENCE = {
  STORAGE_KEY_URL: "tg_qr_url",
  STORAGE_KEY_TIMESTAMP: "tg_qr_timestamp",
  QR_EXPIRY_SECONDS: 30,

  save: function(url) {
    localStorage.setItem(this.STORAGE_KEY_URL, url);
    localStorage.setItem(this.STORAGE_KEY_TIMESTAMP, Date.now().toString());
  },

  load: function() {
    const url = localStorage.getItem(this.STORAGE_KEY_URL);
    const timestamp = localStorage.getItem(this.STORAGE_KEY_TIMESTAMP);
    if (!url || !timestamp) return null;

    const ageSeconds = (Date.now() - parseInt(timestamp)) / 1000;
    if (ageSeconds > this.QR_EXPIRY_SECONDS) {
      this.clear();
      return null;
    }
    return { url, ageSeconds: Math.floor(ageSeconds) };
  },

  clear: function() {
    localStorage.removeItem(this.STORAGE_KEY_URL);
    localStorage.removeItem(this.STORAGE_KEY_TIMESTAMP);
  }
};
```

#### Initialization Flow

```javascript
function init() {
  // 1. Check current auth status
  fetch("/api/v1/scrape/status")
    .then(r => r.json())
    .then(data => {
      if (data.telegram_status === "READY") {
        // Already connected - show success state
        showConnected();
        QR_PERSISTENCE.clear();
      } else {
        // Not connected - check for saved QR
        const savedQR = QR_PERSISTENCE.load();
        if (savedQR) {
          // Restore QR with remaining time
          displayQR(savedQR.url, 30 - savedQR.ageSeconds);
        }
      }
    });
}
```

---

## 7. State Machine

### Frontend States

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  ┌─────────────┐                        ┌─────────────────────┐ │
│  │   LOADING   │──────────────────────▶│   DISCONNECTED      │ │
│  │             │   status != READY      │                     │ │
│  └─────────────┘                        │ - Show Connect btn  │ │
│        │                                │ - Check localStorage│ │
│        │ status == READY                │   for saved QR      │ │
│        ▼                                └──────────┬──────────┘ │
│  ┌─────────────┐                                   │            │
│  │  CONNECTED  │◀──────────────────────────────────┤            │
│  │             │   tg_auth_success event           │            │
│  │ - Hide btn  │                                   │            │
│  │ - Show OK   │                                   │            │
│  └─────────────┘                                   │            │
│        ▲                                           │            │
│        │                                           │            │
│        │                                           ▼            │
│        │                                ┌─────────────────────┐ │
│        │                                │   QR_DISPLAYED      │ │
│        │                                │                     │ │
│        │ tg_auth_success                │ - Show QR code      │ │
│        └────────────────────────────────│ - 30s countdown     │ │
│                                         │ - Save to storage   │ │
│                                         └──────────┬──────────┘ │
│                                                    │            │
│                                                    │ timeout    │
│                                                    ▼            │
│                                         ┌─────────────────────┐ │
│                                         │   QR_EXPIRED        │ │
│                                         │                     │ │
│                                         │ - Clear storage     │ │
│                                         │ - Show "Try again"  │ │
│                                         └─────────────────────┘ │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Backend States (Manager.status)

| Status | Meaning | Frontend Action |
|--------|---------|-----------------|
| `INITIALIZING` | App starting, checking DB for session | Show "Loading..." |
| `UNAUTHORIZED` | No valid session, need QR auth | Show "Connect" button |
| `READY` | Session valid, Telegram connected | Show "Connected" |
| `ERROR` | Connection failed | Show error, retry option |

---

## 8. Common Pitfalls

### Frontend Issues

#### 1. Duplicate QR Display

**Problem**: Same QR displayed multiple times on WebSocket reconnect.

**Solution**: Track last displayed URL, skip if identical:

```javascript
let lastQRUrl = null;

document.addEventListener("tg_qr", function(e) {
  if (e.detail.url === lastQRUrl) return; // Skip duplicate
  lastQRUrl = e.detail.url;
  displayQR(e.detail.url);
});
```

#### 2. QR Lost on Page Reload

**Problem**: User refreshes page while QR is displayed, loses progress.

**Solution**: Save QR URL and timestamp to localStorage, restore on init.

#### 3. Timer Desync After Tab Sleep

**Problem**: Browser throttles timers when tab is backgrounded.

**Solution**: Use timestamp-based calculation instead of interval counting:

```javascript
function getRemainingTime(timestamp) {
  const elapsed = (Date.now() - timestamp) / 1000;
  return Math.max(0, 30 - Math.floor(elapsed));
}
```

#### 4. Button Double-Click

**Problem**: User clicks "Connect" multiple times, creates race conditions.

**Solution**: Disable button immediately, re-enable after timeout:

```javascript
connectBtn.addEventListener("click", function() {
  if (connectBtn.disabled) return;
  connectBtn.disabled = true;
  setTimeout(() => { connectBtn.disabled = false; }, 3000);
});
```

### Backend Issues

#### 1. Dispatcher Passed by Value

**Problem**: `tg.UpdateDispatcher` passed by value, events go to wrong instance.

**Solution**: Always use pointer: `&tg.UpdateDispatcher{}` or `tg.NewUpdateDispatcher()`.

See `docs/summary-tg-auth-fix.md` for details.

#### 2. Blocking gotgproto Initialization

**Problem**: `gotgproto.NewClient` blocks waiting for phone input if no session.

**Solution**: Use dual-factory architecture:
- Check DB for existing session first
- Use raw gotd client for QR auth (no interactive prompts)
- Only use gotgproto after session is established

#### 3. WebSocket Connection Drop

**Problem**: WS connection drops during auth flow.

**Solution**: Implement reconnection with exponential backoff:

```javascript
function connect() {
  const ws = new WebSocket(wsUrl);
  ws.onclose = () => setTimeout(connect, 2000);
}
```

---

## 9. Testing Checklist

### Manual Testing

- [ ] Fresh install: "Connect" button visible, status shows "Disconnected"
- [ ] Click "Connect": QR code appears, 30s timer starts
- [ ] Scan QR with Telegram app: Success message, QR hides, status shows "Connected"
- [ ] Refresh page after auth: Status still shows "Connected" (session persisted)
- [ ] Refresh page during QR display: QR restored with correct remaining time
- [ ] Wait for QR to expire: Message shows "QR expired", button re-enabled
- [ ] Double-click "Connect": Only one QR flow starts
- [ ] Kill WebSocket connection: Reconnects automatically, auth still works
- [ ] Container restart: Session restored from DB, no re-auth needed

### Unit Tests

Located in `internal/telegram/`:

- `qr_client_test.go`: QR client creation doesn't block
- `manager_test.go`: Manager state transitions
- `session_converter_test.go`: Session format conversion

### Integration Tests

- [ ] Full QR flow with mock Telegram server
- [ ] Session persistence across app restarts
- [ ] Concurrent QR requests (should reject duplicates)

---

## 10. Session Invalidation Detection (Implementation Plan)

### Problem Statement

When a user terminates their Telegram session from another device (e.g., phone: Settings → Devices → Terminate Session), the backend doesn't detect this immediately. The frontend continues to show "Connected" until the next API call fails.

**Current Behavior:**
1. User is authenticated, frontend shows "Connected"
2. User terminates session from phone
3. Frontend still shows "Connected" (stale state)
4. Next API call fails with `AUTH_KEY_UNREGISTERED`
5. Only then does the user see an error

**Desired Behavior:**
1. Backend detects session invalidation proactively
2. Broadcasts `tg_session_revoked` event via WebSocket
3. Frontend immediately shows "Disconnected" and "Connect" button

### How Telegram Session Termination Works

According to [Telegram's Error Handling documentation](https://core.telegram.org/api/errors):

1. **AUTH_KEY_UNREGISTERED (401)**: The key is not registered in the system. Session was terminated.
2. **SESSION_REVOKED (401)**: Authorization invalidated because user terminated all sessions.
3. **AUTH_KEY_DUPLICATED**: Session invalidated due to auth key conflict (same key used from two connections).

When a session is terminated:
1. Telegram sends `updatesTooLong` to the client
2. Client should call `updates.getDifference`
3. Server responds with 401 `AUTH_KEY_UNREGISTERED`
4. Client should clear local session and prompt re-authentication

Reference: [gotd/td Issue #1456](https://github.com/gotd/td/issues/1456)

### Implementation Plan

#### Phase 1: Backend Health Check Service

**New file: `internal/telegram/health.go`**

```go
type HealthChecker struct {
    manager    *Manager
    hub        HubBroadcaster
    interval   time.Duration
    stopCh     chan struct{}
}

func NewHealthChecker(manager *Manager, hub HubBroadcaster) *HealthChecker {
    return &HealthChecker{
        manager:  manager,
        hub:      hub,
        interval: 30 * time.Second, // Check every 30s
        stopCh:   make(chan struct{}),
    }
}

func (h *HealthChecker) Start(ctx context.Context) {
    ticker := time.NewTicker(h.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-h.stopCh:
            return
        case <-ticker.C:
            h.checkHealth(ctx)
        }
    }
}

func (h *HealthChecker) checkHealth(ctx context.Context) {
    if h.manager.GetStatus() != StatusReady {
        return // Not connected, nothing to check
    }

    client := h.manager.GetClient()
    if client == nil {
        return
    }

    // Lightweight API call to verify session
    _, err := client.API().UsersGetFullUser(ctx, &tg.InputUserSelf{})
    if err != nil {
        if isSessionInvalid(err) {
            h.handleSessionRevoked()
        }
    }
}

func isSessionInvalid(err error) bool {
    errStr := err.Error()
    return strings.Contains(errStr, "AUTH_KEY_UNREGISTERED") ||
           strings.Contains(errStr, "SESSION_REVOKED") ||
           strings.Contains(errStr, "401")
}

func (h *HealthChecker) handleSessionRevoked() {
    // 1. Update manager status
    h.manager.SetStatus(StatusUnauthorized)

    // 2. Clear session from database
    h.manager.ClearSession()

    // 3. Broadcast to all connected clients
    if h.hub != nil {
        h.hub.Broadcast(map[string]string{
            "type":    "tg_session_revoked",
            "message": "Session terminated from another device",
        })
    }
}
```

**Changes to `internal/telegram/manager.go`:**

```go
// Add method to update status
func (m *Manager) SetStatus(status Status) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.status = status
    m.client = nil // Clear invalid client
}

// Add method to clear session from DB
func (m *Manager) ClearSession() error {
    return m.db.Exec("DELETE FROM sessions").Error
}
```

#### Phase 2: WebSocket Event

**New event type: `tg_session_revoked`**

```json
{
  "type": "tg_session_revoked",
  "message": "Session terminated from another device"
}
```

**Update `internal/web/templates/layout.html`:**

```javascript
// Add handler for session revocation
else if (msg.type === "tg_session_revoked") {
    document.dispatchEvent(new CustomEvent("tg_session_revoked", { detail: msg }));
}
```

#### Phase 3: React Frontend Implementation

**File: `frontend/src/hooks/useTelegramAuth.ts`**

```typescript
import { useState, useEffect, useCallback } from 'react';
import { useWebSocket } from './useWebSocket';

type TelegramStatus = 'loading' | 'connected' | 'disconnected' | 'qr_displayed' | 'error';

interface UseTelegramAuthReturn {
  status: TelegramStatus;
  qrUrl: string | null;
  error: string | null;
  startQRLogin: () => Promise<void>;
  checkStatus: () => Promise<void>;
}

export function useTelegramAuth(): UseTelegramAuthReturn {
  const [status, setStatus] = useState<TelegramStatus>('loading');
  const [qrUrl, setQrUrl] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const { lastMessage, isConnected } = useWebSocket();

  // Handle WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    switch (lastMessage.type) {
      case 'tg_qr':
        setQrUrl(lastMessage.url);
        setStatus('qr_displayed');
        // Persist to localStorage for page reload recovery
        localStorage.setItem('tg_qr_url', lastMessage.url);
        localStorage.setItem('tg_qr_timestamp', Date.now().toString());
        break;

      case 'tg_auth_success':
        setStatus('connected');
        setQrUrl(null);
        localStorage.removeItem('tg_qr_url');
        localStorage.removeItem('tg_qr_timestamp');
        break;

      case 'tg_session_revoked':
        // SESSION INVALIDATION HANDLING
        setStatus('disconnected');
        setQrUrl(null);
        setError(lastMessage.message || 'Session terminated');
        localStorage.removeItem('tg_qr_url');
        localStorage.removeItem('tg_qr_timestamp');
        // Show toast notification
        showToast('Telegram session terminated. Please reconnect.', 'warning');
        break;

      case 'error':
        setStatus('error');
        setError(lastMessage.message);
        setQrUrl(null);
        break;
    }
  }, [lastMessage]);

  // Check status on mount and restore QR if needed
  const checkStatus = useCallback(async () => {
    try {
      const response = await fetch('/api/v1/auth/status');
      const data = await response.json();

      if (data.is_ready) {
        setStatus('connected');
        localStorage.removeItem('tg_qr_url');
        localStorage.removeItem('tg_qr_timestamp');
      } else {
        setStatus('disconnected');

        // Try to restore QR from localStorage
        const savedUrl = localStorage.getItem('tg_qr_url');
        const savedTimestamp = localStorage.getItem('tg_qr_timestamp');

        if (savedUrl && savedTimestamp) {
          const age = (Date.now() - parseInt(savedTimestamp)) / 1000;
          if (age < 30) {
            setQrUrl(savedUrl);
            setStatus('qr_displayed');
          } else {
            localStorage.removeItem('tg_qr_url');
            localStorage.removeItem('tg_qr_timestamp');
          }
        }
      }
    } catch (err) {
      setStatus('error');
      setError('Failed to check status');
    }
  }, []);

  // Start QR login flow
  const startQRLogin = useCallback(async () => {
    try {
      setError(null);
      const response = await fetch('/api/v1/auth/qr', { method: 'POST' });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || 'Failed to start QR login');
      }

      // QR will arrive via WebSocket
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      setStatus('error');
    }
  }, []);

  // Initial status check
  useEffect(() => {
    checkStatus();
  }, [checkStatus]);

  return {
    status,
    qrUrl,
    error,
    startQRLogin,
    checkStatus,
  };
}
```

**File: `frontend/src/components/TelegramAuthCard.tsx`**

```tsx
import React, { useEffect, useState } from 'react';
import QRCode from 'qrcode.react';
import { useTelegramAuth } from '../hooks/useTelegramAuth';

export function TelegramAuthCard() {
  const { status, qrUrl, error, startQRLogin, checkStatus } = useTelegramAuth();
  const [timeLeft, setTimeLeft] = useState(30);
  const [isLoading, setIsLoading] = useState(false);

  // QR expiration timer
  useEffect(() => {
    if (status !== 'qr_displayed' || !qrUrl) {
      setTimeLeft(30);
      return;
    }

    // Calculate initial time from localStorage timestamp
    const timestamp = localStorage.getItem('tg_qr_timestamp');
    if (timestamp) {
      const elapsed = (Date.now() - parseInt(timestamp)) / 1000;
      setTimeLeft(Math.max(0, 30 - Math.floor(elapsed)));
    }

    const timer = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          clearInterval(timer);
          checkStatus(); // Re-check status when QR expires
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [status, qrUrl, checkStatus]);

  const handleConnect = async () => {
    setIsLoading(true);
    try {
      await startQRLogin();
    } finally {
      setTimeout(() => setIsLoading(false), 2000);
    }
  };

  return (
    <div className="card">
      <div className="card-header">
        <h2>Telegram Connection</h2>
      </div>

      <div className="card-body">
        {/* Status Display */}
        <div className="status-row">
          <span className="label">Status:</span>
          <StatusBadge status={status} />
        </div>

        {/* Error Message */}
        {error && (
          <div className="alert alert-error">
            {error}
          </div>
        )}

        {/* QR Code Display */}
        {status === 'qr_displayed' && qrUrl && (
          <div className="qr-container">
            <p>Scan with Telegram App</p>
            <div className="qr-code-wrapper">
              <QRCode value={qrUrl} size={200} />
            </div>
            <p className="qr-timer">
              Expires in <strong>{timeLeft}s</strong>
            </p>
          </div>
        )}

        {/* Connect Button */}
        {status === 'disconnected' && (
          <button
            onClick={handleConnect}
            disabled={isLoading}
            className="btn btn-primary"
          >
            {isLoading ? 'Connecting...' : 'Connect Telegram'}
          </button>
        )}

        {/* Connected State */}
        {status === 'connected' && (
          <div className="connected-indicator">
            <CheckIcon /> Connected
          </div>
        )}
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const config = {
    loading: { label: 'Loading...', className: 'badge-neutral' },
    connected: { label: 'Connected', className: 'badge-success' },
    disconnected: { label: 'Disconnected', className: 'badge-error' },
    qr_displayed: { label: 'Scan QR Code', className: 'badge-warning' },
    error: { label: 'Error', className: 'badge-error' },
  }[status] || { label: status, className: 'badge-neutral' };

  return <span className={`badge ${config.className}`}>{config.label}</span>;
}
```

#### Phase 4: Integration

**Update `cmd/collector/main.go`:**

```go
// After creating manager and hub
healthChecker := telegram.NewHealthChecker(manager, hub)
go healthChecker.Start(ctx)

// On shutdown
defer healthChecker.Stop()
```

### Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SESSION HEALTH MONITORING                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────┐        ┌─────────────────┐                        │
│  │ HealthChecker   │───────▶│   Telegram API  │                        │
│  │                 │        │                 │                        │
│  │ ticker: 30s     │◀───────│ UsersGetFullUser│                        │
│  └────────┬────────┘        └─────────────────┘                        │
│           │                                                             │
│           │ AUTH_KEY_UNREGISTERED (401)                                │
│           ▼                                                             │
│  ┌─────────────────┐        ┌─────────────────┐        ┌─────────────┐│
│  │ handleRevoked() │───────▶│  Hub.Broadcast  │───────▶│  WebSocket  ││
│  │                 │        │                 │        │  Clients    ││
│  │ - SetStatus()   │        │ tg_session_     │        └──────┬──────┘│
│  │ - ClearSession()│        │ revoked         │               │       │
│  └─────────────────┘        └─────────────────┘               │       │
│                                                                │       │
└────────────────────────────────────────────────────────────────┼───────┘
                                                                 │
                                                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          REACT FRONTEND                                  │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────┐        ┌─────────────────┐        ┌─────────────┐│
│  │ useWebSocket    │───────▶│ useTelegramAuth │───────▶│ AuthCard    ││
│  │                 │        │                 │        │ Component   ││
│  │ lastMessage:    │        │ case            │        │             ││
│  │ tg_session_     │        │ 'tg_session_    │        │ status:     ││
│  │ revoked         │        │  revoked':      │        │ disconnected││
│  └─────────────────┘        │   setStatus()   │        │             ││
│                             │   showToast()   │        │ Show        ││
│                             └─────────────────┘        │ "Connect"   ││
│                                                        └─────────────┘│
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Configuration Options

| Setting | Default | Description |
|---------|---------|-------------|
| `TG_HEALTH_CHECK_INTERVAL` | `30s` | How often to verify session validity |
| `TG_HEALTH_CHECK_ENABLED` | `true` | Enable/disable health checking |

### Testing Scenarios

- [ ] Health checker detects `AUTH_KEY_UNREGISTERED` error
- [ ] WebSocket broadcasts `tg_session_revoked` to all clients
- [ ] React component transitions to "disconnected" state
- [ ] Toast notification appears with appropriate message
- [ ] "Connect" button becomes visible
- [ ] LocalStorage QR data is cleared
- [ ] Session is cleared from database
- [ ] User can re-authenticate via QR flow

### References

- [Telegram Error Handling](https://core.telegram.org/api/errors) - Official 401 error codes
- [gotd/td Issue #1456](https://github.com/gotd/td/issues/1456) - AUTH_KEY_UNREGISTERED handling
- [Microservices Health Check Pattern](https://microservices.io/patterns/observability/health-check-api.html) - General health check best practices
- `docs/session-logout-root-cause-analysis.md` - Previous session invalidation analysis

---

## References

- `internal/web/handlers/auth.go` - HTTP endpoints
- `internal/telegram/manager.go` - Client lifecycle management
- `internal/telegram/qr_client.go` - Raw gotd QR client
- `internal/web/hub.go` - WebSocket broadcast
- `internal/web/templates/pages/settings.html` - Go+HTMX frontend reference
- `internal/web/templates/layout.html` - WebSocket event dispatcher
- `docs/telegram-qr-auth-guide.md` - Session format details
- `docs/summary-tg-auth-fix.md` - Historical bug fixes
