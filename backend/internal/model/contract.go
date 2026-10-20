package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Contract statuses (PRD 6.4).
const (
	ContractDraft      = "draft"
	ContractActive     = "active"
	ContractCompleted  = "completed"
	ContractTerminated = "terminated"
)

// Allowance calculation bases.
const (
	BasisFixedMonth   = "fixed_month"
	BasisFixedDay     = "fixed_day"
	BasisPercentBasic = "percent_basic"
)

// Contract links one crew member to one vessel, rank and principal for a period.
type Contract struct {
	ID                 uuid.UUID
	PublicID           string
	CrewID             uuid.UUID
	VesselID           uuid.UUID
	RankID             uuid.UUID
	PrincipalID        uuid.UUID
	EmployerCompanyID  uuid.UUID
	SignOnDate         time.Time
	DurationMonths     int
	PlannedSignOffDate time.Time
	ActualSignOffDate  *time.Time
	PointOfHire        string
	SignOnPort         string
	SignOffPort        string
	CurrencyCode       string
	BasicSalary        decimal.Decimal
	WorkingHoursWeek   *int
	LeaveDaysMonth     *decimal.Decimal
	ProbationMonths    *int
	Status             string
	TerminationReason  string
	Notes              string
	Allowances         []ContractAllowance
	CreatedAt          time.Time
	UpdatedAt          time.Time

	CrewPublicID     string
	CrewName         string
	CrewStatusCode   string
	VesselName       string
	VesselTypeID     *uuid.UUID
	VesselTypeName   string
	VesselGRT        *decimal.Decimal
	VesselHPKW       string
	RankName         string
	PrincipalName    string
	EmployerName     string
	CurrencyDecimals int
	TotalMonthlyPay  decimal.Decimal
}

// ContractAllowance is one allowance row with its computed monthly amount.
type ContractAllowance struct {
	ID              uuid.UUID
	AllowanceTypeID uuid.UUID
	AllowanceType   string
	Basis           string
	Value           decimal.Decimal
	MonthlyAmount   decimal.Decimal
	SortOrder       int
}

// ContractFilter holds the contract list search, filters and sort.
type ContractFilter struct {
	Q             string
	Statuses      []string
	VesselIDs     []uuid.UUID
	PrincipalIDs  []uuid.UUID
	RankIDs       []uuid.UUID
	CrewPublicID  string
	SignOnFrom    *time.Time
	SignOnTo      *time.Time
	SignOffBefore *time.Time
	Sort          string
	Desc          bool
}

// Relief plan statuses (PRD 6.5).
const (
	ReliefPlanned   = "planned"
	ReliefConfirmed = "confirmed"
	ReliefCompleted = "completed"
	ReliefCancelled = "cancelled"
)

// ReliefPlan assigns a reliever to an active contract.
type ReliefPlan struct {
	ID                uuid.UUID
	ContractID        uuid.UUID
	RelieverCrewID    uuid.UUID
	PlannedReliefDate time.Time
	JoiningPort       string
	Status            string
	Notes             string
	NewContractID     *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time

	ContractPublicID    string
	ContractStatus      string
	ContractDuration    int
	RankID              uuid.UUID
	RankName            string
	VesselID            uuid.UUID
	VesselName          string
	RelievedCrewID      uuid.UUID
	RelievedPublicID    string
	RelievedName        string
	RelieverPublicID    string
	RelieverName        string
	NewContractPublicID string
}

// ReliefPlanFilter holds the relief plan list filters.
type ReliefPlanFilter struct {
	Statuses         []string
	ContractPublicID string
	RelieverPublicID string
	VesselIDs        []uuid.UUID
	DateFrom         *time.Time
	DateTo           *time.Time
}

// ReliefWindow is a reliever commitment from the relief date for the relieved contract's duration.
type ReliefWindow struct {
	PlanID         uuid.UUID
	ReliefDate     time.Time
	DurationMonths int
}

// ReliefCandidate is one reliever suggestion.
type ReliefCandidate struct {
	CrewID         uuid.UUID
	PublicID       string
	FullName       string
	Nationality    string
	DateOfBirth    *time.Time
	MobilePhone    string
	SeaServiceDays int
}

// TimelineBar is one active contract on a vessel lane.
type TimelineBar struct {
	ContractID         uuid.UUID
	ContractPublicID   string
	VesselID           uuid.UUID
	VesselName         string
	CrewPublicID       string
	CrewName           string
	RankName           string
	SignOnDate         time.Time
	PlannedSignOffDate time.Time
}

// TimelineVessel is one vessel lane.
type TimelineVessel struct {
	ID   uuid.UUID
	Name string
	Type string
}
