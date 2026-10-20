package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/money"
)

// FileReader loads stored file content for embedding in documents.
type FileReader interface {
	Content(ctx context.Context, id uuid.UUID) ([]byte, string, error)
}

// docFormat renders dates and money the way Settings ask for.
type docFormat struct {
	pattern string
	today   time.Time
}

func (f docFormat) date(t *time.Time) string {
	if t == nil {
		return ""
	}
	return datetime.FormatDate(*t, f.pattern)
}

func (f docFormat) day(t time.Time) string { return datetime.FormatDate(t, f.pattern) }

func moneyText(code string, v decimal.Decimal, decimals int32) string {
	return money.FormatWithCode(code, v, decimals)
}

func optInt(v *int16) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(int(*v))
}

func titleCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func joinNonEmpty(sep string, parts ...string) string {
	var out []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, sep)
}

func crewVars(p repository.CrewProfile, f docFormat) Vars {
	v := NewVars()
	age := ""
	if a := datetime.Age(p.DateOfBirth, f.today); a != nil {
		age = strconv.Itoa(*a)
	}
	for k, val := range map[string]string{
		"crew_id": p.PublicID, "crew_name": p.FullName, "date_of_birth": f.date(p.DateOfBirth), "place_of_birth": p.PlaceOfBirth,
		"nationality": p.Nationality, "passport_no": p.PassportNo, "passport_expiry": f.date(p.PassportExpiry),
		"seaman_book_no": p.SeamanBookNo, "mobile_phone": p.MobilePhone, "email": p.Email, "gender": titleCase(p.Gender),
		"marital_status": titleCase(p.MaritalStatus), "religion": p.Religion, "age": age, "nearest_airport": p.NearestAirport,
		"applied_rank": p.AppliedRank,
		"address":      joinNonEmpty(", ", p.Address, p.City, p.Province, p.PostalCode, p.Country),
	} {
		v.Text[k] = val
	}
	return v
}

// QuotationVars builds the variables of a quotation (PRD 6.6 Quotation group).
func QuotationVars(q model.Quotation, f docFormat) Vars {
	v := NewVars()
	m := func(d decimal.Decimal) string { return moneyText(q.CurrencyCode, d, q.CurrencyDecimal) }
	for k, val := range map[string]string{
		"quotation_id": q.PublicID, "client_name": q.ClientName, "quotation_date": f.day(q.QuotationDate),
		"valid_until": f.day(q.ValidUntil), "subtotal": m(q.Subtotal), "discount": m(q.DiscountAmount),
		"tax_percent": q.TaxPercent.StringFixed(2) + "%", "tax": m(q.TaxAmount), "grand_total": m(q.GrandTotal),
		"currency": q.CurrencyCode, "vessel_name": q.VesselName,
	} {
		v.Text[k] = val
	}
	v.HTML["terms"] = textToHTML(q.Terms)
	v.HTML["pricing_table"] = pricingTable(q, m)
	return v
}

func pricingTable(q model.Quotation, m func(decimal.Decimal) string) string {
	var daily, lump [][]string
	for _, l := range q.Lines {
		if l.LineType == model.LineDailyRate {
			rate := decimal.Zero
			if l.DailyRate != nil {
				rate = *l.DailyRate
			}
			rank := l.RankName
			if l.Description != "" {
				rank = joinNonEmpty(" – ", rank, l.Description)
			}
			daily = append(daily, []string{rank, intText(l.CrewCount), m(rate), intText(l.Days), m(l.Amount)})
			continue
		}
		lump = append(lump, []string{l.Description, m(l.Amount)})
	}
	var b strings.Builder
	if len(daily) > 0 {
		b.WriteString(`<h4 class="pricing-heading">Daily rate</h4>`)
		b.WriteString(HTMLTable([]string{"Rank", "Crew", "Daily rate", "Days", "Amount"}, daily, nil, ""))
	}
	if len(lump) > 0 {
		b.WriteString(`<h4 class="pricing-heading">Lump sum</h4>`)
		b.WriteString(HTMLTable([]string{"Description", "Amount"}, lump, nil, ""))
	}
	return b.String()
}

func intText(n *int) string {
	if n == nil {
		return ""
	}
	return strconv.Itoa(*n)
}

func textToHTML(s string) string {
	var b strings.Builder
	for _, para := range strings.Split(strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n"), "\n\n") {
		if strings.TrimSpace(para) == "" {
			continue
		}
		b.WriteString("<p>" + strings.ReplaceAll(html.EscapeString(para), "\n", "<br>") + "</p>")
	}
	return b.String()
}

// CV sections that can be hidden per generation.
const (
	CVHidePhoto        = "photo"
	CVHidePersonal     = "personal"
	CVHideDocuments    = "documents"
	CVHideCertificates = "certificates"
	CVHideExperiences  = "experiences"
	CVHideSeaService   = "sea_service"
)

