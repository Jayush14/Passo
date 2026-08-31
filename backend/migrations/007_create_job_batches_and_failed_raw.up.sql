-- Create job_batches table for tracking batch processing
CREATE TABLE job_batches (
    batch_id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL,
    total_jobs INTEGER NOT NULL,
    processed_ok INTEGER NOT NULL DEFAULT 0,
    processed_fail INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create index for user_id and status for faster lookups
CREATE INDEX ix_job_batches_user_id_status ON job_batches(user_id, status);

-- Create job_failed_raw table for storing raw descriptions of failed jobs
CREATE TABLE job_failed_raw (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL,
    batch_id UUID NOT NULL,
    user_id BIGINT NOT NULL,
    provider VARCHAR(50) NOT NULL,
    model VARCHAR(100) NOT NULL,
    job_description TEXT NOT NULL,
    failed_at TIMESTAMP NOT NULL DEFAULT NOW(),
    retry_count INTEGER NOT NULL DEFAULT 0
);

-- Add indexes for job_failed_raw
CREATE INDEX ix_job_failed_raw_batch_id ON job_failed_raw(batch_id);
CREATE INDEX ix_job_failed_raw_user_id ON job_failed_raw(user_id);