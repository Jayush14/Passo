-- Add company_id column to job_postings table
ALTER TABLE job_postings
    ADD COLUMN IF NOT EXISTS company_id UUID;

-- Make company_id nullable (it already is by default, but let's be explicit)
ALTER TABLE job_postings
    ALTER COLUMN company_id DROP NOT NULL;

-- Add foreign key constraint to companies table
ALTER TABLE job_postings
    ADD CONSTRAINT job_postings_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id);