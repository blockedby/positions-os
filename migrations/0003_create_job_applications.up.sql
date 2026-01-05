-- migration: create job_applications table
-- stores tailored resumes and application tracking

CREATE TYPE delivery_channel AS ENUM (
    'TG_DM',
    'EMAIL',
    'HH_RESPONSE'
);

CREATE TYPE delivery_status AS ENUM (
    'PENDING',
    'SENT',
    'DELIVERED',
    'READ',
    'FAILED'
);

CREATE TABLE job_applications (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    job_id                  UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    
    -- generated content
    tailored_resume_md      TEXT,           -- markdown resume
    cover_letter_md         TEXT,           -- markdown cover letter
    
    -- generated files
    resume_pdf_path         VARCHAR(512),   -- path to pdf in volume
    cover_letter_pdf_path   VARCHAR(512),
    
    -- delivery
    delivery_channel        delivery_channel,
    delivery_status         delivery_status DEFAULT 'PENDING',
    recipient               VARCHAR(255),   -- @username or email
    
    -- tracking
    sent_at                 TIMESTAMPTZ,
    delivered_at            TIMESTAMPTZ,
    read_at                 TIMESTAMPTZ,
    response_received_at    TIMESTAMPTZ,
    
    -- recruiter response (if any)
    recruiter_response      TEXT,
    
    -- timestamps
    created_at              TIMESTAMPTZ DEFAULT NOW(),
    updated_at              TIMESTAMPTZ DEFAULT NOW(),
    
    -- can create multiple versions for one job
    version                 INT DEFAULT 1
);

-- index for finding by job
CREATE INDEX idx_job_applications_job ON job_applications (job_id);

-- index for pending deliveries
CREATE INDEX idx_job_applications_pending ON job_applications (created_at) 
    WHERE delivery_status = 'PENDING';

COMMENT ON TABLE job_applications IS 'tailoring results and application tracking';
COMMENT ON COLUMN job_applications.version IS 'application version (can create multiple iterations)';
