package request

import (
	"net/url"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// ApplicantStage is the body of PATCH /applicants/{id}/stage.
type ApplicantStage struct {
	Stage string `json:"stage"`
	Note  string `json:"note"`
}

// ParseApplicantFilter reads applicant list filters: q, vacancy (Vacancy ID), stage, rank_id, date_from, date_to.
func ParseApplicantFilter(q url.Values) (model.ApplicantFilter, Errors) {
	e := Errors{}
	p := ParseList(q)
	f := model.ApplicantFilter{
		Q: p.Q, Sort: p.Sort, Desc: p.Desc, VacancyPublic: strings.ToUpper(strings.TrimSpace(q.Get("vacancy"))),
		Stages: CSV(q, "stage"),
	}
	for _, st := range f.Stages {
		e.OneOf("stage", st, model.ApplicantStages...)
	}
	f.RankIDs = uuids(e, q, "rank_id")
	f.DateFrom = e.Date("date_from", q.Get("date_from"))
	f.DateTo = e.Date("date_to", q.Get("date_to"))
	e.After("date_to", f.DateFrom, f.DateTo, "The end date must be on or after the start date.")
	return f, e
}

// ContactPatch is the body of PATCH /contact-messages/{id}.
type ContactPatch struct {
	IsRead     *bool `json:"is_read"`
	IsArchived *bool `json:"is_archived"`
}

// ParseContactFilter reads inbox filters: q and status (all, unread, read, archived; default all).
func ParseContactFilter(q url.Values) (model.ContactFilter, Errors) {
	e := Errors{}
	f := model.ContactFilter{Q: ParseList(q).Q, Status: strings.TrimSpace(q.Get("status"))}
	if f.Status == "" {
		f.Status = model.ContactAll
	}
	e.OneOf("status", f.Status, model.ContactAll, model.ContactUnread, model.ContactRead, model.ContactArchived)
	return f, e
}
