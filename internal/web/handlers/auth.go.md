# auth.go

Telegram authentication HTTP handler — manages QR code login flow.

## Endpoints

### POST /api/v1/auth/qr

Initiates QR code login flow.

**Request:** None (body empty)

**Response:**
- `200 OK` — `{"status":"started"}` — QR flow initiated
- `202 Accepted` — `{"status":"already in progress"}` — QR flow already running
- `400 Bad Request` — `{"error":"already logged in"}` — Already authenticated

## Behavior

1. Checks if user already logged in → returns 400
2. Checks if QR flow already in progress → returns 202
3. Starts QR flow in background goroutine
4. Returns immediately (async operation)

## WebSocket Events

Broadcasts to connected clients:
- `{"type":"tg_qr","url":"tg://login?token=..."}` — QR code generated
- `{"type":"tg_auth_success"}` — Authentication successful
- `{"type":"error","message":"..."}` — Authentication failed

## Flow Protection

- Only one QR flow can run at a time
- `IsQRInProgress()` check before starting new flow
- Ignores context.Canceled errors (normal user cancellation)
