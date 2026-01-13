# config.go

Environment-based configuration loader for the application.

- `Config` struct holds all configuration (database, NATS, LLM, Telegram, HTTP, logging)
- `Load()` reads from environment variables with sensible defaults
- Helper functions: `getEnv()`, `getEnvInt()`, `getEnvFloat()`
- Default port: 3100, default NATS: nats://localhost:4222
