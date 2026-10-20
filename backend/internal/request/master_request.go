package request

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

var (
	sevenDigits   = regexp.MustCompile(`^[0-9]{7}$`)
	currencyCode  = regexp.MustCompile(`^[A-Z]{3}$`)
	shortCode     = regexp.MustCompile(`^[A-Z0-9/-]+$`)
	hexColor      = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	maxGRT        = decimal.RequireFromString("9999999999.99")
	departments   = []string{"deck", "engine", "catering_other"}
	companyTypes  = []string{model.CompanyTypeCompany, model.CompanyTypePrincipal, model.CompanyTypeBoth}
	vesselStatus  = []string{"active", "laid_up", "sold"}
	bases         = []string{"fixed_month", "fixed_day", "percent_basic"}
	certificateCa = []string{"coc", "cop", "endorsement", "other"}
)

// Departments lists rank departments.
func Departments() []string { return cloneStrings(departments) }

// CompanyTypes lists company types.
func CompanyTypes() []string { return cloneStrings(companyTypes) }

// VesselStatuses lists vessel statuses.
func VesselStatuses() []string { return cloneStrings(vesselStatus) }

// AllowanceBases lists allowance calculation bases.
func AllowanceBases() []string { return cloneStrings(bases) }

// CertificateCategories lists certificate categories.
func CertificateCategories() []string { return cloneStrings(certificateCa) }

func cloneStrings(s []string) []string { return append([]string(nil), s...) }

// Reorder is the body of PATCH /{master}/reorder.
type Reorder struct {
	IDs []uuid.UUID `json:"ids"`
}

// Validate implements Validator.
func (r *Reorder) Validate() map[string]string {
	if len(r.IDs) == 0 {
		return map[string]string{"ids": "The ids are required."}
	}
	return nil
}

