DROP INDEX IF EXISTS applicant_stage_logs_admin_id_idx;
ALTER TABLE applicant_stage_logs DROP COLUMN IF EXISTS admin_id;
