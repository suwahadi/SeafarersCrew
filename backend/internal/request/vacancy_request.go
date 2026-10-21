package request

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// Vacancy limits.
const (
	MaxVacancyDocuments    = 20
	maxVacancyDocumentName = 100
	maxVacancyHTML         = 100_000
)

var vacancyDepartments = []string{model.DepartmentDeck, model.DepartmentEngine, model.DepartmentCateringOthers}

// Vacancy is the body of POST /vacancies and PUT /vacancies/{id}.
type Vacancy struct {
	Title             string   `json:"title"`
	Slug              string   `json:"slug"`
	RankID            string   `json:"rank_id"`
	VesselTypeID      string   `json:"vessel_type_id"`
	Department        string   `json:"department"`
	Location          string   `json:"location"`
	EmploymentType    string   `json:"employment_type"`
	Duration          string   `json:"duration"`
	SalaryText        string   `json:"salary_text"`
	DescriptionHTML   string   `json:"description_html"`
	RequirementsHTML  string   `json:"requirements_html"`
	RequiredDocuments []string `json:"required_documents"`
	ClosingDate       string   `json:"closing_date"`
	Status            string   `json:"status"`

	parsed model.Vacancy
}

// Validate implements Validator.
func (r *Vacancy) Validate() map[string]string {
	Trim(&r.Title, &r.Slug, &r.RankID, &r.VesselTypeID, &r.Department, &r.Location, &r.EmploymentType, &r.Duration, &r.SalaryText, &r.ClosingDate, &r.Status)
	if r.EmploymentType == "" {
		r.EmploymentType = model.EmploymentContract
	}
	if r.Status == "" {
		r.Status = model.VacancyDraft
	}
	e := Errors{}
	v := model.Vacancy{
		Title: r.Title, Slug: r.Slug, Department: r.Department, Location: r.Location, EmploymentType: r.EmploymentType,
		Duration: r.Duration, SalaryText: r.SalaryText, DescriptionHTML: r.DescriptionHTML, RequirementsHTML: r.RequirementsHTML,
		Status: r.Status,
	}
	e.Required("title", r.Title, 200)
	e.MaxLen("slug", r.Slug, 80)
	v.RankID = e.RequiredUUID("rank_id", r.RankID)
	v.VesselTypeID = e.UUID("vessel_type_id", r.VesselTypeID)
	if r.Department == "" {
		e["department"] = "This field is required."
	}
	e.OneOf("department", r.Department, vacancyDepartments...)
	e.OneOf("employment_type", r.EmploymentType, model.EmploymentContract, model.EmploymentPermanent)
	e.OneOf("status", r.Status, model.VacancyDraft, model.VacancyOpen, model.VacancyClosed)
	e.MaxLen("location", r.Location, 120)
	e.MaxLen("duration", r.Duration, 80)
	e.MaxLen("salary_text", r.SalaryText, 120)
	if len(r.DescriptionHTML) > maxVacancyHTML {
		e["description_html"] = "The description is too large."
	}
	if len(r.RequirementsHTML) > maxVacancyHTML {
		e["requirements_html"] = "The requirements are too large."
	}
	v.ClosingDate = e.Date("closing_date", r.ClosingDate)
	v.RequiredDocuments = requiredDocuments(e, r.RequiredDocuments)
	r.parsed = v
	return e
}

func requiredDocuments(e Errors, in []string) []string {
	out := make([]string, 0, len(in))
	for _, d := range in {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		out = append(out, d)
		e.MaxLen("required_documents", d, maxVacancyDocumentName)
	}
	if len(out) > MaxVacancyDocuments {
		e["required_documents"] = fmt.Sprintf("At most %d required documents are allowed.", MaxVacancyDocuments)
	}
	return out
}

// Model returns the validated vacancy.
func (r *Vacancy) Model() model.Vacancy { return r.parsed }

// VacancyStatus is the body of PATCH /vacancies/{id}/status.
type VacancyStatus struct {
	Status string `json:"status"`
}

// Validate implements Validator.
func (r *VacancyStatus) Validate() map[string]string {
	r.Status = strings.TrimSpace(r.Status)
	e := Errors{}
	if r.Status == "" {
		e["status"] = "This field is required."
	}
	e.OneOf("status", r.Status, model.VacancyDraft, model.VacancyOpen, model.VacancyClosed)
	return e
}

// ParseVacancyFilter reads admin vacancy list filters.
func ParseVacancyFilter(q url.Values) (model.VacancyFilter, Errors) {
	e := Errors{}
	p := ParseList(q)
	f := model.VacancyFilter{Q: p.Q, Sort: p.Sort, Desc: p.Desc, Statuses: CSV(q, "status"), Departments: CSV(q, "department")}
	for _, st := range f.Statuses {
		e.OneOf("status", st, model.VacancyDraft, model.VacancyOpen, model.VacancyClosed)
	}
	for _, d := range f.Departments {
		e.OneOf("department", d, vacancyDepartments...)
	}
	f.RankIDs = uuids(e, q, "rank_id")
	f.VesselTypeIDs = uuids(e, q, "vessel_type_id")
	return f, e
}

// Public vacancy list paging.
const (
	PublicDefaultPageSize = 12
	PublicMaxPageSize     = 48
)

// ParsePublicVacancyQuery reads the career page search: q, rank (rank id), vessel_type (vessel type id), department, location, page, per_page.
func ParsePublicVacancyQuery(q url.Values) (model.PublicVacancyQuery, Errors) {
	e := Errors{}
	out := model.PublicVacancyQuery{
		Q:          truncateRunes(strings.TrimSpace(q.Get("q")), 100),
		Department: strings.TrimSpace(q.Get("department")),
		Location:   truncateRunes(strings.TrimSpace(q.Get("location")), 120),
		Page:       1,
		PerPage:    PublicDefaultPageSize,
	}
	e.OneOf("department", out.Department, vacancyDepartments...)
	out.RankID = e.UUID("rank", strings.TrimSpace(q.Get("rank")))
	out.VesselTypeID = e.UUID("vessel_type", strings.TrimSpace(q.Get("vessel_type")))
	if n, err := strconv.Atoi(q.Get("page")); err == nil && n > 0 && n <= 100_000 {
		out.Page = n
	}
	if n, err := strconv.Atoi(q.Get("per_page")); err == nil && n > 0 {
		out.PerPage = min(n, PublicMaxPageSize)
	}
	return out, e
}

func truncateRunes(s string, max int) string {
	if r := []rune(s); len(r) > max {
		return string(r[:max])
	}
	return s
}
