ALTER TABLE files ADD COLUMN purged_at timestamptz;
CREATE INDEX files_purge_idx ON files (created_at) WHERE purged_at IS NULL;
