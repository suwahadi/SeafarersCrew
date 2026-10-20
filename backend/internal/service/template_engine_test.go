package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

func TestRenderTemplateEscapesTextValues(t *testing.T) {
	v := NewVars()
	v.Text["crew_name"] = `<script>alert("x")</script> & Sons`
	v.Text["rank"] = `"><img src=x onerror=alert(1)>`
	out, unknown := RenderTemplate(`<p title="{rank}">{crew_name}</p>`, v)
	assert.Empty(t, unknown)
	assert.NotContains(t, out, "<script>")
	assert.NotContains(t, out, "<img")
	assert.Contains(t, out, "&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt; &amp; Sons")
}

func TestRenderTemplateKeepsServerBuiltHTMLAndReportsUnknownTags(t *testing.T) {
	v := NewVars()
	v.HTML["allowance_table"] = "<table><tr><td>A</td></tr></table>"
	v.Text["crew_id"] = "{allowance_table}"
	out, unknown := RenderTemplate("<div>{allowance_table}</div>{crew_id}{mystery}{mystery}{Upper}{ not_a_tag}", v)
	assert.Equal(t, "<div><table><tr><td>A</td></tr></table></div>{allowance_table}{Upper}{ not_a_tag}", out, "values are not re-scanned for tags")
	assert.Equal(t, []string{"mystery"}, unknown)
}

func TestUnknownTagsDependOnKind(t *testing.T) {
	assert.Empty(t, UnknownTags(model.DocumentSEA, "{crew_name}{allowance_table}{company_name}{today}"))
	assert.Equal(t, []string{"pricing_table"}, UnknownTags(model.DocumentSEA, "{crew_name}{pricing_table}"))
	assert.Equal(t, []string{"crew_name"}, UnknownTags(model.DocumentQuotation, "{crew_name}{grand_total}"))
	assert.Empty(t, UnknownTags(model.DocumentCV, "{photo}{experience_table}{passport_no}"))
}

func TestVariableCatalogueCoversPRDTags(t *testing.T) {
	prd := map[string][]string{
		model.DocumentSEA: {"crew_id", "crew_name", "date_of_birth", "place_of_birth", "nationality", "passport_no", "passport_expiry",
			"seaman_book_no", "address", "mobile_phone", "contract_id", "vessel_name", "imo_number", "flag", "rank", "principal_name",
			"sign_on_date", "sign_off_date", "duration_months", "currency", "basic_salary", "allowance_table", "total_monthly_pay"},
		model.DocumentQuotation: {"quotation_id", "client_name", "quotation_date", "valid_until", "pricing_table", "subtotal", "tax", "grand_total", "terms"},
		model.DocumentCV:        {"photo", "experience_table", "certificate_table", "document_table", "sea_service_summary"},
	}
	for kind, tags := range prd {
		known := KnownTags(kind)
		for _, tag := range append(tags, "company_name", "company_address", "company_phone", "company_email", "today") {
			assert.True(t, known[tag], "%s should offer {%s}", kind, tag)
		}
	}
}

func TestHTMLTableEscapesCellsAndShowsEmptyText(t *testing.T) {
	out := HTMLTable([]string{"A", "B"}, [][]string{{"<b>x</b>", "y"}}, [][]string{{"Total", "1"}}, "none")
	assert.Contains(t, out, "<td>&lt;b&gt;x&lt;/b&gt;</td>")
	assert.Contains(t, out, "<tfoot><tr><th>Total</th><th>1</th></tr></tfoot>")
	empty := HTMLTable([]string{"A", "B"}, nil, nil, "No rows & more")
	assert.Contains(t, empty, `<td colspan="2" class="empty">No rows &amp; more</td>`)
}

func TestTemplatePolicyStripsActiveContentAndKeepsLayout(t *testing.T) {
	p := NewTemplatePolicy()
	in := `<h1 onclick="x()">Title {crew_name}</h1><script>alert(1)</script>
<a href="javascript:alert(1)">bad</a><a href="https://example.com">ok</a>
<img src="data:image/png;base64,iVBORw0KGgo=" alt="logo"><img src="https://evil.example/x.png" onerror="x()">
<table class="data-table"><tr><td colspan="2" style="text-align: right; position: fixed; width: 40mm">{allowance_table}</td></tr></table>
<div class="page-break" style="page-break-after: always"></div><span class="pageNumber"></span>
<iframe src="https://example.com"></iframe><style>body{display:none}</style><p style="background-image:url(x)">p</p>`
	out := p.Sanitize(in)
	for _, bad := range []string{"<script", "onclick", "javascript:", "onerror", "<iframe", "<style", "position", "background-image"} {
		assert.NotContains(t, out, bad)
	}
	for _, good := range []string{"{crew_name}", "{allowance_table}", `href="https://example.com"`, "data:image/png;base64,iVBORw0KGgo=",
		`colspan="2"`, "text-align: right", "width: 40mm", `class="page-break"`, "page-break-after: always", `class="pageNumber"`} {
		assert.Contains(t, out, good)
	}
	assert.True(t, strings.Contains(out, "<table"), out)
}
