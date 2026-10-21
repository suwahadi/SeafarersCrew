DROP INDEX files_purge_idx;
ALTER TABLE files DROP COLUMN purged_at;
