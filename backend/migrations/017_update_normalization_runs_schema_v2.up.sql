-- -- Add new columns with defaults
-- ALTER TABLE normalization_runs
--     ADD COLUMN IF NOT EXISTS run_type VARCHAR NOT NULL DEFAULT 'worker',
--     ADD COLUMN IF NOT EXISTS status VARCHAR NOT NULL DEFAULT 'running',
--     ADD COLUMN IF NOT EXISTS error_message TEXT,
--     ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
--     ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- -- Rename finished_at to completed_at and make nullable (split into two statements)
-- ALTER TABLE normalization_runs
--     ALTER COLUMN finished_at DROP NOT NULL;
-- ALTER TABLE normalization_runs
--     RENAME COLUMN finished_at TO completed_at;

-- -- Rename existing columns to match model
-- ALTER TABLE normalization_runs
--     RENAME COLUMN rows_processed TO records_processed;
-- ALTER TABLE normalization_runs
--     -- RENAME COLUMN success_count TO records_updated;
-- ALTER TABLE normalization_runs
--     RENAME COLUMN failure_count TO records_failed;

-- -- Drop foreign key constraint on taxonomy_version
-- ALTER TABLE normalization_runs
--     DROP CONSTRAINT IF EXISTS normalization_runs_taxonomy_version_fkey;

-- -- Drop old columns
-- ALTER TABLE normalization_runs
--     DROP COLUMN IF EXISTS entities_sent_to_review,
--     DROP COLUMN IF EXISTS new_entities_approved,
--     DROP COLUMN IF EXISTS taxonomy_version;