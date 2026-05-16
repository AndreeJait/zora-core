CREATE TABLE IF NOT EXISTS admin_entries (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone      VARCHAR(255) NOT NULL,
    name       VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_admin_entries_phone ON admin_entries (phone);
CREATE INDEX idx_admin_entries_created_at ON admin_entries (created_at DESC);