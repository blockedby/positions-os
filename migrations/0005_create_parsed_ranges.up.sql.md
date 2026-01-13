# 0005_create_parsed_ranges.up.sql

Creates `parsed_ranges` table for message ID deduplication.

Tracks min/max message IDs scraped per target.
Primary key is `target_id` (one-to-one).
