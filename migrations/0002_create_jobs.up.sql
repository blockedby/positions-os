-- migration: create jobs table
-- central table for storing parsed job postings

CREATE TYPE job_status AS ENUM (
    'RAW',          -- just parsed
    'ANALYZED',     -- processed by llm
    'REJECTED',     -- rejected by user
    'INTERESTED',   -- user is interested
    'TAILORED',     -- resume adapted
    'SENT',         -- application sent
    'RESPONDED'     -- got response
);

CREATE TABLE jobs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_id       UUID NOT NULL REFERENCES scraping_targets(id) ON DELETE CASCADE,
    
    -- identification
    external_id     VARCHAR(255) NOT NULL,  -- message/vacancy id on source
    content_hash    VARCHAR(64),            -- sha256 for deduplication
    
    -- content
    raw_content     TEXT NOT NULL,
    
    -- structured data (filled by analyzer)
    structured_data JSONB DEFAULT '{}',
    -- example structure:
    -- {
    --   "title": "Go Developer",
    --   "description": "...",
    --   "salary_min": 3000,
    --   "salary_max": 5000,
    --   "currency": "USD",
    --   "location": "Remote",
    --   "is_remote": true,
    --   "language": "EN",
    --   "technologies": ["go", "postgresql", "docker"],
    --   "experience_years": 3,
    --   "company": "TechCorp",
    --   "contacts": ["@recruiter", "hr@company.com"]
    -- }
    
    -- source metadata
    source_url      TEXT,
    source_date     TIMESTAMPTZ,
    
    -- telegram specific
    tg_message_id   BIGINT,
    tg_topic_id     BIGINT,  -- if from forum topic
    
    -- status
    status          job_status DEFAULT 'RAW',
    
    -- timestamps
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    analyzed_at     TIMESTAMPTZ,
    
    -- uniqueness: one external_id per source
    CONSTRAINT uq_jobs_target_external UNIQUE (target_id, external_id)
);

-- index for filtering by status
CREATE INDEX idx_jobs_status ON jobs (status);

-- index for finding RAW jobs (for analyzer)
CREATE INDEX idx_jobs_raw ON jobs (created_at) WHERE status = 'RAW';

-- index for searching by technologies (gin for jsonb)
CREATE INDEX idx_jobs_technologies ON jobs USING GIN ((structured_data -> 'technologies'));

-- index for full-text search (russian language)
CREATE INDEX idx_jobs_content_search ON jobs USING GIN (to_tsvector('russian', raw_content));

COMMENT ON TABLE jobs IS 'central table for job postings';
COMMENT ON COLUMN jobs.external_id IS 'job id on source (message_id for tg, vacancy_id for hh)';
COMMENT ON COLUMN jobs.content_hash IS 'sha256 of raw_content for duplicate detection';
