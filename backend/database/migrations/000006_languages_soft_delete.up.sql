ALTER TABLE languages ADD COLUMN deleted_at timestamptz;
ALTER TABLE languages DROP CONSTRAINT languages_code_key;
CREATE UNIQUE INDEX languages_code_key ON languages (lower(code)) WHERE deleted_at IS NULL;
