package request

import (
	"net/url"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

// PayrollPeriod is the body of POST /payroll-periods and PUT /payroll-periods/{period}.
type PayrollPeriod struct {
	Title        string `json:"title"`
	PeriodStart  string `json:"period_start"`
	PeriodEnd    string `json:"period_end"`
	TransferDate string `json:"transfer_date"`

	parsed model.PayrollPeriod
}

// Validate implements Validator.
func (r *PayrollPeriod) Validate() map[string]string {
	Trim(&r.Title, &r.PeriodStart, &r.PeriodEnd, &r.TransferDate)
	e := Errors{}
	e.Required("title", r.Title, 120)
	start := e.Date("period_start", r.PeriodStart)
	end := e.Date("period_end", r.PeriodEnd)
	if r.PeriodStart == "" {
		e.set("period_start", "This field is required.")
	}
	if r.PeriodEnd == "" {
		e.set("period_end", "This field is required.")
	}
	if start != nil && end != nil && end.Before(*start) {
		e.set("period_end", "The end date must be on or after the start date.")
	}
	r.parsed = model.PayrollPeriod{Title: r.Title, TransferDate: e.Date("transfer_date", r.TransferDate)}
	if start != nil && end != nil {
		r.parsed.PeriodStart, r.parsed.PeriodEnd = *start, *end
	}
	return e
}

// Model returns the validated period.
func (r *PayrollPeriod) Model() model.PayrollPeriod { return r.parsed }

// PayrollLine is the body of POST /payroll-periods/{period}/lines and PUT /payroll-lines/{id}.
type PayrollLine struct {
	CrewID        string       `json:"crew_id"`
	BankAccountID string       `json:"bank_account_id"`
	AmountUSD     DecimalInput `json:"amount_usd"`
	AmountIDR     DecimalInput `json:"amount_idr"`
	TransferDate  string       `json:"transfer_date"`
	Remark        string       `json:"remark"`

	parsed model.PayrollLineInput
}

// Validate implements Validator.
func (r *PayrollLine) Validate() map[string]string {
	Trim(&r.CrewID, &r.BankAccountID, &r.TransferDate, &r.Remark)
	e := Errors{}
	e.Required("crew_id", r.CrewID, 20)
	e.MaxLen("remark", r.Remark, 300)
	r.parsed = model.PayrollLineInput{
		CrewPublicID:  r.CrewID,
		BankAccountID: e.RequiredUUID("bank_account_id", r.BankAccountID),
		AmountUSD:     e.Decimal("amount_usd", r.AmountUSD, decimal.Zero, 2),
		AmountIDR:     e.RequiredDecimal("amount_idr", r.AmountIDR, decimal.Zero, 2),
		TransferDate:  e.Date("transfer_date", r.TransferDate),
		Remark:        r.Remark,
	}
	return e
}

// Input returns the validated line.
func (r *PayrollLine) Input() model.PayrollLineInput { return r.parsed }

func payrollValues(q url.Values, key string) []string {
	var out []string
	for _, v := range q[key] {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func payrollFlag(q url.Values, key string) bool {
	switch q.Get(key) {
	case "1", "true":
		return true
	}
	return false
}

// ParsePayrollLineFilter reads the payroll lines filter from the query string; bank, vessel and rank repeat.
func ParsePayrollLineFilter(q url.Values) (model.PayrollLineFilter, map[string]string) {
	f := model.PayrollLineFilter{
		Q:         strings.TrimSpace(q.Get("q")),
		Banks:     payrollValues(q, "bank"),
		Vessels:   payrollValues(q, "vessel"),
		Ranks:     payrollValues(q, "rank"),
		HasRemark: payrollFlag(q, "has_remark"),
		NoUSD:     payrollFlag(q, "no_usd"),
		Sort:      strings.TrimSpace(q.Get("sort")),
		Desc:      q.Get("order") == "desc",
	}
	errs := map[string]string{}
	if f.Sort != "" && !repository.PayrollLineSortable(f.Sort) {
		errs["sort"] = "Unknown sort column."
	}
	return f, errs
}
