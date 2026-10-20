package repository

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

const baseColumns = "id, is_active, sort_order, created_at, updated_at"

func baseDest(b *model.MasterBase) []any {
	return []any{&b.ID, &b.IsActive, &b.SortOrder, &b.CreatedAt, &b.UpdatedAt}
}

func scanWith(row pgx.Row, b *model.MasterBase, fields ...any) error {
	return row.Scan(append(baseDest(b), fields...)...)
}

var commonSorts = map[string]string{
	"sort_order": "sort_order",
	"is_active":  "is_active",
	"created_at": "created_at",
	"updated_at": "updated_at",
}

func sorts(extra map[string]string) map[string]string {
	out := make(map[string]string, len(commonSorts)+len(extra))
	for k, v := range commonSorts {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func filters(extra map[string]string) map[string]string {
	out := map[string]string{"is_active": "is_active"}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// NewCompanyRepository returns the companies repository.
func NewCompanyRepository(pool *pgxpool.Pool) *MasterRepository[model.Company] {
	return NewMasterRepository(pool, MasterSpec[model.Company]{
		Table:    "companies",
		Select:   baseColumns + ", code, name, type, address, country, phone, email, website, logo_file_id",
		Writable: []string{"code", "name", "type", "address", "country", "phone", "email", "website", "logo_file_id", "is_active"},
		Values: func(c model.Company) []any {
			return []any{c.Code, c.Name, c.Type, c.Address, c.Country, c.Phone, c.Email, c.Website, c.LogoFileID, c.IsActive}
		},
		Scan: func(row pgx.Row) (model.Company, error) {
			var c model.Company
			err := scanWith(row, &c.MasterBase, &c.Code, &c.Name, &c.Type, &c.Address, &c.Country, &c.Phone, &c.Email, &c.Website, &c.LogoFileID)
			return c, err
		},
		Search:       []string{"code", "name", "country", "email"},
		Sorts:        sorts(map[string]string{"code": "code", "name": "name", "type": "type", "country": "country"}),
		Filters:      filters(map[string]string{"type": "type"}),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"code": "lower(code) = lower($1) AND deleted_at IS NULL"},
		Usages: []UsageQuery{
			{"vessels", "SELECT count(*) FROM vessels WHERE (owner_company_id = $1 OR principal_id = $1) AND deleted_at IS NULL"},
			{"contracts", "SELECT count(*) FROM contracts WHERE (principal_id = $1 OR employer_company_id = $1) AND deleted_at IS NULL"},
			{"experiences", "SELECT count(*) FROM experiences WHERE company_id = $1 OR principal_id = $1"},
			{"quotations", "SELECT count(*) FROM quotations WHERE client_company_id = $1 AND deleted_at IS NULL"},
		},
	})
}

// NewVesselRepository returns the vessels repository.
func NewVesselRepository(pool *pgxpool.Pool) *MasterRepository[model.Vessel] {
	return NewMasterRepository(pool, MasterSpec[model.Vessel]{
		Table:    "vessels",
		Select:   baseColumns + ", name, imo_number, flag, vessel_type_id, grt, hp_kw, year_built, owner_company_id, principal_id, status",
		Writable: []string{"name", "imo_number", "flag", "vessel_type_id", "grt", "hp_kw", "year_built", "owner_company_id", "principal_id", "status", "is_active"},
		Values: func(v model.Vessel) []any {
			return []any{v.Name, v.IMONumber, v.Flag, v.VesselTypeID, v.GRT, v.HPKW, v.YearBuilt, v.OwnerCompanyID, v.PrincipalID, v.Status, v.IsActive}
		},
		Scan: func(row pgx.Row) (model.Vessel, error) {
			var v model.Vessel
			err := scanWith(row, &v.MasterBase, &v.Name, &v.IMONumber, &v.Flag, &v.VesselTypeID, &v.GRT, &v.HPKW, &v.YearBuilt, &v.OwnerCompanyID, &v.PrincipalID, &v.Status)
			return v, err
		},
		Search:       []string{"name", "imo_number", "flag"},
		Sorts:        sorts(map[string]string{"name": "name", "imo_number": "imo_number", "flag": "flag", "year_built": "year_built", "grt": "grt", "status": "status"}),
		Filters:      filters(map[string]string{"status": "status", "vessel_type_id": "vessel_type_id", "principal_id": "principal_id", "owner_company_id": "owner_company_id"}),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"imo_number": "imo_number = $1 AND deleted_at IS NULL"},
		Usages: []UsageQuery{
			{"contracts", "SELECT count(*) FROM contracts WHERE vessel_id = $1 AND deleted_at IS NULL"},
			{"experiences", "SELECT count(*) FROM experiences WHERE vessel_id = $1"},
			{"quotations", "SELECT count(*) FROM quotations WHERE vessel_id = $1 AND deleted_at IS NULL"},
		},
	})
}

// NewRankRepository returns the ranks repository.
func NewRankRepository(pool *pgxpool.Pool) *MasterRepository[model.Rank] {
	return NewMasterRepository(pool, MasterSpec[model.Rank]{
		Table:    "ranks",
		Select:   baseColumns + ", name, short_code, department, is_officer",
		Writable: []string{"name", "short_code", "department", "is_officer", "is_active"},
		Values: func(r model.Rank) []any {
			return []any{r.Name, r.ShortCode, r.Department, r.IsOfficer, r.IsActive}
		},
		Scan: func(row pgx.Row) (model.Rank, error) {
			var r model.Rank
			err := scanWith(row, &r.MasterBase, &r.Name, &r.ShortCode, &r.Department, &r.IsOfficer)
			return r, err
		},
		Search:       []string{"name", "short_code"},
		Sorts:        sorts(map[string]string{"name": "name", "short_code": "short_code", "department": "department"}),
		Filters:      filters(map[string]string{"department": "department", "is_officer": "is_officer"}),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"short_code": "upper(short_code) = upper($1) AND deleted_at IS NULL"},
		Usages: []UsageQuery{
			{"crews", "SELECT count(*) FROM crews WHERE applied_rank_id = $1 AND deleted_at IS NULL"},
			{"contracts", "SELECT count(*) FROM contracts WHERE rank_id = $1 AND deleted_at IS NULL"},
			{"experiences", "SELECT count(*) FROM experiences WHERE rank_id = $1"},
			{"vacancies", "SELECT count(*) FROM vacancies WHERE rank_id = $1 AND deleted_at IS NULL"},
			{"applicants", "SELECT count(*) FROM applicants WHERE rank_applied_id = $1 AND deleted_at IS NULL"},
			{"quotation lines", "SELECT count(*) FROM quotation_lines l JOIN quotations q ON q.id = l.quotation_id WHERE l.rank_id = $1 AND q.deleted_at IS NULL"},
		},
	})
}

