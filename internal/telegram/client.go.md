# client.go

Telegram API client wrapper — provides high-level Telegram operations using gotgproto.

## Methods

- **ResolveChannel()** — Convert username to Channel info with flood wait handling
- **GetMessages()** — Fetch messages by offset/limit (max 100)
- **GetTopics()** — List forum topics for a channel
- **GetTopicMessages()** — Fetch messages from a specific forum topic
- **ChannelExists()** — Check if channel exists and is accessible
- **GetStatus()** — Current connection status
- **StartQR()** — Proxy to Manager's QR login flow
- **IsQRInProgress()** — Check if QR login is running
- **CancelQR()** — Cancel ongoing QR login flow

## Rate Limiting

- Automatic rate limiting via `RateLimiter`
- FloodWait detection and backoff
- Per-request delay enforcement

## Status Values

- `INITIALIZING` — Client starting up
- `READY` — Authenticated and ready
- `UNAUTHORIZED` — No session, needs auth
- `ERROR` — Connection failed
