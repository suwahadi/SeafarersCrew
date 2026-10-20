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

// ContractService is what the contract controller needs.
type ContractService interface {
	Create(ctx context.Context, in model.Contract) (model.Contract, error)
	Get(ctx context.Context, publicID string) (model.Contract, error)
	List(ctx context.Context, f model.ContractFilter, withinDays *int, page, perPage int) (repository.Page[model.Contract], error)
	Update(ctx context.Context, publicID string, in model.Contract) (model.Contract, error)
	Delete(ctx context.Context, publicID string) error
	Activate(ctx context.Context, publicID string) (model.Contract, error)
	Complete(ctx context.Context, publicID string, actual time.Time) (model.Contract, error)
	Terminate(ctx context.Context, publicID string, actual time.Time, reason string) (model.Contract, error)
	CurrentAssignment(ctx context.Context, crewPublicID string) (*service.Assignment, error)
}

// ContractHandler serves contracts and their lifecycle actions.
type ContractHandler struct {
	contracts ContractService
}

// NewContractHandler returns a ContractHandler.
func NewContractHandler(contracts ContractService) *ContractHandler {
	return &ContractHandler{contracts: contracts}
}

func contractID(r *http.Request) string { return chi.URLParam(r, "contract") }

// Index handles GET /contracts.
func (h *ContractHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, within, errs := request.ParseContractFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	p := request.ParseList(q)
	page, err := h.contracts.List(r.Context(), f, within, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.NewContracts(page.Items), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// Show handles GET /contracts/{contract}.
func (h *ContractHandler) Show(w http.ResponseWriter, r *http.Request) {
	k, err := h.contracts.Get(r.Context(), contractID(r))
	respondContract(w, r, http.StatusOK, k, err)
}

// Store handles POST /contracts.
func (h *ContractHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in request.Contract
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	k, err := h.contracts.Create(r.Context(), in.Model())
	respondContract(w, r, http.StatusCreated, k, err)
}

// Update handles PUT /contracts/{contract}.
func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in request.Contract
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	k, err := h.contracts.Update(r.Context(), contractID(r), in.Model())
	respondContract(w, r, http.StatusOK, k, err)
}

// Destroy handles DELETE /contracts/{contract}.
func (h *ContractHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.contracts.Delete(r.Context(), contractID(r)); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

// Activate handles POST /contracts/{contract}/activate.
func (h *ContractHandler) Activate(w http.ResponseWriter, r *http.Request) {
	k, err := h.contracts.Activate(r.Context(), contractID(r))
	respondContract(w, r, http.StatusOK, k, err)
}

// Complete handles POST /contracts/{contract}/complete.
func (h *ContractHandler) Complete(w http.ResponseWriter, r *http.Request) {
	var in request.ContractFinish
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	k, err := h.contracts.Complete(r.Context(), contractID(r), in.Date())
	respondContract(w, r, http.StatusOK, k, err)
}

// Terminate handles POST /contracts/{contract}/terminate.
func (h *ContractHandler) Terminate(w http.ResponseWriter, r *http.Request) {
	var in request.ContractFinish
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	k, err := h.contracts.Terminate(r.Context(), contractID(r), in.Date(), in.Reason)
	respondContract(w, r, http.StatusOK, k, err)
}

// CurrentAssignment handles GET /crews/{crew}/current-assignment; data is null when the crew member is not on board.
func (h *ContractHandler) CurrentAssignment(w http.ResponseWriter, r *http.Request) {
	a, err := h.contracts.CurrentAssignment(r.Context(), chi.URLParam(r, "crew"))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewAssignment(a))
}

func respondContract(w http.ResponseWriter, r *http.Request, status int, k model.Contract, err error) {
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, status, resource.NewContract(k))
}

