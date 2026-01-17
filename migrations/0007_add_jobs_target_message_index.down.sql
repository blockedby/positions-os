-- Remove the partial index for target_id + tg_message_id
DROP INDEX IF EXISTS idx_jobs_target_message;
