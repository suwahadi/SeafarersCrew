package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// Rotation limits.
const (
	MaxSuggestions      = 50
	DefaultTimelineDays = 180
	MaxTimelineDays     = 731
)

// ReliefStore persists relief plans and answers rotation queries.
type ReliefStore interface {
	Get(ctx context.Context, id uuid.UUID) (model.ReliefPlan, error)
	List(ctx context.Context, f model.ReliefPlanFilter, limit, offset int) (repository.Page[model.ReliefPlan], error)
	Create(ctx context.Context, p model.ReliefPlan) (uuid.UUID, error)
	Update(ctx context.Context, p model.ReliefPlan) error
	Delete(ctx context.Context, id uuid.UUID) error
	OpenWindows(ctx context.Context, relieverID, excludePlanID uuid.UUID) ([]model.ReliefWindow, error)
	Suggestions(ctx context.Context, rankID, excludeCrewID uuid.UUID, date, end time.Time, limit int) ([]model.ReliefCandidate, error)
	TimelineVessels(ctx context.Context, vesselIDs []uuid.UUID) ([]model.TimelineVessel, error)
	TimelineBars(ctx context.Context, from, to time.Time, vesselIDs []uuid.UUID) ([]model.TimelineBar, error)
	OpenPlansForContracts(ctx context.Context, contractIDs []uuid.UUID) ([]model.ReliefPlan, error)
	Complete(ctx context.Context, planID uuid.UUID, newID func() (string, error), attempts int) (uuid.UUID, error)
}

// ContractReader loads contracts.
type ContractReader interface {
	Get(ctx context.Context, publicID string) (model.Contract, error)
}

// ComplianceChecker evaluates a crew member for a rank on a date.
type ComplianceChecker interface {
	Compliance(ctx context.Context, publicID string, rankID *uuid.UUID, checkDate *time.Time) (CrewCompliance, error)
}

// ContractByID loads a contract by internal ID.
type ContractByID interface {
	GetByID(ctx context.Context, id uuid.UUID) (model.Contract, error)
}

// RotationService implements relief planning (PRD 6.5).
type RotationService struct {
	plans      ReliefStore
	contracts  ContractReader
	byID       ContractByID
	crews      CrewFinder
	compliance ComplianceChecker
	settings   SettingsReader
	now        func() time.Time
	newID      func() (string, error)
}

// NewRotationService builds a RotationService.
func NewRotationService(plans ReliefStore, contracts ContractReader, byID ContractByID, crews CrewFinder, compliance ComplianceChecker, settings SettingsReader, now func() time.Time) *RotationService {
	return &RotationService{plans: plans, contracts: contracts, byID: byID, crews: crews, compliance: compliance, settings: settings, now: now, newID: publicid.New}
}

// Relief warning reasons.
const (
	WarnExpiresBeforeEnd = "expires_before_contract_end"
	WarnExpiredOnRelief  = "expired_on_relief_date"
	WarnMissing          = "missing"
)

// ReliefWarning flags a mandatory item that is not valid for the whole planned contract.
type ReliefWarning struct {
	Kind        string     `json:"kind"`
	TypeName    string     `json:"type_name"`
	ExpiredDate *time.Time `json:"expired_date"`
	Reason      string     `json:"reason"`
}

// ReliefInput is a relief plan as submitted by the admin.
type ReliefInput struct {
	ContractPublicID  string
	RelieverPublicID  string
	PlannedReliefDate time.Time
	JoiningPort       string
	Status            string
	Notes             string
}

// ReliefResult is a saved plan with its compliance warnings.
type ReliefResult struct {
	Plan     model.ReliefPlan
	Warnings []ReliefWarning
}

// Get returns a relief plan.
func (s *RotationService) Get(ctx context.Context, id uuid.UUID) (model.ReliefPlan, error) {
	p, err := s.plans.Get(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return p, apperr.NotFound("Relief plan")
	}
	return p, err
}

// List returns one page of relief plans.
func (s *RotationService) List(ctx context.Context, f model.ReliefPlanFilter, page, perPage int) (repository.Page[model.ReliefPlan], error) {
	return s.plans.List(ctx, f, perPage, (page-1)*perPage)
}

