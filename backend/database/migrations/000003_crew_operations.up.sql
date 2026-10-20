CREATE TABLE crews (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    public_id char(9) NOT NULL UNIQUE,
    full_name text NOT NULL,
    place_of_birth text NOT NULL DEFAULT '',
    date_of_birth date,
    gender text CHECK (gender IN ('male', 'female')),
    nationality text NOT NULL DEFAULT '',
    religion text NOT NULL DEFAULT '',
    marital_status text CHECK (marital_status IN ('single', 'married', 'divorced', 'widowed')),
    height_cm smallint CHECK (height_cm BETWEEN 50 AND 250),
    weight_kg smallint CHECK (weight_kg BETWEEN 20 AND 300),
    shoe_size text NOT NULL DEFAULT '',
    coverall_size text NOT NULL DEFAULT '',
    applied_rank_id uuid REFERENCES ranks (id),
    crew_status_id uuid NOT NULL REFERENCES crew_statuses (id),
    status_reason text NOT NULL DEFAULT '',
    photo_file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    cv_file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    address text NOT NULL DEFAULT '',
    city text NOT NULL DEFAULT '',
    province text NOT NULL DEFAULT '',
    country text NOT NULL DEFAULT '',
    postal_code text NOT NULL DEFAULT '',
    home_phone text NOT NULL DEFAULT '',
    mobile_phone text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    nearest_airport text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX crews_full_name_trgm_idx ON crews USING gin (full_name gin_trgm_ops);
CREATE INDEX crews_status_rank_idx ON crews (crew_status_id, applied_rank_id) WHERE deleted_at IS NULL;
CREATE INDEX crews_applied_rank_id_idx ON crews (applied_rank_id);
CREATE INDEX crews_nationality_idx ON crews (nationality) WHERE deleted_at IS NULL;
CREATE INDEX crews_date_of_birth_idx ON crews (date_of_birth) WHERE deleted_at IS NULL;

CREATE TABLE crew_bank_accounts (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    crew_id uuid NOT NULL REFERENCES crews (id) ON DELETE CASCADE,
    bank_name text NOT NULL,
    account_name text NOT NULL,
    account_number text NOT NULL,
    currency_code char(3) NOT NULL REFERENCES currencies (code),
    cover_bank text NOT NULL DEFAULT '',
    remarks text NOT NULL DEFAULT '',
    is_primary boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX crew_bank_accounts_crew_id_idx ON crew_bank_accounts (crew_id);
CREATE UNIQUE INDEX crew_bank_accounts_primary_key ON crew_bank_accounts (crew_id) WHERE is_primary;

CREATE TABLE crew_next_of_kins (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    crew_id uuid NOT NULL REFERENCES crews (id) ON DELETE CASCADE,
    name text NOT NULL,
    relationship text NOT NULL CHECK (relationship IN ('spouse', 'parent', 'child', 'sibling', 'other')),
    phone text NOT NULL DEFAULT '',
    address text NOT NULL DEFAULT '',
    is_primary boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX crew_next_of_kins_crew_id_idx ON crew_next_of_kins (crew_id);
CREATE UNIQUE INDEX crew_next_of_kins_primary_key ON crew_next_of_kins (crew_id) WHERE is_primary;

CREATE TABLE travel_documents (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    crew_id uuid NOT NULL REFERENCES crews (id) ON DELETE CASCADE,
    document_type_id uuid NOT NULL REFERENCES document_types (id),
    number text NOT NULL,
    issued_date date,
    expired_date date,
    place_of_issue text NOT NULL DEFAULT '',
    file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expired_date IS NULL OR issued_date IS NULL OR expired_date >= issued_date)
);
CREATE INDEX travel_documents_crew_id_idx ON travel_documents (crew_id);
CREATE INDEX travel_documents_document_type_id_idx ON travel_documents (document_type_id);
CREATE INDEX travel_documents_expired_date_idx ON travel_documents (expired_date);
CREATE INDEX travel_documents_number_trgm_idx ON travel_documents USING gin (number gin_trgm_ops);

CREATE TABLE certificates (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    crew_id uuid NOT NULL REFERENCES crews (id) ON DELETE CASCADE,
    certificate_type_id uuid NOT NULL REFERENCES certificate_types (id),
    number text NOT NULL,
    issued_date date,
    expired_date date,
    issuing_authority text NOT NULL DEFAULT '',
    place_of_issue text NOT NULL DEFAULT '',
    file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (expired_date IS NULL OR issued_date IS NULL OR expired_date >= issued_date)
);
CREATE INDEX certificates_crew_id_idx ON certificates (crew_id);
CREATE INDEX certificates_certificate_type_id_idx ON certificates (certificate_type_id);
CREATE INDEX certificates_expired_date_idx ON certificates (expired_date);

CREATE TABLE contracts (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    public_id char(9) NOT NULL UNIQUE,
    crew_id uuid NOT NULL REFERENCES crews (id),
    vessel_id uuid NOT NULL REFERENCES vessels (id),
    rank_id uuid NOT NULL REFERENCES ranks (id),
    principal_id uuid NOT NULL REFERENCES companies (id),
    employer_company_id uuid NOT NULL REFERENCES companies (id),
    sign_on_date date NOT NULL,
    duration_months smallint NOT NULL CHECK (duration_months BETWEEN 1 AND 60),
    planned_sign_off_date date NOT NULL,
    actual_sign_off_date date,
    point_of_hire text NOT NULL DEFAULT '',
    sign_on_port text NOT NULL DEFAULT '',
    sign_off_port text NOT NULL DEFAULT '',
    currency_code char(3) NOT NULL REFERENCES currencies (code),
    basic_salary numeric(18, 2) NOT NULL CHECK (basic_salary >= 0),
    working_hours_week smallint CHECK (working_hours_week BETWEEN 0 AND 168),
    leave_days_month numeric(5, 2) CHECK (leave_days_month >= 0),
    probation_months smallint CHECK (probation_months >= 0),
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'completed', 'terminated')),
    termination_reason text NOT NULL DEFAULT '',
    notes text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK (planned_sign_off_date >= sign_on_date),
    CHECK (actual_sign_off_date IS NULL OR actual_sign_off_date >= sign_on_date),
    CHECK (status <> 'terminated' OR termination_reason <> '')
);
CREATE UNIQUE INDEX contracts_one_active_per_crew ON contracts (crew_id) WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX contracts_crew_id_idx ON contracts (crew_id);
CREATE INDEX contracts_vessel_id_idx ON contracts (vessel_id);
CREATE INDEX contracts_rank_id_idx ON contracts (rank_id);
CREATE INDEX contracts_principal_id_idx ON contracts (principal_id);
CREATE INDEX contracts_employer_company_id_idx ON contracts (employer_company_id);
CREATE INDEX contracts_status_idx ON contracts (status, planned_sign_off_date) WHERE deleted_at IS NULL;

