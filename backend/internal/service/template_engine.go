package service

import (
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

var tagPattern = regexp.MustCompile(`\{([a-z][a-z0-9_]*)\}`)

// Vars holds the values merged into a template: Text values are HTML-escaped, HTML values are trusted markup built by the server.
type Vars struct {
	Text map[string]string
	HTML map[string]string
}

// NewVars returns empty Vars.
func NewVars() Vars {
	return Vars{Text: map[string]string{}, HTML: map[string]string{}}
}

// Merge copies every value of other into v.
func (v Vars) Merge(other Vars) {
	for k, val := range other.Text {
		v.Text[k] = val
	}
	for k, val := range other.HTML {
		v.HTML[k] = val
	}
}

// RenderTemplate replaces {snake_case} tags; tags without a value render empty and are returned as unknown.
func RenderTemplate(src string, v Vars) (string, []string) {
	var unknown []string
	out := tagPattern.ReplaceAllStringFunc(src, func(m string) string {
		name := m[1 : len(m)-1]
		if h, ok := v.HTML[name]; ok {
			return h
		}
		if t, ok := v.Text[name]; ok {
			return html.EscapeString(t)
		}
		if !slices.Contains(unknown, name) {
			unknown = append(unknown, name)
		}
		return ""
	})
	slices.Sort(unknown)
	return out, unknown
}

// TemplateTags returns the distinct tags used in src.
func TemplateTags(src string) []string {
	var tags []string
	for _, m := range tagPattern.FindAllStringSubmatch(src, -1) {
		if !slices.Contains(tags, m[1]) {
			tags = append(tags, m[1])
		}
	}
	return tags
}

// Variable is one tag of the variable picker.
type Variable struct {
	Tag         string `json:"tag"`
	Description string `json:"description"`
}

// VariableGroup groups tags for the variable picker.
type VariableGroup struct {
	Name  string     `json:"name"`
	Kinds []string   `json:"kinds"`
	Vars  []Variable `json:"variables"`
}

var allKinds = []string{model.DocumentSEA, model.DocumentQuotation, model.DocumentCV}

// VariableCatalogue lists every supported tag grouped as in PRD 6.6.
var VariableCatalogue = []VariableGroup{
	{Name: "Crew", Kinds: []string{model.DocumentSEA, model.DocumentCV}, Vars: []Variable{
		{"crew_id", "Crew ID"}, {"crew_name", "Full name"}, {"date_of_birth", "Date of birth"},
		{"place_of_birth", "Place of birth"}, {"nationality", "Nationality"}, {"passport_no", "Passport number"},
		{"passport_expiry", "Passport expiry date"}, {"seaman_book_no", "Seaman book number"}, {"address", "Full address"},
		{"mobile_phone", "Mobile phone"}, {"email", "Email"}, {"gender", "Gender"}, {"marital_status", "Marital status"},
		{"religion", "Religion"}, {"age", "Age in years"}, {"nearest_airport", "Nearest airport"},
	}},
	{Name: "Contract", Kinds: []string{model.DocumentSEA}, Vars: []Variable{
		{"contract_id", "Contract ID"}, {"vessel_name", "Vessel name"}, {"imo_number", "IMO number"}, {"flag", "Vessel flag"},
		{"rank", "Rank"}, {"principal_name", "Principal"}, {"employer_name", "Employer company"},
		{"sign_on_date", "Sign-on date"}, {"sign_off_date", "Planned sign-off date"}, {"duration_months", "Contract duration in months"},
		{"point_of_hire", "Point of hire"}, {"sign_on_port", "Place of sign-on"}, {"sign_off_port", "Place of sign-off"},
		{"currency", "Currency code"}, {"basic_salary", "Basic monthly salary"}, {"allowance_table", "Table of allowances"},
		{"total_monthly_pay", "Basic salary plus allowances"}, {"working_hours_week", "Working hours per week"},
		{"leave_days_month", "Leave days per month"}, {"probation_months", "Probation period in months"},
	}},
	{Name: "Quotation", Kinds: []string{model.DocumentQuotation}, Vars: []Variable{
		{"quotation_id", "Quotation ID"}, {"client_name", "Client"}, {"quotation_date", "Quotation date"},
		{"valid_until", "Valid until"}, {"pricing_table", "Daily rate and lump sum tables"}, {"subtotal", "Subtotal"},
		{"discount", "Discount"}, {"tax_percent", "Tax percentage"}, {"tax", "Tax amount"}, {"grand_total", "Grand total"},
		{"terms", "Terms and conditions"}, {"currency", "Currency code"}, {"vessel_name", "Vessel name"},
	}},
	{Name: "CV", Kinds: []string{model.DocumentCV}, Vars: []Variable{
		{"photo", "Profile photo"}, {"applied_rank", "Applied rank"}, {"experience_table", "Sea service history"},
		{"certificate_table", "STCW certificates"}, {"document_table", "Travel documents"},
		{"sea_service_summary", "Total sea service per rank"},
	}},
	{Name: "Company", Kinds: allKinds, Vars: []Variable{
		{"company_name", "Legal company name"}, {"company_address", "Company address"}, {"company_phone", "Company phone"},
		{"company_email", "Company email"}, {"company_website", "Company website"}, {"company_logo", "Company logo"},
		{"signatory_name", "Signatory name"}, {"signatory_title", "Signatory title"}, {"today", "Today's date"},
	}},
}

// KnownTags returns the tags available for a document kind.
func KnownTags(kind string) map[string]bool {
	out := map[string]bool{}
	for _, g := range VariableCatalogue {
		if !slices.Contains(g.Kinds, kind) {
			continue
		}
		for _, v := range g.Vars {
			out[v.Tag] = true
		}
	}
	return out
}

// UnknownTags returns tags in src that are not available for kind.
func UnknownTags(kind string, src string) []string {
	known := KnownTags(kind)
	var out []string
	for _, t := range TemplateTags(src) {
		if !known[t] {
			out = append(out, t)
		}
	}
	slices.Sort(out)
	return out
}

// HTMLTable builds an escaped table; footer rows render bold in a tfoot.
func HTMLTable(headers []string, rows [][]string, footer [][]string, emptyText string) string {
	var b strings.Builder
	b.WriteString(`<table class="data-table"><thead><tr>`)
	for _, h := range headers {
		b.WriteString("<th>" + html.EscapeString(h) + "</th>")
	}
	b.WriteString("</tr></thead><tbody>")
	if len(rows) == 0 {
		b.WriteString(`<tr><td colspan="` + strconv.Itoa(len(headers)) + `" class="empty">` + html.EscapeString(emptyText) + "</td></tr>")
	}
	for _, r := range rows {
		writeRow(&b, "td", r)
	}
	b.WriteString("</tbody>")
	if len(footer) > 0 {
		b.WriteString("<tfoot>")
		for _, r := range footer {
			writeRow(&b, "th", r)
		}
		b.WriteString("</tfoot>")
	}
	b.WriteString("</table>")
	return b.String()
}

func writeRow(b *strings.Builder, cell string, r []string) {
	b.WriteString("<tr>")
	for _, c := range r {
		b.WriteString("<" + cell + ">" + html.EscapeString(c) + "</" + cell + ">")
	}
	b.WriteString("</tr>")
}
