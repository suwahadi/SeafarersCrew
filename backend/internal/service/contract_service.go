package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/money"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// ContractStore persists contracts.
type ContractStore interface {
	Create(ctx context.Context, k model.Contract, publicID string) (uuid.UUID, error)
	Update(ctx context.Context, k model.Contract, status string) error
	GetByPublicID(ctx context.Context, publicID string) (model.Contract, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Contract, error)
	ActiveForCrew(ctx context.Context, crewID uuid.UUID) (model.Contract, error)
	List(ctx context.Context, f model.ContractFilter, limit, offset int) (repository.Page[model.Contract], error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Activate(ctx context.Context, id, crewID uuid.UUID) error
	Finish(ctx context.Context, id, crewID uuid.UUID, status string, actual time.Time, reason string, x model.Experience) error
}

// ContractLookups validates contract references.
type ContractLookups interface {
	CompanyType(ctx context.Context, id uuid.UUID) (string, error)
	VesselActive(ctx context.Context, id uuid.UUID) (bool, error)
	RankActive(ctx context.Context, id uuid.UUID) (bool, error)
	CurrencyDecimals(ctx context.Context, code string) (int, error)
	AllowanceType(ctx context.Context, id uuid.UUID) (string, string, error)
}

// CrewFinder resolves crew members by Crew ID.
type CrewFinder interface {
	Get(ctx context.Context, publicID string) (model.Crew, error)
}

// NextReliefFinder returns the next open relief date of a contract.
type NextReliefFinder interface {
	NextReliefDate(ctx context.Context, contractID uuid.UUID) (*time.Time, error)
}

// ContractService implements contract records and their lifecycle (PRD 6.4, 11.4).
type ContractService struct {
	contracts ContractStore
	lookups   ContractLookups
	crews     CrewFinder
	reliefs   NextReliefFinder
	settings  SettingsReader
	now       func() time.Time
	newID     func() (string, error)
}

// NewContractService builds a ContractService.
func NewContractService(contracts ContractStore, lookups ContractLookups, crews CrewFinder, reliefs NextReliefFinder, settings SettingsReader, now func() time.Time) *ContractService {
	return &ContractService{contracts: contracts, lookups: lookups, crews: crews, reliefs: reliefs, settings: settings, now: now, newID: publicid.New}
}

// Create stores a new Draft contract.
func (s *ContractService) Create(ctx context.Context, in model.Contract) (model.Contract, error) {
	k, err := s.prepare(ctx, in)
	if err != nil {
		return model.Contract{}, err
	}
	for range publicid.MaxAttempts {
		pid, err := s.newID()
		if err != nil {
			return model.Contract{}, err
		}
		id, err := s.contracts.Create(ctx, k, pid)
		if postgres.IsUniqueViolation(err, "contracts_public_id_key") {
			continue
		}
		if err != nil {
			return model.Contract{}, err
		}
		return s.contracts.GetByID(ctx, id)
	}
	return model.Contract{}, fmt.Errorf("create contract: no unique public id after %d attempts", publicid.MaxAttempts)
}

// Get returns a contract by its Contract ID.
func (s *ContractService) Get(ctx context.Context, publicID string) (model.Contract, error) {
	publicID = strings.ToUpper(publicID)
	if !publicid.Valid(publicID) {
		return model.Contract{}, apperr.NotFound("Contract")
	}
	k, err := s.contracts.GetByPublicID(ctx, publicID)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Contract{}, apperr.NotFound("Contract")
	}
	return k, err
}

// List returns one page of contracts.
func (s *ContractService) List(ctx context.Context, f model.ContractFilter, withinDays *int, page, perPage int) (repository.Page[model.Contract], error) {
	if withinDays != nil {
		today, err := s.today(ctx)
		if err != nil {
			return repository.Page[model.Contract]{}, err
		}
		before := today.AddDate(0, 0, *withinDays)
		f.SignOffBefore = &before
	}
	return s.contracts.List(ctx, f, perPage, (page-1)*perPage)
}

