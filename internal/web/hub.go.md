# hub.go

WebSocket connection hub for real-time updates.

- Broadcasts messages to all connected clients
- `Register()` — Add client
- `Unregister()` — Remove client
- `Broadcast()` — Send to all clients
- Each client runs its own read/write pump goroutine
- Used for live scrape status updates
