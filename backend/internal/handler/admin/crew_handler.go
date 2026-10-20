package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// CrewService is what the crew controller needs.
type CrewService interface {
	Create(ctx context.Context, c model.Crew) (model.Crew, error)
	Get(ctx context.Context, publicID string) (model.Crew, error)
	Update(ctx context.Context, publicID string, c model.Crew) (model.Crew, error)
	Delete(ctx context.Context, publicID string) error
	ChangeStatus(ctx context.Context, publicID, action, reason string) (model.Crew, error)
	Compliance(ctx context.Context, publicID string, rankID *uuid.UUID, checkDate *time.Time) (service.CrewCompliance, error)
	SeaService(ctx context.Context, publicID string) (service.CrewSeaService, error)
	ListBankAccounts(ctx context.Context, publicID string) ([]model.BankAccount, error)
	SaveBankAccount(ctx context.Context, publicID string, b model.BankAccount) (model.BankAccount, error)
	DeleteBankAccount(ctx context.Context, publicID string, id uuid.UUID) error
	ListNextOfKins(ctx context.Context, publicID string) ([]model.NextOfKin, error)
	SaveNextOfKin(ctx context.Context, publicID string, k model.NextOfKin) (model.NextOfKin, error)
	DeleteNextOfKin(ctx context.Context, publicID string, id uuid.UUID) error
	ListDocuments(ctx context.Context, k repository.DocumentKind, publicID string) ([]service.DocumentView, error)
	SaveDocument(ctx context.Context, k repository.DocumentKind, publicID string, d model.CrewDocument) (service.DocumentView, error)
	DeleteDocument(ctx context.Context, k repository.DocumentKind, publicID string, id uuid.UUID) error
	SaveExperience(ctx context.Context, publicID string, x model.Experience) (model.Experience, error)
	DeleteExperience(ctx context.Context, publicID string, id uuid.UUID) error
	Today(ctx context.Context) (time.Time, error)
	List(ctx context.Context, f model.CrewFilter, page, perPage int) (service.CrewList, error)
}

// CrewHandler serves crew records and their tabs.
type CrewHandler struct {
	crews CrewService
}

// NewCrewHandler returns a CrewHandler.
func NewCrewHandler(crews CrewService) *CrewHandler {
	return &CrewHandler{crews: crews}
}

func crewID(r *http.Request) string { return chi.URLParam(r, "crew") }

func rowID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperr.NotFound("Record")
	}
	return id, nil
}

func (h *CrewHandler) crew(w http.ResponseWriter, r *http.Request, status int, c model.Crew) {
	today, err := h.crews.Today(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewCrew(c, today))
}

// Index handles GET /crews with datagrid search, filters, sort and grouping.
func (h *CrewHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, errs := request.ParseCrewFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	p := request.ParseList(q)
	res, err := h.crews.List(r.Context(), f, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.NewCrewList(res), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: res.Total})
}