// Create plans a reliever for an Active contract.
func (s *RotationService) Create(ctx context.Context, in ReliefInput) (ReliefResult, error) {
	if in.Status == "" {
		in.Status = model.ReliefPlanned
	}
	if in.Status != model.ReliefPlanned && in.Status != model.ReliefConfirmed {
		return ReliefResult{}, apperr.Validation(map[string]string{"status": "A new relief plan must be planned or confirmed."})
	}
	k, err := s.contracts.Get(ctx, in.ContractPublicID)
	if err != nil {
		return ReliefResult{}, err
	}
	plan, warnings, err := s.validate(ctx, k, in, uuid.Nil)
	if err != nil {
		return ReliefResult{}, err
	}
	id, err := s.plans.Create(ctx, plan)
	if err != nil {
		return ReliefResult{}, err
	}
	saved, err := s.plans.Get(ctx, id)
	return ReliefResult{Plan: saved, Warnings: warnings}, err
}

// Update changes an open relief plan; cancelling skips the reliever checks.
func (s *RotationService) Update(ctx context.Context, id uuid.UUID, in ReliefInput) (ReliefResult, error) {
	cur, err := s.Get(ctx, id)
	if err != nil {
		return ReliefResult{}, err
	}
	if cur.Status == model.ReliefCompleted || cur.Status == model.ReliefCancelled {
		return ReliefResult{}, apperr.Conflict("relief_plan_closed", "A completed or cancelled relief plan cannot be changed.")
	}
	switch in.Status {
	case "":
		in.Status = cur.Status
	case model.ReliefPlanned, model.ReliefConfirmed, model.ReliefCancelled:
	default:
		return ReliefResult{}, apperr.Validation(map[string]string{"status": "The status must be planned, confirmed or cancelled; use complete to finish a relief."})
	}
	plan := cur
	var warnings []ReliefWarning
	if in.Status == model.ReliefCancelled {
		plan.Status, plan.Notes = in.Status, in.Notes
	} else {
		k, err := s.contracts.Get(ctx, cur.ContractPublicID)
		if err != nil {
			return ReliefResult{}, err
		}
		if plan, warnings, err = s.validate(ctx, k, in, cur.ID); err != nil {
			return ReliefResult{}, err
		}
		plan.ID = cur.ID
	}
	if err := s.plans.Update(ctx, plan); err != nil {
		if errors.Is(err, repository.ErrStateChanged) {
			return ReliefResult{}, apperr.Conflict("relief_plan_closed", "A completed or cancelled relief plan cannot be changed.")
		}
		return ReliefResult{}, err
	}
	saved, err := s.plans.Get(ctx, id)
	return ReliefResult{Plan: saved, Warnings: warnings}, err
}

// Delete removes a relief plan that is not completed.
func (s *RotationService) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.plans.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrStateChanged) {
			return apperr.Conflict("relief_plan_closed", "A completed relief plan cannot be deleted.")
		}
		return err
	}
	return nil
}

