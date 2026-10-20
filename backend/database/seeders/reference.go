// Package seeders loads idempotent reference data and optional demo data.
package seeders

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsSeeder inserts default settings.
type SettingsSeeder interface {
	SeedDefaults(ctx context.Context, overrides map[string]map[string]any) error
}

// Reference loads every reference data set; running it twice changes nothing.
func Reference(ctx context.Context, pool *pgxpool.Pool, settings SettingsSeeder, settingOverrides map[string]map[string]any) error {
	steps := []struct {
		name string
		run  func(ctx context.Context, tx pgx.Tx) error
	}{
		{"ranks", seedRanks},
		{"vessel types", seedVesselTypes},
		{"currencies", seedCurrencies},
		{"crew statuses", seedCrewStatuses},
		{"allowance types", seedAllowanceTypes},
		{"document types", seedDocumentTypes},
		{"certificate types", seedCertificateTypes},
		{"compliance matrix", seedComplianceMatrix},
		{"languages", seedLanguages},
		{"email templates", seedEmailTemplates},
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, s := range steps {
		if err := s.run(ctx, tx); err != nil {
			return fmt.Errorf("seed %s: %w", s.name, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	if err := settings.SeedDefaults(ctx, settingOverrides); err != nil {
		return fmt.Errorf("seed settings: %w", err)
	}
	return nil
}

func execBatch(ctx context.Context, tx pgx.Tx, batch *pgx.Batch) error {
	return tx.SendBatch(ctx, batch).Close()
}

type rankSeed struct {
	name, code, department string
	officer                bool
}

var ranks = []rankSeed{
	{"Master", "MST", "deck", true},
	{"Chief Officer", "CO", "deck", true},
	{"2nd Officer", "2O", "deck", true},
	{"3rd Officer", "3O", "deck", true},
	{"Bosun", "BSN", "deck", false},
	{"AB", "AB", "deck", false},
	{"OS", "OS", "deck", false},
	{"Chief Engineer", "CE", "engine", true},
	{"2nd Engineer", "2E", "engine", true},
	{"3rd Engineer", "3E", "engine", true},
	{"4th Engineer", "4E", "engine", true},
	{"Engine Cadet", "EC", "engine", false},
	{"Oiler", "OLR", "engine", false},
	{"Wiper", "WPR", "engine", false},
	{"Cook", "CK", "catering_other", false},
	{"Steward", "STW", "catering_other", false},
}

func seedRanks(ctx context.Context, tx pgx.Tx) error {
	b := &pgx.Batch{}
	for i, r := range ranks {
		b.Queue(`INSERT INTO ranks (name, short_code, department, is_officer, sort_order)
			SELECT $1, $2, $3, $4, $5
			WHERE NOT EXISTS (SELECT 1 FROM ranks WHERE upper(short_code) = upper($2) AND deleted_at IS NULL)`,
			r.name, r.code, r.department, r.officer, (i+1)*10)
	}
	return execBatch(ctx, tx, b)
}

func seedVesselTypes(ctx context.Context, tx pgx.Tx) error {
	b := &pgx.Batch{}
	for i, name := range []string{"AHTS", "Crew Boat", "ASD Tug", "Tanker", "Bulk Carrier", "RORO", "Other"} {
		b.Queue(`INSERT INTO vessel_types (name, sort_order)
			SELECT $1, $2 WHERE NOT EXISTS (SELECT 1 FROM vessel_types WHERE lower(name) = lower($1) AND deleted_at IS NULL)`,
			name, (i+1)*10)
	}
	return execBatch(ctx, tx, b)
}

func seedCurrencies(ctx context.Context, tx pgx.Tx) error {
	rows := []struct {
		code, name, symbol string
		decimals           int
	}{
		{"IDR", "Indonesian Rupiah", "Rp", 0},
		{"USD", "US Dollar", "$", 2},
		{"EUR", "Euro", "€", 2},
		{"SGD", "Singapore Dollar", "S$", 2},
	}
	b := &pgx.Batch{}
	for i, c := range rows {
		b.Queue(`INSERT INTO currencies (code, name, symbol, decimals, sort_order) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (code) DO NOTHING`,
			c.code, c.name, c.symbol, c.decimals, (i+1)*10)
	}
	return execBatch(ctx, tx, b)
}

// Crew status codes referenced by business rules.
const (
	StatusAvailable    = "available"
	StatusOnBoard      = "on_board"
	StatusSignOff      = "sign_off"
	StatusMedicalLeave = "medical_leave"
	StatusBlacklisted  = "blacklisted"
)

func seedCrewStatuses(ctx context.Context, tx pgx.Tx) error {
	rows := []struct{ code, name, color string }{
		{StatusAvailable, "Available", "#16A34A"},
		{StatusOnBoard, "On Board", "#2563EB"},
		{StatusSignOff, "Sign Off", "#64748B"},
		{StatusMedicalLeave, "Medical Leave", "#D97706"},
		{StatusBlacklisted, "Inactive/Blacklisted", "#DC2626"},
	}
	b := &pgx.Batch{}
	for i, s := range rows {
		b.Queue(`INSERT INTO crew_statuses (code, name, color, is_system, sort_order) VALUES ($1, $2, $3, true, $4)
			ON CONFLICT (code) DO UPDATE SET is_system = true`, s.code, s.name, s.color, (i+1)*10)
	}
	return execBatch(ctx, tx, b)
}

func seedAllowanceTypes(ctx context.Context, tx pgx.Tx) error {
	rows := []struct{ name, basis string }{
		{"Overtime", "fixed_month"},
		{"Leave Pay", "fixed_month"},
		{"Standby Pay", "fixed_day"},
		{"Offshore Allowance", "fixed_day"},
		{"Bonus", "fixed_month"},
	}
	b := &pgx.Batch{}
	for i, a := range rows {
		b.Queue(`INSERT INTO allowance_types (name, basis, sort_order)
			SELECT $1, $2, $3 WHERE NOT EXISTS (SELECT 1 FROM allowance_types WHERE lower(name) = lower($1) AND deleted_at IS NULL)`,
			a.name, a.basis, (i+1)*10)
	}
	return execBatch(ctx, tx, b)
}

func seedDocumentTypes(ctx context.Context, tx pgx.Tx) error {
	rows := []struct {
		code, name, hint string
		expiry           bool
	}{
		{"ktp", "KTP", "16 digits", false},
		{"passport", "Passport", "e.g. X1234567", true},
		{"seaman_book", "Seaman Book (CDC)", "e.g. F123456", true},
		{"buku_pelaut", "Buku Pelaut", "e.g. G012345", true},
		{"visa", "Visa", "", true},
		{"mcu", "Medical Check Up", "", true},
		{"yellow_fever", "Yellow Fever", "", false},
	}
	b := &pgx.Batch{}
	for i, d := range rows {
		b.Queue(`INSERT INTO document_types (code, name, number_hint, has_expiry, sort_order) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (code) DO NOTHING`,
			d.code, d.name, d.hint, d.expiry, (i+1)*10)
	}
	return execBatch(ctx, tx, b)
}

type certSeed struct {
	code, name, stcw, category string
	expiry                     bool
}

var certificates = []certSeed{
	{"coc_master", "COC Master", "II/2", "coc", true},
	{"coc_chief_officer", "COC Chief Officer", "II/2", "coc", true},
	{"coc_oow_deck", "COC Officer in Charge of a Navigational Watch", "II/1", "coc", true},
	{"coc_chief_engineer", "COC Chief Engineer", "III/2", "coc", true},
	{"coc_second_engineer", "COC Second Engineer", "III/2", "coc", true},
	{"coc_oow_engine", "COC Officer in Charge of an Engineering Watch", "III/1", "coc", true},
	{"bst", "Basic Safety Training", "VI/1", "cop", true},
	{"aff", "Advanced Fire Fighting", "VI/3", "cop", true},
	{"mfa", "Medical First Aid", "VI/4", "cop", true},
	{"pscrb", "Proficiency in Survival Craft and Rescue Boats", "VI/2", "cop", true},
	{"sat", "Security Awareness Training", "VI/6", "cop", false},
	{"rating_nav_watch", "Rating Forming Part of a Navigational Watch", "II/4", "cop", false},
	{"rating_eng_watch", "Rating Forming Part of an Engineering Watch", "III/4", "cop", false},
	{"gmdss", "GMDSS General Operator", "IV/2", "coc", true},
	{"bosiet", "BOSIET", "", "other", true},
	{"huet", "HUET", "", "other", true},
}

func seedCertificateTypes(ctx context.Context, tx pgx.Tx) error {
	b := &pgx.Batch{}
	for i, c := range certificates {
		b.Queue(`INSERT INTO certificate_types (code, name, stcw_reference, category, has_expiry, sort_order) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (code) DO NOTHING`,
			c.code, c.name, c.stcw, c.category, c.expiry, (i+1)*10)
	}
	return execBatch(ctx, tx, b)
}

var cocByRank = map[string]string{
	"MST": "coc_master",
	"CO":  "coc_chief_officer",
	"2O":  "coc_oow_deck",
	"3O":  "coc_oow_deck",
	"CE":  "coc_chief_engineer",
	"2E":  "coc_second_engineer",
	"3E":  "coc_oow_engine",
	"4E":  "coc_oow_engine",
}

var (
	baseDocuments    = []string{"passport", "seaman_book", "mcu"}
	baseCertificates = []string{"bst", "sat"}
)

func seedComplianceMatrix(ctx context.Context, tx pgx.Tx) error {
	b := &pgx.Batch{}
	for _, r := range ranks {
		for _, doc := range baseDocuments {
			b.Queue(complianceDocumentSQL, r.code, doc)
		}
		certs := baseCertificates
		if coc, ok := cocByRank[r.code]; ok {
			certs = append([]string{coc}, certs...)
		}
		for _, cert := range certs {
			b.Queue(complianceCertificateSQL, r.code, cert)
		}
	}
	return execBatch(ctx, tx, b)
}

const complianceDocumentSQL = `
INSERT INTO compliance_requirements (rank_id, requirement_type, document_type_id)
SELECT r.id, 'document', d.id FROM ranks r, document_types d
WHERE upper(r.short_code) = upper($1) AND r.deleted_at IS NULL AND d.code = $2
  AND NOT EXISTS (SELECT 1 FROM compliance_requirements c WHERE c.rank_id = r.id AND c.created_at < now())
ON CONFLICT DO NOTHING`

const complianceCertificateSQL = `
INSERT INTO compliance_requirements (rank_id, requirement_type, certificate_type_id)
SELECT r.id, 'certificate', t.id FROM ranks r, certificate_types t
WHERE upper(r.short_code) = upper($1) AND r.deleted_at IS NULL AND t.code = $2
  AND NOT EXISTS (SELECT 1 FROM compliance_requirements c WHERE c.rank_id = r.id AND c.created_at < now())
ON CONFLICT DO NOTHING`

func seedLanguages(ctx context.Context, tx pgx.Tx) error {
	b := &pgx.Batch{}
	b.Queue(`INSERT INTO languages (name, code, flag_code, sort_order, is_active, is_source) VALUES ('English', 'en', 'gb', 10, true, true) ON CONFLICT (code) DO NOTHING`)
	b.Queue(`INSERT INTO languages (name, code, flag_code, sort_order, is_active, is_source) VALUES ('Indonesian', 'id', 'id', 20, true, false) ON CONFLICT (code) DO NOTHING`)
	return execBatch(ctx, tx, b)
}

func seedEmailTemplates(ctx context.Context, tx pgx.Tx) error {
	b := &pgx.Batch{}
	for _, t := range emailTemplates {
		b.Queue(`INSERT INTO email_templates (code, name, subject, body_html) VALUES ($1, $2, $3, $4) ON CONFLICT (code) DO NOTHING`,
			t.code, t.name, t.subject, t.body)
	}
	return execBatch(ctx, tx, b)
}
