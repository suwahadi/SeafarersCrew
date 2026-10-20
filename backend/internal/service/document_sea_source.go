package service

import (
	"context"
	"strconv"

	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

// SEADataSource loads the contract data of a Seafarer Employment Agreement.
type SEADataSource interface {
	SEA(ctx context.Context, contractPublicID string) (repository.SEARecord, error)
}

var basisLabels = map[string]string{
	"fixed_month":   "Fixed per month",
	"fixed_day":     "Fixed per day (× 30)",
	"percent_basic": "Percentage of basic salary",
}

// SEAVars builds the variables of a Seafarer Employment Agreement (PRD 6.6 Crew and Contract groups).
func SEAVars(s repository.SEARecord, f docFormat) Vars {
	v := crewVars(s.Crew, f)
	total := s.BasicSalary
	rows := make([][]string, 0, len(s.Allowances))
	for _, a := range s.Allowances {
		value := moneyText(s.Currency, a.Value, s.CurrencyDecimal)
		if a.Basis == "percent_basic" {
			value = a.Value.StringFixed(2) + "%"
		}
		rows = append(rows, []string{a.Name, basisLabels[a.Basis], value, moneyText(s.Currency, a.MonthlyAmount, s.CurrencyDecimal)})
		total = total.Add(a.MonthlyAmount)
	}
	leave := ""
	if s.LeaveDays != nil {
		leave = s.LeaveDays.String()
	}
	for k, val := range map[string]string{
		"contract_id": s.ContractPublicID, "vessel_name": s.VesselName, "imo_number": s.IMONumber, "flag": s.Flag,
		"rank": s.Rank, "principal_name": s.PrincipalName, "employer_name": s.EmployerName, "sign_on_date": f.day(s.SignOnDate),
		"sign_off_date": f.day(s.PlannedSignOff), "duration_months": strconv.Itoa(int(s.DurationMonths)),
		"point_of_hire": s.PointOfHire, "sign_on_port": s.SignOnPort, "sign_off_port": s.SignOffPort, "currency": s.Currency,
		"basic_salary":      moneyText(s.Currency, s.BasicSalary, s.CurrencyDecimal),
		"total_monthly_pay": moneyText(s.Currency, total, s.CurrencyDecimal), "working_hours_week": optInt(s.WorkingHours),
		"leave_days_month": leave, "probation_months": optInt(s.ProbationMonths),
	} {
		v.Text[k] = val
	}
	v.HTML["allowance_table"] = HTMLTable([]string{"Allowance", "Basis", "Value", "Monthly amount"}, rows, [][]string{
		{"Basic salary", "", "", moneyText(s.Currency, s.BasicSalary, s.CurrencyDecimal)},
		{"Total monthly pay", "", "", moneyText(s.Currency, total, s.CurrencyDecimal)},
	}, "No allowances.")
	return v
}
