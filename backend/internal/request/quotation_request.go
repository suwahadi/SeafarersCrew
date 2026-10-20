package request

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// MaxQuotationLines limits the lines of one quotation.
const MaxQuotationLines = 100

// QuotationLine is one line of a quotation body.
type QuotationLine struct {
	LineType    string       `json:"line_type"`
	RankID      string       `json:"rank_id"`
	Description string       `json:"description"`
	CrewCount   *int         `json:"crew_count"`
	DailyRate   DecimalInput `json:"daily_rate"`
	Days        *int         `json:"days"`
	Amount      DecimalInput `json:"amount"`
}

// Quotation is the body of POST /quotations and PUT /quotations/{id}.
type Quotation struct {
	ClientCompanyID string          `json:"client_company_id"`
	VesselID        string          `json:"vessel_id"`
	CurrencyCode    string          `json:"currency_code"`
	QuotationDate   string          `json:"quotation_date"`
	ValidUntil      string          `json:"valid_until"`
	TaxPercent      DecimalInput    `json:"tax_percent"`
	DiscountAmount  DecimalInput    `json:"discount_amount"`
	Terms           string          `json:"terms"`
	Notes           string          `json:"notes"`
	Lines           []QuotationLine `json:"lines"`

	parsed model.Quotation
}

// Validate implements Validator.
func (r *Quotation) Validate() map[string]string {
	Trim(&r.ClientCompanyID, &r.VesselID, &r.CurrencyCode, &r.QuotationDate, &r.ValidUntil, &r.Terms, &r.Notes)
	r.CurrencyCode = strings.ToUpper(r.CurrencyCode)
	e := Errors{}
	q := model.Quotation{CurrencyCode: r.CurrencyCode, Terms: r.Terms, Notes: r.Notes}
	q.ClientCompanyID = e.RequiredUUID("client_company_id", r.ClientCompanyID)
	q.VesselID = e.UUID("vessel_id", r.VesselID)
	e.Required("currency_code", r.CurrencyCode, 3)
	q.QuotationDate = e.RequiredDate("quotation_date", r.QuotationDate)
	q.ValidUntil = e.RequiredDate("valid_until", r.ValidUntil)
	if !q.QuotationDate.IsZero() && !q.ValidUntil.IsZero() && q.ValidUntil.Before(q.QuotationDate) {
		e["valid_until"] = "The valid-until date must be on or after the quotation date."
	}
	if tax := e.Decimal("tax_percent", r.TaxPercent, decimal.Zero, 2); tax != nil {
		if tax.GreaterThan(decimal.NewFromInt(100)) {
			e["tax_percent"] = "The tax percentage may not exceed 100."
		}
		q.TaxPercent = *tax
	}
	if d := e.Decimal("discount_amount", r.DiscountAmount, decimal.Zero, 2); d != nil {
		q.DiscountAmount = *d
	}
	e.MaxLen("terms", r.Terms, 20000)
	e.MaxLen("notes", r.Notes, 2000)
	switch {
	case len(r.Lines) == 0:
		e["lines"] = "Add at least one daily rate or lump sum line."
	case len(r.Lines) > MaxQuotationLines:
		e["lines"] = fmt.Sprintf("A quotation may have at most %d lines.", MaxQuotationLines)
	}
	for i, l := range r.Lines {
		q.Lines = append(q.Lines, parseQuotationLine(e, fmt.Sprintf("lines.%d.", i), l))
	}
	r.parsed = q
	return e
}

func parseQuotationLine(e Errors, prefix string, l QuotationLine) model.QuotationLine {
	line := model.QuotationLine{LineType: l.LineType, Description: strings.TrimSpace(l.Description)}
	e.MaxLen(prefix+"description", line.Description, 500)
	switch l.LineType {
	case model.LineDailyRate:
		id := e.RequiredUUID(prefix+"rank_id", strings.TrimSpace(l.RankID))
		line.RankID = &id
		if l.CrewCount == nil {
			e[prefix+"crew_count"] = "This field is required."
		} else {
			e.Range(prefix+"crew_count", l.CrewCount, 1, 10000)
		}
		if l.Days == nil {
			e[prefix+"days"] = "This field is required."
		} else {
			e.Range(prefix+"days", l.Days, 1, 3660)
		}
		rate := e.RequiredDecimal(prefix+"daily_rate", l.DailyRate, decimal.Zero, 2)
		line.CrewCount, line.Days, line.DailyRate = l.CrewCount, l.Days, &rate
	case model.LineLumpSum:
		if line.Description == "" {
			e[prefix+"description"] = "This field is required."
		}
		line.Amount = e.RequiredDecimal(prefix+"amount", l.Amount, decimal.Zero, 2)
	default:
		e[prefix+"line_type"] = "The line type must be daily_rate or lump_sum."
	}
	return line
}

// Model returns the validated quotation.
func (r *Quotation) Model() model.Quotation { return r.parsed }

// QuotationStatus is the body of PATCH /quotations/{id}/status.
type QuotationStatus struct {
	Status string `json:"status"`
}

// ParseQuotationFilter reads quotation list filters.
func ParseQuotationFilter(q url.Values) (model.QuotationFilter, Errors) {
	e := Errors{}
	list := ParseList(q)
	f := model.QuotationFilter{Q: list.Q, Sort: list.Sort, Desc: list.Desc, Statuses: CSV(q, "status"), Limit: list.PerPage, Offset: list.Offset()}
	for _, s := range f.Statuses {
		e.OneOf("status", s, model.QuotationDraft, model.QuotationSent, model.QuotationAccepted, model.QuotationRejected, model.QuotationExpired)
	}
	for _, c := range CSV(q, "currency") {
		f.Currencies = append(f.Currencies, strings.ToUpper(c))
	}
	f.ClientIDs = uuids(e, q, "client_company_id")
	f.DateFrom = e.Date("date_from", q.Get("date_from"))
	f.DateTo = e.Date("date_to", q.Get("date_to"))
	return f, e
}
