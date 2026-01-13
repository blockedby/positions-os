# service.go

Core scraping orchestration service.

- `Scrape()` — Main scraping loop with batch fetching, deduplication, job creation
- `ListTopics()` — Fetches forum topics for a channel
- `GetTelegramStatus()` — Returns Telegram client connection status
- Message filter integration via `RangesRepository.NewFilter()`
- NATS event publishing (`JobNewEvent`) after each job creation
- Safety limits: max 100 batches, 100ms delay between batches
- Creates `ScrapeResult` with statistics (TotalFetched, NewJobs, SkippedOld, SkippedEmpty, Errors)
