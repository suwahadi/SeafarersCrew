package resource

import (
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

func vesselTypeRef(v model.Vacancy) *Ref {
	if v.VesselTypeID == nil {
		return nil
	}
	return &Ref{ID: *v.VesselTypeID, Name: v.VesselTypeName}
}

// Vacancy is the admin view of a vacancy.
type Vacancy struct {
	ID                   string     `json:"id"`
	Title                string     `json:"title"`
	Slug                 string     `json:"slug"`
	Rank                 Ref        `json:"rank"`
	VesselType           *Ref       `json:"vessel_type"`
	Department           string     `json:"department"`
	Location             string     `json:"location"`
	EmploymentType       string     `json:"employment_type"`
	Duration             string     `json:"duration"`
	SalaryText           string     `json:"salary_text"`
	DescriptionHTML      string     `json:"description_html"`
	RequirementsHTML     string     `json:"requirements_html"`
	RequiredDocuments    []string   `json:"required_documents"`
	ClosingDate          *Date      `json:"closing_date"`
	Status               string     `json:"status"`
	IsExpired            bool       `json:"is_expired"`
	PublishedAt          *time.Time `json:"published_at"`
	ApplicationsCount    int        `json:"applications_count"`
	NewApplicationsCount int        `json:"new_applications_count"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// NewVacancy maps a vacancy for the admin.
func NewVacancy(v model.Vacancy) Vacancy {
	return Vacancy{
		ID: v.PublicID, Title: v.Title, Slug: v.Slug, Rank: Ref{ID: v.RankID, Name: v.RankName}, VesselType: vesselTypeRef(v),
		Department: v.Department, Location: v.Location, EmploymentType: v.EmploymentType, Duration: v.Duration, SalaryText: v.SalaryText,
		DescriptionHTML: v.DescriptionHTML, RequirementsHTML: v.RequirementsHTML, RequiredDocuments: nonNil(v.RequiredDocuments),
		ClosingDate: NewDate(v.ClosingDate), Status: v.Status, IsExpired: v.IsExpired, PublishedAt: v.PublishedAt,
		ApplicationsCount: v.ApplicationsCount, NewApplicationsCount: v.NewApplicationsCount, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
}

// NewVacancies maps a page of vacancies for the admin.
func NewVacancies(items []model.Vacancy) []Vacancy {
	out := make([]Vacancy, len(items))
	for i, v := range items {
		out[i] = NewVacancy(v)
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// PublicVacancyCard is a vacancy on the career list.
type PublicVacancyCard struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Rank           Ref        `json:"rank"`
	VesselType     *Ref       `json:"vessel_type"`
	Department     string     `json:"department"`
	Location       string     `json:"location"`
	EmploymentType string     `json:"employment_type"`
	ClosingDate    *Date      `json:"closing_date"`
	PublishedAt    *time.Time `json:"published_at"`
}

// NewPublicVacancyCards maps a page of open vacancies for the website.
func NewPublicVacancyCards(items []model.Vacancy) []PublicVacancyCard {
	out := make([]PublicVacancyCard, len(items))
	for i, v := range items {
		out[i] = newPublicVacancyCard(v)
	}
	return out
}

func newPublicVacancyCard(v model.Vacancy) PublicVacancyCard {
	return PublicVacancyCard{
		ID: v.PublicID, Title: v.Title, Slug: v.Slug, Rank: Ref{ID: v.RankID, Name: v.RankName}, VesselType: vesselTypeRef(v),
		Department: v.Department, Location: v.Location, EmploymentType: v.EmploymentType, ClosingDate: NewDate(v.ClosingDate), PublishedAt: v.PublishedAt,
	}
}

// PublicVacancy is the vacancy detail for the website.
type PublicVacancy struct {
	PublicVacancyCard
	Duration          string    `json:"duration"`
	SalaryText        string    `json:"salary_text"`
	DescriptionHTML   string    `json:"description_html"`
	RequirementsHTML  string    `json:"requirements_html"`
	RequiredDocuments []string  `json:"required_documents"`
	IsOpen            bool      `json:"is_open"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// NewPublicVacancy maps a vacancy detail for the website.
func NewPublicVacancy(v service.PublicVacancy) PublicVacancy {
	return PublicVacancy{
		PublicVacancyCard: newPublicVacancyCard(v.Vacancy), Duration: v.Duration, SalaryText: v.SalaryText,
		DescriptionHTML: v.DescriptionHTML, RequirementsHTML: v.RequirementsHTML, RequiredDocuments: nonNil(v.RequiredDocuments),
		IsOpen: v.IsOpen, UpdatedAt: v.UpdatedAt,
	}
}

// FilterRank is a rank option of the career filters.
type FilterRank struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	ShortCode string    `json:"short_code"`
}

// PublicVacancyFilters are the filter values used by open vacancies.
type PublicVacancyFilters struct {
	Ranks       []FilterRank `json:"ranks"`
	Departments []string     `json:"departments"`
	VesselTypes []Ref        `json:"vessel_types"`
	Locations   []string     `json:"locations"`
}

// NewPublicVacancyFilters maps the filter options for the website.
func NewPublicVacancyFilters(o model.VacancyFilterOptions) PublicVacancyFilters {
	out := PublicVacancyFilters{
		Ranks: make([]FilterRank, len(o.Ranks)), Departments: nonNil(o.Departments), VesselTypes: make([]Ref, len(o.VesselTypes)),
		Locations: nonNil(o.Locations),
	}
	for i, r := range o.Ranks {
		out.Ranks[i] = FilterRank{ID: r.ID, Name: r.Name, ShortCode: r.ShortCode}
	}
	for i, t := range o.VesselTypes {
		out.VesselTypes[i] = Ref{ID: t.ID, Name: t.Name}
	}
	return out
}

// ApplicantVacancy is the vacancy summary on an applicant.
type ApplicantVacancy struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

// Applicant is an applicant row of the list.
type Applicant struct {
	ID              string           `json:"id"`
	Vacancy         ApplicantVacancy `json:"vacancy"`
	FullName        string           `json:"full_name"`
	Email           string           `json:"email"`
	Phone           string           `json:"phone"`
	Nationality     string           `json:"nationality"`
	RankApplied     *Ref             `json:"rank_applied"`
	YearsExperience *int             `json:"years_experience"`
	Stage           string           `json:"stage"`
	FilesCount      int              `json:"files_count"`
	CreatedAt       time.Time        `json:"created_at"`
}

func newApplicantVacancy(a model.Applicant) ApplicantVacancy {
	return ApplicantVacancy{ID: a.VacancyPublicID, Title: a.VacancyTitle, Slug: a.VacancySlug, Status: a.VacancyStatus}
}

func rankAppliedRef(a model.Applicant) *Ref {
	if a.RankAppliedID == nil {
		return nil
	}
	return &Ref{ID: *a.RankAppliedID, Name: a.RankAppliedName}
}

// NewApplicants maps a page of applicants.
func NewApplicants(items []model.Applicant) []Applicant {
	out := make([]Applicant, len(items))
	for i, a := range items {
		out[i] = Applicant{
			ID: a.PublicID, Vacancy: newApplicantVacancy(a), FullName: a.FullName, Email: a.Email, Phone: a.Phone,
			Nationality: a.Nationality, RankApplied: rankAppliedRef(a), YearsExperience: a.YearsExperience, Stage: a.Stage,
			FilesCount: a.FilesCount, CreatedAt: a.CreatedAt,
		}
	}
	return out
}

// ApplicantFile is a file of an applicant; download it through GET /files/{file_id}/url.
type ApplicantFile struct {
	FileID    uuid.UUID `json:"file_id"`
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
}

// StageLogEntry is one pipeline move.
type StageLogEntry struct {
	ID        uuid.UUID  `json:"id"`
	FromStage *string    `json:"from_stage"`
	ToStage   string     `json:"to_stage"`
	Note      string     `json:"note"`
	Admin     *AdminName `json:"admin"`
	CreatedAt time.Time  `json:"created_at"`
}

// AdminName is the admin who made a change.
type AdminName struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ApplicantDetail is the full applicant record.
type ApplicantDetail struct {
	Applicant
	DateOfBirth *Date           `json:"date_of_birth"`
	Message     string          `json:"message"`
	ConsentAt   time.Time       `json:"consent_at"`
	IPAddress   string          `json:"ip_address"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Files       []ApplicantFile `json:"files"`
	StageLogs   []StageLogEntry `json:"stage_logs"`
}

// NewApplicantDetail maps a full applicant record.
func NewApplicantDetail(a model.Applicant) ApplicantDetail {
	out := ApplicantDetail{
		Applicant:   NewApplicants([]model.Applicant{a})[0],
		DateOfBirth: NewDate(a.DateOfBirth), Message: a.Message, ConsentAt: a.ConsentAt, IPAddress: a.IPAddress, UpdatedAt: a.UpdatedAt,
		Files: make([]ApplicantFile, len(a.Files)), StageLogs: make([]StageLogEntry, len(a.StageLogs)),
	}
	out.FilesCount = len(a.Files)
	for i, f := range a.Files {
		name := service.DownloadName(model.File{OriginalName: f.OriginalName, StorageKey: f.StorageKey})
		out.Files[i] = ApplicantFile{FileID: f.FileID, Kind: f.Kind, Name: name, Size: f.Size, MimeType: f.MimeType, CreatedAt: f.CreatedAt}
	}
	for i, l := range a.StageLogs {
		entry := StageLogEntry{ID: l.ID, ToStage: l.ToStage, Note: l.Note, CreatedAt: l.CreatedAt}
		if l.FromStage != "" {
			from := l.FromStage
			entry.FromStage = &from
		}
		if l.AdminID != nil {
			entry.Admin = &AdminName{ID: *l.AdminID, Name: l.AdminName}
		}
		out.StageLogs[i] = entry
	}
	return out
}

// ContactMessage is a contact inbox message.
type ContactMessage struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Phone      string    `json:"phone"`
	Subject    string    `json:"subject"`
	Message    string    `json:"message"`
	IPAddress  string    `json:"ip_address"`
	IsRead     bool      `json:"is_read"`
	IsArchived bool      `json:"is_archived"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewContactMessage maps one message.
func NewContactMessage(m model.ContactMessage) ContactMessage {
	return ContactMessage{ID: m.ID, Name: m.Name, Email: m.Email, Phone: m.Phone, Subject: m.Subject, Message: m.Message,
		IPAddress: m.IPAddress, IsRead: m.IsRead, IsArchived: m.IsArchived, CreatedAt: m.CreatedAt}
}

// NewContactMessages maps a page of messages.
func NewContactMessages(items []model.ContactMessage) []ContactMessage {
	out := make([]ContactMessage, len(items))
	for i, m := range items {
		out[i] = NewContactMessage(m)
	}
	return out
}
