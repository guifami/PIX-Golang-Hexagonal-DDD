CREATE TABLE IF NOT EXISTS accounts (
    id          UUID PRIMARY KEY,
    owner_name  VARCHAR(255)   NOT NULL,
    cpf         VARCHAR(11)    UNIQUE NOT NULL,
    balance_cents BIGINT       NOT NULL DEFAULT 0,
    status      VARCHAR(20)    NOT NULL DEFAULT 'ACTIVE',
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);
