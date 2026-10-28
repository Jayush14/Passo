-- Drop foreign key constraint
ALTER TABLE job_postings
    DROP CONSTRAINT IF EXISTS job_postings_company_id_fkey;

-- Drop company_id column
ALTER TABLE job_postings
    DROP COLUMN IF EXISTS company_id;