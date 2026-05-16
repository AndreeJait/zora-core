DROP INDEX IF EXISTS idx_whitelist_entries_lid;
DROP INDEX IF EXISTS idx_admin_entries_lid;

ALTER TABLE whitelist_entries DROP COLUMN IF EXISTS scope;
ALTER TABLE whitelist_entries DROP COLUMN IF EXISTS chat_ids;
ALTER TABLE whitelist_entries DROP COLUMN IF EXISTS lid;

ALTER TABLE admin_entries DROP COLUMN IF EXISTS scope;
ALTER TABLE admin_entries DROP COLUMN IF EXISTS chat_ids;
ALTER TABLE admin_entries DROP COLUMN IF EXISTS lid;