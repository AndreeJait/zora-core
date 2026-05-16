-- Task tracking table for background processing
CREATE TABLE IF NOT EXISTS zora_tasks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type            VARCHAR(20) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    source          VARCHAR(20) NOT NULL,
    input           JSONB NOT NULL DEFAULT '{}',
    result          JSONB DEFAULT '{}',
    error           TEXT,
    retry_count     INT NOT NULL DEFAULT 0,
    max_retry       INT NOT NULL DEFAULT 3,
    chat_id         VARCHAR(255),
    message_id      VARCHAR(255),
    session_id      VARCHAR(255),
    thread_id       VARCHAR(255),
    graph_mermaid   TEXT,
    next_retry_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_status ON zora_tasks (status);
CREATE INDEX idx_tasks_chat_id ON zora_tasks (chat_id);
CREATE INDEX idx_tasks_session_id ON zora_tasks (session_id);
CREATE INDEX idx_tasks_retry_sweep ON zora_tasks (status, next_retry_at) WHERE status = 'retrying';
CREATE INDEX idx_tasks_created_at ON zora_tasks (created_at DESC);

-- Runtime settings table
CREATE TABLE IF NOT EXISTS zora_settings (
    key         VARCHAR(255) PRIMARY KEY,
    value       TEXT NOT NULL,
    description TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);