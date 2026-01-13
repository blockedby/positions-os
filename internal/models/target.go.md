# target.go

Scraping target entity.

**ScrapingTarget** represents a source to scrape:
- `ID`, `Name`, `Type` (TG_CHANNEL, TG_FORUM)
- `URL` — Channel username or invite link
- `IsActive` — Enable/disable scraping
- `Metadata` — Telegram channel_id, access_hash
- `LastScrapedAt`, `LastScrapedMaxMsgID` — Progress tracking
