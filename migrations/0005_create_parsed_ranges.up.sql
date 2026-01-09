-- create parsed_ranges table to track scraped message id ranges
CREATE TABLE parsed_ranges (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    target_id   UUID NOT NULL REFERENCES scraping_targets(id) ON DELETE CASCADE,
    min_msg_id  BIGINT NOT NULL,
    max_msg_id  BIGINT NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT uq_parsed_ranges_target UNIQUE (target_id),
    CONSTRAINT check_valid_range CHECK (max_msg_id >= min_msg_id)
);

CREATE INDEX idx_parsed_ranges_target ON parsed_ranges (target_id);

COMMENT ON TABLE parsed_ranges IS 'tracks ranges of scraped telegram message ids for incremental parsing';
COMMENT ON COLUMN parsed_ranges.min_msg_id IS 'lowest message id in the scraped range';
COMMENT ON COLUMN parsed_ranges.max_msg_id IS 'highest message id in the scraped range (most recent)';
