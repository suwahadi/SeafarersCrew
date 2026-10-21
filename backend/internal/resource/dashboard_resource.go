package resource

import (
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// DashboardSummary is the KPI payload with the date it was computed for.
type DashboardSummary struct {
	model.DashboardSummary
	Today Date `json:"today"`
}

// ExpiringItem is one row of the expiring table.
type ExpiringItem struct {
	Kind        string `json:"kind"`
	CrewID      string `json:"crew_id"`
	CrewName    string `json:"crew_name"`
	RankName    string `json:"rank_name"`
	TypeName    string `json:"type_name"`
	Number      string `json:"number"`
	ExpiredDate Date   `json:"expired_date"`
	DaysLeft    int    `json:"days_left"`
	Badge       string `json:"badge"`
}

// UpcomingRelief is one row of the reliefs table.
type UpcomingRelief struct {
	ContractID        string     `json:"contract_id"`
	CrewID            string     `json:"crew_id"`
	CrewName          string     `json:"crew_name"`
	RankName          string     `json:"rank_name"`
	VesselName        string     `json:"vessel_name"`
	PrincipalName     string     `json:"principal_name"`
	PlannedSignOff    Date       `json:"planned_sign_off"`
	ReliefPlanID      *uuid.UUID `json:"relief_plan_id"`
	PlannedReliefDate *Date      `json:"planned_relief_date"`
	ReliefStatus      string     `json:"relief_status"`
	RelieverID        string     `json:"reliever_id"`
	RelieverName      string     `json:"reliever_name"`
	DueDate           Date       `json:"due_date"`
	DaysLeft          int        `json:"days_left"`
	State             string     `json:"state"`
}

// NewDashboardSummary maps the KPI payload.
func NewDashboardSummary(s model.DashboardSummary, dc service.DashboardContext) DashboardSummary {
	return DashboardSummary{DashboardSummary: s, Today: Date(dc.Today)}
}

// NewExpiringItems maps expiring rows with badges.
func NewExpiringItems(in []model.ExpiringItem, dc service.DashboardContext) []ExpiringItem {
	out := make([]ExpiringItem, len(in))
	for i, it := range in {
		exp := it.ExpiredDate
		out[i] = ExpiringItem{Kind: it.Kind, CrewID: it.CrewID, CrewName: it.CrewName, RankName: it.RankName, TypeName: it.TypeName,
			Number: it.Number, ExpiredDate: Date(exp), DaysLeft: datetime.DaysBetween(dc.Today, exp), Badge: service.ExpiryBadge(&exp, dc.Today, dc.Thresholds)}
	}
	return out
}

// NewUpcomingReliefs maps relief rows; state is overdue, due_soon or planned.
func NewUpcomingReliefs(in []model.UpcomingRelief, dc service.DashboardContext) []UpcomingRelief {
	out := make([]UpcomingRelief, len(in))
	for i, u := range in {
		due := u.PlannedSignOff
		if u.PlannedReliefDate != nil {
			due = *u.PlannedReliefDate
		}
		days := datetime.DaysBetween(dc.Today, due)
		out[i] = UpcomingRelief{ContractID: u.ContractID, CrewID: u.CrewID, CrewName: u.CrewName, RankName: u.RankName,
			VesselName: u.VesselName, PrincipalName: u.PrincipalName, PlannedSignOff: Date(u.PlannedSignOff), ReliefPlanID: u.ReliefPlanID,
			PlannedReliefDate: NewDate(u.PlannedReliefDate), ReliefStatus: u.ReliefStatus, RelieverID: u.RelieverID, RelieverName: u.RelieverName,
			DueDate: Date(due), DaysLeft: days, State: reliefState(days, u.ReliefPlanID != nil)}
	}
	return out
}

func reliefState(days int, planned bool) string {
	switch {
	case days < 0:
		return "overdue"
	case planned:
		return "planned"
	default:
		return "due_soon"
	}
}
