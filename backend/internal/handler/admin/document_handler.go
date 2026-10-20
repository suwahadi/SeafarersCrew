package admin

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// TemplateService is what the template controller needs.
type TemplateService interface {
	List(ctx context.Context, kind string) ([]model.DocumentTemplate, error)
	Get(ctx context.Context, id uuid.UUID) (service.TemplateResult, error)
	Save(ctx context.Context, t model.DocumentTemplate) (service.TemplateResult, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// DocumentService is what the document controller needs.
type DocumentService interface {
	Preview(ctx context.Context, req service.DocumentRequest) (service.Preview, error)
	Generate(ctx context.Context, req service.DocumentRequest) (model.GeneratedDocument, error)
	History(ctx context.Context, relatedType, relatedPublicID string) ([]model.GeneratedDocument, error)
}

// DocumentHandler serves templates, previews, generation and history.
type DocumentHandler struct {
	templates TemplateService
	documents DocumentService
}

// NewDocumentHandler returns a DocumentHandler.
func NewDocumentHandler(templates TemplateService, documents DocumentService) *DocumentHandler {
	return &DocumentHandler{templates: templates, documents: documents}
}

// Variables handles GET /document-templates/variables.
func (h *DocumentHandler) Variables(w http.ResponseWriter, _ *http.Request) {
	resource.JSON(w, http.StatusOK, map[string]any{"groups": service.VariableCatalogue, "cv_sections": service.CVSections})
}

// Templates handles GET /document-templates?kind=.
func (h *DocumentHandler) Templates(w http.ResponseWriter, r *http.Request) {
	list, err := h.templates.List(r.Context(), r.URL.Query().Get("kind"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.MapAll(list, resource.NewTemplateSummary))
}

// Template handles GET /document-templates/{id}.
func (h *DocumentHandler) Template(w http.ResponseWriter, r *http.Request) {
	id, err := templateID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	t, err := h.templates.Get(r.Context(), id)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewTemplate(t))
}

// SaveTemplate handles POST /document-templates and PUT /document-templates/{id}.
func (h *DocumentHandler) SaveTemplate(w http.ResponseWriter, r *http.Request) {
	var in request.DocumentTemplate
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	t := in.Model()
	status := http.StatusCreated
	if chi.URLParam(r, "id") != "" {
		id, err := templateID(r)
		if err != nil {
			resource.Error(w, r, err)
			return
		}
		t.ID, status = id, http.StatusOK
	}
	saved, err := h.templates.Save(r.Context(), t)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewTemplate(saved))
}

// DeleteTemplate handles DELETE /document-templates/{id}.
func (h *DocumentHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id, err := templateID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	if err := h.templates.Delete(r.Context(), id); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

// Preview handles POST /documents/preview.
func (h *DocumentHandler) Preview(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRender(w, r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	p, err := h.documents.Preview(r.Context(), req)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, p)
}

// Generate handles POST /documents/generate.
func (h *DocumentHandler) Generate(w http.ResponseWriter, r *http.Request) {
	req, err := decodeRender(w, r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	g, err := h.documents.Generate(r.Context(), req)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, resource.NewGeneratedDocument(g))
}

// History handles GET /generated-documents?related_type=&related_id=.
func (h *DocumentHandler) History(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list, err := h.documents.History(r.Context(), q.Get("related_type"), q.Get("related_id"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.MapAll(list, resource.NewGeneratedDocument))
}

func decodeRender(w http.ResponseWriter, r *http.Request) (service.DocumentRequest, error) {
	var in request.DocumentRender
	if err := request.Decode(w, r, &in); err != nil {
		return service.DocumentRequest{}, err
	}
	return service.DocumentRequest{Kind: in.Kind, TemplateID: in.Parsed(), RelatedID: in.RelatedID, Hide: in.Hide}, nil
}

func templateID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperr.NotFound("Template")
	}
	return id, nil
}
