CREATE TABLE payroll_periods (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    public_id char(9) NOT NULL UNIQUE,
    title text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 120),
    period_start date NOT NULL,
    period_end date NOT NULL,
    transfer_date date,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (period_end >= period_start)
);
CREATE INDEX payroll_periods_period_start_idx ON payroll_periods (period_start DESC);

CREATE TABLE payroll_lines (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    period_id uuid NOT NULL REFERENCES payroll_periods (id) ON DELETE CASCADE,
    line_no integer NOT NULL CHECK (line_no > 0),
    crew_id uuid NOT NULL REFERENCES crews (id),
    bank_account_id uuid REFERENCES crew_bank_accounts (id) ON DELETE SET NULL,
    employee_no text NOT NULL DEFAULT '',
    crew_name text NOT NULL,
    rank_code text NOT NULL DEFAULT '',
    vessel_name text NOT NULL DEFAULT '',
    bank_name text NOT NULL DEFAULT '',
    account_name text NOT NULL DEFAULT '',
    account_number text NOT NULL DEFAULT '',
    amount_usd numeric(18, 2) CHECK (amount_usd >= 0),
    amount_idr numeric(18, 2) NOT NULL CHECK (amount_idr >= 0),
    transfer_date date NOT NULL,
    remark text NOT NULL DEFAULT '' CHECK (char_length(remark) <= 300),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX payroll_lines_period_line_idx ON payroll_lines (period_id, line_no);
CREATE INDEX payroll_lines_period_bank_idx ON payroll_lines (period_id, bank_name);
CREATE INDEX payroll_lines_period_vessel_idx ON payroll_lines (period_id, vessel_name);
CREATE INDEX payroll_lines_period_rank_idx ON payroll_lines (period_id, rank_code);
CREATE INDEX payroll_lines_crew_id_idx ON payroll_lines (crew_id);
CREATE INDEX payroll_lines_bank_account_id_idx ON payroll_lines (bank_account_id);

CREATE TRIGGER payroll_periods_touch BEFORE UPDATE ON payroll_periods FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER payroll_lines_touch BEFORE UPDATE ON payroll_lines FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
