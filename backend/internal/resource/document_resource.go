package resource

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

// DocumentTemplate is a document builder template.
type DocumentTemplate struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	PaperSize   string    `json:"paper_size"`
	Orientation string    `json:"orientation"`
	HeaderHTML  string    `json:"header_html,omitempty"`
	BodyHTML    string    `json:"body_html,omitempty"`
	FooterHTML  string    `json:"footer_html,omitempty"`
	IsDefault   bool      `json:"is_default"`
	UnknownTags []string  `json:"unknown_tags,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewTemplateSummary maps a template without its HTML, for lists.
func NewTemplateSummary(t model.DocumentTemplate) DocumentTemplate {
	return DocumentTemplate{ID: t.ID, Name: t.Name, Kind: t.Kind, PaperSize: t.PaperSize, Orientation: t.Orientation,
		IsDefault: t.IsDefault, UpdatedAt: t.UpdatedAt}
}

// NewTemplate maps a template with its HTML and unknown tag warnings.
func NewTemplate(r service.TemplateResult) DocumentTemplate {
	t := NewTemplateSummary(r.Template)
	t.HeaderHTML, t.BodyHTML, t.FooterHTML = r.Template.HeaderHTML, r.Template.BodyHTML, r.Template.FooterHTML
	t.UnknownTags = r.UnknownTags
	if t.UnknownTags == nil {
		t.UnknownTags = []string{}
	}
	return t
}

// QuotationLine is a priced line.
type QuotationLine struct {
	ID          uuid.UUID        `json:"id"`
	LineType    string           `json:"line_type"`
	RankID      *uuid.UUID       `json:"rank_id"`
	RankName    string           `json:"rank_name"`
	Description string           `json:"description"`
	CrewCount   *int             `json:"crew_count"`
	DailyRate   *decimal.Decimal `json:"daily_rate"`
	Days        *int             `json:"days"`
	Amount      decimal.Decimal  `json:"amount"`
}

// Quotation is a quotation with totals; Lines is omitted in lists.
type Quotation struct {
	ID              string          `json:"id"`
	ClientCompanyID uuid.UUID       `json:"client_company_id"`
	ClientName      string          `json:"client_name"`
	VesselID        *uuid.UUID      `json:"vessel_id"`
	VesselName      string          `json:"vessel_name"`
	CurrencyCode    string          `json:"currency_code"`
	CurrencyDecimal int32           `json:"currency_decimals"`
	QuotationDate   Date            `json:"quotation_date"`
	ValidUntil      Date            `json:"valid_until"`
	TaxPercent      decimal.Decimal `json:"tax_percent"`
	DiscountAmount  decimal.Decimal `json:"discount_amount"`
	Subtotal        decimal.Decimal `json:"subtotal"`
	TaxAmount       decimal.Decimal `json:"tax_amount"`
	GrandTotal      decimal.Decimal `json:"grand_total"`
	Terms           string          `json:"terms,omitempty"`
	Notes           string          `json:"notes,omitempty"`
	Status          string          `json:"status"`
	Lines           []QuotationLine `json:"lines,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// NewQuotationRow maps a quotation for the datagrid.
func NewQuotationRow(q model.Quotation) Quotation {
	return Quotation{
		ID: q.PublicID, ClientCompanyID: q.ClientCompanyID, ClientName: q.ClientName, VesselID: q.VesselID, VesselName: q.VesselName,
		CurrencyCode: q.CurrencyCode, CurrencyDecimal: q.CurrencyDecimal, QuotationDate: Date(q.QuotationDate),
		ValidUntil: Date(q.ValidUntil), TaxPercent: q.TaxPercent, DiscountAmount: q.DiscountAmount, Subtotal: q.Subtotal,
		TaxAmount: q.TaxAmount, GrandTotal: q.GrandTotal, Status: q.Status, CreatedAt: q.CreatedAt, UpdatedAt: q.UpdatedAt,
	}
}

// NewQuotation maps a quotation with lines, terms and notes.
func NewQuotation(q model.Quotation) Quotation {
	out := NewQuotationRow(q)
	out.Terms, out.Notes = q.Terms, q.Notes
	out.Lines = make([]QuotationLine, len(q.Lines))
	for i, l := range q.Lines {
		out.Lines[i] = QuotationLine{ID: l.ID, LineType: l.LineType, RankID: l.RankID, RankName: l.RankName, Description: l.Description,
			CrewCount: l.CrewCount, DailyRate: l.DailyRate, Days: l.Days, Amount: l.Amount}
	}
	return out
}

// GeneratedDocument is one entry of the generated documents history.
type GeneratedDocument struct {
	ID           string     `json:"id"`
	Kind         string     `json:"kind"`
	TemplateID   *uuid.UUID `json:"template_id"`
	TemplateName string     `json:"template_name"`
	RelatedType  string     `json:"related_type"`
	FileID       uuid.UUID  `json:"file_id"`
	FileSize     int64      `json:"file_size"`
	CreatedAt    time.Time  `json:"created_at"`
}

// NewGeneratedDocument maps a generated document.
func NewGeneratedDocument(g model.GeneratedDocument) GeneratedDocument {
	return GeneratedDocument{ID: g.PublicID, Kind: g.Kind, TemplateID: g.TemplateID, TemplateName: g.TemplateName,
		RelatedType: g.RelatedType, FileID: g.FileID, FileSize: g.FileSize, CreatedAt: g.CreatedAt}
}
