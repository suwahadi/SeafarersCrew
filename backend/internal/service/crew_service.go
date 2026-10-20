package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// CrewStore persists crew records.
type CrewStore interface {
	Create(ctx context.Context, c model.Crew, publicID, statusCode string) (uuid.UUID, error)
	Update(ctx context.Context, id uuid.UUID, c model.Crew) error
	GetByPublicID(ctx context.Context, publicID string) (model.Crew, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Crew, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	SetStatus(ctx context.Context, id uuid.UUID, code, reason string) error
	HasActiveContract(ctx context.Context, crewID uuid.UUID) (bool, error)
	ActiveContractRank(ctx context.Context, crewID uuid.UUID) (*uuid.UUID, error)
	Requirements(ctx context.Context, rankID uuid.UUID) ([]repository.RequirementRow, error)
	HeldItems(ctx context.Context, crewID uuid.UUID) ([]repository.HeldRow, error)
	ListCrews(ctx context.Context, f model.CrewFilter, limit, offset int) (repository.Page[model.CrewRow], []model.CrewGroup, error)
	StreamCrews(ctx context.Context, f model.CrewFilter, batch int, fn func(model.CrewRow) error) error
}

// SettingsReader reads decrypted settings groups.
type SettingsReader interface {
	Values(ctx context.Context, group string) (map[string]any, error)
}

// CrewService implements crew records, status changes, compliance and sea service.
type CrewService struct {
	crews    CrewStore
	records  CrewRecordStore
	settings SettingsReader
	now      func() time.Time
	newID    func() (string, error)
}

// NewCrewService builds a CrewService.
func NewCrewService(crews CrewStore, records CrewRecordStore, settings SettingsReader, now func() time.Time) *CrewService {
	return &CrewService{crews: crews, records: records, settings: settings, now: now, newID: publicid.New}
}

// Create stores a new crew member with status Available and a unique 9-character Crew ID.
func (s *CrewService) Create(ctx context.Context, c model.Crew) (model.Crew, error) {
	for range publicid.MaxAttempts {
		pid, err := s.newID()
		if err != nil {
			return model.Crew{}, err
		}
		id, err := s.crews.Create(ctx, c, pid, StatusAvailable)
		if postgres.IsUniqueViolation(err, "crews_public_id_key") {
			continue
		}
		if err != nil {
			return model.Crew{}, crewWriteError(err)
		}
		return s.crews.GetByID(ctx, id)
	}
	return model.Crew{}, fmt.Errorf("create crew: no unique public id after %d attempts", publicid.MaxAttempts)
}

// CrewList is one datagrid page.
type CrewList struct {
	Items      []model.CrewRow
	Total      int
	Groups     []model.CrewGroup
	Today      time.Time
	Thresholds Thresholds
}

// List returns one datagrid page filtered, sorted and optionally grouped (PRD 6.9).
func (s *CrewService) List(ctx context.Context, f model.CrewFilter, page, perPage int) (CrewList, error) {
	loc, thresholds, err := s.displaySettings(ctx)
	if err != nil {
		return CrewList{}, err
	}
	f.Today = datetime.Today(s.now(), loc)
	res, groups, err := s.crews.ListCrews(ctx, f, perPage, (page-1)*perPage)
	if err != nil {
		return CrewList{}, err
	}
	return CrewList{Items: res.Items, Total: res.Total, Groups: groups, Today: f.Today, Thresholds: thresholds}, nil
}

// Get returns a crew member by Crew ID.
func (s *CrewService) Get(ctx context.Context, publicID string) (model.Crew, error) {
	if !publicid.Valid(publicID) {
		return model.Crew{}, apperr.NotFound("Crew")
	}
	c, err := s.crews.GetByPublicID(ctx, publicID)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Crew{}, apperr.NotFound("Crew")
	}
	return c, err
}

// Update replaces the profile fields of a crew member.
func (s *CrewService) Update(ctx context.Context, publicID string, c model.Crew) (model.Crew, error) {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Crew{}, err
	}
	if err := s.crews.Update(ctx, cur.ID, c); err != nil {
		return model.Crew{}, crewWriteError(err)
	}
	return s.crews.GetByID(ctx, cur.ID)
}

// Delete soft-deletes a crew member that holds no Active contract.
func (s *CrewService) Delete(ctx context.Context, publicID string) error {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	active, err := s.crews.HasActiveContract(ctx, c.ID)
	if err != nil {
		return err
	}
	if active {
		return apperr.Conflict("crew_on_board", "The crew member has an active contract and cannot be deleted.")
	}
	return s.crews.SoftDelete(ctx, c.ID)
}

// ChangeStatus applies a manual status action (PRD 11.2).
func (s *CrewService) ChangeStatus(ctx context.Context, publicID, action, reason string) (model.Crew, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Crew{}, err
	}
	active, err := s.crews.HasActiveContract(ctx, c.ID)
	if err != nil {
		return model.Crew{}, err
	}
	change, err := ManualStatusChange(action, reason, active)
	if err != nil {
		return model.Crew{}, err
	}
	if err := s.crews.SetStatus(ctx, c.ID, change.Code, change.Reason); err != nil {
		return model.Crew{}, err
	}
	return s.crews.GetByID(ctx, c.ID)
}

// CrewCompliance is the compliance checklist with the rank it was evaluated for.
type CrewCompliance struct {
	RankID *uuid.UUID `json:"rank_id"`
	ComplianceResult
}

// Compliance evaluates the crew member for rankID (default: active contract rank, else applied rank) on checkDate (default: today).
func (s *CrewService) Compliance(ctx context.Context, publicID string, rankID *uuid.UUID, checkDate *time.Time) (CrewCompliance, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return CrewCompliance{}, err
	}
	return s.complianceFor(ctx, c, rankID, checkDate)
}

