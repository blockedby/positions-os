# events.go

Event types for WebSocket communication.

**Event** types:
- `EventTypeScrapeStatus` — Scrape job status updates
- `EventTypeJobNew` — New job created
- `EventTypeError` — Error notifications

Events serialized as JSON to WebSocket clients.
