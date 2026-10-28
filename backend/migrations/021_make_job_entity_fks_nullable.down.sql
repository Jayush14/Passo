-- Re-add foreign key constraints and set columns to NOT NULL

-- First, we need to ensure there are no NULL values
-- This will fail if there are NULL values - you must clean the data first.

-- job_skills table
ALTER TABLE job_skills
    ALTER COLUMN skill_id SET NOT NULL,
    ADD CONSTRAINT job_skills_skill_id_fkey FOREIGN KEY (skill_id) REFERENCES skills(id);

-- job_capabilities table
ALTER TABLE job_capabilities
    ALTER COLUMN capability_id SET NOT NULL,
    ADD CONSTRAINT job_capabilities_capability_id_fkey FOREIGN KEY (capability_id) REFERENCES capabilities(id);

-- job_domains table
ALTER TABLE job_domains
    ALTER COLUMN domain_id SET NOT NULL,
    ADD CONSTRAINT job_domains_domain_id_fkey FOREIGN KEY (domain_id) REFERENCES domains(id);