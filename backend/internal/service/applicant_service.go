package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// StageNoteMaxLength limits the internal note of a stage change.
const StageNoteMaxLength = 1000

// ApplicantStore reads and moves applicants.
type ApplicantStore interface {
	GetByPublicID(ctx context.Context, publicID string) (model.Applicant, error)
	List(ctx context.Context, f model.ApplicantFilter, limit, offset int) (repository.Page[model.Applicant], error)
	StageCounts(ctx context.Context, f model.ApplicantFilter) (map[string]int, error)
	ChangeStage(ctx context.Context, id uuid.UUID, from, to, note string, adminID uuid.UUID) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// ApplicantService manages the applicant pipeline (PRD 7.9).
type ApplicantService struct {
	store    ApplicantStore
	settings SettingsReader
	now      func() time.Time
}

// NewApplicantService builds an ApplicantService.
func NewApplicantService(store ApplicantStore, settings SettingsReader, now func() time.Time) *ApplicantService {
	return &ApplicantService{store: store, settings: settings, now: now}
}

// Get returns an applicant with files and stage history.
func (s *ApplicantService) Get(ctx context.Context, publicID string) (model.Applicant, error) {
	publicID = strings.ToUpper(publicID)
	if !publicid.Valid(publicID) {
		return model.Applicant{}, apperr.NotFound("Applicant")
	}
	a, err := s.store.GetByPublicID(ctx, publicID)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Applicant{}, apperr.NotFound("Applicant")
	}
	return a, err
}

// List returns one page of applicants; DateFrom and DateTo are inclusive calendar dates in the configured timezone.
func (s *ApplicantService) List(ctx context.Context, f model.ApplicantFilter, page, perPage int) (repository.Page[model.Applicant], error) {
	f, err := s.localize(ctx, f)
	if err != nil {
		return repository.Page[model.Applicant]{}, err
	}
	return s.store.List(ctx, f, perPage, (page-1)*perPage)
}

// StageCounts returns the applicants per stage for the filter without its stage condition, with every stage present.
func (s *ApplicantService) StageCounts(ctx context.Context, f model.ApplicantFilter) (map[string]int, error) {
	f, err := s.localize(ctx, f)
	if err != nil {
		return nil, err
	}
	counts, err := s.store.StageCounts(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(model.ApplicantStages))
	for _, stage := range model.ApplicantStages {
		out[stage] = counts[stage]
	}
	return out, nil
}

func (s *ApplicantService) localize(ctx context.Context, f model.ApplicantFilter) (model.ApplicantFilter, error) {
	if f.DateFrom == nil && f.DateTo == nil {
		return f, nil
	}
	loc, _, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return f, err
	}
	if f.DateFrom != nil {
		y, m, d := f.DateFrom.Date()
		start := time.Date(y, m, d, 0, 0, 0, 0, loc).UTC()
		f.CreatedFrom = &start
	}
	if f.DateTo != nil {
		y, m, d := f.DateTo.Date()
		end := time.Date(y, m, d+1, 0, 0, 0, 0, loc).UTC()
		f.CreatedBefore = &end
	}
	return f, nil
}

// ChangeStage moves an applicant to any other stage and records the move with the admin and note; a note is required for Rejected.
func (s *ApplicantService) ChangeStage(ctx context.Context, publicID string, adminID uuid.UUID, stage, note string) (model.Applicant, error) {
	note = strings.TrimSpace(note)
	fields := map[string]string{}
	if !slices.Contains(model.ApplicantStages, stage) {
		fields["stage"] = "The stage must be one of: " + strings.Join(model.ApplicantStages, ", ") + "."
	}
	if utf8.RuneCountInString(note) > StageNoteMaxLength {
		fields["note"] = "The note may not exceed 1000 characters."
	}
	if stage == model.StageRejected && note == "" {
		fields["note"] = "A note is required when rejecting an applicant."
	}
	if len(fields) > 0 {
		return model.Applicant{}, apperr.Validation(fields)
	}
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Applicant{}, err
	}
	if cur.Stage == stage {
		return model.Applicant{}, apperr.Conflict("stage_unchanged", "The applicant is already in this stage.")
	}
	if err := s.store.ChangeStage(ctx, cur.ID, cur.Stage, stage, note, adminID); err != nil {
		if errors.Is(err, repository.ErrStateChanged) {
			return model.Applicant{}, apperr.Conflict("stage_changed", "The applicant was moved by someone else. Reload and try again.")
		}
		return model.Applicant{}, err
	}
	return s.Get(ctx, publicID)
}

// Delete soft-deletes an applicant.
func (s *ApplicantService) Delete(ctx context.Context, publicID string) error {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	if err := s.store.SoftDelete(ctx, cur.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperr.NotFound("Applicant")
		}
		return err
	}
	return nil
}
