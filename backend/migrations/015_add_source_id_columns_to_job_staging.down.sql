-- Remove source-specific ID columns from job_staging table
ALTER TABLE job_staging
    DROP COLUMN IF EXISTS linkedin_job_id,
    DROP COLUMN IF EXISTS naukri_job_id,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS company,
    DROP COLUMN IF EXISTS location,
    DROP COLUMN IF EXISTS employment_type,
    DROP COLUMN IF EXISTS work_mode,
    DROP COLUMN IF EXISTS seniority,
    DROP COLUMN IF EXISTS min_years,
    DROP COLUMN IF EXISTS max_years;

-- Drop indexes if they exist
DROP INDEX IF EXISTS ix_job_staging_linkedin_job_id;
DROP INDEX IF EXISTS ix_job_staging_naukri_job_id;