CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE OR REPLACE FUNCTION uuid_generate_v7() RETURNS uuid
LANGUAGE plpgsql VOLATILE AS $$
DECLARE
    bytes bytea;
BEGIN
    bytes := substring(int8send(floor(extract(epoch FROM clock_timestamp()) * 1000)::bigint) FROM 3) || gen_random_bytes(10);
    bytes := set_byte(bytes, 6, (b'0111' || get_byte(bytes, 6)::bit(4))::bit(8)::int);
    bytes := set_byte(bytes, 8, (b'10' || get_byte(bytes, 8)::bit(6))::bit(8)::int);
    RETURN encode(bytes, 'hex')::uuid;
END
$$;

CREATE OR REPLACE FUNCTION touch_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END
$$;

CREATE TABLE admins (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    password_changed_at timestamptz NOT NULL DEFAULT now(),
    must_change_password boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX admins_email_key ON admins (lower(email));

CREATE TABLE recovery_codes (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    admin_id uuid NOT NULL REFERENCES admins (id) ON DELETE CASCADE,
    code_hash text NOT NULL,
    used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX recovery_codes_admin_id_idx ON recovery_codes (admin_id);

CREATE TABLE settings (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    group_name text NOT NULL,
    key text NOT NULL,
    value jsonb NOT NULL,
    is_secret boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (group_name, key)
);

CREATE TABLE files (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    storage_key text NOT NULL UNIQUE,
    original_name text NOT NULL,
    mime_type text NOT NULL,
    size_original bigint NOT NULL,
    size_stored bigint NOT NULL,
    sha256 text NOT NULL,
    owner_type text,
    owner_id uuid,
    is_public boolean NOT NULL DEFAULT false,
    variants jsonb NOT NULL DEFAULT '[]'::jsonb,
    status text NOT NULL DEFAULT 'ready' CHECK (status IN ('processing', 'ready', 'failed')),
    error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX files_owner_idx ON files (owner_type, owner_id);

CREATE TABLE email_templates (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    code text NOT NULL UNIQUE,
    name text NOT NULL,
    subject text NOT NULL,
    body_html text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE email_logs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    template_code text,
    to_addresses text[] NOT NULL,
    cc_addresses text[] NOT NULL DEFAULT '{}',
    subject text NOT NULL,
    related_type text,
    related_id uuid,
    attachment_file_ids uuid[] NOT NULL DEFAULT '{}',
    brevo_message_id text,
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'sent', 'failed')),
    attempts integer NOT NULL DEFAULT 0,
    error text,
    sent_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX email_logs_related_idx ON email_logs (related_type, related_id);

CREATE TABLE backups (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    storage_key text,
    size_bytes bigint,
    status text NOT NULL CHECK (status IN ('running', 'succeeded', 'failed')),
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE redirects (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    from_path text NOT NULL UNIQUE,
    to_path text NOT NULL,
    status_code smallint NOT NULL DEFAULT 301 CHECK (status_code IN (301, 302, 308)),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE jobs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    kind text NOT NULL,
    status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'succeeded', 'failed')),
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    result jsonb,
    error text,
    attempts integer NOT NULL DEFAULT 0,
    started_at timestamptz,
    finished_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX jobs_status_idx ON jobs (status, created_at);

CREATE TRIGGER admins_touch BEFORE UPDATE ON admins FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER recovery_codes_touch BEFORE UPDATE ON recovery_codes FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER settings_touch BEFORE UPDATE ON settings FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER files_touch BEFORE UPDATE ON files FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER email_templates_touch BEFORE UPDATE ON email_templates FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER email_logs_touch BEFORE UPDATE ON email_logs FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER backups_touch BEFORE UPDATE ON backups FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER redirects_touch BEFORE UPDATE ON redirects FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER jobs_touch BEFORE UPDATE ON jobs FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
