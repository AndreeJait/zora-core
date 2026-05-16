CREATE TABLE IF NOT EXISTS whitelist_entries (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone            VARCHAR(255) NOT NULL,
    name             VARCHAR(255) NOT NULL,
    tokens_per_hour  INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_whitelist_entries_phone ON whitelist_entries (phone);
CREATE INDEX idx_whitelist_entries_created_at ON whitelist_entries (created_at DESC);

CREATE TABLE IF NOT EXISTS token_usages (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone         VARCHAR(255) NOT NULL,
    tokens_used   INT NOT NULL DEFAULT 1,
    window_start  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_token_usages_phone_window ON token_usages (phone, window_start);
CREATE INDEX idx_token_usages_window_start ON token_usages (window_start);