// Update edits a Draft contract freely and an Active one only in ActiveEditableFields; closed contracts are read-only.
func (s *ContractService) Update(ctx context.Context, publicID string, in model.Contract) (model.Contract, error) {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Contract{}, err
	}
	switch cur.Status {
	case model.ContractDraft, model.ContractActive:
	default:
		return model.Contract{}, apperr.Conflict("contract_closed", "A completed or terminated contract cannot be changed.")
	}
	next, err := s.prepare(ctx, in)
	if err != nil {
		return model.Contract{}, err
	}
	if cur.Status == model.ContractActive {
		if locked := LockedChanges(cur, next); len(locked) > 0 {
			e := apperr.Conflict("contract_locked", "An active contract only allows changes to the planned sign-off date, ports and notes.")
			e.Fields = map[string]string{}
			for _, f := range locked {
				e.Fields[f] = "This field cannot change once the contract is active."
			}
			return model.Contract{}, e
		}
	}
	next.ID = cur.ID
	if err := s.contracts.Update(ctx, next, cur.Status); err != nil {
		if errors.Is(err, repository.ErrStateChanged) {
			return model.Contract{}, apperr.Conflict("contract_state_changed", "The contract changed status; reload and try again.")
		}
		return model.Contract{}, err
	}
	return s.contracts.GetByID(ctx, cur.ID)
}

// Delete soft-deletes a Draft contract.
func (s *ContractService) Delete(ctx context.Context, publicID string) error {
	k, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	if k.Status != model.ContractDraft {
		return apperr.Conflict("contract_not_draft", "Only a draft contract can be deleted.")
	}
	if err := s.contracts.SoftDelete(ctx, k.ID); err != nil {
		if errors.Is(err, repository.ErrStateChanged) {
			return apperr.Conflict("contract_not_draft", "Only a draft contract can be deleted.")
		}
		return err
	}
	return nil
}

// Activate confirms sign-on: the contract becomes Active and the crew member On Board.
func (s *ContractService) Activate(ctx context.Context, publicID string) (model.Contract, error) {
	k, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Contract{}, err
	}
	active, err := s.contracts.ActiveForCrew(ctx, k.CrewID)
	hasActive := err == nil && active.ID != k.ID
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.Contract{}, err
	}
	if err := CheckActivation(k.Status, k.CrewStatusCode, hasActive); err != nil {
		return model.Contract{}, err
	}
	switch err := s.contracts.Activate(ctx, k.ID, k.CrewID); {
	case errors.Is(err, repository.ErrCrewHasActiveContract):
		return model.Contract{}, apperr.Conflict("crew_has_active_contract", "The crew member already holds an active contract.")
	case errors.Is(err, repository.ErrStateChanged):
		return model.Contract{}, apperr.Conflict("contract_not_draft", "Only a draft contract can be activated.")
	case err != nil:
		return model.Contract{}, err
	}
	return s.contracts.GetByID(ctx, k.ID)
}

// Complete records the actual sign-off, creates the Experience row and signs the crew member off.
func (s *ContractService) Complete(ctx context.Context, publicID string, actual time.Time) (model.Contract, error) {
	return s.finish(ctx, publicID, actual, "", false)
}

// Terminate ends a contract early with a mandatory reason, which becomes the Experience remarks.
func (s *ContractService) Terminate(ctx context.Context, publicID string, actual time.Time, reason string) (model.Contract, error) {
	return s.finish(ctx, publicID, actual, strings.TrimSpace(reason), true)
}

func (s *ContractService) finish(ctx context.Context, publicID string, actual time.Time, reason string, terminate bool) (model.Contract, error) {
	k, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Contract{}, err
	}
	if err := CheckFinish(k, actual, reason, terminate); err != nil {
		return model.Contract{}, err
	}
	status := model.ContractCompleted
	if terminate {
		status = model.ContractTerminated
	}
	if err := s.contracts.Finish(ctx, k.ID, k.CrewID, status, actual, reason, ContractExperience(k, actual, reason)); err != nil {
		if errors.Is(err, repository.ErrStateChanged) {
			return model.Contract{}, apperr.Conflict("contract_not_active", "Only an active contract can be completed or terminated.")
		}
		return model.Contract{}, err
	}
	return s.contracts.GetByID(ctx, k.ID)
}

// ContractExperience builds the sea service row recorded when a contract ends.
func ContractExperience(k model.Contract, actual time.Time, remarks string) model.Experience {
	vessel, contractID := k.VesselID, k.ID
	employer, principal := k.EmployerCompanyID, k.PrincipalID
	var parts []string
	if k.VesselGRT != nil {
		parts = append(parts, "GRT "+money.Format(*k.VesselGRT, 0))
	}
	if k.VesselHPKW != "" {
		parts = append(parts, k.VesselHPKW)
	}
	return model.Experience{
		CrewID: k.CrewID, VesselName: k.VesselName, VesselID: &vessel, RankID: k.RankID, VesselTypeID: k.VesselTypeID,
		GRTHP: strings.Join(parts, " / "), CompanyID: &employer, CompanyName: k.EmployerName, PrincipalID: &principal,
		PrincipalName: k.PrincipalName, SignOnDate: k.SignOnDate, SignOffDate: &actual, Status: "sign_off", Remarks: remarks,
		Source: "contract", ContractID: &contractID,
	}
}

