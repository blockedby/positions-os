# manager.go

Thread-safe scrape job lifecycle manager.

- Ensures only one scrape job runs at a time (returns `ErrAlreadyRunning` if busy)
- Uses `context.Background()` for long-running jobs (not HTTP request context)
- `Start()` — Launches scrape in goroutine, returns immediately with `ScrapeJob`
- `Stop()` — Cancels running job via `context.CancelFunc`
- `Current()` — Returns currently running job or nil
- Important: HTTP handler returns before scrape completes (async pattern)
