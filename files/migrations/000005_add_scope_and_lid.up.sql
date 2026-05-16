-- Add scope, chat_ids, and lid columns to whitelist_entries
ALTER TABLE whitelist_entries ADD COLUMN IF NOT EXISTS scope VARCHAR(20) NOT NULL DEFAULT 'both';
ALTER TABLE whitelist_entries ADD COLUMN IF NOT EXISTS chat_ids TEXT NOT NULL DEFAULT '';
ALTER TABLE whitelist_entries ADD COLUMN IF NOT EXISTS lid VARCHAR(255) NOT NULL DEFAULT '';

-- Add scope, chat_ids, and lid columns to admin_entries
ALTER TABLE admin_entries ADD COLUMN IF NOT EXISTS scope VARCHAR(20) NOT NULL DEFAULT 'both';
ALTER TABLE admin_entries ADD COLUMN IF NOT EXISTS chat_ids TEXT NOT NULL DEFAULT '';
ALTER TABLE admin_entries ADD COLUMN IF NOT EXISTS lid VARCHAR(255) NOT NULL DEFAULT '';

-- Index on lid for fallback lookups
CREATE INDEX IF NOT EXISTS idx_whitelist_entries_lid ON whitelist_entries (lid);
CREATE INDEX IF NOT EXISTS idx_admin_entries_lid ON admin_entries (lid);