-- Recreate job_failed_raw table (for rollback)

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