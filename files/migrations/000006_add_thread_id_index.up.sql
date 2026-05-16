-- Add index on thread_id for worker graph step lookups
CREATE INDEX IF NOT EXISTS idx_tasks_thread_id ON zora_tasks (thread_id);