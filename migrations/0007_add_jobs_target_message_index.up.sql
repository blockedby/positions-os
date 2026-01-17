-- Add partial index for GetExistingMessageIDs query performance
-- Filters by target_id and tg_message_id, so index only rows where tg_message_id IS NOT NULL
CREATE INDEX IF NOT EXISTS idx_jobs_target_message
ON jobs(target_id, tg_message_id)
WHERE tg_message_id IS NOT NULL;
