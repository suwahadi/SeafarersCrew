package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestMonthlyAllowance(t *testing.T) {
	cases := []struct {
		name     string
		basis    string
		value    string
		basic    string
		decimals int
		want     string
	}{
		{"fixed per month is the value", model.BasisFixedMonth, "250.50", "5000", 2, "250.5"},
		{"fixed per day times 30", model.BasisFixedDay, "45.25", "5000", 2, "1357.5"},
		{"percentage of basic", model.BasisPercentBasic, "12.5", "4800", 2, "600"},
		{"percentage rounds to currency decimals", model.BasisPercentBasic, "33.333", "1000", 2, "333.33"},
		{"IDR rounds to whole rupiah", model.BasisPercentBasic, "10", "12345678", 0, "1234568"},
		{"IDR daily allowance", model.BasisFixedDay, "150000", "0", 0, "4500000"},
		{"zero value", model.BasisFixedMonth, "0", "5000", 2, "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MonthlyAllowance(tc.basis, dec(tc.value), dec(tc.basic), tc.decimals)
			assert.True(t, dec(tc.want).Equal(got), "got %s", got)
		})
	}
}

func TestTotalMonthlyPaySumsInContractCurrency(t *testing.T) {
	total := TotalMonthlyPay(dec("5000"), []model.ContractAllowance{{MonthlyAmount: dec("600")}, {MonthlyAmount: dec("1350.25")}})
	assert.True(t, dec("6950.25").Equal(total))
	assert.True(t, dec("5000").Equal(TotalMonthlyPay(dec("5000"), nil)))
}

func TestPlannedSignOffAddsDurationMonths(t *testing.T) {
	assert.Equal(t, date(2027, 4, 15), PlannedSignOff(date(2026, 10, 15), 6))
	assert.Equal(t, date(2027, 2, 28), PlannedSignOff(date(2026, 8, 31), 6), "clamps to month end")
}

func TestCheckActivation(t *testing.T) {
	cases := []struct {
		name      string
		contract  string
		crew      string
		hasActive bool
		code      string
	}{
		{"draft and available", model.ContractDraft, StatusAvailable, false, ""},
		{"already active", model.ContractActive, StatusAvailable, false, "contract_not_draft"},
		{"completed", model.ContractCompleted, StatusAvailable, false, "contract_not_draft"},
		{"crew holds another active contract", model.ContractDraft, StatusOnBoard, true, "crew_has_active_contract"},
		{"crew on medical leave", model.ContractDraft, StatusMedicalLeave, false, "crew_not_available"},
		{"crew signed off", model.ContractDraft, StatusSignOff, false, "crew_not_available"},
		{"crew blacklisted", model.ContractDraft, StatusBlacklisted, false, "crew_not_available"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckActivation(tc.contract, tc.crew, tc.hasActive)
			if tc.code == "" {
				assert.NoError(t, err)
				return
			}
			requireAppErr(t, err, tc.code)
		})
	}
}

func TestCheckFinish(t *testing.T) {
	active := model.Contract{Status: model.ContractActive, SignOnDate: date(2026, 1, 10)}
	cases := []struct {
		name      string
		contract  model.Contract
		actual    time.Time
		reason    string
		terminate bool
		code      string
		field     string
	}{
		{"complete on sign-on date", active, date(2026, 1, 10), "", false, "", ""},
		{"sign-off before sign-on", active, date(2026, 1, 9), "", false, "validation_failed", "actual_sign_off_date"},
		{"terminate needs a reason", active, date(2026, 3, 1), "", true, "validation_failed", "reason"},
		{"terminate with reason", active, date(2026, 3, 1), "Medical repatriation", true, "", ""},
		{"draft cannot finish", model.Contract{Status: model.ContractDraft}, date(2026, 3, 1), "", false, "contract_not_active", ""},
		{"completed cannot finish twice", model.Contract{Status: model.ContractCompleted}, date(2026, 3, 1), "", false, "contract_not_active", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckFinish(tc.contract, tc.actual, tc.reason, tc.terminate)
			if tc.code == "" {
				assert.NoError(t, err)
				return
			}
			e := requireAppErr(t, err, tc.code)
			if tc.field != "" {
				assert.Contains(t, e.Fields, tc.field)
			}
		})
	}
}

func TestLockedChangesAllowsOnlyActiveEditableFields(t *testing.T) {
	hours := 48
	cur := model.Contract{CrewID: uuid.New(), VesselID: uuid.New(), SignOnDate: date(2026, 1, 1), DurationMonths: 6,
		PlannedSignOffDate: date(2026, 7, 1), BasicSalary: dec("5000"), CurrencyCode: "USD", WorkingHoursWeek: &hours,
		Allowances: []model.ContractAllowance{{AllowanceTypeID: uuid.New(), Value: dec("100")}}}

	editable := cur
	editable.PlannedSignOffDate = date(2026, 8, 1)
	editable.SignOnPort, editable.SignOffPort, editable.Notes = "Batam", "Singapore", "Extended by principal"
	editable.BasicSalary = dec("5000.00")
	assert.Empty(t, LockedChanges(cur, editable))

	locked := cur
	locked.BasicSalary = dec("5100")
	locked.VesselID = uuid.New()
	other := 40
	locked.WorkingHoursWeek = &other
	locked.Allowances = []model.ContractAllowance{{AllowanceTypeID: cur.Allowances[0].AllowanceTypeID, Value: dec("150")}}
	assert.ElementsMatch(t, []string{"basic_salary", "vessel_id", "working_hours_week", "allowances"}, LockedChanges(cur, locked))
}

