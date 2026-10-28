-- Make foreign key columns nullable in job entity tables and drop constraints

-- job_skills table
ALTER TABLE job_skills
    ALTER COLUMN skill_id DROP NOT NULL,
    DROP CONSTRAINT IF EXISTS job_skills_skill_id_fkey;

-- job_capabilities table
ALTER TABLE job_capabilities
    ALTER COLUMN capability_id DROP NOT NULL,
    DROP CONSTRAINT IF EXISTS job_capabilities_capability_id_fkey;

-- job_domains table
ALTER TABLE job_domains
    ALTER COLUMN domain_id DROP NOT NULL,
    DROP CONSTRAINT IF EXISTS job_domains_domain_id_fkey;