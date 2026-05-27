CREATE TABLE IF NOT EXISTS pix_transactions (
    id                  UUID PRIMARY KEY,
    payer_account_id    UUID           NOT NULL REFERENCES accounts(id),
    receiver_key        VARCHAR(255)   NOT NULL,
    receiver_account_id UUID           REFERENCES accounts(id),
    amount_cents        BIGINT         NOT NULL CHECK (amount_cents > 0),
    status              VARCHAR(20)    NOT NULL DEFAULT 'PENDING',
    description         VARCHAR(140),
    initiated_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ,
    failure_reason      VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_pix_transactions_payer ON pix_transactions(payer_account_id);
CREATE INDEX IF NOT EXISTS idx_pix_transactions_status ON pix_transactions(status);
