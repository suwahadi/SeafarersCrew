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
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type stubQuotations struct{ created model.Quotation }

func (s *stubQuotations) List(context.Context, model.QuotationFilter) (repository.Page[model.Quotation], error) {
	return repository.Page[model.Quotation]{Items: []model.Quotation{{PublicID: "QTAAAAAA2", GrandTotal: decimal.RequireFromString("1100.50")}}, Total: 1}, nil
}

func (s *stubQuotations) Get(_ context.Context, pid string) (model.Quotation, error) {
	return model.Quotation{}, apperr.NotFound("Quotation")
}

func (s *stubQuotations) Create(_ context.Context, q model.Quotation) (model.Quotation, error) {
	q.PublicID, q.Status = "QTAAAAAA2", model.QuotationDraft
	s.created = q
	return q, nil
}

func (s *stubQuotations) Update(context.Context, string, model.Quotation) (model.Quotation, error) {
	return model.Quotation{}, nil
}

func (s *stubQuotations) ChangeStatus(context.Context, string, string) (model.Quotation, error) {
	return model.Quotation{}, apperr.Conflict("invalid_transition", "no")
}

func (s *stubQuotations) Delete(context.Context, string) error { return nil }

func TestQuotationStoreValidatesAndReturnsMoneyAsStrings(t *testing.T) {
	stub := &stubQuotations{}
	h := NewQuotationHandler(stub)

	rec := httptest.NewRecorder()
	h.Store(rec, httptest.NewRequest(http.MethodPost, "/quotations", strings.NewReader(`{"lines":[]}`)))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), `"lines"`)

	body := `{"client_company_id":"` + uuid.NewString() + `","currency_code":"USD","quotation_date":"2026-10-01","valid_until":"2026-10-31",
		"lines":[{"line_type":"lump_sum","description":"Fee","amount":"1000.5"}]}`
	rec = httptest.NewRecorder()
	h.Store(rec, httptest.NewRequest(http.MethodPost, "/quotations", strings.NewReader(body)))
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"quotation_date":"2026-10-01"`)
	assert.Contains(t, rec.Body.String(), `"amount":"1000.5"`)
	assert.True(t, decimal.RequireFromString("1000.5").Equal(stub.created.Lines[0].Amount))

	rec = httptest.NewRecorder()
	h.Index(rec, httptest.NewRequest(http.MethodGet, "/quotations?status=open", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = httptest.NewRecorder()
	h.Index(rec, httptest.NewRequest(http.MethodGet, "/quotations", nil))
	assert.Contains(t, rec.Body.String(), `"grand_total":"1100.5"`)
	assert.Contains(t, rec.Body.String(), `"total":1`)
}

type stubTemplates struct{}

func (stubTemplates) List(context.Context, string) ([]model.DocumentTemplate, error) {
	return []model.DocumentTemplate{{Name: "SEA", Kind: "sea", BodyHTML: "<p>secret body</p>"}}, nil
}

func (stubTemplates) Get(context.Context, uuid.UUID) (service.TemplateResult, error) {
	return service.TemplateResult{}, apperr.NotFound("Template")
}

func (stubTemplates) Save(_ context.Context, t model.DocumentTemplate) (service.TemplateResult, error) {
	return service.TemplateResult{Template: t, UnknownTags: []string{"oops"}}, nil
}

func (stubTemplates) Delete(context.Context, uuid.UUID) error { return nil }

type stubDocuments struct{ got service.DocumentRequest }

func (s *stubDocuments) Preview(_ context.Context, req service.DocumentRequest) (service.Preview, error) {
	s.got = req
	return service.Preview{HTML: "<p>x</p>", UnknownTags: []string{}}, nil
}

func (s *stubDocuments) Generate(context.Context, service.DocumentRequest) (model.GeneratedDocument, error) {
	return model.GeneratedDocument{PublicID: "GENAAAAA2", Kind: "sea", FileID: uuid.New()}, nil
}

func (s *stubDocuments) History(context.Context, string, string) ([]model.GeneratedDocument, error) {
	return nil, nil
}

func TestDocumentHandlerEndpoints(t *testing.T) {
	docs := &stubDocuments{}
	h := NewDocumentHandler(stubTemplates{}, docs)

	rec := httptest.NewRecorder()
	h.Variables(rec, httptest.NewRequest(http.MethodGet, "/document-templates/variables", nil))
	var vars struct {
		Data struct {
			Groups []service.VariableGroup `json:"groups"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &vars))
	assert.Len(t, vars.Data.Groups, len(service.VariableCatalogue))

	rec = httptest.NewRecorder()
	h.Templates(rec, httptest.NewRequest(http.MethodGet, "/document-templates", nil))
	assert.NotContains(t, rec.Body.String(), "secret body", "lists omit template HTML")

	rec = httptest.NewRecorder()
	h.SaveTemplate(rec, httptest.NewRequest(http.MethodPost, "/document-templates", strings.NewReader(`{"name":"CV","kind":"cv","body_html":"<p>{x}</p>"}`)))
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"unknown_tags":["oops"]`)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	req := httptest.NewRequest(http.MethodGet, "/document-templates/x", nil).WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rctx))
	rec = httptest.NewRecorder()
	h.Template(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	tplID := uuid.NewString()
	rec = httptest.NewRecorder()
	h.Preview(rec, httptest.NewRequest(http.MethodPost, "/documents/preview", strings.NewReader(`{"kind":"cv","related_id":"CREWAAAA2","template_id":"`+tplID+`","hide":["photo"]}`)))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, tplID, docs.got.TemplateID.String())
	assert.Equal(t, []string{"photo"}, docs.got.Hide)

	rec = httptest.NewRecorder()
	h.Generate(rec, httptest.NewRequest(http.MethodPost, "/documents/generate", strings.NewReader(`{"kind":"sea"}`)))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	rec = httptest.NewRecorder()
	h.Generate(rec, httptest.NewRequest(http.MethodPost, "/documents/generate", strings.NewReader(`{"kind":"sea","related_id":"KNTRAAAA2"}`)))
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":"GENAAAAA2"`)
}
