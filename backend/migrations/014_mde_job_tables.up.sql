-- Job-centric tables for Market Data Engine

-- Jobs table
CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_title_id UUID NOT NULL REFERENCES job_titles(id),
    company_id UUID NOT NULL REFERENCES companies(id),
    seniority_id UUID NOT NULL REFERENCES seniority_levels(id),
    job_family_id UUID NOT NULL REFERENCES job_families(id),
    normalization_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job postings table
CREATE TABLE IF NOT EXISTS job_postings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    location_id UUID NOT NULL REFERENCES locations(id),
    employment_type_id UUID NOT NULL REFERENCES employment_types(id),
    work_mode_id UUID NOT NULL REFERENCES work_modes(id),
    source VARCHAR(50) NOT NULL,
    source_job_id VARCHAR(200),
    url TEXT,
    posted_at TIMESTAMPTZ,
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    salary_min NUMERIC(10,2),
    salary_max NUMERIC(10,2),
    currency CHAR(3),
    dedup_key CHAR(64) NOT NULL,
    UNIQUE (dedup_key),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job descriptions table
CREATE TABLE IF NOT EXISTS job_descriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_posting_id UUID NOT NULL REFERENCES job_postings(id) ON DELETE CASCADE,
    raw_text TEXT NOT NULL,
    extracted_json JSONB NOT NULL,
    parser_version VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job skills table
CREATE TABLE IF NOT EXISTS job_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES skills(id),
    requirement_type VARCHAR(20) NOT NULL CHECK (requirement_type IN ('required','preferred','nice_to_have')),
    requirement_weight DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    source_section VARCHAR(30),
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    evidence TEXT,
    raw_skill_name TEXT NOT NULL,
    normalization_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job skill groups table
CREATE TABLE IF NOT EXISTS job_skill_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    group_label VARCHAR(20) NOT NULL,
    operator VARCHAR(10) NOT NULL CHECK (operator IN ('ANY','ALL')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job capabilities table (stub for Phase 2)
CREATE TABLE IF NOT EXISTS job_capabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    capability_id UUID NULL REFERENCES capabilities(id), -- placeholder for Phase 2
    importance DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    evidence TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job domains table
CREATE TABLE IF NOT EXISTS job_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    domain_id UUID NOT NULL REFERENCES domains(id),
    importance DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job soft skills table (optional)
CREATE TABLE IF NOT EXISTS job_soft_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    evidence TEXT,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auxiliary tables

-- Entity resolution queue
CREATE TABLE IF NOT EXISTS entity_resolution_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_type VARCHAR(20) NOT NULL, -- skill|title|company|location|seniority|employment_type|work_mode
    raw_value TEXT NOT NULL,
    evidence TEXT,
    extraction_id UUID NOT NULL REFERENCES job_extractions(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending_review', -- approved|rejected
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Failed extractions table
CREATE TABLE IF NOT EXISTS failed_extractions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    extraction_id UUID NOT NULL REFERENCES job_extractions(id),
    error_message TEXT NOT NULL,
    failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    retry_count INT NOT NULL DEFAULT 0
);

-- Normalization runs table
CREATE TABLE IF NOT EXISTS normalization_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    rows_processed INT NOT NULL DEFAULT 0,
    success_count INT NOT NULL DEFAULT 0,
    failure_count INT NOT NULL DEFAULT 0,
    entities_sent_to_review INT NOT NULL DEFAULT 0,
    new_entities_approved INT NOT NULL DEFAULT 0,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id)
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS ix_jobs_canonical_title_id ON jobs(canonical_title_id);
CREATE INDEX IF NOT EXISTS ix_jobs_company_id ON jobs(company_id);
CREATE INDEX IF NOT EXISTS ix_jobs_seniority_id ON jobs(seniority_id);
CREATE INDEX IF NOT EXISTS ix_jobs_job_family_id ON jobs(job_family_id);
CREATE INDEX IF NOT EXISTS ix_jobs_normalization_version ON jobs(normalization_version);
CREATE INDEX IF NOT EXISTS ix_job_postings_job_id ON job_postings(job_id);
CREATE INDEX IF NOT EXISTS ix_job_postings_location_id ON job_postings(location_id);
CREATE INDEX IF NOT EXISTS ix_job_postings_employment_type_id ON job_postings(employment_type_id);
CREATE INDEX IF NOT EXISTS ix_job_postings_work_mode_id ON job_postings(work_mode_id);
CREATE INDEX IF NOT EXISTS ix_job_postings_dedup_key ON job_postings(dedup_key);
CREATE INDEX IF NOT EXISTS ix_job_descriptions_job_posting_id ON job_descriptions(job_posting_id);
CREATE INDEX IF NOT EXISTS ix_job_skills_job_id ON job_skills(job_id);
CREATE INDEX IF NOT EXISTS ix_job_skills_skill_id ON job_skills(skill_id);
CREATE INDEX IF NOT EXISTS ix_job_skills_normalization_version ON job_skills(normalization_version);
CREATE INDEX IF NOT EXISTS ix_job_skill_groups_job_id ON job_skill_groups(job_id);
CREATE INDEX IF NOT EXISTS ix_job_capabilities_job_id ON job_capabilities(job_id);
CREATE INDEX IF NOT EXISTS ix_job_domains_job_id ON job_domains(job_id);
CREATE INDEX IF NOT EXISTS ix_job_soft_skills_job_id ON job_soft_skills(job_id);
CREATE INDEX IF NOT EXISTS ix_entity_resolution_queue_field_type ON entity_resolution_queue(field_type);
CREATE INDEX IF NOT EXISTS ix_entity_resolution_queue_status ON entity_resolution_queue(status);
CREATE INDEX IF NOT EXISTS ix_failed_extractions_extraction_id ON failed_extractions(extraction_id);
CREATE INDEX IF NOT EXISTS ix_normalization_runs_taxonomy_version ON normalization_runs(taxonomy_version);