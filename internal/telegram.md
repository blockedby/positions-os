# telegram

Telegram API client wrapper using gotgproto.

## Core

- **client.go** → [client.go.md](client.go.md) — API client methods
- **manager.go** → [manager.go.md](manager.go.md) — Client lifecycle
- **factory.go** → [factory.go.md](factory.go.md) — Client initialization

## Auth

- **qr_client.go** → [qr_client.go.md](qr_client.go.md) — QR code authentication
- **session_converter.go** → [session_converter.go.md](session_converter.go.md) — Session import/export

## Support

- **types.go** → [types.go.md](types.go.md) — Data structures
- **ratelimit.go** → [ratelimit.go.md](ratelimit.go.md) — FloodWait handling

## Tests

- **client_test.go**, **manager_test.go** — Core tests
- **qr_test.go**, **persistence_test.go** — Auth tests
- **session_converter_test.go**, **types_test.go** — Unit tests