// Store handles POST /crews.
func (h *CrewHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in request.Crew
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	c, err := h.crews.Create(r.Context(), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	h.crew(w, r, http.StatusCreated, c)
}

// Show handles GET /crews/{crew}.
func (h *CrewHandler) Show(w http.ResponseWriter, r *http.Request) {
	c, err := h.crews.Get(r.Context(), crewID(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	h.crew(w, r, http.StatusOK, c)
}

// Update handles PUT /crews/{crew}.
func (h *CrewHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in request.Crew
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	c, err := h.crews.Update(r.Context(), crewID(r), in.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	h.crew(w, r, http.StatusOK, c)
}

// Destroy handles DELETE /crews/{crew}.
func (h *CrewHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.crews.Delete(r.Context(), crewID(r)); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

// Status handles PATCH /crews/{crew}/status.
func (h *CrewHandler) Status(w http.ResponseWriter, r *http.Request) {
	var in request.CrewStatusChange
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	c, err := h.crews.ChangeStatus(r.Context(), crewID(r), in.Action, in.Reason)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	h.crew(w, r, http.StatusOK, c)
}

// Compliance handles GET /crews/{crew}/compliance?rank_id=&date=.
func (h *CrewHandler) Compliance(w http.ResponseWriter, r *http.Request) {
	e := request.Errors{}
	rankID := e.UUID("rank_id", r.URL.Query().Get("rank_id"))
	date := e.Date("date", r.URL.Query().Get("date"))
	if len(e) > 0 {
		resource.Error(w, r, apperr.Validation(e))
		return
	}
	res, err := h.crews.Compliance(r.Context(), crewID(r), rankID, date)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewCompliance(res))
}

// SeaService handles GET /crews/{crew}/sea-service and GET /crews/{crew}/experiences.
func (h *CrewHandler) SeaService(w http.ResponseWriter, r *http.Request) {
	res, err := h.crews.SeaService(r.Context(), crewID(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewSeaService(res))
}

// BankAccounts handles GET /crews/{crew}/bank-accounts.
func (h *CrewHandler) BankAccounts(w http.ResponseWriter, r *http.Request) {
	rows, err := h.crews.ListBankAccounts(r.Context(), crewID(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewBankAccounts(rows))
}

// SaveBankAccount handles POST /crews/{crew}/bank-accounts and PUT /crews/{crew}/bank-accounts/{id}.
func (h *CrewHandler) SaveBankAccount(w http.ResponseWriter, r *http.Request) {
	var in request.BankAccount
	id, status, err := decodeRow(w, r, &in)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	b := in.Model(uuid.Nil)
	b.ID = id
	saved, err := h.crews.SaveBankAccount(r.Context(), crewID(r), b)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewBankAccount(saved))
}

// DeleteBankAccount handles DELETE /crews/{crew}/bank-accounts/{id}.
func (h *CrewHandler) DeleteBankAccount(w http.ResponseWriter, r *http.Request) {
	h.deleteRow(w, r, h.crews.DeleteBankAccount)
}

// NextOfKins handles GET /crews/{crew}/next-of-kins.
func (h *CrewHandler) NextOfKins(w http.ResponseWriter, r *http.Request) {
	rows, err := h.crews.ListNextOfKins(r.Context(), crewID(r))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewNextOfKins(rows))
}

// SaveNextOfKin handles POST and PUT on /crews/{crew}/next-of-kins.
func (h *CrewHandler) SaveNextOfKin(w http.ResponseWriter, r *http.Request) {
	var in request.NextOfKin
	id, status, err := decodeRow(w, r, &in)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	k := in.Model(uuid.Nil)
	k.ID = id
	saved, err := h.crews.SaveNextOfKin(r.Context(), crewID(r), k)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewNextOfKin(saved))
}

// DeleteNextOfKin handles DELETE /crews/{crew}/next-of-kins/{id}.
func (h *CrewHandler) DeleteNextOfKin(w http.ResponseWriter, r *http.Request) {
	h.deleteRow(w, r, h.crews.DeleteNextOfKin)
}

// Documents returns handlers for one document kind (documents or certificates).
func (h *CrewHandler) Documents(k repository.DocumentKind) (list, save, remove http.HandlerFunc) {
	list = func(w http.ResponseWriter, r *http.Request) {
		rows, err := h.crews.ListDocuments(r.Context(), k, crewID(r))
		if err != nil {
			resource.Error(w, r, err)
			return
		}
		resource.JSON(w, http.StatusOK, resource.NewDocuments(rows))
	}
	save = func(w http.ResponseWriter, r *http.Request) {
		var in request.CrewDocument
		id, status, err := decodeRow(w, r, &in)
		if err != nil {
			resource.Error(w, r, err)
			return
		}
		d := in.Model(uuid.Nil)
		d.ID = id
		saved, err := h.crews.SaveDocument(r.Context(), k, crewID(r), d)
		if err != nil {
			resource.Error(w, r, err)
			return
		}
		resource.JSON(w, status, resource.NewDocument(saved))
	}
	remove = func(w http.ResponseWriter, r *http.Request) {
		h.deleteRow(w, r, func(ctx context.Context, publicID string, id uuid.UUID) error {
			return h.crews.DeleteDocument(ctx, k, publicID, id)
		})
	}
	return list, save, remove
}

// SaveExperience handles POST and PUT on /crews/{crew}/experiences.
func (h *CrewHandler) SaveExperience(w http.ResponseWriter, r *http.Request) {
	var in request.Experience
	id, status, err := decodeRow(w, r, &in)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	x := in.Model(uuid.Nil)
	x.ID = id
	saved, err := h.crews.SaveExperience(r.Context(), crewID(r), x)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewExperience(saved))
}

// DeleteExperience handles DELETE /crews/{crew}/experiences/{id}.
func (h *CrewHandler) DeleteExperience(w http.ResponseWriter, r *http.Request) {
	h.deleteRow(w, r, h.crews.DeleteExperience)
}

func (h *CrewHandler) deleteRow(w http.ResponseWriter, r *http.Request, del func(ctx context.Context, publicID string, id uuid.UUID) error) {
	id, err := rowID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	if err := del(r.Context(), crewID(r), id); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

func decodeRow(w http.ResponseWriter, r *http.Request, dst any) (uuid.UUID, int, error) {
	if err := request.Decode(w, r, dst); err != nil {
		return uuid.Nil, 0, err
	}
	if chi.URLParam(r, "id") == "" {
		return uuid.Nil, http.StatusCreated, nil
	}
	id, err := rowID(r)
	return id, http.StatusOK, err
}
