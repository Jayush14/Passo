-- Add back the foreign key constraints and set columns to NOT NULL
-- Note: This will fail if there are NULL values in these columns. You must clean the data first.

-- job_family_id
ALTER TABLE jobs
    ADD CONSTRAINT jobs_job_family_id_fkey FOREIGN KEY (job_family_id) REFERENCES job_families(id),
    ALTER COLUMN job_family_id SET NOT NULL;

-- seniority_id
ALTER TABLE jobs
    ADD CONSTRAINT jobs_seniority_id_fkey FOREIGN KEY (seniority_id) REFERENCES seniority_levels(id),
    ALTER COLUMN seniority_id SET NOT NULL;

-- company_id
ALTER TABLE jobs
    ADD CONSTRAINT jobs_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id),
    ALTER COLUMN company_id SET NOT NULL;

-- canonical_title_id
ALTER TABLE jobs
    ADD CONSTRAINT jobs_canonical_title_id_fkey FOREIGN KEY (canonical_title_id) REFERENCES job_titles(id),
    ALTER COLUMN canonical_title_id SET NOT NULL;

-- company_id in job_postings
ALTER TABLE job_postings
    ADD CONSTRAINT job_postings_company_id_fkey FOREIGN KEY (company_id) REFERENCES companies(id);

-- work_mode_id
ALTER TABLE job_postings
    ADD CONSTRAINT job_postings_work_mode_id_fkey FOREIGN KEY (work_mode_id) REFERENCES work_modes(id),
    ALTER COLUMN work_mode_id SET NOT NULL;

-- employment_type_id
ALTER TABLE job_postings
    ADD CONSTRAINT job_postings_employment_type_id_fkey FOREIGN KEY (employment_type_id) REFERENCES employment_types(id),
    ALTER COLUMN employment_type_id SET NOT NULL;

-- location_id
ALTER TABLE job_postings
    ADD CONSTRAINT job_postings_location_id_fkey FOREIGN KEY (location_id) REFERENCES locations(id),
    ALTER COLUMN location_id SET NOT NULL;