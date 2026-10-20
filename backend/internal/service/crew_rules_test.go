package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func date(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func ptr(t time.Time) *time.Time { return &t }

func TestExpiryBadgeBoundaries(t *testing.T) {
	today := date(2026, 10, 20)
	cases := []struct {
		name   string
		expiry *time.Time
		want   string
	}{
		{"no expiry", nil, BadgeGreen},
		{"expired yesterday", ptr(today.AddDate(0, 0, -1)), BadgeExpired},
		{"expires today", ptr(today), BadgeRed},
		{"30 days", ptr(today.AddDate(0, 0, 30)), BadgeRed},
		{"31 days", ptr(today.AddDate(0, 0, 31)), BadgeOrange},
		{"60 days", ptr(today.AddDate(0, 0, 60)), BadgeOrange},
		{"61 days", ptr(today.AddDate(0, 0, 61)), BadgeYellow},
		{"90 days", ptr(today.AddDate(0, 0, 90)), BadgeYellow},
		{"91 days", ptr(today.AddDate(0, 0, 91)), BadgeGreen},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ExpiryBadge(tc.expiry, today, DefaultThresholds))
		})
	}
}

func TestExpiryBadgeUsesConfiguredThresholds(t *testing.T) {
	today := date(2026, 1, 1)
	custom := Thresholds{14, 45, 120}
	assert.Equal(t, BadgeOrange, ExpiryBadge(ptr(today.AddDate(0, 0, 20)), today, custom))
	assert.Equal(t, BadgeYellow, ExpiryBadge(ptr(today.AddDate(0, 0, 100)), today, custom))
}

func TestEvaluateCompliance(t *testing.T) {
	passport, bst, coc, ktp := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	reqs := []Requirement{
		{Kind: "document", TypeID: passport, TypeName: "Passport", HasExpiry: true},
		{Kind: "document", TypeID: ktp, TypeName: "KTP", HasExpiry: false},
		{Kind: "certificate", TypeID: bst, TypeName: "BST", HasExpiry: true},
		{Kind: "certificate", TypeID: coc, TypeName: "COC", HasExpiry: true},
	}
	today := date(2026, 10, 20)

	t.Run("all valid or expiring is compliant", func(t *testing.T) {
		held := []HeldItem{
			{Kind: "document", TypeID: passport, ExpiredDate: ptr(date(2030, 1, 1))},
			{Kind: "document", TypeID: ktp, ExpiredDate: ptr(date(2000, 1, 1))},
			{Kind: "certificate", TypeID: bst, ExpiredDate: ptr(today.AddDate(0, 0, 10))},
			{Kind: "certificate", TypeID: coc, ExpiredDate: ptr(date(2029, 1, 1))},
		}
		res := EvaluateCompliance(reqs, held, today, DefaultThresholds)
		assert.True(t, res.Compliant)
		assert.Equal(t, ItemValid, res.Items[0].Status)
		assert.Equal(t, ItemValid, res.Items[1].Status, "a type without expiry is valid once present")
		assert.Equal(t, ItemExpiring, res.Items[2].Status)
		assert.Equal(t, BadgeRed, res.Items[2].Badge)
	})

	t.Run("expired mandatory certificate makes crew not compliant", func(t *testing.T) {
		held := []HeldItem{
			{Kind: "document", TypeID: passport, ExpiredDate: ptr(date(2030, 1, 1))},
			{Kind: "document", TypeID: ktp},
			{Kind: "certificate", TypeID: bst, ExpiredDate: ptr(date(2030, 1, 1))},
			{Kind: "certificate", TypeID: coc, ExpiredDate: ptr(today.AddDate(0, 0, -1))},
		}
		res := EvaluateCompliance(reqs, held, today, DefaultThresholds)
		assert.False(t, res.Compliant)
		assert.Equal(t, ItemExpired, res.Items[3].Status)
	})

	t.Run("missing item", func(t *testing.T) {
		res := EvaluateCompliance(reqs, nil, today, DefaultThresholds)
		assert.False(t, res.Compliant)
		for _, it := range res.Items {
			assert.Equal(t, ItemMissing, it.Status)
		}
	})

	t.Run("latest expiry wins when several items of one type exist", func(t *testing.T) {
		held := []HeldItem{
			{Kind: "document", TypeID: passport, Number: "OLD", ExpiredDate: ptr(date(2020, 1, 1))},
			{Kind: "document", TypeID: passport, Number: "NEW", ExpiredDate: ptr(date(2031, 1, 1))},
		}
		res := EvaluateCompliance(reqs[:1], held, today, DefaultThresholds)
		assert.True(t, res.Compliant)
		assert.Equal(t, "NEW", res.Items[0].Number)
	})

	t.Run("check against a future relief date", func(t *testing.T) {
		held := []HeldItem{{Kind: "document", TypeID: passport, ExpiredDate: ptr(date(2026, 12, 1))}}
		assert.True(t, EvaluateCompliance(reqs[:1], held, today, DefaultThresholds).Compliant)
		assert.False(t, EvaluateCompliance(reqs[:1], held, date(2027, 1, 15), DefaultThresholds).Compliant)
	})

	t.Run("no requirements is compliant", func(t *testing.T) {
		assert.True(t, EvaluateCompliance(nil, nil, today, DefaultThresholds).Compliant)
	})
}

