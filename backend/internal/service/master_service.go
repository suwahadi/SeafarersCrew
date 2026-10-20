package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// MaxReorderItems bounds a single reorder request.
const MaxReorderItems = 1000

// MasterStore persists one master table.
type MasterStore[T any] interface {
	List(ctx context.Context, q repository.ListQuery) (repository.Page[T], error)
	Get(ctx context.Context, id uuid.UUID) (T, error)
	Insert(ctx context.Context, item T) (T, error)
	Update(ctx context.Context, id uuid.UUID, item T) (T, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Reorder(ctx context.Context, ids []uuid.UUID) error
	Taken(ctx context.Context, field, value string, exclude uuid.UUID) (bool, error)
	Usages(ctx context.Context, id uuid.UUID) ([]model.Usage, error)
}

// UniqueField declares a field that must not repeat across records.
type UniqueField[T any] struct {
	Field   string
	Message string
	Value   func(T) string
}

// MasterRules holds the business rules of one master.
type MasterRules[T any] struct {
	Noun      string
	Prepare   func(ctx context.Context, item *T, existing *T) error
	Unique    []UniqueField[T]
	CanDelete func(item T) error
}

// MasterService applies shared CRUD rules to one master table.
type MasterService[T any] struct {
	store    MasterStore[T]
	rules    MasterRules[T]
	onChange func(ctx context.Context) error
}

// NewMasterService returns a MasterService; onChange runs after every successful write.
func NewMasterService[T any](store MasterStore[T], rules MasterRules[T], onChange func(ctx context.Context) error) *MasterService[T] {
	return &MasterService[T]{store: store, rules: rules, onChange: onChange}
}

// List returns one page of records.
func (s *MasterService[T]) List(ctx context.Context, q repository.ListQuery) (repository.Page[T], error) {
	return s.store.List(ctx, q)
}

// Get returns one record.
func (s *MasterService[T]) Get(ctx context.Context, id uuid.UUID) (T, error) {
	item, err := s.store.Get(ctx, id)
	return item, s.mapErr(err)
}

// Create validates and stores a new record.
func (s *MasterService[T]) Create(ctx context.Context, item T) (T, error) {
	var zero T
	if err := s.prepare(ctx, &item, nil, uuid.Nil); err != nil {
		return zero, err
	}
	created, err := s.store.Insert(ctx, item)
	if err != nil {
		return zero, s.mapErr(err)
	}
	return created, s.changed(ctx)
}

// Update validates and stores changes to an existing record.
func (s *MasterService[T]) Update(ctx context.Context, id uuid.UUID, item T) (T, error) {
	var zero T
	existing, err := s.store.Get(ctx, id)
	if err != nil {
		return zero, s.mapErr(err)
	}
	if err := s.prepare(ctx, &item, &existing, id); err != nil {
		return zero, err
	}
	updated, err := s.store.Update(ctx, id, item)
	if err != nil {
		return zero, s.mapErr(err)
	}
	return updated, s.changed(ctx)
}

// Delete soft-deletes a record that is not protected and not in use.
func (s *MasterService[T]) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.store.Get(ctx, id)
	if err != nil {
		return s.mapErr(err)
	}
	if s.rules.CanDelete != nil {
		if err := s.rules.CanDelete(existing); err != nil {
			return err
		}
	}
	usages, err := s.store.Usages(ctx, id)
	if err != nil {
		return err
	}
	if len(usages) > 0 {
		return InUseError(s.rules.Noun, usages)
	}
	if err := s.store.SoftDelete(ctx, id); err != nil {
		return s.mapErr(err)
	}
	return s.changed(ctx)
}

// Reorder sets the display order to the order of ids.
func (s *MasterService[T]) Reorder(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 || len(ids) > MaxReorderItems {
		return apperr.Validation(map[string]string{"ids": fmt.Sprintf("Between 1 and %d IDs are required.", MaxReorderItems)})
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return apperr.Validation(map[string]string{"ids": "IDs must not repeat."})
		}
		seen[id] = struct{}{}
	}
	if err := s.store.Reorder(ctx, ids); err != nil {
		return err
	}
	return s.changed(ctx)
}

func (s *MasterService[T]) prepare(ctx context.Context, item *T, existing *T, id uuid.UUID) error {
	if s.rules.Prepare != nil {
		if err := s.rules.Prepare(ctx, item, existing); err != nil {
			return err
		}
	}
	fields := map[string]string{}
	for _, u := range s.rules.Unique {
		value := u.Value(*item)
		if value == "" {
			continue
		}
		taken, err := s.store.Taken(ctx, u.Field, value, id)
		if err != nil {
			return err
		}
		if taken {
			fields[u.Field] = u.Message
		}
	}
	if len(fields) > 0 {
		return conflictFields(fields)
	}
	return nil
}

func (s *MasterService[T]) changed(ctx context.Context) error {
	if s.onChange == nil {
		return nil
	}
	return s.onChange(ctx)
}

func (s *MasterService[T]) mapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return apperr.NotFound(s.rules.Noun)
	case errors.Is(err, repository.ErrDuplicate):
		return apperr.Conflict("duplicate", "A "+strings.ToLower(s.rules.Noun)+" with the same unique value already exists.").Wrap(err)
	}
	return err
}

func conflictFields(fields map[string]string) error {
	e := apperr.Conflict("duplicate", "The given data conflicts with an existing record.")
	e.Fields = fields
	return e
}

// InUseError builds the 409 returned when a referenced record would be deleted.
func InUseError(noun string, usages []model.Usage) error {
	parts := make([]string, len(usages))
	fields := make(map[string]string, len(usages))
	for i, u := range usages {
		parts[i] = strconv.Itoa(u.Count) + " " + u.Label
		fields[u.Label] = strconv.Itoa(u.Count)
	}
	e := apperr.Conflict("in_use", "This "+strings.ToLower(noun)+" is in use and cannot be deleted: "+strings.Join(parts, ", ")+".")
	e.Fields = fields
	return e
}
