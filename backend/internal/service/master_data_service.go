package service

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// MasterLookup answers cross-record questions needed by master data rules.
type MasterLookup interface {
	CompanyType(ctx context.Context, id uuid.UUID) (string, error)
	VesselTypeExists(ctx context.Context, id uuid.UUID) (bool, error)
	FileExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// MasterStores groups the stores of every master table.
type MasterStores struct {
	Companies        MasterStore[model.Company]
	Vessels          MasterStore[model.Vessel]
	Ranks            MasterStore[model.Rank]
	VesselTypes      MasterStore[model.VesselType]
	Currencies       MasterStore[model.Currency]
	CrewStatuses     MasterStore[model.CrewStatus]
	AllowanceTypes   MasterStore[model.AllowanceType]
	DocumentTypes    MasterStore[model.DocumentType]
	CertificateTypes MasterStore[model.CertificateType]
}

// MasterData groups the services of every master table.
type MasterData struct {
	Companies        *MasterService[model.Company]
	Vessels          *MasterService[model.Vessel]
	Ranks            *MasterService[model.Rank]
	VesselTypes      *MasterService[model.VesselType]
	Currencies       *MasterService[model.Currency]
	CrewStatuses     *MasterService[model.CrewStatus]
	AllowanceTypes   *MasterService[model.AllowanceType]
	DocumentTypes    *MasterService[model.DocumentType]
	CertificateTypes *MasterService[model.CertificateType]
}

// NewMasterData builds every master service; onChange runs after each write, e.g. to invalidate cached options.
func NewMasterData(stores MasterStores, lookup MasterLookup, onChange func(ctx context.Context) error) MasterData {
	return MasterData{
		Companies:        NewMasterService(stores.Companies, companyRules(lookup), onChange),
		Vessels:          NewMasterService(stores.Vessels, vesselRules(lookup), onChange),
		Ranks:            NewMasterService(stores.Ranks, rankRules(), onChange),
		VesselTypes:      NewMasterService(stores.VesselTypes, vesselTypeRules(), onChange),
		Currencies:       NewMasterService(stores.Currencies, currencyRules(), onChange),
		CrewStatuses:     NewMasterService(stores.CrewStatuses, crewStatusRules(), onChange),
		AllowanceTypes:   NewMasterService(stores.AllowanceTypes, allowanceTypeRules(), onChange),
		DocumentTypes:    NewMasterService(stores.DocumentTypes, documentTypeRules(), onChange),
		CertificateTypes: NewMasterService(stores.CertificateTypes, certificateTypeRules(), onChange),
	}
}

// ValidIMO reports whether s is a 7-digit IMO number with a correct check digit.
func ValidIMO(s string) bool {
	if len(s) != 7 {
		return false
	}
	sum := 0
	for i := 0; i < 7; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
		if i < 6 {
			sum += int(s[i]-'0') * (7 - i)
		}
	}
	return sum%10 == int(s[6]-'0')
}

func companyRules(lookup MasterLookup) MasterRules[model.Company] {
	return MasterRules[model.Company]{
		Noun: "Company",
		Prepare: func(ctx context.Context, c *model.Company, _ *model.Company) error {
			return requireFile(ctx, lookup, c.LogoFileID, "logo_file_id")
		},
		Unique: []UniqueField[model.Company]{
			{Field: "code", Message: "The code is already used by another company.", Value: func(c model.Company) string { return c.Code }},
		},
	}
}

func vesselRules(lookup MasterLookup) MasterRules[model.Vessel] {
	return MasterRules[model.Vessel]{
		Noun: "Vessel",
		Prepare: func(ctx context.Context, v *model.Vessel, _ *model.Vessel) error {
			fields := map[string]string{}
			if !ValidIMO(v.IMONumber) {
				fields["imo_number"] = "The IMO number must be 7 digits with a valid check digit."
			}
			if v.VesselTypeID != nil {
				ok, err := lookup.VesselTypeExists(ctx, *v.VesselTypeID)
				if err != nil {
					return err
				}
				if !ok {
					fields["vessel_type_id"] = "The vessel type does not exist."
				}
			}
			if err := checkCompanyType(ctx, lookup, v.OwnerCompanyID, "owner_company_id", "The owner must be a company.", model.CompanyTypeCompany, fields); err != nil {
				return err
			}
			if err := checkCompanyType(ctx, lookup, v.PrincipalID, "principal_id", "The principal must be a principal.", model.CompanyTypePrincipal, fields); err != nil {
				return err
			}
			if len(fields) > 0 {
				return apperr.Validation(fields)
			}
			return nil
		},
		Unique: []UniqueField[model.Vessel]{
			{Field: "imo_number", Message: "The IMO number is already used by another vessel.", Value: func(v model.Vessel) string { return v.IMONumber }},
		},
	}
}

