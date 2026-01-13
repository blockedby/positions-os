# migrations

Database schema migrations using golang-migrate.

## Files

| Migration | Up | Down |
|-----------|-----|------|
| 0001 | Create `scraping_targets` table | Drop table |
| 0002 | Create `jobs` table | Drop table |
| 0003 | Create `job_applications` table | Drop table |
| 0004 | Add update triggers | Remove triggers |
| 0005 | Create `parsed_ranges` table | Drop table |

## scraping_targets

```sql
- id (UUID, PK)
- name (VARCHAR)
- type (VARCHAR) — TG_CHANNEL, TG_FORUM
- url (VARCHAR)
- is_active (BOOLEAN)
- metadata (JSONB) — telegram channel_id, access_hash
- last_scraped_at (TIMESTAMP)
- last_scraped_max_msg_id (BIGINT)
- created_at, updated_at
```

## jobs

```sql
- id (UUID, PK)
- target_id (UUID, FK)
- external_id (VARCHAR)
- content_hash (VARCHAR)
- raw_content (TEXT)
- structured_data (JSONB)
- source_url (VARCHAR)
- source_date (TIMESTAMP)
- tg_message_id (BIGINT)
- tg_topic_id (BIGINT)
- status (VARCHAR) — RAW, ANALYZED, REJECTED, INTERESTED, TAILORED, SENT, RESPONDED
- created_at, updated_at, analyzed_at
```

## job_applications

```sql
- id (UUID, PK)
- job_id (UUID, FK)
- status (VARCHAR)
- resume_path (VARCHAR)
- cover_letter (TEXT)
- sent_at (TIMESTAMP)
- responded_at (TIMESTAMP)
- response_notes (TEXT)
- created_at, updated_at
```

## parsed_ranges

```sql
- id (BIGINT, PK) — target_id
- min_msg_id (BIGINT)
- max_msg_id (BIGINT)
- updated_at (TIMESTAMP)
```

## Running Migrations

```bash
# Up
task migrate-up

# Down (rollback last)
task migrate-down

# Create new
task migrate-create name=add_feature
```
