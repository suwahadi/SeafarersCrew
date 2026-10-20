package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type stubRanks struct {
	lastQuery repository.ListQuery
	created   model.Rank
}

func (s *stubRanks) List(_ context.Context, q repository.ListQuery) (repository.Page[model.Rank], error) {
	s.lastQuery = q
	return repository.Page[model.Rank]{Items: []model.Rank{{Name: "Master", ShortCode: "MST", Department: "deck"}}, Total: 41}, nil
}

func (s *stubRanks) Get(_ context.Context, id uuid.UUID) (model.Rank, error) {
	return model.Rank{}, apperr.NotFound("Rank")
}

func (s *stubRanks) Create(_ context.Context, r model.Rank) (model.Rank, error) {
	s.created = r
	r.ID = uuid.New()
	return r, nil
}

func (s *stubRanks) Update(_ context.Context, _ uuid.UUID, r model.Rank) (model.Rank, error) {
	return r, nil
}

func (s *stubRanks) Delete(context.Context, uuid.UUID) error { return nil }

func (s *stubRanks) Reorder(context.Context, []uuid.UUID) error { return nil }

func rankRouter(svc *stubRanks) http.Handler {
	h := NewMasterHandler[model.Rank, request.Rank](svc, present(resource.NewRank), map[string]Filter{
		"department": {Kind: FilterEnum, Options: request.Departments()},
	})
	r := chi.NewRouter()
	r.Get("/ranks", h.List)
	r.Post("/ranks", h.Create)
	r.Get("/ranks/{id}", h.Show)
	r.Patch("/ranks/reorder", h.Reorder)
	return r
}

func TestMasterListReturnsEnvelopeWithMeta(t *testing.T) {
	svc := &stubRanks{}
	rec := httptest.NewRecorder()
	rankRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ranks?page=2&per_page=10&sort=name&order=desc&q=mas&department=deck&is_active=true", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Data []map[string]any `json:"data"`
		Meta resource.Meta    `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, resource.Meta{Page: 2, PerPage: 10, Total: 41}, body.Meta)
	assert.Equal(t, "MST", body.Data[0]["short_code"])
	assert.Equal(t, map[string]any{"department": "deck", "is_active": true}, svc.lastQuery.Filters)
	assert.True(t, svc.lastQuery.Desc)
	assert.Equal(t, "mas", svc.lastQuery.Q)
}

func TestMasterListRejectsInvalidFilters(t *testing.T) {
	rec := httptest.NewRecorder()
	rankRouter(&stubRanks{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ranks?department=bridge&is_active=maybe", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), `"department"`)
	assert.Contains(t, rec.Body.String(), `"is_active"`)
}

func TestMasterCreateValidatesAndNormalizes(t *testing.T) {
	svc := &stubRanks{}
	rec := httptest.NewRecorder()
	rankRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/ranks", strings.NewReader(`{"name":" Chief Officer ","short_code":"co","department":"deck"}`)))
	require.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "CO", svc.created.ShortCode)
	assert.Equal(t, "Chief Officer", svc.created.Name)
	assert.True(t, svc.created.IsActive, "new records are active by default")

	rec = httptest.NewRecorder()
	rankRouter(svc).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/ranks", strings.NewReader(`{"name":"","short_code":"c o","department":"bridge"}`)))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	for _, field := range []string{"name", "short_code", "department"} {
		assert.Contains(t, rec.Body.String(), `"`+field+`"`)
	}
}

func TestMasterShowWithMalformedIDIsNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	rankRouter(&stubRanks{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ranks/not-a-uuid", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMasterReorderRequiresIDs(t *testing.T) {
	rec := httptest.NewRecorder()
	rankRouter(&stubRanks{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/ranks/reorder", strings.NewReader(`{"ids":[]}`)))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = httptest.NewRecorder()
	rankRouter(&stubRanks{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPatch, "/ranks/reorder", strings.NewReader(`{"ids":["`+uuid.NewString()+`"]}`)))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestVesselRequestValidation(t *testing.T) {
	cases := []struct {
		body  string
		field string
	}{
		{`{"name":"MV","imo_number":"12345"}`, "imo_number"},
		{`{"name":"MV","imo_number":"9074729","year_built":1800}`, "year_built"},
		{`{"name":"MV","imo_number":"9074729","grt":-1}`, "grt"},
		{`{"name":"MV","imo_number":"9074729","status":"scrapped"}`, "status"},
	}
	for _, tc := range cases {
		var in request.Vessel
		require.NoError(t, json.Unmarshal([]byte(tc.body), &in))
		assert.Contains(t, in.Validate(), tc.field, tc.body)
	}
	var ok request.Vessel
	require.NoError(t, json.Unmarshal([]byte(`{"name":"MV","imo_number":"9074729","grt":"1234.567"}`), &ok))
	assert.Empty(t, ok.Validate())
	assert.Equal(t, "active", ok.Model().Status)
	assert.Equal(t, "1234.57", ok.Model().GRT.StringFixed(2))
}

func TestCurrencyAndStatusRequestValidation(t *testing.T) {
	var c request.Currency
	require.NoError(t, json.Unmarshal([]byte(`{"code":"usd","decimals":5}`), &c))
	f := c.Validate()
	assert.Equal(t, "USD", c.Code)
	assert.Contains(t, f, "decimals")

	var bad request.Currency
	require.NoError(t, json.Unmarshal([]byte(`{"code":"US1"}`), &bad))
	f = bad.Validate()
	assert.Contains(t, f, "code")
	assert.Contains(t, f, "decimals")

	var s request.CrewStatus
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Leave","color":"orange"}`), &s))
	assert.Contains(t, s.Validate(), "color")
}
