-- Drop job-centric tables in reverse order to avoid foreign key constraints
DROP TABLE IF EXISTS normalization_runs;
DROP TABLE IF EXISTS failed_extractions;
DROP TABLE IF EXISTS entity_resolution_queue;
DROP TABLE IF EXISTS job_soft_skills;
DROP TABLE IF EXISTS job_domains;
DROP TABLE IF EXISTS job_capabilities;
DROP TABLE IF EXISTS job_skill_groups;
DROP TABLE IF EXISTS job_skills;
DROP TABLE IF EXISTS job_descriptions;
DROP TABLE IF EXISTS job_postings;
DROP TABLE IF EXISTS jobs;