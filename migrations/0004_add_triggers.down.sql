-- rollback: drop triggers and function

DROP TRIGGER IF EXISTS update_job_applications_updated_at ON job_applications;
DROP TRIGGER IF EXISTS update_jobs_updated_at ON jobs;
DROP TRIGGER IF EXISTS update_scraping_targets_updated_at ON scraping_targets;
DROP FUNCTION IF EXISTS update_updated_at_column();
