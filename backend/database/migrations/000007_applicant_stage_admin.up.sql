ALTER TABLE applicant_stage_logs ADD COLUMN admin_id uuid REFERENCES admins (id) ON DELETE SET NULL;
CREATE INDEX applicant_stage_logs_admin_id_idx ON applicant_stage_logs (admin_id);