func (s *RotationService) validate(ctx context.Context, k model.Contract, in ReliefInput, planID uuid.UUID) (model.ReliefPlan, []ReliefWarning, error) {
	if k.Status != model.ContractActive {
		return model.ReliefPlan{}, nil, apperr.Conflict("contract_not_active", "Reliefs can only be planned for an active contract.")
	}
	reliever, err := s.crews.Get(ctx, in.RelieverPublicID)
	if err != nil {
		var ae *apperr.Error
		if errors.As(err, &ae) {
			return model.ReliefPlan{}, nil, apperr.Validation(map[string]string{"reliever_crew_id": "The selected crew member does not exist."})
		}
		return model.ReliefPlan{}, nil, err
	}
	fields := map[string]string{}
	if reliever.ID == k.CrewID {
		fields["reliever_crew_id"] = "The reliever must differ from the crew member being relieved."
	}
	if in.PlannedReliefDate.Before(k.SignOnDate) {
		fields["planned_relief_date"] = "The relief date must be on or after the contract sign-on date."
	}
	if len(fields) > 0 {
		return model.ReliefPlan{}, nil, apperr.Validation(fields)
	}
	if !CanBeAssigned(reliever.StatusCode) {
		return model.ReliefPlan{}, nil, apperr.Conflict("reliever_not_available", "Only an Available crew member can be planned as a reliever.")
	}
	window := model.ReliefWindow{PlanID: planID, ReliefDate: in.PlannedReliefDate, DurationMonths: k.DurationMonths}
	others, err := s.plans.OpenWindows(ctx, reliever.ID, planID)
	if err != nil {
		return model.ReliefPlan{}, nil, err
	}
	for _, o := range others {
		if ReliefWindowsOverlap(window, o) {
			return model.ReliefPlan{}, nil, apperr.Conflict("reliever_overlap",
				"The reliever is already planned on another relief from "+datetime.FormatDate(o.ReliefDate, datetime.DefaultDisplayFormat)+
					" to "+datetime.FormatDate(ReliefWindowEnd(o), datetime.DefaultDisplayFormat)+".")
		}
	}
	warnings, err := s.warnings(ctx, reliever.PublicID, k.RankID, window)
	if err != nil {
		return model.ReliefPlan{}, nil, err
	}
	plan := model.ReliefPlan{ContractID: k.ID, RelieverCrewID: reliever.ID, PlannedReliefDate: in.PlannedReliefDate,
		JoiningPort: in.JoiningPort, Status: in.Status, Notes: in.Notes}
	return plan, warnings, nil
}

func (s *RotationService) warnings(ctx context.Context, relieverPublicID string, rankID uuid.UUID, w model.ReliefWindow) ([]ReliefWarning, error) {
	start, end := w.ReliefDate, ReliefWindowEnd(w)
	atStart, err := s.compliance.Compliance(ctx, relieverPublicID, &rankID, &start)
	if err != nil {
		return nil, err
	}
	atEnd, err := s.compliance.Compliance(ctx, relieverPublicID, &rankID, &end)
	if err != nil {
		return nil, err
	}
	return ReliefWarnings(atStart.ComplianceResult, atEnd.ComplianceResult), nil
}

// ReliefWarnings lists mandatory items that are missing, expired on the relief date or expire before the contract end.
func ReliefWarnings(atStart, atEnd ComplianceResult) []ReliefWarning {
	startStatus := map[uuid.UUID]string{}
	for _, it := range atStart.Items {
		startStatus[it.TypeID] = it.Status
	}
	out := []ReliefWarning{}
	for _, it := range atEnd.Items {
		if it.Status != ItemExpired && it.Status != ItemMissing {
			continue
		}
		reason := WarnExpiresBeforeEnd
		switch startStatus[it.TypeID] {
		case ItemMissing:
			reason = WarnMissing
		case ItemExpired:
			reason = WarnExpiredOnRelief
		}
		out = append(out, ReliefWarning{Kind: it.Kind, TypeName: it.TypeName, ExpiredDate: it.ExpiredDate, Reason: reason})
	}
	return out
}

// Suggestions lists relievers for a contract on date (default: its planned sign-off).
func (s *RotationService) Suggestions(ctx context.Context, contractPublicID string, date *time.Time) ([]model.ReliefCandidate, time.Time, error) {
	k, err := s.contracts.Get(ctx, contractPublicID)
	if err != nil {
		return nil, time.Time{}, err
	}
	if k.Status != model.ContractActive {
		return nil, time.Time{}, apperr.Conflict("contract_not_active", "Reliefs can only be planned for an active contract.")
	}
	d := k.PlannedSignOffDate
	if date != nil {
		d = *date
	}
	end := ReliefWindowEnd(model.ReliefWindow{ReliefDate: d, DurationMonths: k.DurationMonths})
	list, err := s.plans.Suggestions(ctx, k.RankID, k.CrewID, d, end, MaxSuggestions)
	return list, d, err
}

// Complete finishes a relief and returns the reliever's new Draft contract.
func (s *RotationService) Complete(ctx context.Context, id uuid.UUID) (model.Contract, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return model.Contract{}, err
	}
	contractID, err := s.plans.Complete(ctx, id, s.newID, publicid.MaxAttempts)
	if errors.Is(err, repository.ErrPlanCancelled) {
		return model.Contract{}, apperr.Conflict("relief_plan_closed", "A cancelled relief plan cannot be completed.")
	}
	if err != nil {
		return model.Contract{}, err
	}
	return s.byID.GetByID(ctx, contractID)
}

