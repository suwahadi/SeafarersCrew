ALTER TABLE files
    ADD COLUMN width integer CHECK (width > 0),
    ADD COLUMN height integer CHECK (height > 0);
