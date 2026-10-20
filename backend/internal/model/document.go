package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Document kinds produced by the document builder.
const (
	DocumentSEA       = "sea"
	DocumentQuotation = "quotation"
	DocumentCV        = "cv"
)

// DocumentTemplate is an editable template for one document kind.
type DocumentTemplate struct {
	ID          uuid.UUID
	Name        string
	Kind        string
	PaperSize   string
	Orientation string
	HeaderHTML  string
	BodyHTML    string
	FooterHTML  string
	IsDefault   bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Quotation line types.
const (
	LineDailyRate = "daily_rate"
	LineLumpSum   = "lump_sum"
)

// Quotation statuses.
const (
	QuotationDraft    = "draft"
	QuotationSent     = "sent"
	QuotationAccepted = "accepted"
	QuotationRejected = "rejected"
	QuotationExpired  = "expired"
)

// QuotationLine is one priced line of a quotation.
type QuotationLine struct {
	ID          uuid.UUID
	LineType    string
	RankID      *uuid.UUID
	RankName    string
	Description string
	CrewCount   *int
	DailyRate   *decimal.Decimal
	Days        *int
	Amount      decimal.Decimal
	SortOrder   int
}

// Quotation is a priced offer to a client.
type Quotation struct {
	ID              uuid.UUID
	PublicID        string
	ClientCompanyID uuid.UUID
	ClientName      string
	VesselID        *uuid.UUID
	VesselName      string
	CurrencyCode    string
	CurrencyDecimal int32
	QuotationDate   time.Time
	ValidUntil      time.Time
	TaxPercent      decimal.Decimal
	DiscountAmount  decimal.Decimal
	Terms           string
	Notes           string
	Status          string
	Subtotal        decimal.Decimal
	TaxAmount       decimal.Decimal
	GrandTotal      decimal.Decimal
	Lines           []QuotationLine
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// QuotationFilter holds quotation list filters.
type QuotationFilter struct {
	Q          string
	Statuses   []string
	ClientIDs  []uuid.UUID
	Currencies []string
	DateFrom   *time.Time
	DateTo     *time.Time
	Sort       string
	Desc       bool
	Limit      int
	Offset     int
}

// GeneratedDocument is a PDF produced from a template.
type GeneratedDocument struct {
	ID           uuid.UUID
	PublicID     string
	Kind         string
	TemplateID   *uuid.UUID
	TemplateName string
	RelatedType  string
	RelatedID    uuid.UUID
	FileID       uuid.UUID
	FileSize     int64
	CreatedAt    time.Time
}
