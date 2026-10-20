package service

import (
	"html"
	"regexp"
	"strings"
)

var emailTagPattern = regexp.MustCompile(`\{([a-z][a-z0-9_]*)\}`)

// RenderEmail replaces {snake_case} tags with vars; values are HTML-escaped when escapeHTML is set and unknown tags render empty.
func RenderEmail(tpl string, vars map[string]string, escapeHTML bool) string {
	return emailTagPattern.ReplaceAllStringFunc(tpl, func(m string) string {
		v := vars[m[1:len(m)-1]]
		if escapeHTML {
			return html.EscapeString(v)
		}
		return v
	})
}

// RenderEmailSubject renders a subject line as plain text on a single line.
func RenderEmailSubject(tpl string, vars map[string]string) string {
	return strings.Join(strings.Fields(RenderEmail(tpl, vars, false)), " ")
}

// EmailTags lists the distinct tags used in tpl, in order of first use.
func EmailTags(tpl string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range emailTagPattern.FindAllStringSubmatch(tpl, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

var emailCompanyTags = []string{"company_name", "company_address", "company_phone", "company_email", "company_website", "today"}

var emailTemplateTags = map[string][]string{
	"send_contract":          {"crew_name", "crew_id", "contract_id", "rank", "vessel_name", "sign_on_date"},
	"send_quotation":         {"client_name", "quotation_id", "quotation_date", "valid_until"},
	"send_cv":                {"crew_name", "crew_id", "rank"},
	"applicant_confirmation": {"applicant_name", "application_id", "vacancy_title"},
	"admin_new_application":  {"application_id", "vacancy_title", "vacancy_id", "applicant_name", "applicant_email", "admin_url"},
	"admin_new_contact":      {"subject", "sender_name", "sender_email", "sender_phone", "message", "admin_url"},
}

// EmailAvailableTags returns the tags a template may use, its own first and then the company tags.
func EmailAvailableTags(code string) []string {
	return append(append([]string{}, emailTemplateTags[code]...), emailCompanyTags...)
}

var emailSampleValues = map[string]string{
	"crew_name": "Budi Santoso", "crew_id": "SR6H72QU9", "contract_id": "CT4K9P2XW", "rank": "Chief Officer",
	"vessel_name": "MV Ocean Star", "sign_on_date": "01-Nov-2026", "client_name": "Alpha Offshore Pte Ltd",
	"quotation_id": "QT7M3N8RD", "quotation_date": "20-Oct-2026", "valid_until": "19-Nov-2026",
	"applicant_name": "Siti Aminah", "application_id": "AP5H2J9KQ", "vacancy_title": "Chief Engineer – AHTS",
	"vacancy_id": "VC8D3F6TW", "applicant_email": "siti@example.com", "admin_url": "https://admin.example.com",
	"subject": "Crew availability", "sender_name": "John Doe", "sender_email": "john@example.com",
	"sender_phone": "+6281234567890", "message": "Do you have a 2nd Engineer available next month?",
}

// EmailSampleVariables returns sample values for previews.
func EmailSampleVariables() map[string]string {
	out := make(map[string]string, len(emailSampleValues))
	for k, v := range emailSampleValues {
		out[k] = v
	}
	return out
}
