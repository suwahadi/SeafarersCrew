// Package money formats and totals amounts per currency without conversion.
package money

import (
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

// Amount is a decimal value tagged with its ISO 4217 currency code.
type Amount struct {
	Currency string
	Value    decimal.Decimal
}

// Format renders value with thousands separators and the given decimal places, e.g. 45,000.00.
func Format(value decimal.Decimal, decimals int32) string {
	fixed := value.Abs().StringFixed(decimals)
	intPart, fracPart, hasFrac := strings.Cut(fixed, ".")
	var b strings.Builder
	if value.Round(decimals).IsNegative() {
		b.WriteByte('-')
	}
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if hasFrac {
		b.WriteByte('.')
		b.WriteString(fracPart)
	}
	return b.String()
}

// FormatWithCode renders an amount prefixed by its currency code, e.g. USD 45,000.00.
func FormatWithCode(currency string, value decimal.Decimal, decimals int32) string {
	return currency + " " + Format(value, decimals)
}

// TotalsByCurrency sums amounts grouped by currency and returns them sorted by currency code.
func TotalsByCurrency(amounts []Amount) []Amount {
	sums := make(map[string]decimal.Decimal)
	for _, a := range amounts {
		sums[a.Currency] = sums[a.Currency].Add(a.Value)
	}
	out := make([]Amount, 0, len(sums))
	for code, v := range sums {
		out = append(out, Amount{Currency: code, Value: v})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Currency < out[j].Currency })
	return out
}

// JoinTotals renders grouped totals as "USD 45,000.00 | IDR 120,000,000" using each currency's decimals.
func JoinTotals(totals []Amount, decimals func(currency string) int32) string {
	parts := make([]string, 0, len(totals))
	for _, t := range totals {
		parts = append(parts, FormatWithCode(t.Currency, t.Value, decimals(t.Currency)))
	}
	return strings.Join(parts, " | ")
}
