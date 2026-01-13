# handler.go

HTTP request handlers for collector API.

- `Health` — GET /health — Status check
- `StartScrape` — POST /api/v1/scrape/telegram — Start scraping job
- `StopScrape` — DELETE /api/v1/scrape/current — Stop current job
- `Status` — GET /api/v1/scrape/status — Get current job status
- `ListTargets` — GET /api/v1/targets — List all scraping targets
- `CreateTarget` — POST /api/v1/targets — Create new target
- `ListForumTopics` — GET /api/v1/tools/telegram/topics — Get forum topics
