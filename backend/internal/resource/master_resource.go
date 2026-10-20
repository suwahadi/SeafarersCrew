package resource

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// MasterBase is the shared part of every master resource.
type MasterBase struct {
	ID        uuid.UUID `json:"id"`
	IsActive  bool      `json:"is_active"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func masterBase(b model.MasterBase) MasterBase {
	return MasterBase{ID: b.ID, IsActive: b.IsActive, SortOrder: b.SortOrder, CreatedAt: b.CreatedAt.UTC(), UpdatedAt: b.UpdatedAt.UTC()}
}

// Company is the API shape of a company.
type Company struct {
	MasterBase
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Address    string     `json:"address"`
	Country    string     `json:"country"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email"`
	Website    string     `json:"website"`
	LogoFileID *uuid.UUID `json:"logo_file_id"`
}

// NewCompany maps a company.
func NewCompany(c model.Company) Company {
	return Company{masterBase(c.MasterBase), c.Code, c.Name, c.Type, c.Address, c.Country, c.Phone, c.Email, c.Website, c.LogoFileID}
}

// Vessel is the API shape of a vessel.
type Vessel struct {
	MasterBase
	Name           string           `json:"name"`
	IMONumber      string           `json:"imo_number"`
	Flag           string           `json:"flag"`
	VesselTypeID   *uuid.UUID       `json:"vessel_type_id"`
	GRT            *decimal.Decimal `json:"grt"`
	HPKW           string           `json:"hp_kw"`
	YearBuilt      *int             `json:"year_built"`
	OwnerCompanyID *uuid.UUID       `json:"owner_company_id"`
	PrincipalID    *uuid.UUID       `json:"principal_id"`
	Status         string           `json:"status"`
}

// NewVessel maps a vessel.
func NewVessel(v model.Vessel) Vessel {
	return Vessel{masterBase(v.MasterBase), v.Name, v.IMONumber, v.Flag, v.VesselTypeID, v.GRT, v.HPKW, v.YearBuilt, v.OwnerCompanyID, v.PrincipalID, v.Status}
}

// Rank is the API shape of a rank.
type Rank struct {
	MasterBase
	Name       string `json:"name"`
	ShortCode  string `json:"short_code"`
	Department string `json:"department"`
	IsOfficer  bool   `json:"is_officer"`
}

// NewRank maps a rank.
func NewRank(r model.Rank) Rank {
	return Rank{masterBase(r.MasterBase), r.Name, r.ShortCode, r.Department, r.IsOfficer}
}

// VesselType is the API shape of a vessel type.
type VesselType struct {
	MasterBase
	Name string `json:"name"`
}

// NewVesselType maps a vessel type.
func NewVesselType(v model.VesselType) VesselType {
	return VesselType{masterBase(v.MasterBase), v.Name}
}

