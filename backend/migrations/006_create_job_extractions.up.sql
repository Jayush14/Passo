-- Create job_extractions table for storing raw AI responses
CREATE TABLE job_extractions (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL,
    provider VARCHAR(50),
    model VARCHAR(100),
    prompt_version VARCHAR(50),
    raw_response JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- Add indexes for common query patterns
CREATE INDEX ix_job_extractions_job_id ON job_extractions(job_id);
CREATE INDEX ix_job_extractions_provider ON job_extractions(provider);
CREATE INDEX ix_job_extractions_created_at ON job_extractions(created_at);