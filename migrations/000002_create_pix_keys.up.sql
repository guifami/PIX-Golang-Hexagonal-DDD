CREATE TABLE IF NOT EXISTS pix_keys (
    id          UUID PRIMARY KEY,
    account_id  UUID           NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    key_type    VARCHAR(20)    NOT NULL,
    key_value   VARCHAR(255)   UNIQUE NOT NULL,
    status      VARCHAR(20)    NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pix_keys_account_id ON pix_keys(account_id);