// NewVesselTypeRepository returns the vessel types repository.
func NewVesselTypeRepository(pool *pgxpool.Pool) *MasterRepository[model.VesselType] {
	return NewMasterRepository(pool, MasterSpec[model.VesselType]{
		Table:    "vessel_types",
		Select:   baseColumns + ", name",
		Writable: []string{"name", "is_active"},
		Values:   func(v model.VesselType) []any { return []any{v.Name, v.IsActive} },
		Scan: func(row pgx.Row) (model.VesselType, error) {
			var v model.VesselType
			err := scanWith(row, &v.MasterBase, &v.Name)
			return v, err
		},
		Search:       []string{"name"},
		Sorts:        sorts(map[string]string{"name": "name"}),
		Filters:      filters(nil),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"name": "lower(name) = lower($1) AND deleted_at IS NULL"},
		Usages: []UsageQuery{
			{"vessels", "SELECT count(*) FROM vessels WHERE vessel_type_id = $1 AND deleted_at IS NULL"},
			{"experiences", "SELECT count(*) FROM experiences WHERE vessel_type_id = $1"},
			{"vacancies", "SELECT count(*) FROM vacancies WHERE vessel_type_id = $1 AND deleted_at IS NULL"},
		},
	})
}

// NewCurrencyRepository returns the currencies repository.
func NewCurrencyRepository(pool *pgxpool.Pool) *MasterRepository[model.Currency] {
	return NewMasterRepository(pool, MasterSpec[model.Currency]{
		Table:    "currencies",
		Select:   baseColumns + ", code, name, symbol, decimals",
		Writable: []string{"code", "name", "symbol", "decimals", "is_active"},
		Values: func(c model.Currency) []any {
			return []any{c.Code, c.Name, c.Symbol, c.Decimals, c.IsActive}
		},
		Scan: func(row pgx.Row) (model.Currency, error) {
			var c model.Currency
			err := scanWith(row, &c.MasterBase, &c.Code, &c.Name, &c.Symbol, &c.Decimals)
			return c, err
		},
		Search:       []string{"code", "name"},
		Sorts:        sorts(map[string]string{"code": "code", "name": "name", "decimals": "decimals"}),
		Filters:      filters(nil),
		DefaultOrder: "sort_order, code",
		Unique:       map[string]string{"code": "code = $1"},
		Usages: []UsageQuery{
			{"contracts", "SELECT count(*) FROM contracts WHERE currency_code = (SELECT code FROM currencies WHERE id = $1) AND deleted_at IS NULL"},
			{"quotations", "SELECT count(*) FROM quotations WHERE currency_code = (SELECT code FROM currencies WHERE id = $1) AND deleted_at IS NULL"},
			{"bank accounts", "SELECT count(*) FROM crew_bank_accounts WHERE currency_code = (SELECT code FROM currencies WHERE id = $1)"},
		},
	})
}

