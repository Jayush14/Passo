-- Drop the old unique constraint that only covered user_id
ALTER TABLE api_keys DROP CONSTRAINT IF EXISTS uk_api_keys_user_id;

-- Add a composite unique constraint so a user can have many providers
ALTER TABLE api_keys
ADD CONSTRAINT uk_api_keys_user_provider UNIQUE (user_id, provider);