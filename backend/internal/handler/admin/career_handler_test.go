package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/middleware"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type stubVacancyService struct {
	created  model.Vacancy
	updated  model.Vacancy
	filter   model.VacancyFilter
	status   string
	deleted  string
	page     int
	perPage  int
	getError error
}

func (s *stubVacancyService) Create(_ context.Context, in model.Vacancy) (model.Vacancy, error) {
	s.created = in
	in.PublicID = "VAC222222"
	return in, nil
}

func (s *stubVacancyService) Get(_ context.Context, id string) (model.Vacancy, error) {
	return model.Vacancy{PublicID: id, Title: "Master", Slug: "master"}, s.getError
}

func (s *stubVacancyService) List(_ context.Context, f model.VacancyFilter, page, perPage int) (repository.Page[model.Vacancy], error) {
	s.filter, s.page, s.perPage = f, page, perPage
	return repository.Page[model.Vacancy]{Items: []model.Vacancy{{PublicID: "VAC222222", ApplicationsCount: 4, NewApplicationsCount: 1}}, Total: 1}, nil
}

func (s *stubVacancyService) Update(_ context.Context, id string, in model.Vacancy) (model.Vacancy, error) {
	s.updated = in
	in.PublicID = id
	return in, nil
}

func (s *stubVacancyService) SetStatus(_ context.Context, id, status string) (model.Vacancy, error) {
	s.status = status
	return model.Vacancy{PublicID: id, Status: status}, nil
}

func (s *stubVacancyService) Delete(_ context.Context, id string) error {
	s.deleted = id
	return nil
}

func vacancyRouter(h *VacancyHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/vacancies", h.Index)
	r.Post("/vacancies", h.Store)
	r.Route("/vacancies/{vacancy}", func(r chi.Router) {
		r.Get("/", h.Show)
		r.Put("/", h.Update)
		r.Delete("/", h.Destroy)
		r.Patch("/status", h.UpdateStatus)
	})
	return r
}

func send(h http.Handler, method, url, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, url, strings.NewReader(body)))
	return rec
}

func TestStoreVacancyParsesBody(t *testing.T) {
	s := &stubVacancyService{}
	rank, vessel := uuid.New(), uuid.New()
	body := `{"title":" Chief Officer ","slug":"","rank_id":"` + rank.String() + `","vessel_type_id":"` + vessel.String() + `","department":"deck",
		"location":"Singapore","employment_type":"permanent","duration":"6 months","salary_text":"USD 5,000","description_html":"<p>x</p>",
		"requirements_html":"<p>y</p>","required_documents":[" Passport ","","COC"],"closing_date":"2026-12-31","status":"open"}`

	rec := send(vacancyRouter(NewVacancyHandler(s)), http.MethodPost, "/vacancies", body)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, "Chief Officer", s.created.Title)
	assert.Equal(t, rank, s.created.RankID)
	assert.Equal(t, vessel, *s.created.VesselTypeID)
	assert.Equal(t, []string{"Passport", "COC"}, s.created.RequiredDocuments)
	assert.Equal(t, time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), *s.created.ClosingDate)
	assert.Equal(t, model.EmploymentPermanent, s.created.EmploymentType)
	var out struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, "VAC222222", out.Data["id"])
	assert.Equal(t, "2026-12-31", out.Data["closing_date"])
	assert.Equal(t, []any{"Passport", "COC"}, out.Data["required_documents"])
}

func TestStoreVacancyAppliesDefaults(t *testing.T) {
	s := &stubVacancyService{}
	body := `{"title":"Master","rank_id":"` + uuid.NewString() + `","department":"engine"}`

	rec := send(vacancyRouter(NewVacancyHandler(s)), http.MethodPost, "/vacancies", body)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, model.VacancyDraft, s.created.Status)
	assert.Equal(t, model.EmploymentContract, s.created.EmploymentType)
	assert.Nil(t, s.created.VesselTypeID)
	assert.Nil(t, s.created.ClosingDate)
	assert.NotNil(t, s.created.RequiredDocuments)
}