// TimelinePlan is a relief plan shown on a bar.
type TimelinePlan struct {
	ID                uuid.UUID `json:"id"`
	RelieverPublicID  string    `json:"reliever_id"`
	RelieverName      string    `json:"reliever_name"`
	PlannedReliefDate time.Time `json:"planned_relief_date"`
	JoiningPort       string    `json:"joining_port"`
	Status            string    `json:"status"`
}

// TimelineBarView is a bar with its state and relief plans.
type TimelineBarView struct {
	model.TimelineBar
	State string
	Plans []TimelinePlan
}

// TimelineLane is one vessel with its bars.
type TimelineLane struct {
	Vessel model.TimelineVessel
	Bars   []TimelineBarView
}

// Timeline is the vessel timeline view.
type Timeline struct {
	From  time.Time
	To    time.Time
	Today time.Time
	Lanes []TimelineLane
}

// Timeline returns one lane per vessel with a bar per active contract between from and to.
func (s *RotationService) Timeline(ctx context.Context, from, to *time.Time, vesselIDs []uuid.UUID) (Timeline, error) {
	loc, _, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return Timeline{}, err
	}
	t := Timeline{Today: datetime.Today(s.now(), loc)}
	t.From, t.To = t.Today.AddDate(0, 0, -30), t.Today.AddDate(0, 0, DefaultTimelineDays)
	if from != nil {
		t.From = *from
	}
	if to != nil {
		t.To = *to
	}
	if t.To.Before(t.From) || datetime.DaysBetween(t.From, t.To) > MaxTimelineDays {
		return Timeline{}, apperr.Validation(map[string]string{"to": "The range must end after it starts and span at most two years."})
	}
	vessels, err := s.plans.TimelineVessels(ctx, vesselIDs)
	if err != nil {
		return Timeline{}, err
	}
	bars, err := s.plans.TimelineBars(ctx, t.From, t.To, vesselIDs)
	if err != nil {
		return Timeline{}, err
	}
	ids := make([]uuid.UUID, len(bars))
	for i, b := range bars {
		ids[i] = b.ContractID
	}
	plans, err := s.plans.OpenPlansForContracts(ctx, ids)
	if err != nil {
		return Timeline{}, err
	}
	t.Lanes = BuildLanes(vessels, bars, plans, t.Today)
	return t, nil
}

// BuildLanes groups bars by vessel, keeping vessel order and adding lanes for vessels outside the list.
func BuildLanes(vessels []model.TimelineVessel, bars []model.TimelineBar, plans []model.ReliefPlan, today time.Time) []TimelineLane {
	byContract := map[uuid.UUID][]TimelinePlan{}
	for _, p := range plans {
		byContract[p.ContractID] = append(byContract[p.ContractID], TimelinePlan{ID: p.ID, RelieverPublicID: p.RelieverPublicID,
			RelieverName: p.RelieverName, PlannedReliefDate: p.PlannedReliefDate, JoiningPort: p.JoiningPort, Status: p.Status})
	}
	lanes := make([]TimelineLane, 0, len(vessels))
	index := map[uuid.UUID]int{}
	for _, v := range vessels {
		index[v.ID] = len(lanes)
		lanes = append(lanes, TimelineLane{Vessel: v, Bars: []TimelineBarView{}})
	}
	for _, b := range bars {
		i, ok := index[b.VesselID]
		if !ok {
			i = len(lanes)
			index[b.VesselID] = i
			lanes = append(lanes, TimelineLane{Vessel: model.TimelineVessel{ID: b.VesselID, Name: b.VesselName}, Bars: []TimelineBarView{}})
		}
		ps := byContract[b.ContractID]
		if ps == nil {
			ps = []TimelinePlan{}
		}
		lanes[i].Bars = append(lanes[i].Bars, TimelineBarView{TimelineBar: b, State: BarState(b.PlannedSignOffDate, today), Plans: ps})
	}
	return lanes
}
