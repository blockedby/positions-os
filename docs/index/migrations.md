# migrations

Database schema migrations using golang-migrate.

## Running Migrations

```bash
task migrate-up     # Apply all migrations
task migrate-down   # Rollback last migration
```

## Schema Versions

| Migration | Description |
|-----------|-------------|
| 0001 | `scraping_targets` table |
| 0002 | `jobs` table |
| 0003 | `job_applications` table |
| 0004 | `updated_at` triggers |
| 0005 | `parsed_ranges` table |

See [README.md](../../migrations/README.md) for full schema details.
