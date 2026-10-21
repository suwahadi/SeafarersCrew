package repository_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func TestDashboardAggregates(t *testing.T) {
	w := setupCrewWorld(t)
	truncate(t, "applicants", "vacancies", "contact_messages", "backups")
	ctx := context.Background()
	crews := repository.NewCrewRepository(testDB)
	dash := repository.NewDashboardRepository(testDB)

	onBoard := createCrew(t, crews, "DASHBOARD", "On Board Crew", &w.master, nil, "")
	available := createCrew(t, crews, "DASHAVAIL", "Available Crew", &w.ab, nil, "")
	saveDoc := func(kind repository.DocumentKind, crewID any, typ any, number string, exp string) {
		exec(t, `INSERT INTO `+kind.Table+` (crew_id, `+kind.TypeColumn+`, number, expired_date) VALUES ($1, $2, $3, $4)`, crewID, typ, number, exp)
	}
	saveDoc(repository.TravelDocuments, onBoard, w.passport, "OLD", "2020-01-01")
	saveDoc(repository.TravelDocuments, onBoard, w.passport, "NEW", "2026-11-10")
	saveDoc(repository.Certificates, onBoard, w.bst, "BST", "2026-10-01")
	saveDoc(repository.TravelDocuments, available, w.passport, "AV1", "2027-01-05")
	saveDoc(repository.TravelDocuments, available, w.seamanBook, "SB1", "2026-12-10")

	exec(t, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id, sign_on_date,
		duration_months, planned_sign_off_date, currency_code, basic_salary, status)
		VALUES ('DASHCONT2', $1, $2, $3, $4, $5, '2026-05-01', 6, '2026-11-01', 'USD', 5000, 'active')`, onBoard, w.vessel, w.master, w.principalA, w.principalB)
	exec(t, `UPDATE crews SET crew_status_id = (SELECT id FROM crew_statuses WHERE code = 'on_board') WHERE id = $1`, onBoard)
	exec(t, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id, sign_on_date,
		duration_months, planned_sign_off_date, actual_sign_off_date, currency_code, basic_salary, status)
		VALUES ('DASHOLD22', $1, $2, $3, $4, $5, '2026-01-10', 3, '2026-04-10', '2026-04-12', 'USD', 5000, 'completed')`, available, w.vessel, w.ab, w.principalA, w.principalB)
	exec(t, `INSERT INTO contact_messages (name, email, subject, message) VALUES ('A', 'a@example.com', 'Hi', 'Hello'), ('B', 'b@example.com', 'Hi', 'Read')`)
	exec(t, `UPDATE contact_messages SET is_read = true WHERE name = 'B'`)
	exec(t, `INSERT INTO backups (status, finished_at) VALUES ('failed', now())`)

	today := d(2026, 10, 20)
	s, err := dash.Summary(ctx, today, [3]int{30, 60, 90})
	require.NoError(t, err)
	assert.Equal(t, 2, s.TotalCrew)
	counts := map[string]int{}
	for _, b := range s.ByStatus {
		counts[b.Key] = b.Count
	}
	assert.Equal(t, 1, counts["on_board"])
	assert.Equal(t, 1, counts["available"])
	assert.Equal(t, 1, s.ActiveContracts)
	assert.Equal(t, 1, s.DueForRelief, "planned sign-off 01-Nov is within 30 days")
	assert.Equal(t, 1, s.UnreadMessages)
	assert.Equal(t, model.ExpiryBuckets{Expired: 0, Within1: 1, Within2: 1, Within3: 1}, s.Documents,
		"the renewed passport hides the old one; buckets 21, 51 and 77 days")
	assert.Equal(t, model.ExpiryBuckets{Expired: 1}, s.Certificates)
	assert.Equal(t, "failed", s.LastBackupStatus)

	ch, err := dash.Charts(ctx, today)
	require.NoError(t, err)
	require.Len(t, ch.Movements, 12)
	assert.Equal(t, "2025-11", ch.Movements[0].Month)
	assert.Equal(t, "2026-10", ch.Movements[11].Month)
	byMonth := map[string]model.MonthlyMovement{}
	for _, m := range ch.Movements {
		byMonth[m.Month] = m
	}
	assert.Equal(t, 1, byMonth["2026-05"].SignOns)
	assert.Equal(t, 1, byMonth["2026-01"].SignOns)
	assert.Equal(t, 1, byMonth["2026-04"].SignOffs)
	require.Len(t, ch.ByPrincipal, 1)
	assert.Equal(t, "Alpha Offshore", ch.ByPrincipal[0].Label)
	ranks := map[string]int{}
	for _, r := range ch.ByRank {
		ranks[r.Label] = r.Count
		assert.Equal(t, "deck", r.Group)
	}
	assert.Equal(t, map[string]int{"Master": 1, "AB": 1}, ranks)

	exp, err := dash.Expiring(ctx, today, 90, 50)
	require.NoError(t, err)
	require.Len(t, exp, 4)
	assert.Equal(t, "BST", exp[0].Number, "expired first")
	assert.Equal(t, "NEW", exp[1].Number)
	assert.Equal(t, "Master", exp[1].RankName, "rank of the active contract")

	rel, err := dash.Reliefs(ctx, today, 30, 50)
	require.NoError(t, err)
	require.Len(t, rel, 1)
	assert.Equal(t, "DASHCONT2", rel[0].ContractID)
	assert.Nil(t, rel[0].ReliefPlanID)

	exec(t, `INSERT INTO relief_plans (contract_id, reliever_crew_id, planned_relief_date, status)
		SELECT id, $1, '2026-10-25', 'planned' FROM contracts WHERE public_id = 'DASHCONT2'`, available)
	rel, err = dash.Reliefs(ctx, today, 30, 50)
	require.NoError(t, err)
	require.Len(t, rel, 1)
	assert.Equal(t, "DASHAVAIL", rel[0].RelieverID)
	assert.Equal(t, d(2026, 10, 25), *rel[0].PlannedReliefDate)
}
