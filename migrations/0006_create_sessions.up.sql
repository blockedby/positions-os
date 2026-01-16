-- Telegram session storage for gotgproto
CREATE TABLE IF NOT EXISTS sessions (
    version INTEGER PRIMARY KEY,
    data    BYTEA
);

COMMENT ON TABLE sessions IS 'Telegram MTProto session storage for gotgproto';
COMMENT ON COLUMN sessions.version IS 'Session version (always 1 for latest)';
COMMENT ON COLUMN sessions.data IS 'JSON-serialized session data';
