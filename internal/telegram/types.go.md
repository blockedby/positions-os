# types.go

Telegram data structures.

**Message** — Parsed message from Telegram
- ID, ChannelID, Text, Date, TopicID
- Views, Forwards counts

**Topic** — Forum topic
- ID, Title, TopMessage, Closed, Pinned

**Channel** — Channel info
- ID, AccessHash, Username, Title, IsForum

**ParsedRange** — Scraped message ID range
- MinMsgID, MaxMsgID

**ScrapeStats** — Scraping statistics
- TotalFetched, NewMessages, SkippedOld, SkippedEmpty, Errors
