# session_converter.go

Session data conversion between gotd and gotgproto formats.

## The Problem

Gotgproto expects session JSON in a wrapped format:
```json
{"Version": 1, "Data": {"DC": 2, "Addr": "...", ...}}
```

Gotd's `session.Data` cannot be marshaled directly — it would produce flat JSON missing the wrapper.

## Solution

`ConvertToGotgprotoSession()` wraps gotd session data in the required structure:

```go
type jsonData struct {
    Version int         `json:"Version"`
    Data    session.Data `json:"Data"`
}
```

## Functions

- **ConvertToGotgprotoSession()** — Converts gotd `session.Data` to gotgproto `storage.Session`
  - Wraps data in `jsonData` struct
  - Marshals to JSON with `Version` and `Data` fields
  - Returns `*storage.Session` ready for database storage

## Database Format

Table: `sessions`
| Column | Type | Description |
|--------|------|-------------|
| version | INTEGER (PK) | Always 1 |
| data | BLOB | JSON: `{"Version":1,"Data":{...}}` |
