package money

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestFormat(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		decimals int32
		want     string
	}{
		{"two decimals with thousands", "45000", 2, "45,000.00"},
		{"zero decimals for IDR", "120000000", 0, "120,000,000"},
		{"rounds half up", "1234.565", 2, "1,234.57"},
		{"small value", "7.5", 2, "7.50"},
		{"zero", "0", 2, "0.00"},
		{"negative", "-1500.25", 2, "-1,500.25"},
		{"negative rounding to zero", "-0.001", 2, "0.00"},
		{"exactly three digits", "999", 0, "999"},
		{"one million", "1000000", 2, "1,000,000.00"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Format(d(tc.value), tc.decimals))
		})
	}
}

func TestFormatWithCode(t *testing.T) {
	assert.Equal(t, "USD 45,000.00", FormatWithCode("USD", d("45000"), 2))
}

func TestTotalsByCurrencyNeverConverts(t *testing.T) {
	totals := TotalsByCurrency([]Amount{
		{Currency: "USD", Value: d("40000")},
		{Currency: "IDR", Value: d("100000000")},
		{Currency: "USD", Value: d("5000")},
		{Currency: "IDR", Value: d("20000000")},
	})
	assert.Equal(t, []Amount{
		{Currency: "IDR", Value: d("120000000")},
		{Currency: "USD", Value: d("45000")},
	}, normalize(totals))
}

func TestJoinTotals(t *testing.T) {
	decimals := func(code string) int32 {
		if code == "IDR" {
			return 0
		}
		return 2
	}
	got := JoinTotals([]Amount{
		{Currency: "USD", Value: d("45000")},
		{Currency: "IDR", Value: d("120000000")},
	}, decimals)
	assert.Equal(t, "USD 45,000.00 | IDR 120,000,000", got)
}

func normalize(in []Amount) []Amount {
	out := make([]Amount, len(in))
	for i, a := range in {
		out[i] = Amount{Currency: a.Currency, Value: d(a.Value.String())}
	}
	return out
}
