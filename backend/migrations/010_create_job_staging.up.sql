-- Create job_staging table to track individual job statuses

CREATE TABLE job_staging (
    id UUID PRIMARY KEY,
    batch_id UUID NOT NULL REFERENCES job_batches(batch_id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    provider VARCHAR(50) NOT NULL DEFAULT 'gemini',
    model VARCHAR(100) NOT NULL DEFAULT 'gemini-2.0-flash',
    job_description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    raw_json JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    retry_count INTEGER NOT NULL DEFAULT 0
);

-- Create indexes for efficient querying
CREATE INDEX ix_job_staging_batch_id ON job_staging(batch_id);
CREATE INDEX ix_job_staging_status ON job_staging(status);
CREATE INDEX ix_job_staging_batch_status ON job_staging(batch_id, status);
CREATE INDEX ix_job_staging_user_id ON job_staging(user_id);