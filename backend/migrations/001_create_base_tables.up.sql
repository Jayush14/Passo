CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- users
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    current_streak INTEGER DEFAULT 0,
    last_activity_date DATE,
    plan_typed TEXT NOT NULL DEFAULT 'byok', -- byok or subscription
    is_admin BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- refresh_tokens
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- api_keys (user's encrypted Gemini key)
CREATE TABLE api_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    encrypted_key TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'gemini',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- house_api_key (single row for subscription tier)
CREATE TABLE house_api_key (
    id SMALLINT PRIMARY KEY DEFAULT 1,
    encrypted_key TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'gemini',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- enforce single row
    CONSTRAINT single_row CHECK (id = 1)
);

-- roadmaps
CREATE TABLE roadmaps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'active', -- active, archived
    onboarding_source TEXT NOT NULL, -- manual, resume
    current_skills TEXT[], -- array of strings
    target_role TEXT NOT NULL,
    hours_per_week_available INTEGER NOT NULL,
    weekly_time_pattern TEXT,
    start_date DATE NOT NULL,
    target_end_date DATE NOT NULL,
    current_company TEXT,
    current_package TEXT,
    expected_company TEXT,
    expected_package TEXT,
    education_background TEXT,
    certifications TEXT[],
    portfolio_url TEXT,
    interview_experience TEXT,
    learning_style TEXT,
    target_domain TEXT,
    network_access TEXT,
    additional_notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_repaced_at TIMESTAMPTZ
);

-- progress (per task)
CREATE TABLE progress (
    id BIGSERIAL PRIMARY KEY,
    roadmap_id UUID NOT NULL REFERENCES roadmaps(id) ON DELETE CASCADE,
    task_id TEXT NOT NULL,
    checked BOOLEAN NOT NULL DEFAULT FALSE,
    note TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- chat_messages
CREATE TABLE chat_messages (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    roadmap_id UUID NOT NULL REFERENCES roadmaps(id) ON DELETE CASCADE,
    role TEXT NOT NULL, -- user or assistant
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- push_subscriptions
CREATE TABLE push_subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    keys JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- nudge_schedule
CREATE TABLE nudge_schedule (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cadence TEXT NOT NULL, -- daily, weekly, etc.
    last_sent_at TIMESTAMPTZ
);

-- analytics_events
CREATE TABLE analytics_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);