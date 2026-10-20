package datetime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func jakarta(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(DefaultTimezone)
	require.NoError(t, err)
	return loc
}

func TestFormatDateUsesDefaultPattern(t *testing.T) {
	assert.Equal(t, "20-Oct-2026", FormatDate(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), DefaultDisplayFormat))
}

func TestFormatDateTimeConvertsUTCToWIB(t *testing.T) {
	ts := time.Date(2026, 10, 20, 18, 30, 0, 0, time.UTC)
	assert.Equal(t, "21-Oct-2026 01:30", FormatDateTime(ts, jakarta(t), DefaultDisplayFormat))
}

func TestTodayUsesConfiguredTimezone(t *testing.T) {
	// 20:00 UTC is already the next day in WIB (UTC+7).
	now := time.Date(2026, 10, 20, 20, 0, 0, 0, time.UTC)
	assert.Equal(t, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), Today(now, jakarta(t)))
}

func TestParseDate(t *testing.T) {
	got, err := ParseDate("2026-02-28")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), got)

	_, err = ParseDate("28-02-2026")
	assert.Error(t, err)
}

func TestDaysBetween(t *testing.T) {
	a := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 0, DaysBetween(a, a))
	assert.Equal(t, 31, DaysBetween(a, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, -1, DaysBetween(a, time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)))
}

func TestAddMonthsClampsToMonthEnd(t *testing.T) {
	cases := []struct {
		from   time.Time
		months int
		want   time.Time
	}{
		{time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), 6, time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 1, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)},
		{time.Date(2027, 11, 30, 0, 0, 0, 0, time.UTC), 3, time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, AddMonths(tc.from, tc.months))
	}
}

func TestAge(t *testing.T) {
	dob := time.Date(1990, 10, 21, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 35, *Age(&dob, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)), "birthday tomorrow")
	assert.Equal(t, 36, *Age(&dob, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)))
	leap := time.Date(2000, 2, 29, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 25, *Age(&leap, time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)))
	assert.Nil(t, Age(nil, time.Now()))
}
