-- Remove MDE processing tracking columns from job_extractions
ALTER TABLE job_extractions
DROP COLUMN IF EXISTS mde_processed,
DROP COLUMN IF EXISTS mde_processed_at,
DROP COLUMN IF EXISTS mde_error_message;