func TestReliefWindowsOverlap(t *testing.T) {
	base := model.ReliefWindow{ReliefDate: date(2026, 11, 1), DurationMonths: 3}
	cases := []struct {
		name  string
		other model.ReliefWindow
		want  bool
	}{
		{"same start", model.ReliefWindow{ReliefDate: date(2026, 11, 1), DurationMonths: 6}, true},
		{"starts inside", model.ReliefWindow{ReliefDate: date(2027, 1, 15), DurationMonths: 6}, true},
		{"starts on the last committed day", model.ReliefWindow{ReliefDate: date(2027, 2, 1), DurationMonths: 6}, true},
		{"starts the day after", model.ReliefWindow{ReliefDate: date(2027, 2, 2), DurationMonths: 6}, false},
		{"ends before", model.ReliefWindow{ReliefDate: date(2026, 5, 1), DurationMonths: 5}, false},
		{"ends on the first day", model.ReliefWindow{ReliefDate: date(2026, 5, 1), DurationMonths: 6}, true},
		{"unknown duration uses six months", model.ReliefWindow{ReliefDate: date(2026, 6, 1)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ReliefWindowsOverlap(base, tc.other))
			assert.Equal(t, tc.want, ReliefWindowsOverlap(tc.other, base), "symmetric")
		})
	}
}

func TestBarState(t *testing.T) {
	today := date(2026, 10, 20)
	assert.Equal(t, BarOverdue, BarState(date(2026, 10, 19), today))
	assert.Equal(t, BarDueSoon, BarState(today, today))
	assert.Equal(t, BarDueSoon, BarState(date(2026, 11, 19), today))
	assert.Equal(t, BarNormal, BarState(date(2026, 11, 20), today))
}

func TestReliefWarnings(t *testing.T) {
	passport, bst, coc, sat := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exp := date(2027, 1, 1)
	atStart := ComplianceResult{Items: []ComplianceItem{
		{TypeID: passport, Status: ItemValid}, {TypeID: bst, Status: ItemExpired}, {TypeID: coc, Status: ItemMissing}, {TypeID: sat, Status: ItemValid},
	}}
	atEnd := ComplianceResult{Items: []ComplianceItem{
		{Kind: "document", TypeID: passport, TypeName: "Passport", Status: ItemExpired, ExpiredDate: &exp},
		{Kind: "certificate", TypeID: bst, TypeName: "BST", Status: ItemExpired},
		{Kind: "certificate", TypeID: coc, TypeName: "COC", Status: ItemMissing},
		{Kind: "certificate", TypeID: sat, TypeName: "SAT", Status: ItemValid},
	}}
	got := ReliefWarnings(atStart, atEnd)
	require.Len(t, got, 3)
	assert.Equal(t, ReliefWarning{Kind: "document", TypeName: "Passport", ExpiredDate: &exp, Reason: WarnExpiresBeforeEnd}, got[0])
	assert.Equal(t, WarnExpiredOnRelief, got[1].Reason)
	assert.Equal(t, WarnMissing, got[2].Reason)
	assert.Empty(t, ReliefWarnings(ComplianceResult{}, ComplianceResult{Items: []ComplianceItem{{Status: ItemExpiring}}}))
}

func TestBuildLanesKeepsVesselOrderAndAttachesPlans(t *testing.T) {
	v1, v2, ghost := uuid.New(), uuid.New(), uuid.New()
	k1, k2 := uuid.New(), uuid.New()
	today := date(2026, 10, 20)
	lanes := BuildLanes(
		[]model.TimelineVessel{{ID: v1, Name: "MV Alpha"}, {ID: v2, Name: "MV Beta"}},
		[]model.TimelineBar{
			{ContractID: k1, VesselID: v2, PlannedSignOffDate: date(2026, 10, 1)},
			{ContractID: k2, VesselID: ghost, VesselName: "MV Laid Up", PlannedSignOffDate: date(2027, 6, 1)},
		},
		[]model.ReliefPlan{{ID: uuid.New(), ContractID: k1, RelieverName: "Reliever", PlannedReliefDate: date(2026, 10, 25)}},
		today)
	require.Len(t, lanes, 3)
	assert.Empty(t, lanes[0].Bars, "vessels without crew keep their lane")
	require.Len(t, lanes[1].Bars, 1)
	assert.Equal(t, BarOverdue, lanes[1].Bars[0].State)
	assert.Len(t, lanes[1].Bars[0].Plans, 1)
	assert.Equal(t, "MV Laid Up", lanes[2].Vessel.Name)
	assert.Equal(t, BarNormal, lanes[2].Bars[0].State)
	assert.NotNil(t, lanes[2].Bars[0].Plans)
}

func TestContractExperienceCopiesTheContract(t *testing.T) {
	grt := dec("1234.6")
	vt := uuid.New()
	k := model.Contract{ID: uuid.New(), CrewID: uuid.New(), VesselID: uuid.New(), VesselName: "MV Ocean", VesselTypeID: &vt,
		VesselGRT: &grt, VesselHPKW: "5150 HP", RankID: uuid.New(), EmployerCompanyID: uuid.New(), EmployerName: "PT Crew",
		PrincipalID: uuid.New(), PrincipalName: "Alpha Offshore", SignOnDate: date(2026, 1, 1)}
	x := ContractExperience(k, date(2026, 6, 30), "Medical")
	assert.Equal(t, "GRT 1,235 / 5150 HP", x.GRTHP)
	assert.Equal(t, "contract", x.Source)
	assert.Equal(t, "sign_off", x.Status)
	assert.Equal(t, k.ID, *x.ContractID)
	assert.Equal(t, k.EmployerCompanyID, *x.CompanyID)
	assert.Equal(t, "Alpha Offshore", x.PrincipalName)
	assert.Equal(t, date(2026, 6, 30), *x.SignOffDate)
	assert.Equal(t, "Medical", x.Remarks)
}
