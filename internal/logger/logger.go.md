# logger.go

Structured logging wrapper around Zerolog.

- `New()` creates logger with level and optional file output
- Multi-writer support: console + file simultaneously
- Global logger singleton for convenience
- Helper functions: `Info()`, `Error()`, `Debug()` use global logger
- Console writer with timestamps, caller info
- Auto-creates log directory if needed
