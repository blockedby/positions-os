# 0002_create_jobs.up.sql

Creates `jobs` table for storing scraped job postings.

Includes indexes on `target_id`, `status`, `created_at`.
Stores raw content and LLM-extracted structured data (JSONB).