func TestStoreVacancyValidation(t *testing.T) {
	rank := uuid.NewString()
	tests := []struct {
		name  string
		body  string
		field string
	}{
		{"missing title", `{"rank_id":"` + rank + `","department":"deck"}`, "title"},
		{"missing rank", `{"title":"x","department":"deck"}`, "rank_id"},
		{"invalid rank", `{"title":"x","rank_id":"abc","department":"deck"}`, "rank_id"},
		{"missing department", `{"title":"x","rank_id":"` + rank + `"}`, "department"},
		{"unknown department", `{"title":"x","rank_id":"` + rank + `","department":"bridge"}`, "department"},
		{"unknown employment type", `{"title":"x","rank_id":"` + rank + `","department":"deck","employment_type":"gig"}`, "employment_type"},
		{"unknown status", `{"title":"x","rank_id":"` + rank + `","department":"deck","status":"published"}`, "status"},
		{"malformed closing date", `{"title":"x","rank_id":"` + rank + `","department":"deck","closing_date":"31/12/2026"}`, "closing_date"},
		{"title too long", `{"title":"` + strings.Repeat("a", 201) + `","rank_id":"` + rank + `","department":"deck"}`, "title"},
		{"too many required documents", `{"title":"x","rank_id":"` + rank + `","department":"deck","required_documents":[` +
			strings.TrimSuffix(strings.Repeat(`"d",`, 21), ",") + `]}`, "required_documents"},
		{"required document too long", `{"title":"x","rank_id":"` + rank + `","department":"deck","required_documents":["` + strings.Repeat("a", 101) + `"]}`, "required_documents"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &stubVacancyService{}
			rec := send(vacancyRouter(NewVacancyHandler(s)), http.MethodPost, "/vacancies", tc.body)
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"`+tc.field+`"`)
			assert.Empty(t, s.created.Title)
		})
	}
}

func TestStoreVacancyAcceptsTwentyRequiredDocuments(t *testing.T) {
	s := &stubVacancyService{}
	body := `{"title":"x","rank_id":"` + uuid.NewString() + `","department":"deck","required_documents":[` + strings.TrimSuffix(strings.Repeat(`"d",`, 20), ",") + `]}`

	rec := send(vacancyRouter(NewVacancyHandler(s)), http.MethodPost, "/vacancies", body)

	assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
}

func TestVacancyIndexParsesFiltersAndPaging(t *testing.T) {
	s := &stubVacancyService{}
	rank := uuid.New()

	rec := send(vacancyRouter(NewVacancyHandler(s)), http.MethodGet,
		"/vacancies?q=master&status=open,draft&rank_id="+rank.String()+"&department=deck&sort=applications_count&order=desc&page=3&per_page=50", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "master", s.filter.Q)
	assert.Equal(t, []string{"open", "draft"}, s.filter.Statuses)
	assert.Equal(t, []uuid.UUID{rank}, s.filter.RankIDs)
	assert.Equal(t, []string{"deck"}, s.filter.Departments)
	assert.True(t, s.filter.Desc)
	assert.Equal(t, 3, s.page)
	assert.Equal(t, 50, s.perPage)
	var out struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.EqualValues(t, 4, out.Data[0]["applications_count"])
	assert.EqualValues(t, 1, out.Data[0]["new_applications_count"])

	for _, url := range []string{"/vacancies?status=archived", "/vacancies?rank_id=x", "/vacancies?department=bridge"} {
		assert.Equal(t, http.StatusUnprocessableEntity, send(vacancyRouter(NewVacancyHandler(s)), http.MethodGet, url, "").Code, url)
	}
}

func TestVacancyStatusUpdateDeleteAndNotFound(t *testing.T) {
	s := &stubVacancyService{}
	h := vacancyRouter(NewVacancyHandler(s))

	rec := send(h, http.MethodPatch, "/vacancies/VAC222222/status", `{"status":"closed"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "closed", s.status)
	assert.Equal(t, http.StatusUnprocessableEntity, send(h, http.MethodPatch, "/vacancies/VAC222222/status", `{"status":"gone"}`).Code)
	assert.Equal(t, http.StatusUnprocessableEntity, send(h, http.MethodPatch, "/vacancies/VAC222222/status", `{}`).Code)

	rec = send(h, http.MethodDelete, "/vacancies/VAC222222", "")
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "VAC222222", s.deleted)

	s.getError = apperr.NotFound("Vacancy")
	assert.Equal(t, http.StatusNotFound, send(h, http.MethodGet, "/vacancies/VAC999999", "").Code)

	rec = send(h, http.MethodPut, "/vacancies/VAC222222", `{"title":"x","rank_id":"`+uuid.NewString()+`","department":"deck"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "x", s.updated.Title)
}

type stubApplicantService struct {
	filter  model.ApplicantFilter
	admin   uuid.UUID
	stage   string
	note    string
	deleted string
	err     error
}

func (s *stubApplicantService) Get(_ context.Context, id string) (model.Applicant, error) {
	rank := uuid.New()
	dob := time.Date(1990, 5, 1, 0, 0, 0, 0, time.UTC)
	adminID := uuid.New()
	return model.Applicant{
		ID: uuid.New(), PublicID: id, VacancyPublicID: "VAC222222", VacancyTitle: "Master", VacancySlug: "master", VacancyStatus: "open",
		FullName: "Siti Aminah", Email: "siti@example.com", Phone: "+6281234567890", DateOfBirth: &dob, RankAppliedID: &rank, RankAppliedName: "Master",
		Stage: model.StageScreening, IPAddress: "203.0.113.9",
		Files: []model.ApplicantFile{{FileID: uuid.New(), Kind: "cv", OriginalName: "cv.docx", StorageKey: "applicant/2026/10/abc.pdf", MimeType: "application/pdf", Size: 1200}},
		StageLogs: []model.StageLog{
			{ID: uuid.New(), ToStage: model.StageNew},
			{ID: uuid.New(), FromStage: model.StageNew, ToStage: model.StageScreening, Note: "ok", AdminID: &adminID, AdminName: "Ambon"},
		},
	}, s.err
}

func (s *stubApplicantService) List(_ context.Context, f model.ApplicantFilter, _, _ int) (repository.Page[model.Applicant], error) {
	s.filter = f
	return repository.Page[model.Applicant]{Items: []model.Applicant{{PublicID: "APP222222", Stage: model.StageNew}}, Total: 1}, nil
}

func (s *stubApplicantService) StageCounts(_ context.Context, f model.ApplicantFilter) (map[string]int, error) {
	s.filter = f
	return map[string]int{"new": 2, "screening": 0, "interview": 0, "accepted": 0, "rejected": 1}, nil
}

func (s *stubApplicantService) ChangeStage(ctx context.Context, id string, admin uuid.UUID, stage, note string) (model.Applicant, error) {
	s.admin, s.stage, s.note = admin, stage, note
	if s.err != nil {
		return model.Applicant{}, s.err
	}
	return s.Get(ctx, id)
}

func (s *stubApplicantService) Delete(_ context.Context, id string) error {
	s.deleted = id
	return nil
}

func applicantRouter(h *ApplicantHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/applicants", h.Index)
	r.Get("/applicants/stage-counts", h.StageCounts)
	r.Route("/applicants/{applicant}", func(r chi.Router) {
		r.Get("/", h.Show)
		r.Delete("/", h.Destroy)
		r.Patch("/stage", h.UpdateStage)
	})
	return r
}

func TestApplicantIndexParsesFilters(t *testing.T) {
	s := &stubApplicantService{}
	rank := uuid.New()

	rec := send(applicantRouter(NewApplicantHandler(s)), http.MethodGet,
		"/applicants?q=siti&vacancy=vac222222&stage=new,interview&rank_id="+rank.String()+"&date_from=2026-10-01&date_to=2026-10-31", "")

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "siti", s.filter.Q)
	assert.Equal(t, "VAC222222", s.filter.VacancyPublic)
	assert.Equal(t, []string{"new", "interview"}, s.filter.Stages)
	assert.Equal(t, []uuid.UUID{rank}, s.filter.RankIDs)
	assert.Equal(t, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), *s.filter.DateFrom)
	assert.Equal(t, time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), *s.filter.DateTo)

	for _, url := range []string{"/applicants?stage=hired", "/applicants?date_from=x", "/applicants?date_from=2026-10-31&date_to=2026-10-01", "/applicants?rank_id=1"} {
		assert.Equal(t, http.StatusUnprocessableEntity, send(applicantRouter(NewApplicantHandler(s)), http.MethodGet, url, "").Code, url)
	}
}

func TestApplicantStageCountsReturnsCountPerStage(t *testing.T) {
	rec := send(applicantRouter(NewApplicantHandler(&stubApplicantService{})), http.MethodGet, "/applicants/stage-counts?vacancy=VAC222222", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":{"new":2,"screening":0,"interview":0,"accepted":0,"rejected":1}}`, rec.Body.String())
}

func TestApplicantShowIncludesFilesAndStageHistory(t *testing.T) {
	rec := send(applicantRouter(NewApplicantHandler(&stubApplicantService{})), http.MethodGet, "/applicants/APP222222", "")

	require.Equal(t, http.StatusOK, rec.Code)
	var out struct {
		Data struct {
			ID          string `json:"id"`
			DateOfBirth string `json:"date_of_birth"`
			FilesCount  int    `json:"files_count"`
			Vacancy     struct {
				ID, Title, Slug, Status string
			} `json:"vacancy"`
			Files []struct {
				FileID   string `json:"file_id"`
				Kind     string `json:"kind"`
				Name     string `json:"name"`
				Size     int64  `json:"size"`
				MimeType string `json:"mime_type"`
			} `json:"files"`
			StageLogs []struct {
				FromStage *string `json:"from_stage"`
				ToStage   string  `json:"to_stage"`
				Note      string  `json:"note"`
				Admin     *struct {
					Name string `json:"name"`
				} `json:"admin"`
			} `json:"stage_logs"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, "APP222222", out.Data.ID)
	assert.Equal(t, "1990-05-01", out.Data.DateOfBirth)
	assert.Equal(t, 1, out.Data.FilesCount)
	assert.Equal(t, "VAC222222", out.Data.Vacancy.ID)
	require.Len(t, out.Data.Files, 1)
	assert.Equal(t, "cv.pdf", out.Data.Files[0].Name, "converted documents are listed under their stored extension")
	assert.Equal(t, "cv", out.Data.Files[0].Kind)
	require.Len(t, out.Data.StageLogs, 2)
	assert.Nil(t, out.Data.StageLogs[0].FromStage)
	assert.Nil(t, out.Data.StageLogs[0].Admin)
	assert.Equal(t, "new", *out.Data.StageLogs[1].FromStage)
	assert.Equal(t, "Ambon", out.Data.StageLogs[1].Admin.Name)
}

func TestApplicantUpdateStageRecordsTheSessionAdmin(t *testing.T) {
	s := &stubApplicantService{}
	adminID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/applicants/APP222222/stage", strings.NewReader(`{"stage":"rejected","note":"No valid COC"}`))
	req = req.WithContext(middleware.WithAdmin(req.Context(), model.Admin{ID: adminID}, "session"))
	rec := httptest.NewRecorder()

	applicantRouter(NewApplicantHandler(s)).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, adminID, s.admin)
	assert.Equal(t, "rejected", s.stage)
	assert.Equal(t, "No valid COC", s.note)
}

func TestApplicantUpdateStagePropagatesConflicts(t *testing.T) {
	s := &stubApplicantService{err: apperr.Conflict("stage_unchanged", "The applicant is already in this stage.")}

	rec := send(applicantRouter(NewApplicantHandler(s)), http.MethodPatch, "/applicants/APP222222/stage", `{"stage":"new"}`)

	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "stage_unchanged")
	assert.Equal(t, http.StatusBadRequest, send(applicantRouter(NewApplicantHandler(s)), http.MethodPatch, "/applicants/APP222222/stage", `nope`).Code)
}

func TestApplicantDestroy(t *testing.T) {
	s := &stubApplicantService{}

	rec := send(applicantRouter(NewApplicantHandler(s)), http.MethodDelete, "/applicants/APP222222", "")

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "APP222222", s.deleted)
}

type stubInbox struct {
	filter  model.ContactFilter
	read    *bool
	archive *bool
	unread  int
}

func (s *stubInbox) List(_ context.Context, f model.ContactFilter, _, _ int) (repository.Page[model.ContactMessage], error) {
	s.filter = f
	return repository.Page[model.ContactMessage]{Items: []model.ContactMessage{{ID: uuid.New(), Name: "John", Subject: "Hi"}}, Total: 1}, nil
}

func (s *stubInbox) Get(_ context.Context, id uuid.UUID) (model.ContactMessage, error) {
	return model.ContactMessage{ID: id, Name: "John", IPAddress: "203.0.113.9"}, nil
}

func (s *stubInbox) SetFlags(_ context.Context, id uuid.UUID, isRead, isArchived *bool) (model.ContactMessage, error) {
	s.read, s.archive = isRead, isArchived
	return model.ContactMessage{ID: id, IsRead: isRead != nil && *isRead}, nil
}

func (s *stubInbox) UnreadCount(context.Context) (int, error) { return s.unread, nil }

func inboxRouter(h *ContactHandler) http.Handler {
	r := chi.NewRouter()
	r.Get("/contact-messages", h.Index)
	r.Get("/contact-messages/unread-count", h.UnreadCount)
	r.Route("/contact-messages/{id}", func(r chi.Router) {
		r.Get("/", h.Show)
		r.Patch("/", h.Update)
	})
	return r
}

func TestContactInboxFiltersDefaultToAll(t *testing.T) {
	s := &stubInbox{}
	h := inboxRouter(NewContactHandler(s))

	require.Equal(t, http.StatusOK, send(h, http.MethodGet, "/contact-messages", "").Code)
	assert.Equal(t, model.ContactFilter{Status: "all"}, s.filter)

	require.Equal(t, http.StatusOK, send(h, http.MethodGet, "/contact-messages?status=unread&q=john", "").Code)
	assert.Equal(t, model.ContactFilter{Status: "unread", Q: "john"}, s.filter)

	assert.Equal(t, http.StatusUnprocessableEntity, send(h, http.MethodGet, "/contact-messages?status=spam", "").Code)
}

func TestContactInboxUnreadCountEndpoint(t *testing.T) {
	rec := send(inboxRouter(NewContactHandler(&stubInbox{unread: 7})), http.MethodGet, "/contact-messages/unread-count", "")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"data":{"unread":7}}`, rec.Body.String())
}

func TestContactInboxPatchPassesOnlyGivenFlags(t *testing.T) {
	s := &stubInbox{}
	h := inboxRouter(NewContactHandler(s))
	id := uuid.NewString()

	rec := send(h, http.MethodPatch, "/contact-messages/"+id, `{"is_read":true}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, s.read)
	assert.True(t, *s.read)
	assert.Nil(t, s.archive)

	send(h, http.MethodPatch, "/contact-messages/"+id, `{"is_archived":false}`)
	assert.Nil(t, s.read)
	require.NotNil(t, s.archive)
	assert.False(t, *s.archive)

	assert.Equal(t, http.StatusNotFound, send(h, http.MethodPatch, "/contact-messages/not-a-uuid", `{"is_read":true}`).Code)
	assert.Equal(t, http.StatusNotFound, send(h, http.MethodGet, "/contact-messages/not-a-uuid", "").Code)
	assert.Equal(t, http.StatusOK, send(h, http.MethodGet, "/contact-messages/"+id, "").Code)
}