CREATE TABLE contract_allowances (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    contract_id uuid NOT NULL REFERENCES contracts (id) ON DELETE CASCADE,
    allowance_type_id uuid NOT NULL REFERENCES allowance_types (id),
    basis text NOT NULL CHECK (basis IN ('fixed_month', 'fixed_day', 'percent_basic')),
    value numeric(18, 2) NOT NULL CHECK (value >= 0),
    monthly_amount numeric(18, 2) NOT NULL CHECK (monthly_amount >= 0),
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX contract_allowances_contract_id_idx ON contract_allowances (contract_id);
CREATE INDEX contract_allowances_allowance_type_id_idx ON contract_allowances (allowance_type_id);

CREATE TABLE experiences (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    crew_id uuid NOT NULL REFERENCES crews (id) ON DELETE CASCADE,
    vessel_name text NOT NULL,
    vessel_id uuid REFERENCES vessels (id) ON DELETE SET NULL,
    rank_id uuid NOT NULL REFERENCES ranks (id),
    vessel_type_id uuid REFERENCES vessel_types (id),
    grt_hp text NOT NULL DEFAULT '',
    company_id uuid REFERENCES companies (id),
    company_name text NOT NULL DEFAULT '',
    principal_id uuid REFERENCES companies (id),
    principal_name text NOT NULL DEFAULT '',
    sign_on_date date NOT NULL,
    sign_off_date date,
    status text NOT NULL DEFAULT 'sign_off' CHECK (status IN ('on_board', 'sign_off')),
    remarks text NOT NULL DEFAULT '',
    source text NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'contract')),
    contract_id uuid UNIQUE REFERENCES contracts (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (sign_off_date IS NULL OR sign_off_date >= sign_on_date)
);
CREATE INDEX experiences_crew_id_idx ON experiences (crew_id);
CREATE INDEX experiences_rank_id_idx ON experiences (rank_id);
CREATE INDEX experiences_vessel_id_idx ON experiences (vessel_id);
CREATE INDEX experiences_vessel_type_id_idx ON experiences (vessel_type_id);
CREATE INDEX experiences_company_id_idx ON experiences (company_id);
CREATE INDEX experiences_principal_id_idx ON experiences (principal_id);

