package model

import (
	"time"

	"github.com/google/uuid"
)

// CountByKey is a labelled count used by dashboard charts.
type CountByKey struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Color string `json:"color,omitempty"`
	Group string `json:"group,omitempty"`
	Count int    `json:"count"`
}

// ExpiryBuckets counts documents and certificates by days to expiry.
type ExpiryBuckets struct {
	Expired int `json:"expired"`
	Within1 int `json:"within_1"`
	Within2 int `json:"within_2"`
	Within3 int `json:"within_3"`
}

// DashboardSummary holds the KPI cards of PRD 6.1.
type DashboardSummary struct {
	TotalCrew          int           `json:"total_crew"`
	ByStatus           []CountByKey  `json:"by_status"`
	ActiveContracts    int           `json:"active_contracts"`
	Documents          ExpiryBuckets `json:"documents"`
	Certificates       ExpiryBuckets `json:"certificates"`
	DueForRelief       int           `json:"due_for_relief"`
	NewApplicants      int           `json:"new_applicants"`
	UnreadMessages     int           `json:"unread_messages"`
	Thresholds         [3]int        `json:"thresholds"`
	LastBackupStatus   string        `json:"last_backup_status,omitempty"`
	LastBackupFinished *time.Time    `json:"last_backup_finished_at,omitempty"`
}

// MonthlyMovement counts sign-ons and sign-offs in one month.
type MonthlyMovement struct {
	Month    string `json:"month"`
	SignOns  int    `json:"sign_ons"`
	SignOffs int    `json:"sign_offs"`
}

// DashboardCharts holds the chart series of PRD 6.1.
type DashboardCharts struct {
	ByStatus    []CountByKey      `json:"by_status"`
	ByRank      []CountByKey      `json:"by_rank"`
	ByPrincipal []CountByKey      `json:"on_board_by_principal"`
	Movements   []MonthlyMovement `json:"movements"`
}

// ExpiringItem is one row of the expiring documents and certificates table.
type ExpiringItem struct {
	Kind        string    `json:"kind"`
	CrewID      string    `json:"crew_id"`
	CrewName    string    `json:"crew_name"`
	RankName    string    `json:"rank_name"`
	TypeName    string    `json:"type_name"`
	Number      string    `json:"number"`
	ExpiredDate time.Time `json:"expired_date"`
}

// UpcomingRelief is one row of the upcoming reliefs table.
type UpcomingRelief struct {
	ContractID        string     `json:"contract_id"`
	CrewID            string     `json:"crew_id"`
	CrewName          string     `json:"crew_name"`
	RankName          string     `json:"rank_name"`
	VesselName        string     `json:"vessel_name"`
	PrincipalName     string     `json:"principal_name"`
	PlannedSignOff    time.Time  `json:"planned_sign_off"`
	ReliefPlanID      *uuid.UUID `json:"relief_plan_id"`
	PlannedReliefDate *time.Time `json:"planned_relief_date"`
	ReliefStatus      string     `json:"relief_status,omitempty"`
	RelieverID        string     `json:"reliever_id,omitempty"`
	RelieverName      string     `json:"reliever_name,omitempty"`
}
