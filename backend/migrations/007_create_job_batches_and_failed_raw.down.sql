-- Drop indexes and tables in reverse order
DROP INDEX IF EXISTS ix_job_failed_raw_user_id;
DROP INDEX IF EXISTS ix_job_failed_raw_batch_id;
DROP TABLE IF EXISTS job_failed_raw;

DROP INDEX IF EXISTS ix_job_batches_user_id_status;
DROP TABLE IF EXISTS job_batches;