// Company is the body of POST and PUT /companies.
type Company struct {
	Code       string     `json:"code"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Address    string     `json:"address"`
	Country    string     `json:"country"`
	Phone      string     `json:"phone"`
	Email      string     `json:"email"`
	Website    string     `json:"website"`
	LogoFileID *uuid.UUID `json:"logo_file_id"`
	IsActive   *bool      `json:"is_active"`
}

// Validate implements Validator.
func (r *Company) Validate() map[string]string {
	trimAll(&r.Code, &r.Name, &r.Type, &r.Address, &r.Country, &r.Phone, &r.Email, &r.Website)
	r.Code = strings.ToUpper(r.Code)
	f := FieldErrors{}
	f.Required("code", r.Code, 20)
	f.Required("name", r.Name, 200)
	f.OneOf("type", r.Type, companyTypes...)
	f.MaxLen("address", r.Address, 500)
	f.MaxLen("country", r.Country, 80)
	f.MaxLen("phone", r.Phone, 40)
	f.Email("email", r.Email)
	f.URL("website", r.Website)
	return f
}

// Model converts the request to a model.
func (r *Company) Model() model.Company {
	return model.Company{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Code:       r.Code, Name: r.Name, Type: r.Type, Address: r.Address, Country: r.Country,
		Phone: r.Phone, Email: r.Email, Website: r.Website, LogoFileID: r.LogoFileID,
	}
}

// Vessel is the body of POST and PUT /vessels.
type Vessel struct {
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
	IsActive       *bool            `json:"is_active"`
}

// Validate implements Validator.
func (r *Vessel) Validate() map[string]string {
	trimAll(&r.Name, &r.IMONumber, &r.Flag, &r.HPKW, &r.Status)
	if r.Status == "" {
		r.Status = "active"
	}
	f := FieldErrors{}
	f.Required("name", r.Name, 120)
	f.Required("imo_number", r.IMONumber, 7)
	f.Matches("imo_number", r.IMONumber, sevenDigits, "The IMO number must be exactly 7 digits.")
	f.MaxLen("flag", r.Flag, 80)
	f.MaxLen("hp_kw", r.HPKW, 40)
	f.Range("year_built", r.YearBuilt, 1900, 2100)
	f.OneOf("status", r.Status, vesselStatus...)
	if r.GRT != nil && (r.GRT.IsNegative() || r.GRT.GreaterThan(maxGRT)) {
		f.set("grt", "The GRT must be between 0 and 9,999,999,999.99.")
	}
	return f
}

// Model converts the request to a model.
func (r *Vessel) Model() model.Vessel {
	var grt *decimal.Decimal
	if r.GRT != nil {
		v := r.GRT.Round(2)
		grt = &v
	}
	return model.Vessel{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Name:       r.Name, IMONumber: r.IMONumber, Flag: r.Flag, VesselTypeID: r.VesselTypeID, GRT: grt,
		HPKW: r.HPKW, YearBuilt: r.YearBuilt, OwnerCompanyID: r.OwnerCompanyID, PrincipalID: r.PrincipalID, Status: r.Status,
	}
}

// Rank is the body of POST and PUT /ranks.
type Rank struct {
	Name       string `json:"name"`
	ShortCode  string `json:"short_code"`
	Department string `json:"department"`
	IsOfficer  bool   `json:"is_officer"`
	IsActive   *bool  `json:"is_active"`
}

// Validate implements Validator.
func (r *Rank) Validate() map[string]string {
	trimAll(&r.Name, &r.ShortCode, &r.Department)
	r.ShortCode = strings.ToUpper(r.ShortCode)
	f := FieldErrors{}
	f.Required("name", r.Name, 80)
	f.Required("short_code", r.ShortCode, 10)
	f.Matches("short_code", r.ShortCode, shortCode, "The short code may contain only letters, digits, / and -.")
	f.OneOf("department", r.Department, departments...)
	return f
}

// Model converts the request to a model.
func (r *Rank) Model() model.Rank {
	return model.Rank{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Name:       r.Name, ShortCode: r.ShortCode, Department: r.Department, IsOfficer: r.IsOfficer,
	}
}

// VesselType is the body of POST and PUT /vessel-types.
type VesselType struct {
	Name     string `json:"name"`
	IsActive *bool  `json:"is_active"`
}

// Validate implements Validator.
func (r *VesselType) Validate() map[string]string {
	trimAll(&r.Name)
	f := FieldErrors{}
	f.Required("name", r.Name, 80)
	return f
}

// Model converts the request to a model.
func (r *VesselType) Model() model.VesselType {
	return model.VesselType{MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)}, Name: r.Name}
}

// Currency is the body of POST and PUT /currencies.
type Currency struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals *int   `json:"decimals"`
	IsActive *bool  `json:"is_active"`
}

// Validate implements Validator.
func (r *Currency) Validate() map[string]string {
	trimAll(&r.Code, &r.Name, &r.Symbol)
	r.Code = strings.ToUpper(r.Code)
	f := FieldErrors{}
	f.Required("code", r.Code, 3)
	f.Matches("code", r.Code, currencyCode, "The code must be a 3-letter ISO 4217 code.")
	f.MaxLen("name", r.Name, 80)
	f.MaxLen("symbol", r.Symbol, 8)
	if r.Decimals == nil {
		f.set("decimals", "The decimals are required.")
	}
	f.Range("decimals", r.Decimals, 0, 4)
	return f
}

// Model converts the request to a model.
func (r *Currency) Model() model.Currency {
	decimals := 2
	if r.Decimals != nil {
		decimals = *r.Decimals
	}
	return model.Currency{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Code:       r.Code, Name: r.Name, Symbol: r.Symbol, Decimals: decimals,
	}
}

// CrewStatus is the body of POST and PUT /crew-statuses.
type CrewStatus struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	IsActive *bool  `json:"is_active"`
}

// Validate implements Validator.
func (r *CrewStatus) Validate() map[string]string {
	trimAll(&r.Name, &r.Color)
	r.Color = strings.ToUpper(r.Color)
	f := FieldErrors{}
	f.Required("name", r.Name, 60)
	f.Required("color", r.Color, 7)
	f.Matches("color", r.Color, hexColor, "The color must be a hex color such as #2563EB.")
	return f
}

// Model converts the request to a model.
func (r *CrewStatus) Model() model.CrewStatus {
	return model.CrewStatus{MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)}, Name: r.Name, Color: r.Color}
}

// AllowanceType is the body of POST and PUT /allowance-types.
type AllowanceType struct {
	Name     string `json:"name"`
	Basis    string `json:"basis"`
	IsActive *bool  `json:"is_active"`
}

// Validate implements Validator.
func (r *AllowanceType) Validate() map[string]string {
	trimAll(&r.Name, &r.Basis)
	f := FieldErrors{}
	f.Required("name", r.Name, 80)
	f.OneOf("basis", r.Basis, bases...)
	return f
}

// Model converts the request to a model.
func (r *AllowanceType) Model() model.AllowanceType {
	return model.AllowanceType{MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)}, Name: r.Name, Basis: r.Basis}
}

// DocumentType is the body of POST and PUT /document-types.
type DocumentType struct {
	Name       string `json:"name"`
	HasExpiry  *bool  `json:"has_expiry"`
	NumberHint string `json:"number_hint"`
	IsActive   *bool  `json:"is_active"`
}

// Validate implements Validator.
func (r *DocumentType) Validate() map[string]string {
	trimAll(&r.Name, &r.NumberHint)
	f := FieldErrors{}
	f.Required("name", r.Name, 80)
	f.MaxLen("number_hint", r.NumberHint, 80)
	return f
}

// Model converts the request to a model.
func (r *DocumentType) Model() model.DocumentType {
	return model.DocumentType{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Name:       r.Name, HasExpiry: boolOr(r.HasExpiry, true), NumberHint: r.NumberHint,
	}
}

// CertificateType is the body of POST and PUT /certificate-types.
type CertificateType struct {
	Name          string `json:"name"`
	STCWReference string `json:"stcw_reference"`
	Category      string `json:"category"`
	HasExpiry     *bool  `json:"has_expiry"`
	IsActive      *bool  `json:"is_active"`
}

// Validate implements Validator.
func (r *CertificateType) Validate() map[string]string {
	trimAll(&r.Name, &r.STCWReference, &r.Category)
	f := FieldErrors{}
	f.Required("name", r.Name, 160)
	f.MaxLen("stcw_reference", r.STCWReference, 20)
	f.OneOf("category", r.Category, certificateCa...)
	return f
}

// Model converts the request to a model.
func (r *CertificateType) Model() model.CertificateType {
	return model.CertificateType{
		MasterBase: model.MasterBase{IsActive: boolOr(r.IsActive, true)},
		Name:       r.Name, STCWReference: r.STCWReference, Category: r.Category, HasExpiry: boolOr(r.HasExpiry, true),
	}
}

// ComplianceCell is one checked cell of the compliance matrix.
type ComplianceCell struct {
	RankID            uuid.UUID  `json:"rank_id"`
	DocumentTypeID    *uuid.UUID `json:"document_type_id"`
	CertificateTypeID *uuid.UUID `json:"certificate_type_id"`
}

// ComplianceMatrix is the body of PUT /compliance-matrix.
type ComplianceMatrix struct {
	Cells []ComplianceCell `json:"cells"`
}

// Validate implements Validator.
func (r *ComplianceMatrix) Validate() map[string]string {
	if r.Cells == nil {
		return map[string]string{"cells": "The cells are required; send an empty list to clear the matrix."}
	}
	if len(r.Cells) > 5000 {
		return map[string]string{"cells": "The matrix may not exceed 5000 cells."}
	}
	return nil
}

// Model converts the request to model cells.
func (r *ComplianceMatrix) Model() []model.ComplianceCell {
	out := make([]model.ComplianceCell, len(r.Cells))
	for i, c := range r.Cells {
		out[i] = model.ComplianceCell{RankID: c.RankID, DocumentTypeID: c.DocumentTypeID, CertificateTypeID: c.CertificateTypeID}
	}
	return out
}

func trimAll(fields ...*string) {
	for _, f := range fields {
		*f = strings.TrimSpace(*f)
	}
}