// NewCrewStatusRepository returns the crew statuses repository.
func NewCrewStatusRepository(pool *pgxpool.Pool) *MasterRepository[model.CrewStatus] {
	return NewMasterRepository(pool, MasterSpec[model.CrewStatus]{
		Table:    "crew_statuses",
		Select:   baseColumns + ", code, name, color, is_system",
		Writable: []string{"name", "color", "is_active"},
		Values:   func(s model.CrewStatus) []any { return []any{s.Name, s.Color, s.IsActive} },
		Scan: func(row pgx.Row) (model.CrewStatus, error) {
			var s model.CrewStatus
			err := scanWith(row, &s.MasterBase, &s.Code, &s.Name, &s.Color, &s.IsSystem)
			return s, err
		},
		Search:       []string{"name", "code"},
		Sorts:        sorts(map[string]string{"name": "name", "is_system": "is_system"}),
		Filters:      filters(map[string]string{"is_system": "is_system"}),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"name": "lower(name) = lower($1) AND deleted_at IS NULL"},
		Usages: []UsageQuery{
			{"crews", "SELECT count(*) FROM crews WHERE crew_status_id = $1 AND deleted_at IS NULL"},
		},
	})
}

// NewAllowanceTypeRepository returns the allowance types repository.
func NewAllowanceTypeRepository(pool *pgxpool.Pool) *MasterRepository[model.AllowanceType] {
	return NewMasterRepository(pool, MasterSpec[model.AllowanceType]{
		Table:    "allowance_types",
		Select:   baseColumns + ", name, basis",
		Writable: []string{"name", "basis", "is_active"},
		Values:   func(a model.AllowanceType) []any { return []any{a.Name, a.Basis, a.IsActive} },
		Scan: func(row pgx.Row) (model.AllowanceType, error) {
			var a model.AllowanceType
			err := scanWith(row, &a.MasterBase, &a.Name, &a.Basis)
			return a, err
		},
		Search:       []string{"name"},
		Sorts:        sorts(map[string]string{"name": "name", "basis": "basis"}),
		Filters:      filters(map[string]string{"basis": "basis"}),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"name": "lower(name) = lower($1) AND deleted_at IS NULL"},
		Usages: []UsageQuery{
			{"contract allowances", "SELECT count(*) FROM contract_allowances a JOIN contracts c ON c.id = a.contract_id WHERE a.allowance_type_id = $1 AND c.deleted_at IS NULL"},
		},
	})
}

// NewDocumentTypeRepository returns the document types repository.
func NewDocumentTypeRepository(pool *pgxpool.Pool) *MasterRepository[model.DocumentType] {
	return NewMasterRepository(pool, MasterSpec[model.DocumentType]{
		Table:    "document_types",
		Select:   baseColumns + ", code, name, has_expiry, number_hint",
		Writable: []string{"name", "has_expiry", "number_hint", "is_active"},
		Values: func(d model.DocumentType) []any {
			return []any{d.Name, d.HasExpiry, d.NumberHint, d.IsActive}
		},
		Scan: func(row pgx.Row) (model.DocumentType, error) {
			var d model.DocumentType
			err := scanWith(row, &d.MasterBase, &d.Code, &d.Name, &d.HasExpiry, &d.NumberHint)
			return d, err
		},
		Search:       []string{"name", "code"},
		Sorts:        sorts(map[string]string{"name": "name", "has_expiry": "has_expiry"}),
		Filters:      filters(map[string]string{"has_expiry": "has_expiry"}),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"name": "lower(name) = lower($1) AND deleted_at IS NULL"},
		Usages: []UsageQuery{
			{"crew documents", "SELECT count(*) FROM travel_documents d JOIN crews c ON c.id = d.crew_id WHERE d.document_type_id = $1 AND c.deleted_at IS NULL"},
			{"compliance matrix", "SELECT count(*) FROM compliance_requirements WHERE document_type_id = $1"},
		},
	})
}

// NewCertificateTypeRepository returns the certificate types repository.
func NewCertificateTypeRepository(pool *pgxpool.Pool) *MasterRepository[model.CertificateType] {
	return NewMasterRepository(pool, MasterSpec[model.CertificateType]{
		Table:    "certificate_types",
		Select:   baseColumns + ", code, name, stcw_reference, category, has_expiry",
		Writable: []string{"name", "stcw_reference", "category", "has_expiry", "is_active"},
		Values: func(c model.CertificateType) []any {
			return []any{c.Name, c.STCWReference, c.Category, c.HasExpiry, c.IsActive}
		},
		Scan: func(row pgx.Row) (model.CertificateType, error) {
			var c model.CertificateType
			err := scanWith(row, &c.MasterBase, &c.Code, &c.Name, &c.STCWReference, &c.Category, &c.HasExpiry)
			return c, err
		},
		Search:       []string{"name", "stcw_reference", "code"},
		Sorts:        sorts(map[string]string{"name": "name", "stcw_reference": "stcw_reference", "category": "category", "has_expiry": "has_expiry"}),
		Filters:      filters(map[string]string{"category": "category", "has_expiry": "has_expiry"}),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"name": "lower(name) = lower($1) AND deleted_at IS NULL"},
		Usages: []UsageQuery{
			{"crew certificates", "SELECT count(*) FROM certificates t JOIN crews c ON c.id = t.crew_id WHERE t.certificate_type_id = $1 AND c.deleted_at IS NULL"},
			{"compliance matrix", "SELECT count(*) FROM compliance_requirements WHERE certificate_type_id = $1"},
		},
	})
}
