package model

import "strings"

// NormalizeEmployeeNo is the form Employee IDs are matched and kept unique in: trimmed, upper case, without leading zeros.
func NormalizeEmployeeNo(s string) string {
	return strings.ToUpper(strings.TrimLeft(strings.TrimSpace(s), "0"))
}
