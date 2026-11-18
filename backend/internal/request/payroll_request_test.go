package request

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePayrollLineFilter(t *testing.T) {
	q := url.Values{}
	q.Add("bank", "BANK A, TBK & CO")
	q.Add("bank", " MANDIRI ")
	q.Add("bank", "")
	q.Add("vessel", "MV. NUSA 52")
	q.Add("rank", "2E")
	q.Set("has_remark", "1")
	q.Set("no_usd", "true")
	q.Set("q", "  santoso ")
	q.Set("sort", "amount_idr")
	q.Set("order", "desc")

	f, errs := ParsePayrollLineFilter(q)
	require.Empty(t, errs)
	assert.Equal(t, []string{"BANK A, TBK & CO", "MANDIRI"}, f.Banks)
	assert.Equal(t, []string{"MV. NUSA 52"}, f.Vessels)
	assert.Equal(t, []string{"2E"}, f.Ranks)
	assert.True(t, f.HasRemark)
	assert.True(t, f.NoUSD)
	assert.Equal(t, "santoso", f.Q)
	assert.Equal(t, "amount_idr", f.Sort)
	assert.True(t, f.Desc)

	empty, errs := ParsePayrollLineFilter(url.Values{})
	require.Empty(t, errs)
	assert.False(t, empty.HasRemark)
	assert.Nil(t, empty.Banks)

	_, errs = ParsePayrollLineFilter(url.Values{"sort": {"password"}})
	assert.Equal(t, "Unknown sort column.", errs["sort"])
}

func TestPayrollPeriodRequest_Validate(t *testing.T) {
	ok := PayrollPeriod{Title: " Salary September 2026 ", PeriodStart: "2026-08-25", PeriodEnd: "2026-09-24", TransferDate: "2026-10-20"}
	require.Empty(t, ok.Validate())
	p := ok.Model()
	assert.Equal(t, "Salary September 2026", p.Title)
	assert.Equal(t, time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC), p.PeriodStart)
	require.NotNil(t, p.TransferDate)

	backwards := PayrollPeriod{Title: "x", PeriodStart: "2026-09-24", PeriodEnd: "2026-08-25"}
	assert.Equal(t, "The end date must be on or after the start date.", backwards.Validate()["period_end"])

	long := PayrollPeriod{Title: strings.Repeat("t", 121), PeriodStart: "2026-08-25", PeriodEnd: "2026-09-24"}
	assert.Contains(t, long.Validate(), "title")

	missing := PayrollPeriod{}
	errs := missing.Validate()
	assert.Contains(t, errs, "title")
	assert.Contains(t, errs, "period_start")
	assert.Contains(t, errs, "period_end")
}

func TestPayrollLineRequest_Validate(t *testing.T) {
	account := uuid.New()
	ok := PayrollLine{CrewID: "CREWAAAA2", BankAccountID: account.String(), AmountUSD: "1800.00", AmountIDR: "32100000", TransferDate: "2026-10-20", Remark: "x"}
	require.Empty(t, ok.Validate())
	in := ok.Input()
	assert.Equal(t, "CREWAAAA2", in.CrewPublicID)
	assert.Equal(t, account, in.BankAccountID)
	require.NotNil(t, in.AmountUSD)
	assert.Equal(t, "1800", in.AmountUSD.String())
	assert.Equal(t, "32100000", in.AmountIDR.String())

	noUSD := PayrollLine{CrewID: "CREWAAAA2", BankAccountID: account.String(), AmountIDR: "1"}
	require.Empty(t, noUSD.Validate())
	assert.Nil(t, noUSD.Input().AmountUSD)
	assert.Nil(t, noUSD.Input().TransferDate)

	bad := PayrollLine{AmountUSD: "-1", AmountIDR: "", Remark: strings.Repeat("r", 301)}
	errs := bad.Validate()
	for _, field := range []string{"crew_id", "bank_account_id", "amount_usd", "amount_idr", "remark"} {
		assert.Contains(t, errs, field)
	}
}
