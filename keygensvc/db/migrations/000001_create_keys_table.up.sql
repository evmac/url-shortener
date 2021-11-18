BEGIN;
CREATE TABLE IF NOT EXISTS keys (
    id SERIAL PRIMARY KEY,
    raw_key VARCHAR(36) NOT NULL,
    source_id BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uix_keys_raw_key_source_id
ON keys (raw_key, source_id);
COMMIT;