func (s *CrewService) complianceFor(ctx context.Context, c model.Crew, rankID *uuid.UUID, checkDate *time.Time) (CrewCompliance, error) {
	loc, thresholds, err := s.displaySettings(ctx)
	if err != nil {
		return CrewCompliance{}, err
	}
	date := datetime.Today(s.now(), loc)
	if checkDate != nil {
		date = *checkDate
	}
	if rankID == nil {
		if rankID, err = s.crews.ActiveContractRank(ctx, c.ID); err != nil {
			return CrewCompliance{}, err
		}
	}
	if rankID == nil {
		rankID = c.AppliedRankID
	}
	if rankID == nil {
		return CrewCompliance{ComplianceResult: ComplianceResult{CheckDate: date, Compliant: false, Items: []ComplianceItem{}}}, nil
	}
	reqRows, err := s.crews.Requirements(ctx, *rankID)
	if err != nil {
		return CrewCompliance{}, err
	}
	heldRows, err := s.crews.HeldItems(ctx, c.ID)
	if err != nil {
		return CrewCompliance{}, err
	}
	reqs := make([]Requirement, len(reqRows))
	for i, r := range reqRows {
		reqs[i] = Requirement{Kind: r.Kind, TypeID: r.TypeID, TypeName: r.TypeName, HasExpiry: r.HasExpiry}
	}
	held := make([]HeldItem, len(heldRows))
	for i, h := range heldRows {
		held[i] = HeldItem{Kind: h.Kind, TypeID: h.TypeID, Number: h.Number, ExpiredDate: h.ExpiredDate}
	}
	return CrewCompliance{RankID: rankID, ComplianceResult: EvaluateCompliance(reqs, held, date, thresholds)}, nil
}

// SeaServiceRow is one experience with its inclusive day count.
type SeaServiceRow struct {
	model.Experience
	Days     int  `json:"days"`
	Duration Span `json:"duration"`
}

// RankSeaService is the merged total of one rank.
type RankSeaService struct {
	RankID   uuid.UUID `json:"rank_id"`
	RankName string    `json:"rank_name"`
	Total    Span      `json:"total"`
}

// CrewSeaService is the Experiences tab with totals.
type CrewSeaService struct {
	Rows   []SeaServiceRow  `json:"rows"`
	ByRank []RankSeaService `json:"by_rank"`
	Total  Span             `json:"total"`
}

// SeaService returns experiences with per-row durations and merged totals per rank (PRD 6.8, 11.5).
func (s *CrewService) SeaService(ctx context.Context, publicID string) (CrewSeaService, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return CrewSeaService{}, err
	}
	exps, err := s.records.ListExperiences(ctx, c.ID)
	if err != nil {
		return CrewSeaService{}, err
	}
	loc, _, err := s.displaySettings(ctx)
	if err != nil {
		return CrewSeaService{}, err
	}
	today := datetime.Today(s.now(), loc)
	out := CrewSeaService{Rows: make([]SeaServiceRow, 0, len(exps))}
	periods := make([]Period, 0, len(exps))
	names := map[uuid.UUID]string{}
	order := []uuid.UUID{}
	for _, x := range exps {
		p := Period{RankID: x.RankID, SignOn: x.SignOnDate, SignOff: x.SignOffDate}
		days := RowDays(p, today)
		out.Rows = append(out.Rows, SeaServiceRow{Experience: x, Days: days, Duration: NewSpan(days)})
		periods = append(periods, p)
		if _, seen := names[x.RankID]; !seen {
			order = append(order, x.RankID)
		}
		names[x.RankID] = x.RankName
	}
	summary := SummarizeSeaService(periods, today)
	out.Total = summary.Total
	for _, id := range order {
		out.ByRank = append(out.ByRank, RankSeaService{RankID: id, RankName: names[id], Total: summary.ByRank[id]})
	}
	if out.ByRank == nil {
		out.ByRank = []RankSeaService{}
	}
	return out, nil
}

// Today returns the current date in the configured timezone.
func (s *CrewService) Today(ctx context.Context) (time.Time, error) {
	loc, _, err := s.displaySettings(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return datetime.Today(s.now(), loc), nil
}

func (s *CrewService) displaySettings(ctx context.Context) (*time.Location, Thresholds, error) {
	return LoadDisplaySettings(ctx, s.settings)
}

// LoadDisplaySettings reads the timezone and expiry thresholds from Settings.
func LoadDisplaySettings(ctx context.Context, settings SettingsReader) (*time.Location, Thresholds, error) {
	general, err := settings.Values(ctx, "general")
	if err != nil {
		return nil, Thresholds{}, err
	}
	tz, _ := general["timezone"].(string)
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		loc, err = time.LoadLocation(datetime.DefaultTimezone)
		if err != nil {
			return nil, Thresholds{}, fmt.Errorf("load default timezone: %w", err)
		}
	}
	alerts, err := settings.Values(ctx, "alerts")
	if err != nil {
		return nil, Thresholds{}, err
	}
	t := DefaultThresholds
	for i, key := range []string{"expiry_threshold_1", "expiry_threshold_2", "expiry_threshold_3"} {
		if n, ok := alerts[key].(int); ok && n > 0 {
			t[i] = n
		}
	}
	return loc, t, nil
}

var crewFieldByColumn = map[string]string{
	"applied_rank_id": "applied_rank_id",
	"photo_file_id":   "photo_file_id",
	"cv_file_id":      "cv_file_id",
}

func crewWriteError(err error) error {
	if col, ok := postgres.ForeignKeyColumn(err, "crews"); ok {
		if field, known := crewFieldByColumn[col]; known {
			return apperr.Validation(map[string]string{field: "The selected value does not exist."})
		}
	}
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound("Crew")
	}
	return err
}
