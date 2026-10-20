package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// Period is one sea service row; a nil SignOff means still on board.
type Period struct {
	RankID  uuid.UUID
	SignOn  time.Time
	SignOff *time.Time
}

// Span is a duration expressed in days with its years, months, days breakdown.
type Span struct {
	TotalDays int `json:"total_days"`
	Years     int `json:"years"`
	Months    int `json:"months"`
	Days      int `json:"days"`
}

// String renders the span as "3 y 4 m 12 d".
func (s Span) String() string {
	return fmt.Sprintf("%d y %d m %d d", s.Years, s.Months, s.Days)
}

// NewSpan splits total days using 365-day years and 30-day months (PRD 11.5).
func NewSpan(total int) Span {
	if total < 0 {
		total = 0
	}
	years := total / 365
	rest := total % 365
	return Span{TotalDays: total, Years: years, Months: rest / 30, Days: rest % 30}
}

// RowDays returns sign-off minus sign-on plus one day; open rows count to today.
func RowDays(p Period, today time.Time) int {
	end := today
	if p.SignOff != nil {
		end = *p.SignOff
	}
	if end.Before(p.SignOn) {
		return 0
	}
	return datetime.DaysBetween(p.SignOn, end) + 1
}

// MergedDays returns the number of distinct calendar days covered by periods.
func MergedDays(periods []Period, today time.Time) int {
	type rng struct{ start, end time.Time }
	ranges := make([]rng, 0, len(periods))
	for _, p := range periods {
		end := today
		if p.SignOff != nil {
			end = *p.SignOff
		}
		if end.Before(p.SignOn) {
			continue
		}
		ranges = append(ranges, rng{p.SignOn, end})
	}
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start.Before(ranges[j].start) })
	total := 0
	var cur *rng
	for i := range ranges {
		r := ranges[i]
		if cur == nil {
			cur = &r
			continue
		}
		if !r.start.After(cur.end.AddDate(0, 0, 1)) {
			if r.end.After(cur.end) {
				cur.end = r.end
			}
			continue
		}
		total += datetime.DaysBetween(cur.start, cur.end) + 1
		cur = &r
	}
	if cur != nil {
		total += datetime.DaysBetween(cur.start, cur.end) + 1
	}
	return total
}

// SeaServiceSummary holds totals per rank and overall.
type SeaServiceSummary struct {
	Total  Span               `json:"total"`
	ByRank map[uuid.UUID]Span `json:"by_rank"`
}

// SummarizeSeaService merges overlapping periods per rank and overall before summing.
func SummarizeSeaService(periods []Period, today time.Time) SeaServiceSummary {
	byRank := map[uuid.UUID][]Period{}
	for _, p := range periods {
		byRank[p.RankID] = append(byRank[p.RankID], p)
	}
	out := SeaServiceSummary{Total: NewSpan(MergedDays(periods, today)), ByRank: make(map[uuid.UUID]Span, len(byRank))}
	for rank, ps := range byRank {
		out.ByRank[rank] = NewSpan(MergedDays(ps, today))
	}
	return out
}
