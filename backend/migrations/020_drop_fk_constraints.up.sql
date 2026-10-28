-- Make location_id nullable and drop FK
ALTER TABLE job_postings
    DROP CONSTRAINT IF EXISTS job_postings_location_id_fkey,
    ALTER COLUMN location_id DROP NOT NULL;

-- Make employment_type_id nullable and drop FK
ALTER TABLE job_postings
    DROP CONSTRAINT IF EXISTS job_postings_employment_type_id_fkey,
    ALTER COLUMN employment_type_id DROP NOT NULL;

-- Make work_mode_id nullable and drop FK
ALTER TABLE job_postings
    DROP CONSTRAINT IF EXISTS job_postings_work_mode_id_fkey,
    ALTER COLUMN work_mode_id DROP NOT NULL;

-- Make company_id nullable and drop FK (already nullable from migration 019, but we drop the FK)
ALTER TABLE job_postings
    DROP CONSTRAINT IF EXISTS job_postings_company_id_fkey;

-- Make canonical_title_id nullable and drop FK
ALTER TABLE jobs
    DROP CONSTRAINT IF EXISTS jobs_canonical_title_id_fkey,
    ALTER COLUMN canonical_title_id DROP NOT NULL;

-- Make company_id nullable and drop FK
ALTER TABLE jobs
    DROP CONSTRAINT IF EXISTS jobs_company_id_fkey,
    ALTER COLUMN company_id DROP NOT NULL;

-- Make seniority_id nullable and drop FK
ALTER TABLE jobs
    DROP CONSTRAINT IF EXISTS jobs_seniority_id_fkey,
    ALTER COLUMN seniority_id DROP NOT NULL;

-- Make job_family_id nullable and drop FK
ALTER TABLE jobs
    DROP CONSTRAINT IF EXISTS jobs_job_family_id_fkey,
    ALTER COLUMN job_family_id DROP NOT NULL;