CREATE TABLE relief_plans (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    contract_id uuid NOT NULL REFERENCES contracts (id) ON DELETE CASCADE,
    reliever_crew_id uuid NOT NULL REFERENCES crews (id),
    planned_relief_date date NOT NULL,
    joining_port text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'confirmed', 'completed', 'cancelled')),
    notes text NOT NULL DEFAULT '',
    new_contract_id uuid REFERENCES contracts (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX relief_plans_contract_id_idx ON relief_plans (contract_id);
CREATE INDEX relief_plans_reliever_crew_id_idx ON relief_plans (reliever_crew_id);
CREATE INDEX relief_plans_new_contract_id_idx ON relief_plans (new_contract_id);
CREATE INDEX relief_plans_date_idx ON relief_plans (planned_relief_date) WHERE status IN ('planned', 'confirmed');

CREATE TABLE document_templates (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('sea', 'quotation', 'cv')),
    paper_size text NOT NULL DEFAULT 'a4' CHECK (paper_size IN ('a4', 'letter')),
    orientation text NOT NULL DEFAULT 'portrait' CHECK (orientation IN ('portrait', 'landscape')),
    header_html text NOT NULL DEFAULT '',
    body_html text NOT NULL DEFAULT '',
    footer_html text NOT NULL DEFAULT '',
    is_default boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX document_templates_default_key ON document_templates (kind) WHERE is_default AND deleted_at IS NULL;

CREATE TABLE quotations (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    public_id char(9) NOT NULL UNIQUE,
    client_company_id uuid NOT NULL REFERENCES companies (id),
    vessel_id uuid REFERENCES vessels (id),
    currency_code char(3) NOT NULL REFERENCES currencies (code),
    quotation_date date NOT NULL,
    valid_until date NOT NULL,
    tax_percent numeric(5, 2) NOT NULL DEFAULT 0 CHECK (tax_percent BETWEEN 0 AND 100),
    discount_amount numeric(18, 2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    terms text NOT NULL DEFAULT '',
    notes text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'sent', 'accepted', 'rejected', 'expired')),
    subtotal numeric(18, 2) NOT NULL DEFAULT 0,
    tax_amount numeric(18, 2) NOT NULL DEFAULT 0,
    grand_total numeric(18, 2) NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK (valid_until >= quotation_date)
);
CREATE INDEX quotations_client_company_id_idx ON quotations (client_company_id);
CREATE INDEX quotations_vessel_id_idx ON quotations (vessel_id);
CREATE INDEX quotations_status_idx ON quotations (status, valid_until) WHERE deleted_at IS NULL;

CREATE TABLE quotation_lines (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    quotation_id uuid NOT NULL REFERENCES quotations (id) ON DELETE CASCADE,
    line_type text NOT NULL CHECK (line_type IN ('daily_rate', 'lump_sum')),
    rank_id uuid REFERENCES ranks (id),
    description text NOT NULL DEFAULT '',
    crew_count integer CHECK (crew_count > 0),
    daily_rate numeric(18, 2) CHECK (daily_rate >= 0),
    days integer CHECK (days > 0),
    amount numeric(18, 2) NOT NULL CHECK (amount >= 0),
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (line_type = 'daily_rate' AND rank_id IS NOT NULL AND crew_count IS NOT NULL AND daily_rate IS NOT NULL AND days IS NOT NULL)
        OR (line_type = 'lump_sum' AND description <> '')
    )
);
CREATE INDEX quotation_lines_quotation_id_idx ON quotation_lines (quotation_id);
CREATE INDEX quotation_lines_rank_id_idx ON quotation_lines (rank_id);

CREATE TABLE generated_documents (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    public_id char(9) NOT NULL UNIQUE,
    kind text NOT NULL CHECK (kind IN ('sea', 'quotation', 'cv')),
    template_id uuid REFERENCES document_templates (id) ON DELETE SET NULL,
    related_type text NOT NULL,
    related_id uuid NOT NULL,
    file_id uuid NOT NULL REFERENCES files (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX generated_documents_related_idx ON generated_documents (related_type, related_id);
CREATE INDEX generated_documents_template_id_idx ON generated_documents (template_id);
CREATE INDEX generated_documents_file_id_idx ON generated_documents (file_id);

CREATE TRIGGER crews_touch BEFORE UPDATE ON crews FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER crew_bank_accounts_touch BEFORE UPDATE ON crew_bank_accounts FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER crew_next_of_kins_touch BEFORE UPDATE ON crew_next_of_kins FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER travel_documents_touch BEFORE UPDATE ON travel_documents FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER certificates_touch BEFORE UPDATE ON certificates FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER contracts_touch BEFORE UPDATE ON contracts FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER contract_allowances_touch BEFORE UPDATE ON contract_allowances FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER experiences_touch BEFORE UPDATE ON experiences FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER relief_plans_touch BEFORE UPDATE ON relief_plans FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER document_templates_touch BEFORE UPDATE ON document_templates FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER quotations_touch BEFORE UPDATE ON quotations FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER quotation_lines_touch BEFORE UPDATE ON quotation_lines FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER generated_documents_touch BEFORE UPDATE ON generated_documents FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
