package resource

import (
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// PayrollPeriod is the JSON shape of a payroll period.
type PayrollPeriod struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	PeriodStart  Date      `json:"period_start"`
	PeriodEnd    Date      `json:"period_end"`
	TransferDate *Date     `json:"transfer_date"`
	LineCount    int       `json:"line_count"`
	TotalUSD     string    `json:"total_usd"`
	TotalIDR     string    `json:"total_idr"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewPayrollPeriod maps a period.
func NewPayrollPeriod(p model.PayrollPeriod) PayrollPeriod {
	return PayrollPeriod{
		ID: p.PublicID, Title: p.Title, PeriodStart: Date(p.PeriodStart), PeriodEnd: Date(p.PeriodEnd), TransferDate: NewDate(p.TransferDate),
		LineCount: p.LineCount, TotalUSD: p.TotalUSD.StringFixed(2), TotalIDR: p.TotalIDR.StringFixed(2), CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

// NewPayrollPeriods maps periods.
func NewPayrollPeriods(items []model.PayrollPeriod) []PayrollPeriod {
	out := make([]PayrollPeriod, len(items))
	for i, p := range items {
		out[i] = NewPayrollPeriod(p)
	}
	return out
}

// PayrollLineCrew links a line to its crew record.
type PayrollLineCrew struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PayrollLine is the JSON shape of a payroll line.
type PayrollLine struct {
	ID            uuid.UUID       `json:"id"`
	LineNo        int             `json:"line_no"`
	Crew          PayrollLineCrew `json:"crew"`
	EmployeeNo    string          `json:"employee_no"`
	CrewName      string          `json:"crew_name"`
	RankCode      string          `json:"rank_code"`
	VesselName    string          `json:"vessel_name"`
	BankAccountID *uuid.UUID      `json:"bank_account_id"`
	BankName      string          `json:"bank_name"`
	AccountName   string          `json:"account_name"`
	AccountNumber string          `json:"account_number"`
	AmountUSD     *string         `json:"amount_usd"`
	AmountIDR     string          `json:"amount_idr"`
	TransferDate  Date            `json:"transfer_date"`
	Remark        string          `json:"remark"`
}

// NewPayrollLine maps a line.
func NewPayrollLine(l model.PayrollLine) PayrollLine {
	out := PayrollLine{
		ID: l.ID, LineNo: l.LineNo, Crew: PayrollLineCrew{ID: l.CrewPublicID, Name: l.CrewName}, EmployeeNo: l.EmployeeNo,
		CrewName: l.CrewName, RankCode: l.RankCode, VesselName: l.VesselName, BankAccountID: l.BankAccountID, BankName: l.BankName,
		AccountName: l.AccountName, AccountNumber: l.AccountNumber, AmountIDR: l.AmountIDR.StringFixed(2), TransferDate: Date(l.TransferDate),
		Remark: l.Remark,
	}
	if l.AmountUSD != nil {
		usd := l.AmountUSD.StringFixed(2)
		out.AmountUSD = &usd
	}
	return out
}

// NewPayrollLines maps lines.
func NewPayrollLines(items []model.PayrollLine) []PayrollLine {
	out := make([]PayrollLine, len(items))
	for i, l := range items {
		out[i] = NewPayrollLine(l)
	}
	return out
}

// PayrollTotals is the JSON shape of the totals of the filtered lines.
type PayrollTotals struct {
	Count int    `json:"count"`
	USD   string `json:"usd"`
	IDR   string `json:"idr"`
}

// NewPayrollTotals maps totals.
func NewPayrollTotals(t model.PayrollTotals) PayrollTotals {
	return PayrollTotals{Count: t.Count, USD: t.USD.StringFixed(2), IDR: t.IDR.StringFixed(2)}
}

// PayrollFacets is the JSON shape of a period's filter options.
type PayrollFacets struct {
	Banks   []string `json:"banks"`
	Vessels []string `json:"vessels"`
	Ranks   []string `json:"ranks"`
}

// NewPayrollFacets maps facets, never returning null lists.
func NewPayrollFacets(f model.PayrollFacets) PayrollFacets {
	nonNil := func(v []string) []string {
		if v == nil {
			return []string{}
		}
		return v
	}
	return PayrollFacets{Banks: nonNil(f.Banks), Vessels: nonNil(f.Vessels), Ranks: nonNil(f.Ranks)}
}
