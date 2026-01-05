-- rollback: drop job_applications table

DROP TABLE IF EXISTS job_applications;
DROP TYPE IF EXISTS delivery_status;
DROP TYPE IF EXISTS delivery_channel;
