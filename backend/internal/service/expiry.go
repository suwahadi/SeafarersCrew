package service

import (
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// Expiry badges from PRD 6.3.
const (
	BadgeExpired = "expired"
	BadgeRed     = "red"
	BadgeOrange  = "orange"
	BadgeYellow  = "yellow"
	BadgeGreen   = "green"
)

// Thresholds are the expiry alert boundaries in days, strictly increasing.
type Thresholds [3]int

// DefaultThresholds are 30, 60 and 90 days.
var DefaultThresholds = Thresholds{30, 60, 90}

// ExpiryBadge classifies an expiry date against the check date.
func ExpiryBadge(expiry *time.Time, checkDate time.Time, t Thresholds) string {
	if expiry == nil {
		return BadgeGreen
	}
	days := datetime.DaysBetween(checkDate, *expiry)
	switch {
	case days < 0:
		return BadgeExpired
	case days <= t[0]:
		return BadgeRed
	case days <= t[1]:
		return BadgeOrange
	case days <= t[2]:
		return BadgeYellow
	default:
		return BadgeGreen
	}
}

// DaysToExpiry returns the days from checkDate to expiry, or nil when there is no expiry.
func DaysToExpiry(expiry *time.Time, checkDate time.Time) *int {
	if expiry == nil {
		return nil
	}
	d := datetime.DaysBetween(checkDate, *expiry)
	return &d
}
