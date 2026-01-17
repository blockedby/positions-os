-- Remove brain output columns
ALTER TABLE jobs DROP COLUMN IF EXISTS tailored_resume_path;
ALTER TABLE jobs DROP COLUMN IF EXISTS cover_letter_text;
