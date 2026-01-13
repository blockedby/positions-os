# router.go

Chi router configuration for collector API.

- Middleware: Logger, Recoverer, RequestID, CORS (open origins)
- Health check: GET /health
- API v1 routes: /api/v1/scrape/*, /api/v1/targets, /api/v1/tools/*
- Uses `github.com/go-chi/chi/v5` package
