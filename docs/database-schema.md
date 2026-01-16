# Database Schema

PostgreSQL database schema for Positions OS.

## Overview

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│  scraping_targets   │────<│        jobs         │────<│  job_applications   │
│  (sources config)   │     │  (job postings)     │     │  (resume/letters)   │
└─────────────────────┘     └─────────────────────┘     └─────────────────────┘
          │
          │
          ▼
┌─────────────────────┐
│   parsed_ranges     │
│ (scraping progress) │
└─────────────────────┘

┌─────────────────────┐     ┌─────────────────────┐
│      sessions       │     │       peers         │
│ (telegram auth)     │     │ (gotgproto cache)   │
└─────────────────────┘     └─────────────────────┘
```

## Tables

### `scraping_targets`

Sources for job parsing (Telegram channels, groups, forums).

| Column | Type | Description |
|--------|------|-------------|
| `id` | `uuid` | Primary key |
| `name` | `varchar(255)` | Human-readable name |
| `type` | `scraping_target_type` | TG_CHANNEL, TG_GROUP, TG_FORUM, HH_SEARCH, LINKEDIN_SEARCH |
| `url` | `text` | Source URL (@channel or https://...) |
| `tg_access_hash` | `bigint` | Telegram access hash (cached) |
| `tg_channel_id` | `bigint` | Telegram channel ID (cached) |
| `metadata` | `jsonb` | Config: keywords, limit, include_topics, etc. |
| `last_scraped_at` | `timestamptz` | Last successful scrape time |
| `last_message_id` | `bigint` | Last processed message ID |
| `is_active` | `boolean` | Enable/disable target |
| `created_at` | `timestamptz` | Created timestamp |
| `updated_at` | `timestamptz` | Updated timestamp (auto) |

**Indexes:**
- `idx_scraping_targets_active` - active targets only
- `idx_scraping_targets_type` - by type

---

### `jobs`

Central table for job postings.

| Column | Type | Description |
|--------|------|-------------|
| `id` | `uuid` | Primary key |
| `target_id` | `uuid` | FK → scraping_targets |
| `external_id` | `varchar(255)` | Source ID (message_id for TG) |
| `content_hash` | `varchar(64)` | SHA256 for deduplication |
| `raw_content` | `text` | Original message text |
| `structured_data` | `jsonb` | LLM extraction result |
| `source_url` | `text` | Direct link to posting |
| `source_date` | `timestamptz` | Original post date |
| `tg_message_id` | `bigint` | Telegram message ID |
| `tg_topic_id` | `bigint` | Forum topic ID |
| `status` | `job_status` | Processing status |
| `created_at` | `timestamptz` | Created timestamp |
| `updated_at` | `timestamptz` | Updated timestamp (auto) |
| `analyzed_at` | `timestamptz` | When LLM processed |

**Indexes:**
- `idx_jobs_status` - by status
- `idx_jobs_raw` - RAW jobs by date
- `idx_jobs_technologies` - GIN on structured_data→technologies
- `idx_jobs_content_search` - Full-text search (Russian)
- `uq_jobs_target_external` - Unique (target_id, external_id)

**structured_data schema:**
```json
{
  "title": "Go Developer",
  "description": "...",
  "salary_min": 200000,
  "salary_max": 350000,
  "currency": "RUB",
  "location": "Moscow",
  "is_remote": true,
  "language": "RU",
  "technologies": ["go", "postgresql", "docker"],
  "experience_years": 3,
  "company": "Yandex",
  "contacts": ["@hr_yandex", "hr@yandex.ru"]
}
```

---

### `job_applications`

Tailoring results and application tracking (Phase 4).

| Column | Type | Description |
|--------|------|-------------|
| `id` | `uuid` | Primary key |
| `job_id` | `uuid` | FK → jobs |
| `tailored_resume_md` | `text` | Tailored resume markdown |
| `cover_letter_md` | `text` | Generated cover letter |
| `resume_pdf_path` | `varchar(512)` | Path to PDF |
| `cover_letter_pdf_path` | `varchar(512)` | Path to PDF |
| `delivery_channel` | `delivery_channel` | TG_DM, EMAIL, HH_RESPONSE |
| `delivery_status` | `delivery_status` | PENDING → SENT → DELIVERED → READ |
| `recipient` | `varchar(255)` | Contact used |
| `sent_at` | `timestamptz` | When sent |
| `delivered_at` | `timestamptz` | When delivered |
| `read_at` | `timestamptz` | When read |
| `response_received_at` | `timestamptz` | When recruiter replied |
| `recruiter_response` | `text` | Response text |
| `version` | `integer` | Application iteration |
| `created_at` | `timestamptz` | Created timestamp |
| `updated_at` | `timestamptz` | Updated timestamp (auto) |

---

### `parsed_ranges`

Tracks scraped Telegram message ID ranges for incremental parsing.

| Column | Type | Description |
|--------|------|-------------|
| `id` | `uuid` | Primary key |
| `target_id` | `uuid` | FK → scraping_targets (unique) |
| `min_msg_id` | `bigint` | Lowest message ID scraped |
| `max_msg_id` | `bigint` | Highest message ID scraped |
| `created_at` | `timestamptz` | Created timestamp |
| `updated_at` | `timestamptz` | Updated timestamp (auto) |

**Constraint:** `max_msg_id >= min_msg_id`

---

### `sessions`

Telegram MTProto session storage for gotgproto.

| Column | Type | Description |
|--------|------|-------------|
| `version` | `integer` | Primary key (always 1) |
| `data` | `bytea` | JSON-serialized session |

---

### `peers`

gotgproto internal cache for Telegram peers (auto-managed).

---

## Enums

### `job_status`

```
RAW → ANALYZED → INTERESTED → TAILORED → SENT → RESPONDED
                     ↓
                 REJECTED
```

| Value | Description |
|-------|-------------|
| `RAW` | Just scraped, not analyzed |
| `ANALYZED` | LLM extracted structured data |
| `REJECTED` | Not interesting |
| `INTERESTED` | Marked for application |
| `TAILORED` | Resume/cover letter generated |
| `SENT` | Application sent |
| `RESPONDED` | Got recruiter response |

### `scraping_target_type`

| Value | Description |
|-------|-------------|
| `TG_CHANNEL` | Telegram channel |
| `TG_GROUP` | Telegram group |
| `TG_FORUM` | Telegram forum (with topics) |
| `HH_SEARCH` | HeadHunter search (planned) |
| `LINKEDIN_SEARCH` | LinkedIn search (planned) |

### `delivery_channel`

| Value | Description |
|-------|-------------|
| `TG_DM` | Telegram direct message |
| `EMAIL` | Email |
| `HH_RESPONSE` | HeadHunter response |

### `delivery_status`

| Value | Description |
|-------|-------------|
| `PENDING` | Not sent yet |
| `SENT` | Sent to recipient |
| `DELIVERED` | Confirmed delivered |
| `READ` | Confirmed read |
| `FAILED` | Delivery failed |

## Triggers

All tables with `updated_at` have auto-update trigger:

```sql
CREATE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';
```
