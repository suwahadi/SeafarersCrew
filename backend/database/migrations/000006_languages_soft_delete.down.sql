DELETE FROM languages WHERE deleted_at IS NOT NULL;
DROP INDEX languages_code_key;
ALTER TABLE languages ADD CONSTRAINT languages_code_key UNIQUE (code);
ALTER TABLE languages DROP COLUMN deleted_at;
