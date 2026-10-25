package request

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

var maxMoney = decimal.RequireFromString("9999999999999999.99")

// ContractAllowance is one allowance row of a contract body.
type ContractAllowance struct {
	AllowanceTypeID string           `json:"allowance_type_id"`
	Value           *decimal.Decimal `json:"value"`
}

// Contract is the body of POST /contracts and PUT /contracts/{id}.
type Contract struct {
	CrewID             string              `json:"crew_id"`
	VesselID           string              `json:"vessel_id"`
	RankID             string              `json:"rank_id"`
	PrincipalID        string              `json:"principal_id"`
	EmployerCompanyID  string              `json:"employer_company_id"`
	SignOnDate         string              `json:"sign_on_date"`
	DurationMonths     *int                `json:"duration_months"`
	PlannedSignOffDate string              `json:"planned_sign_off_date"`
	PointOfHire        string              `json:"point_of_hire"`
	SignOnPort         string              `json:"sign_on_port"`
	SignOffPort        string              `json:"sign_off_port"`
	CurrencyCode       string              `json:"currency_code"`
	BasicSalary        *decimal.Decimal    `json:"basic_salary"`
	WorkingHoursWeek   *int                `json:"working_hours_week"`
	LeaveDaysMonth     *decimal.Decimal    `json:"leave_days_month"`
	ProbationMonths    *int                `json:"probation_months"`
	Notes              string              `json:"notes"`
	Allowances         []ContractAllowance `json:"allowances"`

	parsed model.Contract
}

// Validate implements Validator.
func (r *Contract) Validate() map[string]string {
	Trim(&r.CrewID, &r.VesselID, &r.RankID, &r.PrincipalID, &r.EmployerCompanyID, &r.SignOnDate, &r.PlannedSignOffDate,
		&r.PointOfHire, &r.SignOnPort, &r.SignOffPort, &r.CurrencyCode, &r.Notes)
	r.CurrencyCode = strings.ToUpper(r.CurrencyCode)
	e := Errors{}
	k := model.Contract{CrewPublicID: strings.ToUpper(r.CrewID), PointOfHire: r.PointOfHire, SignOnPort: r.SignOnPort,
		SignOffPort: r.SignOffPort, CurrencyCode: r.CurrencyCode, Notes: r.Notes, WorkingHoursWeek: r.WorkingHoursWeek,
		ProbationMonths: r.ProbationMonths, LeaveDaysMonth: r.LeaveDaysMonth}
	e.Required("crew_id", r.CrewID, 9)
	k.VesselID = e.RequiredUUID("vessel_id", r.VesselID)
	k.RankID = e.RequiredUUID("rank_id", r.RankID)
	k.PrincipalID = e.RequiredUUID("principal_id", r.PrincipalID)
	k.EmployerCompanyID = e.RequiredUUID("employer_company_id", r.EmployerCompanyID)
	k.SignOnDate = e.RequiredDate("sign_on_date", r.SignOnDate)
	if p := e.Date("planned_sign_off_date", r.PlannedSignOffDate); p != nil {
		k.PlannedSignOffDate = *p
	}
	if r.DurationMonths == nil {
		e["duration_months"] = "This field is required."
	} else {
		e.Range("duration_months", r.DurationMonths, 1, 60)
		k.DurationMonths = *r.DurationMonths
	}
	e.Required("currency_code", r.CurrencyCode, 3)
	e.MaxLen("point_of_hire", r.PointOfHire, 100)
	e.MaxLen("sign_on_port", r.SignOnPort, 100)
	e.MaxLen("sign_off_port", r.SignOffPort, 100)
	e.MaxLen("notes", r.Notes, 2000)
	e.Range("working_hours_week", r.WorkingHoursWeek, 0, 168)
	e.Range("probation_months", r.ProbationMonths, 0, 24)
	if r.LeaveDaysMonth != nil && (r.LeaveDaysMonth.IsNegative() || r.LeaveDaysMonth.GreaterThan(decimal.NewFromInt(31))) {
		e["leave_days_month"] = "This must be between 0 and 31."
	}
	switch {
	case r.BasicSalary == nil:
		e["basic_salary"] = "This field is required."
	case r.BasicSalary.IsNegative() || r.BasicSalary.GreaterThan(maxMoney):
		e["basic_salary"] = "This must be a positive amount."
	default:
		k.BasicSalary = *r.BasicSalary
	}
	if len(r.Allowances) > 30 {
		e["allowances"] = "A contract may have at most 30 allowances."
	}
	for i, a := range r.Allowances {
		row := model.ContractAllowance{AllowanceTypeID: e.RequiredUUID(fmt.Sprintf("allowances.%d.allowance_type_id", i), strings.TrimSpace(a.AllowanceTypeID))}
		switch {
		case a.Value == nil:
			e[fmt.Sprintf("allowances.%d.value", i)] = "This field is required."
		case a.Value.IsNegative() || a.Value.GreaterThan(maxMoney):
			e[fmt.Sprintf("allowances.%d.value", i)] = "This must be a positive amount."
		default:
			row.Value = *a.Value
		}
		k.Allowances = append(k.Allowances, row)
	}
	r.parsed = k
	return e
}

