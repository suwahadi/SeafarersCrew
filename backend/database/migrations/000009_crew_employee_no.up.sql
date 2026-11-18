ALTER TABLE crews ADD COLUMN employee_no text NOT NULL DEFAULT '';
CREATE UNIQUE INDEX crews_employee_no_key ON crews (upper(ltrim(employee_no, '0'))) WHERE deleted_at IS NULL AND employee_no <> '';
