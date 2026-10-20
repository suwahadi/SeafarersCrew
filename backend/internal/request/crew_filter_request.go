package request

import (
	"net/url"
	"slices"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// ParseCrewFilter reads datagrid filters from a query string (PRD 6.9).
func ParseCrewFilter(q url.Values) (model.CrewFilter, Errors) {
	e := Errors{}
	list := ParseList(q)
	f := model.CrewFilter{
		Q:             list.Q,
		Sort:          list.Sort,
		Desc:          list.Desc,
		StatusCodes:   CSV(q, "status"),
		Departments:   CSV(q, "department"),
		Nationalities: CSV(q, "nationality"),
		Compliance:    q.Get("compliance"),
		DocValidity:   q.Get("doc_validity"),
		GroupBy:       q.Get("group_by"),
		AgeMin:        Int(q, "age_min"),
		AgeMax:        Int(q, "age_max"),
	}
	f.RankIDs = uuids(e, q, "rank_id")
	f.VesselTypeIDs = uuids(e, q, "vessel_type_id")
	f.VesselIDs = uuids(e, q, "vessel_id")
	f.PrincipalIDs = uuids(e, q, "principal_id")
	f.SeaRankID = e.UUID("sea_rank_id", q.Get("sea_rank_id"))
	if n := Int(q, "sea_min_days"); n != nil {
		f.SeaMinDays = *n
	}
	if n := Int(q, "doc_days"); n != nil {
		f.DocDays = *n
	}
	e.OneOf("compliance", f.Compliance, "compliant", "not_compliant")
	e.OneOf("doc_validity", f.DocValidity, "valid", "expiring", "expired")
	e.OneOf("group_by", f.GroupBy, "principal", "company")
	e.Range("age_min", f.AgeMin, 0, 100)
	e.Range("age_max", f.AgeMax, 0, 100)
	e.Range("doc_days", &f.DocDays, 0, 3650)
	e.Range("sea_min_days", &f.SeaMinDays, 0, 36500)
	for _, d := range f.Departments {
		if !slices.Contains([]string{"deck", "engine", "catering_other"}, d) {
			e["department"] = "Unknown department."
		}
	}
	return f, e
}

func uuids(e Errors, q url.Values, key string) []uuid.UUID {
	raw := CSV(q, key)
	out := make([]uuid.UUID, 0, len(raw))
	for _, v := range raw {
		id, err := uuid.Parse(v)
		if err != nil {
			e[key] = "Contains an invalid identifier."
			continue
		}
		out = append(out, id)
	}
	return out
}