// Model returns the validated contract.
func (r *Contract) Model() model.Contract { return r.parsed }

// ContractFinish is the body of POST /contracts/{id}/complete and /terminate.
type ContractFinish struct {
	ActualSignOffDate string `json:"actual_sign_off_date"`
	Reason            string `json:"reason"`

	date time.Time
}

// Validate implements Validator.
func (r *ContractFinish) Validate() map[string]string {
	Trim(&r.ActualSignOffDate, &r.Reason)
	e := Errors{}
	r.date = e.RequiredDate("actual_sign_off_date", r.ActualSignOffDate)
	e.MaxLen("reason", r.Reason, 1000)
	return e
}

// Date returns the parsed sign-off date.
func (r *ContractFinish) Date() time.Time { return r.date }

// ParseContractFilter reads contract list filters; within_days limits to planned sign-off within N days.
func ParseContractFilter(q url.Values) (model.ContractFilter, *int, Errors) {
	e := Errors{}
	p := ParseList(q)
	f := model.ContractFilter{Q: p.Q, Sort: p.Sort, Desc: p.Desc, Statuses: CSV(q, "status"), CrewPublicID: strings.TrimSpace(q.Get("crew_id"))}
	for _, st := range f.Statuses {
		e.OneOf("status", st, model.ContractDraft, model.ContractActive, model.ContractCompleted, model.ContractTerminated)
	}
	f.VesselIDs = uuids(e, q, "vessel_id")
	f.PrincipalIDs = uuids(e, q, "principal_id")
	f.RankIDs = uuids(e, q, "rank_id")
	f.SignOnFrom = e.Date("sign_on_from", q.Get("sign_on_from"))
	f.SignOnTo = e.Date("sign_on_to", q.Get("sign_on_to"))
	f.SignOffBefore = e.Date("sign_off_before", q.Get("sign_off_before"))
	within := Int(q, "sign_off_within_days")
	e.Range("sign_off_within_days", within, 0, 3650)
	return f, within, e
}

// ReliefPlan is the body of POST /relief-plans and PUT /relief-plans/{id}.
type ReliefPlan struct {
	ContractID        string `json:"contract_id"`
	RelieverCrewID    string `json:"reliever_crew_id"`
	PlannedReliefDate string `json:"planned_relief_date"`
	JoiningPort       string `json:"joining_port"`
	Status            string `json:"status"`
	Notes             string `json:"notes"`

	date time.Time
}

// Validate implements Validator.
func (r *ReliefPlan) Validate() map[string]string {
	Trim(&r.ContractID, &r.RelieverCrewID, &r.PlannedReliefDate, &r.JoiningPort, &r.Status, &r.Notes)
	r.ContractID, r.RelieverCrewID = strings.ToUpper(r.ContractID), strings.ToUpper(r.RelieverCrewID)
	e := Errors{}
	if r.Status != model.ReliefCancelled {
		e.Required("reliever_crew_id", r.RelieverCrewID, 9)
		r.date = e.RequiredDate("planned_relief_date", r.PlannedReliefDate)
	}
	e.MaxLen("joining_port", r.JoiningPort, 100)
	e.MaxLen("notes", r.Notes, 2000)
	e.OneOf("status", r.Status, model.ReliefPlanned, model.ReliefConfirmed, model.ReliefCancelled)
	return e
}

// Date returns the parsed relief date.
func (r *ReliefPlan) Date() time.Time { return r.date }

// ParseReliefFilter reads relief plan list filters.
func ParseReliefFilter(q url.Values) (model.ReliefPlanFilter, Errors) {
	e := Errors{}
	f := model.ReliefPlanFilter{Statuses: CSV(q, "status"), ContractPublicID: strings.TrimSpace(q.Get("contract_id")),
		RelieverPublicID: strings.TrimSpace(q.Get("reliever_id"))}
	for _, st := range f.Statuses {
		e.OneOf("status", st, model.ReliefPlanned, model.ReliefConfirmed, model.ReliefCompleted, model.ReliefCancelled)
	}
	f.VesselIDs = uuids(e, q, "vessel_id")
	f.DateFrom = e.Date("from", q.Get("from"))
	f.DateTo = e.Date("to", q.Get("to"))
	return f, e
}
