// Package datetime handles date-only values and display formatting in the configured timezone.
package datetime

import (
	"fmt"
	"strings"
	"time"
)

// DateLayout is the ISO layout used for date-only values in the API.
const DateLayout = "2006-01-02"

// DefaultTimezone is the timezone used when Settings has none.
const DefaultTimezone = "Asia/Jakarta"

// DefaultDisplayFormat is the default display pattern from Settings.
const DefaultDisplayFormat = "DD-MMM-YYYY"

var displayTokens = strings.NewReplacer(
	"YYYY", "2006",
	"MMM", "Jan",
	"MM", "01",
	"DD", "02",
	"HH", "15",
	"mm", "04",
)

// GoLayout translates a display pattern such as DD-MMM-YYYY into a Go time layout.
func GoLayout(pattern string) string {
	return displayTokens.Replace(pattern)
}

// FormatDate renders a date-only value using a display pattern.
func FormatDate(t time.Time, pattern string) string {
	return t.Format(GoLayout(pattern))
}

// FormatDateTime renders a UTC timestamp in loc using the pattern followed by HH:mm.
func FormatDateTime(t time.Time, loc *time.Location, pattern string) string {
	return t.In(loc).Format(GoLayout(pattern + " HH:mm"))
}

// ParseDate parses a YYYY-MM-DD string into a UTC midnight time.
func ParseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation(DateLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date %q: %w", s, err)
	}
	return t, nil
}

// Today returns the current calendar date in loc as a UTC midnight time.
func Today(now time.Time, loc *time.Location) time.Time {
	y, m, d := now.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// DaysBetween returns the whole number of calendar days from a to b.
func DaysBetween(a, b time.Time) int {
	ad := time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
	bd := time.Date(b.Year(), b.Month(), b.Day(), 0, 0, 0, 0, time.UTC)
	return int(bd.Sub(ad).Hours() / 24)
}

// AddMonths adds months to a date, clamping to the last day of the target month.
func AddMonths(t time.Time, months int) time.Time {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, months, 0)
	lastDay := first.AddDate(0, 1, -1).Day()
	day := t.Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC)
}

// Age returns the completed years between dob and today, or nil when dob is unknown.
func Age(dob *time.Time, today time.Time) *int {
	if dob == nil {
		return nil
	}
	years := today.Year() - dob.Year()
	if today.Month() < dob.Month() || (today.Month() == dob.Month() && today.Day() < dob.Day()) {
		years--
	}
	return &years
}