// Currency is the API shape of a currency.
type Currency struct {
	MasterBase
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// NewCurrency maps a currency.
func NewCurrency(c model.Currency) Currency {
	return Currency{masterBase(c.MasterBase), c.Code, c.Name, c.Symbol, c.Decimals}
}

// CrewStatus is the API shape of a crew status.
type CrewStatus struct {
	MasterBase
	Code     *string `json:"code"`
	Name     string  `json:"name"`
	Color    string  `json:"color"`
	IsSystem bool    `json:"is_system"`
}

// NewCrewStatus maps a crew status.
func NewCrewStatus(s model.CrewStatus) CrewStatus {
	return CrewStatus{masterBase(s.MasterBase), s.Code, s.Name, s.Color, s.IsSystem}
}

// AllowanceType is the API shape of an allowance type.
type AllowanceType struct {
	MasterBase
	Name  string `json:"name"`
	Basis string `json:"basis"`
}

// NewAllowanceType maps an allowance type.
func NewAllowanceType(a model.AllowanceType) AllowanceType {
	return AllowanceType{masterBase(a.MasterBase), a.Name, a.Basis}
}

// DocumentType is the API shape of a document type.
type DocumentType struct {
	MasterBase
	Code       *string `json:"code"`
	Name       string  `json:"name"`
	HasExpiry  bool    `json:"has_expiry"`
	NumberHint string  `json:"number_hint"`
}

// NewDocumentType maps a document type.
func NewDocumentType(d model.DocumentType) DocumentType {
	return DocumentType{masterBase(d.MasterBase), d.Code, d.Name, d.HasExpiry, d.NumberHint}
}

// CertificateType is the API shape of a certificate type.
type CertificateType struct {
	MasterBase
	Code          *string `json:"code"`
	Name          string  `json:"name"`
	STCWReference string  `json:"stcw_reference"`
	Category      string  `json:"category"`
	HasExpiry     bool    `json:"has_expiry"`
}

// NewCertificateType maps a certificate type.
func NewCertificateType(c model.CertificateType) CertificateType {
	return CertificateType{masterBase(c.MasterBase), c.Code, c.Name, c.STCWReference, c.Category, c.HasExpiry}
}

// MasterOptions is the dropdown payload of GET /options.
type MasterOptions struct {
	Companies        []Company         `json:"companies"`
	Vessels          []Vessel          `json:"vessels"`
	Ranks            []Rank            `json:"ranks"`
	VesselTypes      []VesselType      `json:"vessel_types"`
	Currencies       []Currency        `json:"currencies"`
	CrewStatuses     []CrewStatus      `json:"crew_statuses"`
	AllowanceTypes   []AllowanceType   `json:"allowance_types"`
	DocumentTypes    []DocumentType    `json:"document_types"`
	CertificateTypes []CertificateType `json:"certificate_types"`
}

// NewMasterOptions maps all active masters.
func NewMasterOptions(o model.MasterOptions) MasterOptions {
	return MasterOptions{
		Companies:        MapAll(o.Companies, NewCompany),
		Vessels:          MapAll(o.Vessels, NewVessel),
		Ranks:            MapAll(o.Ranks, NewRank),
		VesselTypes:      MapAll(o.VesselTypes, NewVesselType),
		Currencies:       MapAll(o.Currencies, NewCurrency),
		CrewStatuses:     MapAll(o.CrewStatuses, NewCrewStatus),
		AllowanceTypes:   MapAll(o.AllowanceTypes, NewAllowanceType),
		DocumentTypes:    MapAll(o.DocumentTypes, NewDocumentType),
		CertificateTypes: MapAll(o.CertificateTypes, NewCertificateType),
	}
}

// ComplianceCell is one checked cell.
type ComplianceCell struct {
	RankID            uuid.UUID  `json:"rank_id"`
	DocumentTypeID    *uuid.UUID `json:"document_type_id"`
	CertificateTypeID *uuid.UUID `json:"certificate_type_id"`
}

// ComplianceMatrix is the payload of GET and PUT /compliance-matrix.
type ComplianceMatrix struct {
	Ranks            []Rank            `json:"ranks"`
	DocumentTypes    []DocumentType    `json:"document_types"`
	CertificateTypes []CertificateType `json:"certificate_types"`
	Cells            []ComplianceCell  `json:"cells"`
}

// NewComplianceMatrix maps the matrix.
func NewComplianceMatrix(m model.ComplianceMatrix) ComplianceMatrix {
	return ComplianceMatrix{
		Ranks:            MapAll(m.Ranks, NewRank),
		DocumentTypes:    MapAll(m.DocumentTypes, NewDocumentType),
		CertificateTypes: MapAll(m.CertificateTypes, NewCertificateType),
		Cells: MapAll(m.Cells, func(c model.ComplianceCell) ComplianceCell {
			return ComplianceCell{RankID: c.RankID, DocumentTypeID: c.DocumentTypeID, CertificateTypeID: c.CertificateTypeID}
		}),
	}
}

// MapAll maps every item with fn and never returns nil.
func MapAll[T, R any](items []T, fn func(T) R) []R {
	out := make([]R, len(items))
	for i, it := range items {
		out[i] = fn(it)
	}
	return out
}
