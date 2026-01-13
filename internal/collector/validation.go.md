# validation.go

Request validation and DTOs for scraping.

- `ScrapeRequest` — Scraping request with TargetID, Channel, Limit, Until, TopicIDs
- `ScrapeResponse` — Response with ScrapeID, Status, Target, StartedAt
- `TargetInfo` — Brief target info (ID, Name, Channel)
- `Validate()` — Validates request (channel/limit/until date)
- `UntilTime()` — Parses "YYYY-MM-DD" to `*time.Time`
- Validation errors: `ErrChannelRequired`, `ErrInvalidDate`, `ErrFutureDate`, etc.
