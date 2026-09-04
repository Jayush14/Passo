-- Add MDE processing tracking columns to job_extractions
ALTER TABLE job_extractions
ADD COLUMN IF NOT EXISTS mde_processed BOOLEAN DEFAULT FALSE,
ADD COLUMN IF NOT EXISTS mde_processed_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS mde_error_message TEXT;