package service

import (
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// DaysPerMonth converts a daily allowance into a monthly amount (PRD 11.4).
const DaysPerMonth = 30

// DueSoonDays marks timeline bars within this many days of planned sign-off (PRD 6.5).
const DueSoonDays = 30

// DefaultReliefMonths is the reliever commitment used when a contract has no duration.
const DefaultReliefMonths = 6

// Timeline bar states.
const (
	BarNormal  = "normal"
	BarDueSoon = "due_soon"
	BarOverdue = "overdue"
)

var hundred = decimal.NewFromInt(100)

// MonthlyAllowance computes an allowance per month in the contract currency, rounded to its decimals.
func MonthlyAllowance(basis string, value, basic decimal.Decimal, decimals int) decimal.Decimal {
	var amount decimal.Decimal
	switch basis {
	case model.BasisFixedDay:
		amount = value.Mul(decimal.NewFromInt(DaysPerMonth))
	case model.BasisPercentBasic:
		amount = basic.Mul(value).Div(hundred)
	default:
		amount = value
	}
	return amount.Round(int32(min(max(decimals, 0), 4)))
}

// TotalMonthlyPay is the basic salary plus every allowance, without currency conversion.
func TotalMonthlyPay(basic decimal.Decimal, allowances []model.ContractAllowance) decimal.Decimal {
	total := basic
	for _, a := range allowances {
		total = total.Add(a.MonthlyAmount)
	}
	return total
}

// PlannedSignOff returns sign-on plus duration months.
func PlannedSignOff(signOn time.Time, months int) time.Time {
	return datetime.AddMonths(signOn, months)
}

// CheckActivation enforces PRD 11.2 and 11.4 before a contract becomes Active.
func CheckActivation(contractStatus, crewStatus string, crewHasActive bool) error {
	if contractStatus != model.ContractDraft {
		return apperr.Conflict("contract_not_draft", "Only a draft contract can be activated.")
	}
	if crewHasActive {
		return apperr.Conflict("crew_has_active_contract", "The crew member already holds an active contract.")
	}
	if !CanBeAssigned(crewStatus) {
		return apperr.Conflict("crew_not_available", "Only an Available crew member can be assigned to an active contract.")
	}
	return nil
}

// CheckFinish validates completing or terminating an Active contract.
func CheckFinish(c model.Contract, actualSignOff time.Time, reason string, terminate bool) error {
	if c.Status != model.ContractActive {
		return apperr.Conflict("contract_not_active", "Only an active contract can be completed or terminated.")
	}
	fields := map[string]string{}
	if actualSignOff.Before(c.SignOnDate) {
		fields["actual_sign_off_date"] = "The sign-off date must be on or after the sign-on date."
	}
	if terminate && reason == "" {
		fields["reason"] = "A reason is required to terminate a contract."
	}
	if len(fields) > 0 {
		return apperr.Validation(fields)
	}
	return nil
}

// ActiveEditableFields lists what may still change once a contract is Active.
var ActiveEditableFields = []string{"planned_sign_off_date", "sign_on_port", "sign_off_port", "notes"}

// LockedChanges returns the fields of next that differ from cur but are not editable on an Active contract.
func LockedChanges(cur, next model.Contract) []string {
	var changed []string
	add := func(field string, differs bool) {
		if differs && !slices.Contains(ActiveEditableFields, field) {
			changed = append(changed, field)
		}
	}
	add("crew_id", cur.CrewID != next.CrewID)
	add("vessel_id", cur.VesselID != next.VesselID)
	add("rank_id", cur.RankID != next.RankID)
	add("principal_id", cur.PrincipalID != next.PrincipalID)
	add("employer_company_id", cur.EmployerCompanyID != next.EmployerCompanyID)
	add("sign_on_date", !cur.SignOnDate.Equal(next.SignOnDate))
	add("duration_months", cur.DurationMonths != next.DurationMonths)
	add("point_of_hire", cur.PointOfHire != next.PointOfHire)
	add("currency_code", cur.CurrencyCode != next.CurrencyCode)
	add("basic_salary", !cur.BasicSalary.Equal(next.BasicSalary))
	add("working_hours_week", !equalInt(cur.WorkingHoursWeek, next.WorkingHoursWeek))
	add("leave_days_month", !equalDecimal(cur.LeaveDaysMonth, next.LeaveDaysMonth))
	add("probation_months", !equalInt(cur.ProbationMonths, next.ProbationMonths))
	add("allowances", !sameAllowances(cur.Allowances, next.Allowances))
	return changed
}

func equalInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func equalDecimal(a, b *decimal.Decimal) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func sameAllowances(a, b []model.ContractAllowance) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].AllowanceTypeID != b[i].AllowanceTypeID || !a[i].Value.Equal(b[i].Value) {
			return false
		}
	}
	return true
}

// ReliefWindowEnd is the last day a reliever is committed: relief date plus the relieved contract's duration.
func ReliefWindowEnd(w model.ReliefWindow) time.Time {
	months := w.DurationMonths
	if months <= 0 {
		months = DefaultReliefMonths
	}
	return datetime.AddMonths(w.ReliefDate, months)
}

// ReliefWindowsOverlap reports whether two inclusive reliever windows share at least one day.
func ReliefWindowsOverlap(a, b model.ReliefWindow) bool {
	return !a.ReliefDate.After(ReliefWindowEnd(b)) && !b.ReliefDate.After(ReliefWindowEnd(a))
}

// BarState flags a timeline bar against today.
func BarState(plannedSignOff, today time.Time) string {
	days := datetime.DaysBetween(today, plannedSignOff)
	switch {
	case days < 0:
		return BarOverdue
	case days <= DueSoonDays:
		return BarDueSoon
	default:
		return BarNormal
	}
}
