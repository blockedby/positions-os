# ranges.go

Parsed message range tracking for deduplication.

**ParsedRange** — tracks min/max message IDs scraped per target

**Queries:**
- `GetRange()` — Fetch existing range for target
- `UpdateRange()` — Store new min/max
- `NewFilter()` — Create in-memory filter of known message IDs

**Purpose:** Prevent re-processing already scraped messages
