# targets.go

Scraping target repository.

**Queries:**
- `Create()` — Add new target
- `GetByID()` — Fetch by UUID
- `GetByURL()` — Find existing by channel URL
- `GetActive()` — List all active targets
- `UpdateTelegramInfo()` — Store channel_id, access_hash
- `UpdateLastScraped()` — Record scrape progress
