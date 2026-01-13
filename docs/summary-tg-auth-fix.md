# Summary of Telegram QR Authentication Fix

## 1. Initial Problem

The user triggered the Telegram QR login flow, but the backend "hung" inside the Docker container.

- **Symptom 1**: The log showed "Enter Phone Number:", indicating the app was waiting for interactive console input, which is impossible in a containerized environment.
- **Symptom 2**: After scanning the QR code with the mobile app, the scanner showed "Auth token expired" or "Unknown Error", and the web frontend never updated to "Connected".

## 2. Root Cause Analysis

We identified two distinct issues causing this behavior:

1.  **Blocking Initialization**: The library `gotgproto` defaults to an interactive login flow (`ClientTypePhone("")`) if no session exists. This blocks the main thread waiting for stdin.
2.  **Silent Dispatcher Failure**: In the custom QR client implementation, the `tg.UpdateDispatcher` was being passed by **value** (copy) instead of by **pointer**.
    - _Why this matters_: The Telegram client received the "Login Successful" event from the server, but it dispatched it to the _copy_ of the dispatcher. The application's event handler (registered on the _original_ dispatcher) never received the signal. Consequently, the flow timed out, resulting in "Auth token expired".

## 3. The Solution Strategy (TDD)

We adopted a **Test-Driven Development (TDD)** approach to isolate and fix these issues without guessing.

### Phase 1: Test Plan & Repro

- Created `docs/test/tg-auth.md` detailing the attack plan.
- Wrote **Unit Tests**:
  - `TestQRClientFactory_DoesNotBlock`: Verified that client creation returns instantly (<2s).
  - `TestQRClient_DispatcherInstanceMatch`: A regression test that specifically checks if the client's dispatcher matches the bundle's dispatcher (catching the pointer bug).

### Phase 2: Implementation & Fixes

We refactored the architecture to separate "Session Restoration" from "New Authentication":

1.  **Dual-Factory Architecture** (`internal/telegram/manager.go`):

    - **Restoration**: Uses `gotgproto.NewClient` only when a session _already exists_ in the DB.
    - **QR Auth**: Uses a new lightweight `NewQRClient` (`internal/telegram/qr_client.go`) that uses the raw `gotd` library. This client is purely programmatic and never asks for console input.

2.  **Dispatcher Fix** (`internal/telegram/qr_client.go`):

    - Changed `dispatcher := tg.NewUpdateDispatcher()` to `dispatcher := &tg.UpdateDispatcher{}`.
    - This ensured the exact same memory address is shared between the Client and the Event Handler.

3.  **Session Persistence** (`internal/telegram/manager.go`):
    - Implemented `saveSessionToDB`, which takes the raw session data from `gotd`, converts it to JSON (`internal/telegram/session_converter.go`), and saves it to the `sessions` table in Postgres/SQLite.
    - This ensures that after a restart, the app automatically logs in without needing a rescan.

## 4. Result

- **Non-blocking**: The QR flow starts immediately.
- **Reliable Scanning**: Scanning the code instantly triggers the expected success flow.
- **Persistence**: Sessions survive container restarts.

## 5. Next Steps

- **Issue**: User reported WebSocket connections dropping in the browser after these changes.
- **Action**: Investigate `internal/web/hub.go` and `internal/web/handlers/auth.go` to ensure the QR flow completion doesn't accidentally terminate the WebSocket hub or client connection.
