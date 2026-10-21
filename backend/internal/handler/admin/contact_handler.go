package admin

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// ContactInboxService is what the contact inbox controller needs.
type ContactInboxService interface {
	List(ctx context.Context, f model.ContactFilter, page, perPage int) (repository.Page[model.ContactMessage], error)
	Get(ctx context.Context, id uuid.UUID) (model.ContactMessage, error)
	SetFlags(ctx context.Context, id uuid.UUID, isRead, isArchived *bool) (model.ContactMessage, error)
	UnreadCount(ctx context.Context) (int, error)
}

// ContactHandler serves the contact inbox.
type ContactHandler struct {
	inbox ContactInboxService
}

// NewContactHandler returns a ContactHandler.
func NewContactHandler(inbox ContactInboxService) *ContactHandler {
	return &ContactHandler{inbox: inbox}
}

func messageID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		resource.Error(w, r, apperr.NotFound("Message"))
		return uuid.Nil, false
	}
	return id, true
}

// Index handles GET /contact-messages.
func (h *ContactHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, errs := request.ParseContactFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	p := request.ParseList(q)
	page, err := h.inbox.List(r.Context(), f, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.NewContactMessages(page.Items), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// UnreadCount handles GET /contact-messages/unread-count.
func (h *ContactHandler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	n, err := h.inbox.UnreadCount(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, map[string]int{"unread": n})
}

// Show handles GET /contact-messages/{id}.
func (h *ContactHandler) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := messageID(w, r)
	if !ok {
		return
	}
	m, err := h.inbox.Get(r.Context(), id)
	respondMessage(w, r, m, err)
}

// Update handles PATCH /contact-messages/{id}.
func (h *ContactHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := messageID(w, r)
	if !ok {
		return
	}
	var in request.ContactPatch
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	m, err := h.inbox.SetFlags(r.Context(), id, in.IsRead, in.IsArchived)
	respondMessage(w, r, m, err)
}

func respondMessage(w http.ResponseWriter, r *http.Request, m model.ContactMessage, err error) {
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewContactMessage(m))
}
