package resource

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// Ref is a referenced record with its display name.
type Ref struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// CrewRef is a referenced crew member.
type CrewRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ContractAllowance is one allowance row.
type ContractAllowance struct {
	ID              uuid.UUID       `json:"id"`
	AllowanceTypeID uuid.UUID       `json:"allowance_type_id"`
	AllowanceType   string          `json:"allowance_type"`
	Basis           string          `json:"basis"`
	Value           decimal.Decimal `json:"value"`
	MonthlyAmount   decimal.Decimal `json:"monthly_amount"`
}

// Contract is a contract with its allowances and totals.
type Contract struct {
	ID                 string              `json:"id"`
	Crew               CrewRef             `json:"crew"`
	Vessel             Ref                 `json:"vessel"`
	Rank               Ref                 `json:"rank"`
	Principal          Ref                 `json:"principal"`
	Employer           Ref                 `json:"employer"`
	SignOnDate         Date                `json:"sign_on_date"`
	DurationMonths     int                 `json:"duration_months"`
	PlannedSignOffDate Date                `json:"planned_sign_off_date"`
	ActualSignOffDate  *Date               `json:"actual_sign_off_date"`
	PointOfHire        string              `json:"point_of_hire"`
	SignOnPort         string              `json:"sign_on_port"`
	SignOffPort        string              `json:"sign_off_port"`
	CurrencyCode       string              `json:"currency_code"`
	CurrencyDecimals   int                 `json:"currency_decimals"`
	BasicSalary        decimal.Decimal     `json:"basic_salary"`
	WorkingHoursWeek   *int                `json:"working_hours_week"`
	LeaveDaysMonth     *decimal.Decimal    `json:"leave_days_month"`
	ProbationMonths    *int                `json:"probation_months"`
	Status             string              `json:"status"`
	TerminationReason  string              `json:"termination_reason"`
	Notes              string              `json:"notes"`
	Allowances         []ContractAllowance `json:"allowances,omitempty"`
	TotalMonthlyPay    decimal.Decimal     `json:"total_monthly_pay"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

// NewContract maps a contract; allowances are included when loaded.
func NewContract(k model.Contract) Contract {
	out := Contract{
		ID: k.PublicID, Crew: CrewRef{ID: k.CrewPublicID, Name: k.CrewName}, Vessel: Ref{ID: k.VesselID, Name: k.VesselName},
		Rank: Ref{ID: k.RankID, Name: k.RankName}, Principal: Ref{ID: k.PrincipalID, Name: k.PrincipalName},
		Employer: Ref{ID: k.EmployerCompanyID, Name: k.EmployerName}, SignOnDate: Date(k.SignOnDate), DurationMonths: k.DurationMonths,
		PlannedSignOffDate: Date(k.PlannedSignOffDate), ActualSignOffDate: NewDate(k.ActualSignOffDate), PointOfHire: k.PointOfHire,
		SignOnPort: k.SignOnPort, SignOffPort: k.SignOffPort, CurrencyCode: k.CurrencyCode, CurrencyDecimals: k.CurrencyDecimals,
		BasicSalary: k.BasicSalary, WorkingHoursWeek: k.WorkingHoursWeek, LeaveDaysMonth: k.LeaveDaysMonth, ProbationMonths: k.ProbationMonths,
		Status: k.Status, TerminationReason: k.TerminationReason, Notes: k.Notes, TotalMonthlyPay: k.TotalMonthlyPay,
		CreatedAt: k.CreatedAt, UpdatedAt: k.UpdatedAt,
	}
	for _, a := range k.Allowances {
		out.Allowances = append(out.Allowances, ContractAllowance{ID: a.ID, AllowanceTypeID: a.AllowanceTypeID, AllowanceType: a.AllowanceType,
			Basis: a.Basis, Value: a.Value, MonthlyAmount: a.MonthlyAmount})
	}
	if k.Allowances != nil {
		out.TotalMonthlyPay = service.TotalMonthlyPay(k.BasicSalary, k.Allowances)
		if out.Allowances == nil {
			out.Allowances = []ContractAllowance{}
		}
	}
	return out
}

// NewContracts maps a page of contracts.
func NewContracts(in []model.Contract) []Contract {
	out := make([]Contract, len(in))
	for i, k := range in {
		out[i] = NewContract(k)
	}
	return out
}

// Assignment is the Current Assignment tab.
type Assignment struct {
	ContractID         string `json:"contract_id"`
	Vessel             Ref    `json:"vessel"`
	VesselType         string `json:"vessel_type"`
	Rank               Ref    `json:"rank"`
	Principal          Ref    `json:"principal"`
	SignOnDate         Date   `json:"sign_on_date"`
	PlannedSignOffDate Date   `json:"planned_sign_off_date"`
	PlannedReliefDate  Date   `json:"planned_relief_date"`
	DaysOnBoard        int    `json:"days_on_board"`
}

// NewAssignment maps the current assignment or returns nil when not on board.
func NewAssignment(a *service.Assignment) *Assignment {
	if a == nil {
		return nil
	}
	k := a.Contract
	return &Assignment{ContractID: k.PublicID, Vessel: Ref{ID: k.VesselID, Name: k.VesselName}, VesselType: k.VesselTypeName,
		Rank: Ref{ID: k.RankID, Name: k.RankName}, Principal: Ref{ID: k.PrincipalID, Name: k.PrincipalName},
		SignOnDate: Date(k.SignOnDate), PlannedSignOffDate: Date(k.PlannedSignOffDate), PlannedReliefDate: Date(a.PlannedReliefDate),
		DaysOnBoard: a.DaysOnBoard}
}

// ReliefPlan is a relief plan.
type ReliefPlan struct {
	ID                uuid.UUID `json:"id"`
	ContractID        string    `json:"contract_id"`
	ContractStatus    string    `json:"contract_status"`
	Vessel            Ref       `json:"vessel"`
	Rank              Ref       `json:"rank"`
	Relieved          CrewRef   `json:"relieved"`
	Reliever          CrewRef   `json:"reliever"`
	PlannedReliefDate Date      `json:"planned_relief_date"`
	JoiningPort       string    `json:"joining_port"`
	Status            string    `json:"status"`
	Notes             string    `json:"notes"`
	NewContractID     string    `json:"new_contract_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// NewReliefPlan maps a relief plan.
func NewReliefPlan(p model.ReliefPlan) ReliefPlan {
	return ReliefPlan{ID: p.ID, ContractID: p.ContractPublicID, ContractStatus: p.ContractStatus, Vessel: Ref{ID: p.VesselID, Name: p.VesselName},
		Rank: Ref{ID: p.RankID, Name: p.RankName}, Relieved: CrewRef{ID: p.RelievedPublicID, Name: p.RelievedName},
		Reliever: CrewRef{ID: p.RelieverPublicID, Name: p.RelieverName}, PlannedReliefDate: Date(p.PlannedReliefDate),
		JoiningPort: p.JoiningPort, Status: p.Status, Notes: p.Notes, NewContractID: p.NewContractPublicID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

// NewReliefPlans maps relief plans.
func NewReliefPlans(in []model.ReliefPlan) []ReliefPlan {
	out := make([]ReliefPlan, len(in))
	for i, p := range in {
		out[i] = NewReliefPlan(p)
	}
	return out
}

// ReliefWarning flags a mandatory item that is not valid for the whole planned contract.
type ReliefWarning struct {
	Kind        string `json:"kind"`
	TypeName    string `json:"type_name"`
	ExpiredDate *Date  `json:"expired_date"`
	Reason      string `json:"reason"`
}

// ReliefResult is a saved plan with its warnings.
type ReliefResult struct {
	ReliefPlan
	Warnings []ReliefWarning `json:"warnings"`
}

// NewReliefResult maps a saved plan with its warnings.
func NewReliefResult(r service.ReliefResult) ReliefResult {
	warnings := make([]ReliefWarning, len(r.Warnings))
	for i, w := range r.Warnings {
		warnings[i] = ReliefWarning{Kind: w.Kind, TypeName: w.TypeName, ExpiredDate: NewDate(w.ExpiredDate), Reason: w.Reason}
	}
	return ReliefResult{ReliefPlan: NewReliefPlan(r.Plan), Warnings: warnings}
}

// ReliefCandidate is one reliever suggestion.
type ReliefCandidate struct {
	ID          string       `json:"id"`
	FullName    string       `json:"full_name"`
	Nationality string       `json:"nationality"`
	Age         *int         `json:"age"`
	MobilePhone string       `json:"mobile_phone"`
	SeaService  service.Span `json:"sea_service_in_rank"`
}

// Suggestions is the reliever suggestion list for one date.
type Suggestions struct {
	Date       Date              `json:"date"`
	Candidates []ReliefCandidate `json:"candidates"`
}

// NewSuggestions maps reliever suggestions.
func NewSuggestions(list []model.ReliefCandidate, date time.Time) Suggestions {
	out := Suggestions{Date: Date(date), Candidates: make([]ReliefCandidate, len(list))}
	for i, c := range list {
		out.Candidates[i] = ReliefCandidate{ID: c.PublicID, FullName: c.FullName, Nationality: c.Nationality,
			Age: Age(c.DateOfBirth, date), MobilePhone: c.MobilePhone, SeaService: service.NewSpan(c.SeaServiceDays)}
	}
	return out
}

// TimelinePlan is a relief plan drawn on a bar.
type TimelinePlan struct {
	ID                uuid.UUID `json:"id"`
	Reliever          CrewRef   `json:"reliever"`
	PlannedReliefDate Date      `json:"planned_relief_date"`
	JoiningPort       string    `json:"joining_port"`
	Status            string    `json:"status"`
}

// TimelineBar is one active contract on a lane.
type TimelineBar struct {
	ContractID         string         `json:"contract_id"`
	Crew               CrewRef        `json:"crew"`
	RankName           string         `json:"rank_name"`
	SignOnDate         Date           `json:"sign_on_date"`
	PlannedSignOffDate Date           `json:"planned_sign_off_date"`
	State              string         `json:"state"`
	Plans              []TimelinePlan `json:"relief_plans"`
}

// TimelineLane is one vessel.
type TimelineLane struct {
	Vessel     Ref           `json:"vessel"`
	VesselType string        `json:"vessel_type"`
	Bars       []TimelineBar `json:"bars"`
}

// Timeline is the rotation timeline view.
type Timeline struct {
	From  Date           `json:"from"`
	To    Date           `json:"to"`
	Today Date           `json:"today"`
	Lanes []TimelineLane `json:"lanes"`
}

// NewTimeline maps the rotation timeline.
func NewTimeline(t service.Timeline) Timeline {
	out := Timeline{From: Date(t.From), To: Date(t.To), Today: Date(t.Today), Lanes: make([]TimelineLane, len(t.Lanes))}
	for i, l := range t.Lanes {
		lane := TimelineLane{Vessel: Ref{ID: l.Vessel.ID, Name: l.Vessel.Name}, VesselType: l.Vessel.Type, Bars: make([]TimelineBar, len(l.Bars))}
		for j, b := range l.Bars {
			bar := TimelineBar{ContractID: b.ContractPublicID, Crew: CrewRef{ID: b.CrewPublicID, Name: b.CrewName}, RankName: b.RankName,
				SignOnDate: Date(b.SignOnDate), PlannedSignOffDate: Date(b.PlannedSignOffDate), State: b.State, Plans: make([]TimelinePlan, len(b.Plans))}
			for k, p := range b.Plans {
				bar.Plans[k] = TimelinePlan{ID: p.ID, Reliever: CrewRef{ID: p.RelieverPublicID, Name: p.RelieverName},
					PlannedReliefDate: Date(p.PlannedReliefDate), JoiningPort: p.JoiningPort, Status: p.Status}
			}
			lane.Bars[j] = bar
		}
		out.Lanes[i] = lane
	}
	return out
}
