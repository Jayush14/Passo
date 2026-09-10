-- Add source-specific ID columns and basic job facts to job_staging table
ALTER TABLE job_staging
    ADD COLUMN IF NOT EXISTS linkedin_job_id VARCHAR(200),
    ADD COLUMN IF NOT EXISTS naukri_job_id VARCHAR(200),
    ADD COLUMN IF NOT EXISTS source VARCHAR(50),
    ADD COLUMN IF NOT EXISTS title VARCHAR(500),
    ADD COLUMN IF NOT EXISTS company VARCHAR(500),
    ADD COLUMN IF NOT EXISTS location VARCHAR(500),
    ADD COLUMN IF NOT EXISTS employment_type VARCHAR(50),
    ADD COLUMN IF NOT EXISTS work_mode VARCHAR(20),
    ADD COLUMN IF NOT EXISTS seniority VARCHAR(20),
    ADD COLUMN IF NOT EXISTS min_years INTEGER,
    ADD COLUMN IF NOT EXISTS max_years INTEGER;

-- Add indexes for efficient duplicate checking
CREATE INDEX IF NOT EXISTS ix_job_staging_linkedin_job_id ON job_staging(linkedin_job_id) WHERE linkedin_job_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS ix_job_staging_naukri_job_id ON job_staging(naukri_job_id) WHERE naukri_job_id IS NOT NULL;