-- Remove added columns from jobs table
ALTER TABLE jobs
DROP COLUMN IF EXISTS external_id,
DROP COLUMN IF EXISTS source,
DROP COLUMN IF EXISTS source_url,
DROP COLUMN IF EXISTS raw_content,
DROP COLUMN IF EXISTS language,
DROP COLUMN IF EXISTS retrieved_at,
DROP COLUMN IF EXISTS dedup_key,
DROP COLUMN IF EXISTS mde_processed,
DROP COLUMN IF EXISTS mde_processed_at,
DROP COLUMN IF EXISTS mde_error_message;