// CVSections lists the hideable CV sections.
var CVSections = []string{CVHidePhoto, CVHidePersonal, CVHideDocuments, CVHideCertificates, CVHideExperiences, CVHideSeaService}

// CVData is everything a crew CV can show; bank data is never part of it.
type CVData struct {
	Profile      repository.CrewProfile
	Documents    []DocumentView
	Certificates []DocumentView
	SeaService   CrewSeaService
	Photo        []byte
	PhotoMime    string
}

// CVVars builds the variables of a crew CV, leaving hidden sections empty.
func CVVars(d CVData, f docFormat, hide map[string]bool) Vars {
	v := crewVars(d.Profile, f)
	if hide[CVHidePersonal] {
		for _, k := range []string{"address", "mobile_phone", "email", "passport_no", "passport_expiry", "seaman_book_no", "date_of_birth", "place_of_birth", "age", "religion", "marital_status"} {
			v.Text[k] = ""
		}
	}
	v.HTML["photo"] = ""
	if !hide[CVHidePhoto] && len(d.Photo) > 0 {
		v.HTML["photo"] = dataImage(d.Photo, d.PhotoMime, "crew-photo", "Photo of "+d.Profile.FullName)
	}
	docRows := func(views []DocumentView, authority bool) [][]string {
		rows := make([][]string, 0, len(views))
		for _, x := range views {
			row := []string{x.TypeName, x.Number, f.date(x.IssuedDate), f.date(x.ExpiredDate)}
			if authority {
				row = append(row, x.IssuingAuthority)
			}
			rows = append(rows, append(row, x.PlaceOfIssue))
		}
		return rows
	}
	v.HTML["document_table"] = hidden(hide[CVHideDocuments], func() string {
		return HTMLTable([]string{"Document", "Number", "Issued", "Expiry", "Place of issue"}, docRows(d.Documents, false), nil, "No documents.")
	})
	v.HTML["certificate_table"] = hidden(hide[CVHideCertificates], func() string {
		return HTMLTable([]string{"Certificate", "Number", "Issued", "Expiry", "Issuing authority", "Place of issue"}, docRows(d.Certificates, true), nil, "No certificates.")
	})
	v.HTML["experience_table"] = hidden(hide[CVHideExperiences], func() string {
		rows := make([][]string, 0, len(d.SeaService.Rows))
		for _, r := range d.SeaService.Rows {
			rows = append(rows, []string{r.VesselName, r.RankName, r.VesselType, r.GRTHP, r.CompanyName, r.PrincipalName,
				f.day(r.SignOnDate), f.date(r.SignOffDate), r.Duration.String()})
		}
		return HTMLTable([]string{"Vessel", "Rank", "Type", "GRT/HP", "Company", "Principal", "Sign on", "Sign off", "Duration"}, rows, nil, "No sea service recorded.")
	})
	v.HTML["sea_service_summary"] = hidden(hide[CVHideSeaService], func() string {
		rows := make([][]string, 0, len(d.SeaService.ByRank))
		for _, r := range d.SeaService.ByRank {
			rows = append(rows, []string{r.RankName, r.Total.String()})
		}
		return HTMLTable([]string{"Rank", "Sea service"}, rows, [][]string{{"Total", d.SeaService.Total.String()}}, "No sea service recorded.")
	})
	return v
}

func hidden(hide bool, build func() string) string {
	if hide {
		return ""
	}
	return build()
}

func dataImage(data []byte, mime, class, alt string) string {
	if !strings.HasPrefix(mime, "image/") || strings.Contains(mime, "svg") {
		return ""
	}
	return fmt.Sprintf(`<img class="%s" src="data:%s;base64,%s" alt="%s">`, class, mime, base64.StdEncoding.EncodeToString(data), html.EscapeString(alt))
}

// CompanyVars builds the Company group from the Settings company group.
func CompanyVars(ctx context.Context, company map[string]any, files FileReader, f docFormat) Vars {
	v := NewVars()
	str := func(k string) string {
		s, _ := company[k].(string)
		return s
	}
	for k, src := range map[string]string{
		"company_name": "legal_name", "company_address": "address", "company_phone": "phone", "company_email": "email",
		"company_website": "website", "signatory_name": "signatory_name", "signatory_title": "signatory_title",
	} {
		v.Text[k] = str(src)
	}
	v.Text["today"] = f.day(f.today)
	v.HTML["company_logo"] = ""
	if id, err := uuid.Parse(str("logo_light_file_id")); err == nil && files != nil {
		if data, mime, err := files.Content(ctx, id); err == nil {
			v.HTML["company_logo"] = dataImage(data, mime, "company-logo", str("legal_name"))
		}
	}
	return v
}