func checkCompanyType(ctx context.Context, lookup MasterLookup, id *uuid.UUID, field, msg, want string, fields map[string]string) error {
	if id == nil {
		return nil
	}
	t, err := lookup.CompanyType(ctx, *id)
	if errors.Is(err, repository.ErrNotFound) {
		fields[field] = "The company does not exist."
		return nil
	}
	if err != nil {
		return err
	}
	if t != want && t != model.CompanyTypeBoth {
		fields[field] = msg
	}
	return nil
}

func requireFile(ctx context.Context, lookup MasterLookup, id *uuid.UUID, field string) error {
	if id == nil {
		return nil
	}
	ok, err := lookup.FileExists(ctx, *id)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.Validation(map[string]string{field: "The file does not exist."})
	}
	return nil
}

func rankRules() MasterRules[model.Rank] {
	return MasterRules[model.Rank]{
		Noun: "Rank",
		Unique: []UniqueField[model.Rank]{
			{Field: "short_code", Message: "The short code is already used by another rank.", Value: func(r model.Rank) string { return r.ShortCode }},
		},
	}
}

func vesselTypeRules() MasterRules[model.VesselType] {
	return MasterRules[model.VesselType]{
		Noun: "Vessel type",
		Unique: []UniqueField[model.VesselType]{
			{Field: "name", Message: "The name is already used by another vessel type.", Value: func(v model.VesselType) string { return v.Name }},
		},
	}
}

func currencyRules() MasterRules[model.Currency] {
	return MasterRules[model.Currency]{
		Noun: "Currency",
		Prepare: func(_ context.Context, c *model.Currency, existing *model.Currency) error {
			if existing != nil && existing.Code != c.Code {
				return apperr.Validation(map[string]string{"code": "The currency code cannot be changed."})
			}
			return nil
		},
		Unique: []UniqueField[model.Currency]{
			{Field: "code", Message: "The currency code already exists, possibly on a deleted currency.", Value: func(c model.Currency) string { return c.Code }},
		},
	}
}

func crewStatusRules() MasterRules[model.CrewStatus] {
	return MasterRules[model.CrewStatus]{
		Noun: "Crew status",
		Prepare: func(_ context.Context, s *model.CrewStatus, existing *model.CrewStatus) error {
			s.Code, s.IsSystem = nil, false
			if existing == nil {
				return nil
			}
			s.Code, s.IsSystem = existing.Code, existing.IsSystem
			if s.IsSystem && !s.IsActive {
				return apperr.Validation(map[string]string{"is_active": "System statuses cannot be deactivated."})
			}
			return nil
		},
		Unique: []UniqueField[model.CrewStatus]{
			{Field: "name", Message: "The name is already used by another status.", Value: func(s model.CrewStatus) string { return s.Name }},
		},
		CanDelete: func(s model.CrewStatus) error {
			if s.IsSystem {
				return apperr.New(http.StatusConflict, "system_record", "System statuses cannot be deleted.")
			}
			return nil
		},
	}
}

func allowanceTypeRules() MasterRules[model.AllowanceType] {
	return MasterRules[model.AllowanceType]{
		Noun: "Allowance type",
		Unique: []UniqueField[model.AllowanceType]{
			{Field: "name", Message: "The name is already used by another allowance type.", Value: func(a model.AllowanceType) string { return a.Name }},
		},
	}
}

func documentTypeRules() MasterRules[model.DocumentType] {
	return MasterRules[model.DocumentType]{
		Noun: "Document type",
		Prepare: func(_ context.Context, d *model.DocumentType, existing *model.DocumentType) error {
			d.Code = nil
			if existing != nil {
				d.Code = existing.Code
			}
			return nil
		},
		Unique: []UniqueField[model.DocumentType]{
			{Field: "name", Message: "The name is already used by another document type.", Value: func(d model.DocumentType) string { return d.Name }},
		},
	}
}

func certificateTypeRules() MasterRules[model.CertificateType] {
	return MasterRules[model.CertificateType]{
		Noun: "Certificate type",
		Prepare: func(_ context.Context, c *model.CertificateType, existing *model.CertificateType) error {
			c.Code = nil
			if existing != nil {
				c.Code = existing.Code
			}
			return nil
		},
		Unique: []UniqueField[model.CertificateType]{
			{Field: "name", Message: "The name is already used by another certificate type.", Value: func(c model.CertificateType) string { return c.Name }},
		},
	}
}
