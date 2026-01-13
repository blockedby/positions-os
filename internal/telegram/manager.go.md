# manager.go

Telegram client lifecycle manager.

- Thread-safe singleton pattern
- `Start()` — Connects using session string
- `Stop()` — Graceful disconnect
- `GetClient()` — Access the client
- `GetStatus()` — Current connection state
- Handles FloodWait errors automatically
