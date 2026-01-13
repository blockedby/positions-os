# client.go

Telegram API client wrapper using gotgproto.

**Client** wraps gotgproto Client
- `ResolveChannel()` — Convert username to Channel info
- `GetMessages()` — Fetch messages by offset/limit
- `GetTopics()` — List forum topics
- `GetStatus()` — Connection status check

**Status** values:
- `StatusDisconnected` — Not connected
- `StatusConnecting` — In progress
- `StatusReady` — Authenticated and ready
- `StatusError` — Failed state
