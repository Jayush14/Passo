-- Core reference tables for Market Data Engine

-- Taxonomy versions table
CREATE TABLE IF NOT EXISTS taxonomy_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version VARCHAR(20) NOT NULL UNIQUE,
    description TEXT,
    effective_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to TIMESTAMPTZ
);

-- Skill categories table
CREATE TABLE IF NOT EXISTS skill_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    version INT NOT NULL DEFAULT 1
);

-- Skill types table
CREATE TABLE IF NOT EXISTS skill_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    version INT NOT NULL DEFAULT 1
);

-- Skills table
CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_name VARCHAR(100) NOT NULL UNIQUE,
    normalized_name VARCHAR(100) NOT NULL,
    skill_type_id UUID NOT NULL REFERENCES skill_types(id),
    category_id UUID NOT NULL REFERENCES skill_categories(id),
    parent_skill_id UUID REFERENCES skills(id),
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    version INT NOT NULL DEFAULT 1,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Skill aliases table
CREATE TABLE IF NOT EXISTS skill_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    alias VARCHAR(100) NOT NULL,
    normalized_alias VARCHAR(100) NOT NULL,
    source VARCHAR(20) NOT NULL DEFAULT 'discovered', -- curated|discovered|llm_suggested
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    status VARCHAR(20) NOT NULL DEFAULT 'pending_review', -- approved|pending_review|rejected
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (skill_id, normalized_alias)
);

-- Companies table
CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_name VARCHAR(150) NOT NULL UNIQUE,
    normalized_name VARCHAR(150) NOT NULL,
    website VARCHAR(250),
    industry VARCHAR(100),
    size VARCHAR(50),
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Locations table
CREATE TABLE IF NOT EXISTS locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    country VARCHAR(100) NOT NULL,
    country_code CHAR(2),
    lat DOUBLE PRECISION,
    lon DOUBLE PRECISION,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seniority levels table
CREATE TABLE IF NOT EXISTS seniority_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    ordinal INT NOT NULL,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job families table
CREATE TABLE IF NOT EXISTS job_families (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    parent_id UUID REFERENCES job_families(id),
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Job titles table
CREATE TABLE IF NOT EXISTS job_titles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_title VARCHAR(150) NOT NULL UNIQUE,
    job_family_id UUID NOT NULL REFERENCES job_families(id),
    level_id UUID NOT NULL REFERENCES seniority_levels(id),
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Employment types table
CREATE TABLE IF NOT EXISTS employment_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(30) NOT NULL UNIQUE,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Work modes table
CREATE TABLE IF NOT EXISTS work_modes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(20) NOT NULL UNIQUE,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Skill relationships table
CREATE TABLE IF NOT EXISTS skill_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_skill_id UUID NOT NULL REFERENCES skills(id),
    target_skill_id UUID NOT NULL REFERENCES skills(id),
    type VARCHAR(20) NOT NULL, -- required_for, enhances, etc.
    strength DOUBLE PRECISION NOT NULL CHECK (strength >= 0 AND strength <= 1),
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Capabilities table
CREATE TABLE IF NOT EXISTS capabilities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(50) NOT NULL,
    description TEXT,
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Skill categories table (duplicate in spec, but let's ensure it exists)
CREATE TABLE IF NOT EXISTS skill_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    version INT NOT NULL DEFAULT 1
);

-- Skill types table (duplicate in spec, but let's ensure it exists)
CREATE TABLE IF NOT EXISTS skill_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    version INT NOT NULL DEFAULT 1
);

-- Domains table
CREATE TABLE IF NOT EXISTS domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    parent_id UUID REFERENCES domains(id),
    category VARCHAR(50),
    taxonomy_version UUID NOT NULL REFERENCES taxonomy_versions(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert initial taxonomy version
INSERT INTO taxonomy_versions (id, version, description, effective_from)
VALUES ('00000000-0000-0000-0000-000000000001', '1.0.0', 'Initial taxonomy version', NOW())
ON CONFLICT (version) DO NOTHING;