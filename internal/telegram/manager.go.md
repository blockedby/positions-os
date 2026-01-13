# manager.go

Telegram client lifecycle manager — handles QR authentication, session persistence, and connection state.

## State Management

- **Status** values: `INITIALIZING`, `READY`, `UNAUTHORIZED`, `ERROR`
- Thread-safe status checks via `sync.RWMutex`

## Methods

- **Init()** — Restores session from database or returns `UNAUTHORIZED` if empty
- **StartQR()** — Initiates QR login flow with state protection
- **CancelQR()** — Cancels ongoing QR login flow
- **IsQRInProgress()** — Checks if QR flow is currently running
- **GetStatus()** — Returns current connection status
- **GetClient()** — Returns underlying gotgproto client
- **Stop()** — Graceful disconnect

## QR Flow Protection

- Atomic `qrInProgress` flag prevents concurrent QR requests
- `qrCancel` context allows cancellation of ongoing flow
- Only one QR flow can run at a time per server

## Session Persistence

- `saveSessionToDB()` — Converts gotd session.Data to gotgproto format
- Uses `ConvertToGotgprotoSession()` for proper JSON wrapping
- Session stored in `sessions` table with `Version=1`
