-- Revert to the old unique constraint on user_id only
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS uk_api_keys_user_provider;
ALTER TABLE api_keys ADD CONSTRAINT uk_api_keys_user_id UNIQUE (user_id);