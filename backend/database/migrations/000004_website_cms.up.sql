CREATE TABLE pages (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    title text NOT NULL,
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    system_key text UNIQUE CHECK (system_key IN ('home', 'news', 'career', 'contact', 'gallery')),
    intro_html text NOT NULL DEFAULT '',
    blocks jsonb NOT NULL DEFAULT '[]'::jsonb,
    seo_title text NOT NULL DEFAULT '',
    meta_description text NOT NULL DEFAULT '',
    og_image_file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    in_header_menu boolean NOT NULL DEFAULT false,
    in_footer_menu boolean NOT NULL DEFAULT false,
    menu_order integer NOT NULL DEFAULT 0,
    is_system boolean NOT NULL DEFAULT false,
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK (NOT is_system OR system_key IS NOT NULL)
);
CREATE UNIQUE INDEX pages_slug_key ON pages (slug) WHERE deleted_at IS NULL;
CREATE INDEX pages_og_image_file_id_idx ON pages (og_image_file_id);

CREATE TABLE news (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    title text NOT NULL,
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    excerpt text NOT NULL DEFAULT '',
    body_html text NOT NULL DEFAULT '',
    thumbnail_file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    published_at timestamptz,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    seo_title text NOT NULL DEFAULT '',
    meta_description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX news_slug_key ON news (slug) WHERE deleted_at IS NULL;
CREATE INDEX news_published_idx ON news (published_at DESC) WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX news_thumbnail_file_id_idx ON news (thumbnail_file_id);

CREATE TABLE hero_slides (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    image_file_id uuid NOT NULL REFERENCES files (id),
    mobile_image_file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    heading text NOT NULL DEFAULT '',
    subheading text NOT NULL DEFAULT '',
    button_label text NOT NULL DEFAULT '',
    button_link text NOT NULL DEFAULT '',
    sort_order integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX hero_slides_image_file_id_idx ON hero_slides (image_file_id);
CREATE INDEX hero_slides_mobile_image_file_id_idx ON hero_slides (mobile_image_file_id);

CREATE TABLE gallery_images (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    image_file_id uuid NOT NULL REFERENCES files (id),
    caption text NOT NULL DEFAULT '',
    alt_text text NOT NULL CHECK (alt_text <> ''),
    show_on_home boolean NOT NULL DEFAULT false,
    sort_order integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX gallery_images_image_file_id_idx ON gallery_images (image_file_id);

CREATE TABLE testimonials (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    position text NOT NULL DEFAULT '',
    photo_file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    quote text NOT NULL,
    rating smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
    sort_order integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX testimonials_photo_file_id_idx ON testimonials (photo_file_id);

CREATE TABLE client_logos (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    logo_file_id uuid NOT NULL REFERENCES files (id),
    url text NOT NULL DEFAULT '',
    sort_order integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX client_logos_logo_file_id_idx ON client_logos (logo_file_id);

CREATE TABLE stats_items (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    number integer NOT NULL CHECK (number >= 0),
    suffix text NOT NULL DEFAULT '',
    label text NOT NULL,
    sort_order integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE menu_items (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    menu text NOT NULL CHECK (menu IN ('header', 'footer')),
    label text NOT NULL,
    page_id uuid REFERENCES pages (id) ON DELETE CASCADE,
    url text,
    parent_id uuid REFERENCES menu_items (id) ON DELETE CASCADE,
    open_in_new_tab boolean NOT NULL DEFAULT false,
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((page_id IS NOT NULL) <> (url IS NOT NULL))
);
CREATE INDEX menu_items_menu_idx ON menu_items (menu, sort_order);
CREATE INDEX menu_items_page_id_idx ON menu_items (page_id);
CREATE INDEX menu_items_parent_id_idx ON menu_items (parent_id);

CREATE TABLE vacancies (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    public_id char(9) NOT NULL UNIQUE,
    title text NOT NULL,
    slug text NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(slug) <= 80),
    rank_id uuid REFERENCES ranks (id),
    vessel_type_id uuid REFERENCES vessel_types (id),
    department text CHECK (department IN ('deck', 'engine', 'catering_other')),
    location text NOT NULL DEFAULT '',
    employment_type text NOT NULL DEFAULT 'contract' CHECK (employment_type IN ('contract', 'permanent')),
    duration text NOT NULL DEFAULT '',
    salary_text text NOT NULL DEFAULT '',
    description_html text NOT NULL DEFAULT '',
    requirements_html text NOT NULL DEFAULT '',
    required_documents jsonb NOT NULL DEFAULT '[]'::jsonb,
    closing_date date,
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed')),
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX vacancies_slug_key ON vacancies (slug) WHERE deleted_at IS NULL;
CREATE INDEX vacancies_open_idx ON vacancies (closing_date) WHERE status = 'open' AND deleted_at IS NULL;
CREATE INDEX vacancies_rank_id_idx ON vacancies (rank_id);
CREATE INDEX vacancies_vessel_type_id_idx ON vacancies (vessel_type_id);
CREATE INDEX vacancies_title_trgm_idx ON vacancies USING gin (title gin_trgm_ops);

CREATE TABLE applicants (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    public_id char(9) NOT NULL UNIQUE,
    vacancy_id uuid NOT NULL REFERENCES vacancies (id),
    full_name text NOT NULL,
    email text NOT NULL,
    phone text NOT NULL,
    date_of_birth date,
    nationality text NOT NULL DEFAULT '',
    rank_applied_id uuid REFERENCES ranks (id),
    years_experience smallint CHECK (years_experience BETWEEN 0 AND 60),
    message text NOT NULL DEFAULT '',
    stage text NOT NULL DEFAULT 'new' CHECK (stage IN ('new', 'screening', 'interview', 'accepted', 'rejected')),
    consent_at timestamptz NOT NULL,
    ip_address inet,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE INDEX applicants_vacancy_id_idx ON applicants (vacancy_id);
CREATE INDEX applicants_rank_applied_id_idx ON applicants (rank_applied_id);
CREATE INDEX applicants_stage_idx ON applicants (stage, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX applicants_duplicate_guard_idx ON applicants (vacancy_id, lower(email), created_at DESC);

CREATE TABLE applicant_files (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    applicant_id uuid NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    file_id uuid NOT NULL REFERENCES files (id),
    kind text NOT NULL CHECK (kind IN ('cv', 'document')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX applicant_files_applicant_id_idx ON applicant_files (applicant_id);
CREATE INDEX applicant_files_file_id_idx ON applicant_files (file_id);

CREATE TABLE applicant_stage_logs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    applicant_id uuid NOT NULL REFERENCES applicants (id) ON DELETE CASCADE,
    from_stage text,
    to_stage text NOT NULL,
    note text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX applicant_stage_logs_applicant_id_idx ON applicant_stage_logs (applicant_id, created_at);

CREATE TABLE contact_messages (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    email text NOT NULL,
    phone text NOT NULL DEFAULT '',
    subject text NOT NULL,
    message text NOT NULL CHECK (length(message) <= 2000),
    ip_address inet,
    is_read boolean NOT NULL DEFAULT false,
    is_archived boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX contact_messages_inbox_idx ON contact_messages (is_archived, is_read, created_at DESC);

CREATE TABLE languages (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v7(),
    name text NOT NULL,
    code text NOT NULL UNIQUE CHECK (code ~ '^[a-z]{2,3}(-[A-Za-z]{2,4})?$'),
    flag_file_id uuid REFERENCES files (id) ON DELETE SET NULL,
    flag_code text NOT NULL DEFAULT '',
    sort_order integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    is_source boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (NOT is_source OR is_active)
);
CREATE UNIQUE INDEX languages_single_source ON languages (is_source) WHERE is_source;
CREATE INDEX languages_flag_file_id_idx ON languages (flag_file_id);

CREATE TRIGGER pages_touch BEFORE UPDATE ON pages FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER news_touch BEFORE UPDATE ON news FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER hero_slides_touch BEFORE UPDATE ON hero_slides FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER gallery_images_touch BEFORE UPDATE ON gallery_images FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER testimonials_touch BEFORE UPDATE ON testimonials FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER client_logos_touch BEFORE UPDATE ON client_logos FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER stats_items_touch BEFORE UPDATE ON stats_items FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER menu_items_touch BEFORE UPDATE ON menu_items FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER vacancies_touch BEFORE UPDATE ON vacancies FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER applicants_touch BEFORE UPDATE ON applicants FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER applicant_files_touch BEFORE UPDATE ON applicant_files FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER applicant_stage_logs_touch BEFORE UPDATE ON applicant_stage_logs FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER contact_messages_touch BEFORE UPDATE ON contact_messages FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
CREATE TRIGGER languages_touch BEFORE UPDATE ON languages FOR EACH ROW EXECUTE FUNCTION touch_updated_at();
