-- Add columns for storing brain service outputs
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS tailored_resume_path TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS cover_letter_text TEXT;