// RotationService is what the rotation controller needs.
type RotationService interface {
	Get(ctx context.Context, id uuid.UUID) (model.ReliefPlan, error)
	List(ctx context.Context, f model.ReliefPlanFilter, page, perPage int) (repository.Page[model.ReliefPlan], error)
	Create(ctx context.Context, in service.ReliefInput) (service.ReliefResult, error)
	Update(ctx context.Context, id uuid.UUID, in service.ReliefInput) (service.ReliefResult, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Suggestions(ctx context.Context, contractPublicID string, date *time.Time) ([]model.ReliefCandidate, time.Time, error)
	Complete(ctx context.Context, id uuid.UUID) (model.Contract, error)
	Timeline(ctx context.Context, from, to *time.Time, vesselIDs []uuid.UUID) (service.Timeline, error)
}

// RotationHandler serves the timeline and relief plans.
type RotationHandler struct {
	rotation RotationService
}

// NewRotationHandler returns a RotationHandler.
func NewRotationHandler(rotation RotationService) *RotationHandler {
	return &RotationHandler{rotation: rotation}
}

func planID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, apperr.NotFound("Relief plan")
	}
	return id, nil
}

// Timeline handles GET /rotation/timeline?from=&to=&vessel_id=.
func (h *RotationHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	e := request.Errors{}
	from := e.Date("from", q.Get("from"))
	to := e.Date("to", q.Get("to"))
	var vessels []uuid.UUID
	for _, v := range request.CSV(q, "vessel_id") {
		if id := e.UUID("vessel_id", v); id != nil {
			vessels = append(vessels, *id)
		}
	}
	if len(e) > 0 {
		resource.Error(w, r, apperr.Validation(e))
		return
	}
	t, err := h.rotation.Timeline(r.Context(), from, to, vessels)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewTimeline(t))
}

// Index handles GET /relief-plans.
func (h *RotationHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, errs := request.ParseReliefFilter(q)
	if len(errs) > 0 {
		resource.Error(w, r, apperr.Validation(errs))
		return
	}
	p := request.ParseList(q)
	page, err := h.rotation.List(r.Context(), f, p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.NewReliefPlans(page.Items), resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// Show handles GET /relief-plans/{id}.
func (h *RotationHandler) Show(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	p, err := h.rotation.Get(r.Context(), id)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewReliefPlan(p))
}

// Store handles POST /relief-plans.
func (h *RotationHandler) Store(w http.ResponseWriter, r *http.Request) {
	var in request.ReliefPlan
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	if in.ContractID == "" {
		resource.Error(w, r, apperr.Validation(map[string]string{"contract_id": "This field is required."}))
		return
	}
	res, err := h.rotation.Create(r.Context(), reliefInput(in))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, resource.NewReliefResult(res))
}

// Update handles PUT /relief-plans/{id}.
func (h *RotationHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	var in request.ReliefPlan
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	res, err := h.rotation.Update(r.Context(), id, reliefInput(in))
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewReliefResult(res))
}

// Destroy handles DELETE /relief-plans/{id}.
func (h *RotationHandler) Destroy(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	if err := h.rotation.Delete(r.Context(), id); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

// Complete handles POST /relief-plans/{id}/complete and returns the reliever's new Draft contract.
func (h *RotationHandler) Complete(w http.ResponseWriter, r *http.Request) {
	id, err := planID(r)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	k, err := h.rotation.Complete(r.Context(), id)
	respondContract(w, r, http.StatusOK, k, err)
}

// Suggestions handles GET /relief-plans/suggestions?contract_id=&date=.
func (h *RotationHandler) Suggestions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	e := request.Errors{}
	e.Required("contract_id", q.Get("contract_id"), 9)
	date := e.Date("date", q.Get("date"))
	if len(e) > 0 {
		resource.Error(w, r, apperr.Validation(e))
		return
	}
	list, d, err := h.rotation.Suggestions(r.Context(), q.Get("contract_id"), date)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, resource.NewSuggestions(list, d))
}

func reliefInput(in request.ReliefPlan) service.ReliefInput {
	return service.ReliefInput{ContractPublicID: in.ContractID, RelieverPublicID: in.RelieverCrewID, PlannedReliefDate: in.Date(),
		JoiningPort: in.JoiningPort, Status: in.Status, Notes: in.Notes}
}
