# main.go

Collector service entry point — unified web UI + scraping API.

- Initializes Telegram client, database, NATS
- Registers HTTP handlers for scraping, jobs, targets, stats
- Serves web UI on configured port
