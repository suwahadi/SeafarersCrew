package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// MasterBase holds the columns shared by every master data table.
type MasterBase struct {
	ID        uuid.UUID
	IsActive  bool
	SortOrder int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Company types.
const (
	CompanyTypeCompany   = "company"
	CompanyTypePrincipal = "principal"
	CompanyTypeBoth      = "both"
)

// Company is a company, a principal or both.
type Company struct {
	MasterBase
	Code       string
	Name       string
	Type       string
	Address    string
	Country    string
	Phone      string
	Email      string
	Website    string
	LogoFileID *uuid.UUID
}

// Vessel is a ship managed or crewed by the agency.
type Vessel struct {
	MasterBase
	Name           string
	IMONumber      string
	Flag           string
	VesselTypeID   *uuid.UUID
	GRT            *decimal.Decimal
	HPKW           string
	YearBuilt      *int
	OwnerCompanyID *uuid.UUID
	PrincipalID    *uuid.UUID
	Status         string
}

// Rank is a shipboard position.
type Rank struct {
	MasterBase
	Name       string
	ShortCode  string
	Department string
	IsOfficer  bool
}

// VesselType classifies vessels.
type VesselType struct {
	MasterBase
	Name string
}

// Currency is an ISO 4217 currency.
type Currency struct {
	MasterBase
	Code     string
	Name     string
	Symbol   string
	Decimals int
}

// CrewStatus is a crew availability status; system statuses drive business rules.
type CrewStatus struct {
	MasterBase
	Code     *string
	Name     string
	Color    string
	IsSystem bool
}

// AllowanceType is a kind of contract allowance.
type AllowanceType struct {
	MasterBase
	Name  string
	Basis string
}

// DocumentType is a kind of travel or identity document.
type DocumentType struct {
	MasterBase
	Code       *string
	Name       string
	HasExpiry  bool
	NumberHint string
}

// CertificateType is a kind of STCW or other certificate.
type CertificateType struct {
	MasterBase
	Code          *string
	Name          string
	STCWReference string
	Category      string
	HasExpiry     bool
}

// Usage counts active records referencing a master record.
type Usage struct {
	Label string
	Count int
}

// MasterOptions holds every active master list for dropdowns.
type MasterOptions struct {
	Companies        []Company
	Vessels          []Vessel
	Ranks            []Rank
	VesselTypes      []VesselType
	Currencies       []Currency
	CrewStatuses     []CrewStatus
	AllowanceTypes   []AllowanceType
	DocumentTypes    []DocumentType
	CertificateTypes []CertificateType
}

// ComplianceCell marks one mandatory document or certificate type for a rank.
type ComplianceCell struct {
	RankID            uuid.UUID
	DocumentTypeID    *uuid.UUID
	CertificateTypeID *uuid.UUID
}

// ComplianceMatrix is the editable grid of ranks against document and certificate types.
type ComplianceMatrix struct {
	Ranks            []Rank
	DocumentTypes    []DocumentType
	CertificateTypes []CertificateType
	Cells            []ComplianceCell
}
