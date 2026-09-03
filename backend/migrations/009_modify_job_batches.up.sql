-- Modify job_batches table to remove excess columns
-- Keep only essential batch metadata

-- First, check current table structure and preserve data if needed
-- We'll alter the table to drop the columns we no longer need

ALTER TABLE job_batches
    DROP COLUMN IF EXISTS processed_ok,
    DROP COLUMN IF EXISTS processed_fail,
    DROP COLUMN IF EXISTS jobs_in_progress,
    DROP COLUMN IF EXISTS jobs_waiting;