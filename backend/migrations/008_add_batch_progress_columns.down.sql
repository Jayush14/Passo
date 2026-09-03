-- Remove columns added for real-time batch progress
ALTER TABLE job_batches
    DROP COLUMN IF EXISTS jobs_in_progress,
    DROP COLUMN IF EXISTS jobs_waiting;