# server.go

HTTP server with Chi router.

- `NewServer()` — Creates server with middleware and routes
- Middleware: RequestID, RealIP, Logger, Recoverer, Timeout, Compress
- `Start()` — Begins listening on configured port
- `Stop()` — Graceful shutdown
- Serves static files from `/static/*`
- WebSocket endpoint at `/ws`
- Health check at `/health`
