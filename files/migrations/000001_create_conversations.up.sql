CREATE TABLE IF NOT EXISTS conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  VARCHAR(255) NOT NULL,
    task        TEXT NOT NULL,
    messages    JSONB NOT NULL DEFAULT '[]',
    status      VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_conversations_session_id ON conversations (session_id);
CREATE INDEX idx_conversations_status ON conversations (status);
CREATE INDEX idx_conversations_created_at ON conversations (created_at DESC);
