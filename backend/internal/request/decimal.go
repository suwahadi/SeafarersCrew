package request

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

// DecimalInput accepts a JSON number or string holding a decimal amount.
type DecimalInput string

// UnmarshalJSON implements json.Unmarshaler.
func (d *DecimalInput) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if bytes.Equal(b, []byte("null")) {
		*d = ""
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return fmt.Errorf("decode decimal: %w", err)
		}
		*d = DecimalInput(s)
		return nil
	}
	*d = DecimalInput(b)
	return nil
}

var maxAmount = decimal.New(1, 15)

// Decimal parses an optional amount between min and 10^15 with at most scale fractional digits.
func (e Errors) Decimal(field string, raw DecimalInput, min decimal.Decimal, scale int32) *decimal.Decimal {
	if raw == "" {
		return nil
	}
	v, err := decimal.NewFromString(string(raw))
	if err != nil {
		e.set(field, "This must be a number.")
		return nil
	}
	if v.LessThan(min) || v.GreaterThanOrEqual(maxAmount) {
		e.set(field, "This must be between "+min.String()+" and "+maxAmount.String()+".")
		return nil
	}
	if !v.Equal(v.Round(scale)) {
		e.set(field, fmt.Sprintf("This may have at most %d decimal places.", scale))
		return nil
	}
	return &v
}

// RequiredDecimal parses a mandatory amount.
func (e Errors) RequiredDecimal(field string, raw DecimalInput, min decimal.Decimal, scale int32) decimal.Decimal {
	if raw == "" {
		e.set(field, "This field is required.")
		return decimal.Zero
	}
	v := e.Decimal(field, raw, min, scale)
	if v == nil {
		return decimal.Zero
	}
	return *v
}
