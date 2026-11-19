CREATE TABLE master_aliases (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    kind text NOT NULL CHECK (kind IN ('certificate_type', 'document_type', 'rank', 'vessel_type')),
    alias text NOT NULL CHECK (char_length(alias) BETWEEN 1 AND 200),
    target_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX master_aliases_kind_alias_key ON master_aliases (kind, alias);
