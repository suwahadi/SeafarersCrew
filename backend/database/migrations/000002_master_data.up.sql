CREATE TABLE currencies (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    code char(3) NOT NULL UNIQUE CHECK (code ~ '^[A-Z]{3}$'),
    name text NOT NULL DEFAULT '',
    symbol text NOT NULL DEFAULT '',
    decimals smallint NOT NULL DEFAULT 2 CHECK (decimals BETWEEN 0 AND 4),
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE companies (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    code text NOT NULL,
    name text NOT NULL,
    type text NOT NULL CHECK (type IN ('company', 'principal', 'both')),
    address text NOT NULL DEFAULT '',
    country text NOT NULL DEFAULT '',
    phone text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    website text NOT NULL DEFAULT '',
    logo_file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX companies_code_key ON companies (lower(code)) WHERE deleted_at IS NULL;
CREATE INDEX companies_name_trgm_idx ON companies USING gin (name gin_trgm_ops);

CREATE TABLE vessel_types (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX vessel_types_name_key ON vessel_types (lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE vessels (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    imo_number char(7) NOT NULL CHECK (imo_number ~ '^[0-9]{7}$'),
    flag text NOT NULL DEFAULT '',
    vessel_type_id uuid REFERENCES vessel_types (id),
    grt numeric(12, 2),
    hp_kw text NOT NULL DEFAULT '',
    year_built smallint CHECK (year_built BETWEEN 1900 AND 2100),
    owner_company_id uuid REFERENCES companies (id),
    principal_id uuid REFERENCES companies (id),
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'laid_up', 'sold')),
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX vessels_imo_number_key ON vessels (imo_number) WHERE deleted_at IS NULL;
CREATE INDEX vessels_vessel_type_id_idx ON vessels (vessel_type_id);
CREATE INDEX vessels_owner_company_id_idx ON vessels (owner_company_id);
CREATE INDEX vessels_principal_id_idx ON vessels (principal_id);

CREATE TABLE ranks (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    short_code text NOT NULL,
    department text NOT NULL CHECK (department IN ('deck', 'engine', 'catering_other')),
    is_officer boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX ranks_short_code_key ON ranks (upper(short_code)) WHERE deleted_at IS NULL;

CREATE TABLE crew_statuses (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    code text UNIQUE,
    name text NOT NULL,
    color text NOT NULL DEFAULT '#64748B' CHECK (color ~ '^#[0-9A-Fa-f]{6}$'),
    is_system boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK (NOT is_system OR code IS NOT NULL)
);

CREATE TABLE allowance_types (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    basis text NOT NULL CHECK (basis IN ('fixed_month', 'fixed_day', 'percent_basic')),
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX allowance_types_name_key ON allowance_types (lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE document_types (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    code text UNIQUE,
    name text NOT NULL,
    has_expiry boolean NOT NULL DEFAULT true,
    number_hint text NOT NULL DEFAULT '',
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX document_types_name_key ON document_types (lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE certificate_types (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    code text UNIQUE,
    name text NOT NULL,
    stcw_reference text NOT NULL DEFAULT '',
    category text NOT NULL CHECK (category IN ('coc', 'cop', 'endorsement', 'other')),
    has_expiry boolean NOT NULL DEFAULT true,
    is_active boolean NOT NULL DEFAULT true,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX certificate_types_name_key ON certificate_types (lower(name)) WHERE deleted_at IS NULL;

CREATE TABLE compliance_requirements (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    rank_id uuid NOT NULL REFERENCES ranks (id) ON DELETE CASCADE,
    requirement_type text NOT NULL CHECK (requirement_type IN ('document', 'certificate')),
    document_type_id uuid REFERENCES document_types (id) ON DELETE CASCADE,
    certificate_type_id uuid REFERENCES certificate_types (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (requirement_type = 'document' AND document_type_id IS NOT NULL AND certificate_type_id IS NULL)
        OR (requirement_type = 'certificate' AND certificate_type_id IS NOT NULL AND document_type_id IS NULL)
    )
);
CREATE UNIQUE INDEX compliance_requirements_document_key ON compliance_requirements (rank_id, document_type_id) WHERE document_type_id IS NOT NULL;
CREATE UNIQUE INDEX compliance_requirements_certificate_key ON compliance_requirements (rank_id, certificate_type_id) WHERE certificate_type_id IS NOT NULL;

CREATE TRIGGER currencies_touch BEFORE UPDATE ON currencies FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER companies_touch BEFORE UPDATE ON companies FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER vessel_types_touch BEFORE UPDATE ON vessel_types FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER vessels_touch BEFORE UPDATE ON vessels FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER ranks_touch BEFORE UPDATE ON ranks FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER crew_statuses_touch BEFORE UPDATE ON crew_statuses FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER allowance_types_touch BEFORE UPDATE ON allowance_types FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER document_types_touch BEFORE UPDATE ON document_types FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER certificate_types_touch BEFORE UPDATE ON certificate_types FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER compliance_requirements_touch BEFORE UPDATE ON compliance_requirements FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
