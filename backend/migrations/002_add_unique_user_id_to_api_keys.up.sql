ALTER TABLE api_keys ADD CONSTRAINT uk_api_keys_user_id UNIQUE (user_id);
