package request

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

func decodeQuotation(t *testing.T, body string) (*Quotation, map[string]string) {
	t.Helper()
	var q Quotation
	require.NoError(t, json.Unmarshal([]byte(body), &q))
	return &q, q.Validate()
}

func TestQuotationRequestParsesBothLineTypes(t *testing.T) {
	q, errs := decodeQuotation(t, `{"client_company_id":"0190c9a4-0000-7000-8000-000000000001","currency_code":"usd",
		"quotation_date":"2026-10-01","valid_until":"2026-10-31","tax_percent":11,"discount_amount":"50.5",
		"lines":[{"line_type":"daily_rate","rank_id":"0190c9a4-0000-7000-8000-000000000002","crew_count":2,"daily_rate":"275.50","days":30},
		{"line_type":"lump_sum","description":" Mobilisation ","amount":1500}]}`)
	require.Empty(t, errs)
	m := q.Model()
	assert.Equal(t, "USD", m.CurrencyCode)
	assert.True(t, decimal.NewFromInt(11).Equal(m.TaxPercent))
	assert.True(t, decimal.RequireFromString("50.5").Equal(m.DiscountAmount))
	require.Len(t, m.Lines, 2)
	assert.Equal(t, model.LineDailyRate, m.Lines[0].LineType)
	assert.True(t, decimal.RequireFromString("275.5").Equal(*m.Lines[0].DailyRate))
	assert.Equal(t, "Mobilisation", m.Lines[1].Description)
	assert.True(t, decimal.NewFromInt(1500).Equal(m.Lines[1].Amount))
}

func TestQuotationRequestValidation(t *testing.T) {
	_, errs := decodeQuotation(t, `{"currency_code":"USD","quotation_date":"2026-10-31","valid_until":"2026-10-01","tax_percent":"101",
		"discount_amount":"-1","lines":[{"line_type":"daily_rate","daily_rate":"1.234"},{"line_type":"lump_sum"},{"line_type":"other"}]}`)
	for _, field := range []string{"client_company_id", "valid_until", "tax_percent", "discount_amount",
		"lines.0.rank_id", "lines.0.crew_count", "lines.0.days", "lines.0.daily_rate",
		"lines.1.description", "lines.1.amount", "lines.2.line_type"} {
		assert.Contains(t, errs, field)
	}

	_, errs = decodeQuotation(t, `{"lines":[]}`)
	assert.Contains(t, errs, "lines")
	_, errs = decodeQuotation(t, `{"tax_percent":"abc","lines":[]}`)
	assert.Equal(t, "This must be a number.", errs["tax_percent"])
}

func TestParseQuotationFilter(t *testing.T) {
	q, _ := url.ParseQuery("status=draft,sent&currency=usd&date_from=2026-01-01&page=2&per_page=10&sort=grand_total&order=desc")
	f, errs := ParseQuotationFilter(q)
	assert.Empty(t, errs)
	assert.Equal(t, []string{"draft", "sent"}, f.Statuses)
	assert.Equal(t, []string{"USD"}, f.Currencies)
	assert.Equal(t, 10, f.Limit)
	assert.Equal(t, 10, f.Offset)
	assert.True(t, f.Desc)

	q, _ = url.ParseQuery("status=open&date_to=31-12-2026")
	_, errs = ParseQuotationFilter(q)
	assert.Contains(t, errs, "status")
	assert.Contains(t, errs, "date_to")
}

func TestDocumentTemplateRequestDefaultsAndLimits(t *testing.T) {
	r := DocumentTemplate{Name: "SEA", Kind: "sea", BodyHTML: "<p>x</p>"}
	assert.Empty(t, r.Validate())
	assert.Equal(t, "a4", r.PaperSize)
	assert.Equal(t, "portrait", r.Orientation)

	bad := DocumentTemplate{Kind: "invoice", PaperSize: "a3", Orientation: "diagonal"}
	errs := bad.Validate()
	for _, f := range []string{"name", "kind", "paper_size", "orientation"} {
		assert.Contains(t, errs, f)
	}
}
