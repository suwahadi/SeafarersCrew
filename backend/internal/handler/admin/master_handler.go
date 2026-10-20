package admin

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// MasterCRUD is the service contract shared by every master.
type MasterCRUD[T any] interface {
	List(ctx context.Context, q repository.ListQuery) (repository.Page[T], error)
	Get(ctx context.Context, id uuid.UUID) (T, error)
	Create(ctx context.Context, item T) (T, error)
	Update(ctx context.Context, id uuid.UUID, item T) (T, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Reorder(ctx context.Context, ids []uuid.UUID) error
}

// MasterInput is a request body that validates and converts to a model.
type MasterInput[T any, In any] interface {
	*In
	request.Validator
	Model() T
}

// FilterKind tells how a list filter value is parsed.
type FilterKind int

// Filter kinds.
const (
	FilterBool FilterKind = iota
	FilterUUID
	FilterEnum
)

// Filter declares one accepted list filter.
type Filter struct {
	Kind    FilterKind
	Options []string
}

// MasterHandler serves list, show, create, update, delete and reorder for one master.
type MasterHandler[T any, In any, P MasterInput[T, In]] struct {
	svc     MasterCRUD[T]
	present func(T) any
	filters map[string]Filter
}

// NewMasterHandler returns a MasterHandler; is_active is always accepted as a filter.
func NewMasterHandler[T any, In any, P MasterInput[T, In]](svc MasterCRUD[T], present func(T) any, filters map[string]Filter) *MasterHandler[T, In, P] {
	all := map[string]Filter{"is_active": {Kind: FilterBool}}
	for k, v := range filters {
		all[k] = v
	}
	return &MasterHandler[T, In, P]{svc: svc, present: present, filters: all}
}

// List handles GET /{master}.
func (h *MasterHandler[T, In, P]) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := request.ParseList(q)
	filters, err := parseMasterFilters(q, h.filters)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	page, err := h.svc.List(r.Context(), repository.ListQuery{
		Page: params.Page, PerPage: params.PerPage, Sort: params.Sort, Desc: params.Desc, Q: params.Q, Filters: filters,
	})
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.List(w, resource.MapAll(page.Items, h.present), resource.Meta{Page: params.Page, PerPage: params.PerPage, Total: page.Total})
}

// Show handles GET /{master}/{id}.
func (h *MasterHandler[T, In, P]) Show(w http.ResponseWriter, r *http.Request) {
	id, ok := masterPathID(w, r)
	if !ok {
		return
	}
	item, err := h.svc.Get(r.Context(), id)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, h.present(item))
}

// Create handles POST /{master}.
func (h *MasterHandler[T, In, P]) Create(w http.ResponseWriter, r *http.Request) {
	var in In
	body := P(&in)
	if err := request.Decode(w, r, body); err != nil {
		resource.Error(w, r, err)
		return
	}
	item, err := h.svc.Create(r.Context(), body.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, h.present(item))
}

// Update handles PUT /{master}/{id}.
func (h *MasterHandler[T, In, P]) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := masterPathID(w, r)
	if !ok {
		return
	}
	var in In
	body := P(&in)
	if err := request.Decode(w, r, body); err != nil {
		resource.Error(w, r, err)
		return
	}
	item, err := h.svc.Update(r.Context(), id, body.Model())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, h.present(item))
}

// Delete handles DELETE /{master}/{id}.
func (h *MasterHandler[T, In, P]) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := masterPathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

// Reorder handles PATCH /{master}/reorder.
func (h *MasterHandler[T, In, P]) Reorder(w http.ResponseWriter, r *http.Request) {
	var in request.Reorder
	if err := request.Decode(w, r, &in); err != nil {
		resource.Error(w, r, err)
		return
	}
	if err := h.svc.Reorder(r.Context(), in.IDs); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

func masterPathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		resource.Error(w, r, apperr.NotFound("Record"))
		return uuid.Nil, false
	}
	return id, true
}

func parseMasterFilters(q url.Values, allowed map[string]Filter) (map[string]any, error) {
	out := map[string]any{}
	fields := map[string]string{}
	for key, f := range allowed {
		raw := strings.TrimSpace(q.Get(key))
		if raw == "" {
			continue
		}
		switch f.Kind {
		case FilterBool:
			v, err := strconv.ParseBool(raw)
			if err != nil {
				fields[key] = "The filter must be true or false."
				continue
			}
			out[key] = v
		case FilterUUID:
			v, err := uuid.Parse(raw)
			if err != nil {
				fields[key] = "The filter must be a valid ID."
				continue
			}
			out[key] = v
		case FilterEnum:
			if !slices.Contains(f.Options, raw) {
				fields[key] = "The filter must be one of: " + strings.Join(f.Options, ", ") + "."
				continue
			}
			out[key] = raw
		}
	}
	if len(fields) > 0 {
		return nil, apperr.Validation(fields)
	}
	return out, nil
}
