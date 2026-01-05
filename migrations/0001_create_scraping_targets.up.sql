-- migration: create scraping_targets table
-- stores sources for job parsing (telegram channels, hh searches, etc.)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE scraping_target_type AS ENUM (
    'TG_CHANNEL',
    'TG_GROUP',
    'TG_FORUM',
    'HH_SEARCH',
    'LINKEDIN_SEARCH'
);

CREATE TABLE scraping_targets (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(255) NOT NULL,
    type            scraping_target_type NOT NULL,
    url             TEXT NOT NULL,
    
    -- telegram specific
    tg_access_hash  BIGINT,
    tg_channel_id   BIGINT,
    
    -- parsing config
    metadata        JSONB DEFAULT '{}',
    
    -- state
    last_scraped_at TIMESTAMPTZ,
    last_message_id BIGINT,
    is_active       BOOLEAN DEFAULT true,
    
    -- timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

-- index for finding active sources quickly
CREATE INDEX idx_scraping_targets_active ON scraping_targets (is_active) WHERE is_active = true;

-- index for filtering by type
CREATE INDEX idx_scraping_targets_type ON scraping_targets (type);

COMMENT ON TABLE scraping_targets IS 'sources for job parsing';
COMMENT ON COLUMN scraping_targets.metadata IS 'json config: keywords, limit, include_topics, etc.';
COMMENT ON COLUMN scraping_targets.last_message_id IS 'last processed message id (for incremental parsing)';