func TestNewSpan(t *testing.T) {
	cases := []struct {
		days int
		want string
	}{
		{0, "0 y 0 m 0 d"},
		{29, "0 y 0 m 29 d"},
		{30, "0 y 1 m 0 d"},
		{365, "1 y 0 m 0 d"},
		{365*3 + 30*4 + 12, "3 y 4 m 12 d"},
		{-5, "0 y 0 m 0 d"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, NewSpan(tc.days).String(), tc.days)
	}
}

func TestRowDaysIsInclusive(t *testing.T) {
	today := date(2026, 10, 20)
	assert.Equal(t, 1, RowDays(Period{SignOn: today, SignOff: ptr(today)}, today))
	assert.Equal(t, 31, RowDays(Period{SignOn: date(2026, 1, 1), SignOff: ptr(date(2026, 1, 31))}, today))
	assert.Equal(t, 20, RowDays(Period{SignOn: date(2026, 10, 1)}, today), "open row counts to today")
}

func TestSeaServiceDoesNotDoubleCountOverlaps(t *testing.T) {
	master, chief := uuid.New(), uuid.New()
	today := date(2026, 10, 20)
	periods := []Period{
		{RankID: master, SignOn: date(2025, 1, 1), SignOff: ptr(date(2025, 1, 31))},
		{RankID: master, SignOn: date(2025, 1, 15), SignOff: ptr(date(2025, 2, 14))},
		{RankID: master, SignOn: date(2025, 2, 1), SignOff: ptr(date(2025, 2, 10))},
		{RankID: chief, SignOn: date(2024, 1, 1), SignOff: ptr(date(2024, 1, 10))},
		{RankID: chief, SignOn: date(2025, 1, 20), SignOff: ptr(date(2025, 1, 29))},
	}
	sum := SummarizeSeaService(periods, today)
	assert.Equal(t, 45, sum.ByRank[master].TotalDays, "1 Jan to 14 Feb 2025")
	assert.Equal(t, 20, sum.ByRank[chief].TotalDays)
	assert.Equal(t, 55, sum.Total.TotalDays, "chief days inside the master period are not counted twice")
}

func TestSeaServiceAdjacentAndOpenPeriods(t *testing.T) {
	r := uuid.New()
	today := date(2026, 10, 20)
	periods := []Period{
		{RankID: r, SignOn: date(2026, 1, 1), SignOff: ptr(date(2026, 1, 10))},
		{RankID: r, SignOn: date(2026, 1, 11), SignOff: ptr(date(2026, 1, 20))},
		{RankID: r, SignOn: date(2026, 10, 11)},
	}
	assert.Equal(t, 30, SummarizeSeaService(periods, today).ByRank[r].TotalDays)
}

func TestManualStatusChange(t *testing.T) {
	cases := []struct {
		name      string
		action    string
		reason    string
		active    bool
		want      string
		errorCode string
	}{
		{"medical leave", ActionMedicalLeave, "", false, StatusMedicalLeave, ""},
		{"ready", ActionReady, "", false, StatusAvailable, ""},
		{"blacklist with reason", ActionBlacklist, "Misconduct", false, StatusBlacklisted, ""},
		{"blacklist needs reason", ActionBlacklist, "  ", false, "", "validation_failed"},
		{"unknown action", "on_board", "", false, "", "validation_failed"},
		{"blocked while on board", ActionMedicalLeave, "", true, "", "crew_on_board"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ManualStatusChange(tc.action, tc.reason, tc.active)
			if tc.errorCode != "" {
				requireAppErr(t, err, tc.errorCode)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.Code)
		})
	}
}

func TestOnlyAvailableCrewCanBeAssigned(t *testing.T) {
	assert.True(t, CanBeAssigned(StatusAvailable))
	for _, s := range []string{StatusOnBoard, StatusSignOff, StatusMedicalLeave, StatusBlacklisted} {
		assert.False(t, CanBeAssigned(s), s)
	}
}