// Assignment is the Current Assignment tab of a crew member.
type Assignment struct {
	Contract          model.Contract
	PlannedReliefDate time.Time
	DaysOnBoard       int
}

// CurrentAssignment returns the crew member's Active contract, or nil when not on board.
func (s *ContractService) CurrentAssignment(ctx context.Context, crewPublicID string) (*Assignment, error) {
	crew, err := s.crews.Get(ctx, crewPublicID)
	if err != nil {
		return nil, err
	}
	k, err := s.contracts.ActiveForCrew(ctx, crew.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	today, err := s.today(ctx)
	if err != nil {
		return nil, err
	}
	relief := k.PlannedSignOffDate
	next, err := s.reliefs.NextReliefDate(ctx, k.ID)
	if err != nil {
		return nil, err
	}
	if next != nil {
		relief = *next
	}
	return &Assignment{Contract: k, PlannedReliefDate: relief, DaysOnBoard: max(datetime.DaysBetween(k.SignOnDate, today)+1, 0)}, nil
}

func (s *ContractService) today(ctx context.Context) (time.Time, error) {
	loc, _, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return time.Time{}, err
	}
	return datetime.Today(s.now(), loc), nil
}

func (s *ContractService) prepare(ctx context.Context, in model.Contract) (model.Contract, error) {
	k := in
	fields := map[string]string{}
	crew, err := s.crews.Get(ctx, in.CrewPublicID)
	if err != nil {
		var ae *apperr.Error
		if !errors.As(err, &ae) {
			return k, err
		}
		fields["crew_id"] = "The selected crew member does not exist."
	} else {
		k.CrewID = crew.ID
	}
	if err := s.checkRefs(ctx, k, fields); err != nil {
		return k, err
	}
	decimals, err := s.lookups.CurrencyDecimals(ctx, k.CurrencyCode)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		fields["currency_code"] = "The selected currency does not exist or is inactive."
	case err != nil:
		return k, err
	}
	k.BasicSalary = k.BasicSalary.Round(int32(min(max(decimals, 0), 4)))
	if k.PlannedSignOffDate.IsZero() {
		k.PlannedSignOffDate = PlannedSignOff(k.SignOnDate, k.DurationMonths)
	}
	if k.PlannedSignOffDate.Before(k.SignOnDate) {
		fields["planned_sign_off_date"] = "The planned sign-off date must be on or after the sign-on date."
	}
	for i, a := range k.Allowances {
		name, basis, err := s.lookups.AllowanceType(ctx, a.AllowanceTypeID)
		if errors.Is(err, repository.ErrNotFound) {
			fields[fmt.Sprintf("allowances.%d.allowance_type_id", i)] = "The selected allowance type does not exist or is inactive."
			continue
		}
		if err != nil {
			return k, err
		}
		if basis == model.BasisPercentBasic && a.Value.GreaterThan(hundred) {
			fields[fmt.Sprintf("allowances.%d.value", i)] = "A percentage may not exceed 100."
		}
		k.Allowances[i].AllowanceType = name
		k.Allowances[i].Basis = basis
		k.Allowances[i].MonthlyAmount = MonthlyAllowance(basis, a.Value, k.BasicSalary, decimals)
	}
	if len(fields) > 0 {
		return k, apperr.Validation(fields)
	}
	return k, nil
}

func (s *ContractService) checkRefs(ctx context.Context, k model.Contract, fields map[string]string) error {
	if ok, err := s.lookups.VesselActive(ctx, k.VesselID); err != nil {
		return err
	} else if !ok {
		fields["vessel_id"] = "The selected vessel does not exist or is inactive."
	}
	if ok, err := s.lookups.RankActive(ctx, k.RankID); err != nil {
		return err
	} else if !ok {
		fields["rank_id"] = "The selected rank does not exist or is inactive."
	}
	if err := s.checkCompany(ctx, k.PrincipalID, "principal_id", "principal", fields); err != nil {
		return err
	}
	return s.checkCompany(ctx, k.EmployerCompanyID, "employer_company_id", "company", fields)
}

func (s *ContractService) checkCompany(ctx context.Context, id uuid.UUID, field, want string, fields map[string]string) error {
	t, err := s.lookups.CompanyType(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		fields[field] = "The selected company does not exist or is inactive."
		return nil
	}
	if err != nil {
		return err
	}
	if t != want && t != "both" {
		fields[field] = "The selected company must be of type " + want + " or both."
	}
	return nil
}
