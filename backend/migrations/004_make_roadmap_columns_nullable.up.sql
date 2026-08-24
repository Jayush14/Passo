ALTER TABLE roadmaps ALTER COLUMN onboarding_source DROP NOT NULL;
ALTER TABLE roadmaps ALTER COLUMN target_role DROP NOT NULL;
ALTER TABLE roadmaps ALTER COLUMN hours_per_week_available DROP NOT NULL;
ALTER TABLE roadmaps ALTER COLUMN start_date DROP NOT NULL;
ALTER TABLE roadmaps ALTER COLUMN target_end_date DROP NOT NULL;