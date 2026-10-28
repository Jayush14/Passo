-- -- Add back the old columns
-- ALTER TABLE normalization_runs
--     ADD COLUMN IF NOT EXISTS entities_sent_to_review INTEGER NOT NULL DEFAULT 0,
--     ADD COLUMN IF NOT EXISTS new_entities_approved INTEGER NOT NULL DEFAULT 0,
--     ADD COLUMN IF NOT EXISTS taxonomy_version UUID NOT NULL;

-- -- Rename columns back to original names
-- ALTER TABLE normalization_runs
--     RENAME COLUMN records_processed TO rows_processed,
--     RENAME COLUMN records_updated TO success_count,
--     RENAME COLUMN records_failed TO failure_count;

-- -- Rename completed_at back to finished_at and set NOT NULL
-- ALTER TABLE normalization_runs
--     ALTER COLUMN completed_at SET NOT NULL,
--     RENAME COLUMN completed_at TO finished_at;

-- -- Drop new columns
-- ALTER TABLE normalization_runs
--     DROP COLUMN IF EXISTS run_type,
--     DROP COLUMN IF EXISTS status,
--     DROP COLUMN IF EXISTS error_message,
--     DROP COLUMN IF EXISTS created_at,
--     DROP COLUMN IF EXISTS updated_at;

-- -- Re-add foreign key constraint (best effort)
-- ALTER TABLE normalization_runs
--     ADD CONSTRAINT normalization_runs_taxonomy_version_fkey FOREIGN KEY (taxonomy_version) REFERENCES taxonomy_versions(id);