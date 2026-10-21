package model

import (
	"time"

	"github.com/google/uuid"
)

// Vacancy statuses.
const (
	VacancyDraft  = "draft"
	VacancyOpen   = "open"
	VacancyClosed = "closed"
)

// Vacancy departments.
const (
	DepartmentDeck           = "deck"
	DepartmentEngine         = "engine"
	DepartmentCateringOthers = "catering_other"
)

// Vacancy employment types.
const (
	EmploymentContract  = "contract"
	EmploymentPermanent = "permanent"
)

// Applicant pipeline stages.
const (
	StageNew       = "new"
	StageScreening = "screening"
	StageInterview = "interview"
	StageAccepted  = "accepted"
	StageRejected  = "rejected"
)

// ApplicantStages lists the pipeline stages in board order.
var ApplicantStages = []string{StageNew, StageScreening, StageInterview, StageAccepted, StageRejected}

// Applicant file kinds.
const (
	ApplicantFileCV       = "cv"
	ApplicantFileDocument = "document"
)

// VacancyPathPrefix is the public path under which vacancies are published.
const VacancyPathPrefix = "/career/"

// Vacancy is a job vacancy shown on the career page.
type Vacancy struct {
	ID                   uuid.UUID
	PublicID             string
	Title                string
	Slug                 string
	RankID               uuid.UUID
	RankName             string
	VesselTypeID         *uuid.UUID
	VesselTypeName       string
	Department           string
	Location             string
	EmploymentType       string
	Duration             string
	SalaryText           string
	DescriptionHTML      string
	RequirementsHTML     string
	RequiredDocuments    []string
	ClosingDate          *time.Time
	Status               string
	PublishedAt          *time.Time
	ApplicationsCount    int
	NewApplicationsCount int
	IsExpired            bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// VacancyFilter narrows the admin vacancy list.
type VacancyFilter struct {
	Q             string
	Statuses      []string
	RankIDs       []uuid.UUID
	VesselTypeIDs []uuid.UUID
	Departments   []string
	Sort          string
	Desc          bool
}

// PublicVacancyFilter narrows the public vacancy list; Today is the first date that still counts as open.
type PublicVacancyFilter struct {
	Q            string
	RankID       *uuid.UUID
	VesselTypeID *uuid.UUID
	Department   string
	Location     string
	Today        time.Time
}

// PublicVacancyQuery is a career page search with its pagination.
type PublicVacancyQuery struct {
	Q            string
	RankID       *uuid.UUID
	VesselTypeID *uuid.UUID
	Department   string
	Location     string
	Page         int
	PerPage      int
}

// VacancyFilterOptions are the filter values used by open vacancies.
type VacancyFilterOptions struct {
	Ranks       []Option
	Departments []string
	VesselTypes []Option
	Locations   []string
}

// Option is an id and display name pair.
type Option struct {
	ID        uuid.UUID
	Name      string
	ShortCode string
}

// Applicant is a public job application.
type Applicant struct {
	ID              uuid.UUID
	PublicID        string
	VacancyID       uuid.UUID
	VacancyPublicID string
	VacancyTitle    string
	VacancySlug     string
	VacancyStatus   string
	FullName        string
	Email           string
	Phone           string
	DateOfBirth     *time.Time
	Nationality     string
	RankAppliedID   *uuid.UUID
	RankAppliedName string
	YearsExperience *int
	Message         string
	Stage           string
	ConsentAt       time.Time
	IPAddress       string
	FilesCount      int
	Files           []ApplicantFile
	StageLogs       []StageLog
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ApplicantFile links an uploaded file to an applicant.
type ApplicantFile struct {
	FileID       uuid.UUID
	Kind         string
	OriginalName string
	StorageKey   string
	MimeType     string
	Size         int64
	CreatedAt    time.Time
}

// StageLog records one pipeline move.
type StageLog struct {
	ID        uuid.UUID
	FromStage string
	ToStage   string
	Note      string
	AdminID   *uuid.UUID
	AdminName string
	CreatedAt time.Time
}

// ApplicantFilter narrows the applicant list.
type ApplicantFilter struct {
	Q             string
	VacancyPublic string
	Stages        []string
	RankIDs       []uuid.UUID
	DateFrom      *time.Time
	DateTo        *time.Time
	CreatedFrom   *time.Time
	CreatedBefore *time.Time
	Sort          string
	Desc          bool
}

// NewApplication is a validated public application ready to store.
type NewApplication struct {
	ID              uuid.UUID
	PublicID        string
	VacancyID       uuid.UUID
	FullName        string
	Email           string
	Phone           string
	DateOfBirth     time.Time
	Nationality     string
	RankAppliedID   uuid.UUID
	YearsExperience int
	Message         string
	ConsentAt       time.Time
	IPAddress       string
	Files           []NewApplicantFile
	DuplicateSince  time.Time
}

// NewApplicantFile is an uploaded file to link to a new application.
type NewApplicantFile struct {
	FileID uuid.UUID
	Kind   string
}

// ContactMessage is a message sent through the public contact form.
type ContactMessage struct {
	ID         uuid.UUID
	Name       string
	Email      string
	Phone      string
	Subject    string
	Message    string
	IPAddress  string
	IsRead     bool
	IsArchived bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// Contact inbox status filters.
const (
	ContactAll      = "all"
	ContactUnread   = "unread"
	ContactRead     = "read"
	ContactArchived = "archived"
)

// ContactFilter narrows the contact inbox list.
type ContactFilter struct {
	Q      string
	Status string
}

// ExpiredApplicant is an application past its retention period together with its stored files.
type ExpiredApplicant struct {
	ID    uuid.UUID
	Files []File
}
