package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// VacancyStore persists vacancies.
type VacancyStore interface {
	Create(ctx context.Context, v model.Vacancy, publicID string) (uuid.UUID, error)
	Update(ctx context.Context, v model.Vacancy, redirect *model.Redirect) error
	GetByPublicID(ctx context.Context, publicID string) (model.Vacancy, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Vacancy, error)
	GetBySlug(ctx context.Context, slug string) (model.Vacancy, error)
	List(ctx context.Context, f model.VacancyFilter, limit, offset int) (repository.Page[model.Vacancy], error)
	ListOpen(ctx context.Context, f model.PublicVacancyFilter, limit, offset int) (repository.Page[model.Vacancy], error)
	FilterOptions(ctx context.Context, today time.Time) (model.VacancyFilterOptions, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	SlugTaken(ctx context.Context, slug string, exclude uuid.UUID) (bool, error)
	RankActive(ctx context.Context, id uuid.UUID) (bool, error)
	VesselTypeActive(ctx context.Context, id uuid.UUID) (bool, error)
}

// ContentPublisher announces changed public content.
type ContentPublisher interface {
	Changed(ctx context.Context, tags ...string) error
}

// VacancyService manages vacancies and serves them to the public career pages (PRD 7.8, 8.2).
type VacancyService struct {
	store     VacancyStore
	settings  SettingsReader
	publisher ContentPublisher
	sanitize  *bluemonday.Policy
	now       func() time.Time
	newID     func() (string, error)
}

// NewVacancyService builds a VacancyService.
func NewVacancyService(store VacancyStore, settings SettingsReader, publisher ContentPublisher, now func() time.Time) *VacancyService {
	return &VacancyService{store: store, settings: settings, publisher: publisher, sanitize: NewTemplatePolicy(), now: now, newID: publicid.New}
}

func (s *VacancyService) today(ctx context.Context) (time.Time, error) {
	loc, _, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return time.Time{}, err
	}
	return datetime.Today(s.now(), loc), nil
}

// Create validates and stores a vacancy with a unique ID and slug.
func (s *VacancyService) Create(ctx context.Context, in model.Vacancy) (model.Vacancy, error) {
	v, err := s.prepare(ctx, in, nil)
	if err != nil {
		return model.Vacancy{}, err
	}
	for range publicid.MaxAttempts {
		pid, err := s.newID()
		if err != nil {
			return model.Vacancy{}, err
		}
		id, err := s.store.Create(ctx, v, pid)
		if postgres.IsUniqueViolation(err, "vacancies_public_id_key") {
			continue
		}
		if postgres.IsUniqueViolation(err, "vacancies_slug_key") {
			return model.Vacancy{}, errSlugTaken
		}
		if err != nil {
			return model.Vacancy{}, err
		}
		created, err := s.reload(ctx, id)
		if err != nil {
			return model.Vacancy{}, err
		}
		if created.Status != model.VacancyDraft {
			s.published(ctx, model.TagVacancies, model.TagHome, model.TagSitemap)
		}
		return created, nil
	}
	return model.Vacancy{}, fmt.Errorf("create vacancy: no unique public id after %d attempts", publicid.MaxAttempts)
}

var errSlugTaken = apperr.Validation(map[string]string{"slug": "This slug is already used by another vacancy."})

// Get returns a vacancy by its Vacancy ID.
func (s *VacancyService) Get(ctx context.Context, publicID string) (model.Vacancy, error) {
	publicID = strings.ToUpper(publicID)
	if !publicid.Valid(publicID) {
		return model.Vacancy{}, apperr.NotFound("Vacancy")
	}
	v, err := s.store.GetByPublicID(ctx, publicID)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Vacancy{}, apperr.NotFound("Vacancy")
	}
	if err != nil {
		return model.Vacancy{}, err
	}
	return v, s.markExpired(ctx, &v)
}

func (s *VacancyService) reload(ctx context.Context, id uuid.UUID) (model.Vacancy, error) {
	v, err := s.store.GetByID(ctx, id)
	if err != nil {
		return model.Vacancy{}, err
	}
	return v, s.markExpired(ctx, &v)
}

func (s *VacancyService) markExpired(ctx context.Context, items ...*model.Vacancy) error {
	today, err := s.today(ctx)
	if err != nil {
		return err
	}
	for _, v := range items {
		v.IsExpired = v.Status == model.VacancyOpen && !VacancyAcceptsApplications(*v, today)
	}
	return nil
}

// List returns one page of vacancies with their application counts.
func (s *VacancyService) List(ctx context.Context, f model.VacancyFilter, page, perPage int) (repository.Page[model.Vacancy], error) {
	out, err := s.store.List(ctx, f, perPage, (page-1)*perPage)
	if err != nil {
		return out, err
	}
	items := make([]*model.Vacancy, len(out.Items))
	for i := range out.Items {
		items[i] = &out.Items[i]
	}
	return out, s.markExpired(ctx, items...)
}

// Update replaces a vacancy; changing the slug of a vacancy that was ever published stores a 301 redirect.
func (s *VacancyService) Update(ctx context.Context, publicID string, in model.Vacancy) (model.Vacancy, error) {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Vacancy{}, err
	}
	return s.save(ctx, cur, in)
}

// SetStatus changes only the status of a vacancy.
func (s *VacancyService) SetStatus(ctx context.Context, publicID, status string) (model.Vacancy, error) {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Vacancy{}, err
	}
	next := cur
	next.Status = status
	return s.save(ctx, cur, next)
}

func (s *VacancyService) save(ctx context.Context, cur, in model.Vacancy) (model.Vacancy, error) {
	next, err := s.prepare(ctx, in, &cur)
	if err != nil {
		return model.Vacancy{}, err
	}
	next.ID, next.PublicID = cur.ID, cur.PublicID
	var redirect *model.Redirect
	tags := []string{model.TagVacancies, model.TagHome, model.TagSitemap}
	if next.Slug != cur.Slug && cur.PublishedAt != nil {
		redirect = &model.Redirect{FromPath: model.VacancyPathPrefix + cur.Slug, ToPath: model.VacancyPathPrefix + next.Slug}
		tags = append(tags, model.TagRedirects)
	}
	if err := s.store.Update(ctx, next, redirect); err != nil {
		if postgres.IsUniqueViolation(err, "vacancies_slug_key") {
			return model.Vacancy{}, errSlugTaken
		}
		if errors.Is(err, repository.ErrNotFound) {
			return model.Vacancy{}, apperr.NotFound("Vacancy")
		}
		return model.Vacancy{}, err
	}
	if cur.Status != model.VacancyDraft || next.Status != model.VacancyDraft {
		s.published(ctx, tags...)
	}
	return s.reload(ctx, cur.ID)
}

// Delete soft-deletes a vacancy; its applicants keep the reference.
func (s *VacancyService) Delete(ctx context.Context, publicID string) error {
	cur, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	if err := s.store.SoftDelete(ctx, cur.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperr.NotFound("Vacancy")
		}
		return err
	}
	if cur.Status != model.VacancyDraft {
		s.published(ctx, model.TagVacancies, model.TagHome, model.TagSitemap)
	}
	return nil
}

func (s *VacancyService) published(ctx context.Context, tags ...string) {
	if err := s.publisher.Changed(ctx, tags...); err != nil {
		logctx.From(ctx).WarnContext(ctx, "public cache not invalidated after vacancy change", "error", err)
	}
}

func (s *VacancyService) prepare(ctx context.Context, in model.Vacancy, cur *model.Vacancy) (model.Vacancy, error) {
	v := in
	v.Title = strings.TrimSpace(v.Title)
	fields := map[string]string{}
	if cur == nil || v.RankID != cur.RankID {
		if ok, err := s.store.RankActive(ctx, v.RankID); err != nil {
			return model.Vacancy{}, err
		} else if !ok {
			fields["rank_id"] = "Choose an active rank."
		}
	}
	if v.VesselTypeID != nil && (cur == nil || cur.VesselTypeID == nil || *v.VesselTypeID != *cur.VesselTypeID) {
		if ok, err := s.store.VesselTypeActive(ctx, *v.VesselTypeID); err != nil {
			return model.Vacancy{}, err
		} else if !ok {
			fields["vessel_type_id"] = "Choose an active vessel type."
		}
	}
	if err := s.checkClosingDate(ctx, v, cur, fields); err != nil {
		return model.Vacancy{}, err
	}
	if len(fields) > 0 {
		return model.Vacancy{}, apperr.Validation(fields)
	}
	if err := s.assignSlug(ctx, &v, cur); err != nil {
		return model.Vacancy{}, err
	}
	v.DescriptionHTML = s.sanitize.Sanitize(v.DescriptionHTML)
	v.RequirementsHTML = s.sanitize.Sanitize(v.RequirementsHTML)
	v.PublishedAt = nil
	if cur != nil {
		v.PublishedAt = cur.PublishedAt
	}
	if v.Status == model.VacancyOpen && v.PublishedAt == nil {
		now := s.now().UTC()
		v.PublishedAt = &now
	}
	return v, nil
}

func (s *VacancyService) checkClosingDate(ctx context.Context, v model.Vacancy, cur *model.Vacancy, fields map[string]string) error {
	if v.Status != model.VacancyOpen || v.ClosingDate == nil {
		return nil
	}
	if cur != nil && cur.Status == model.VacancyOpen && cur.ClosingDate != nil && cur.ClosingDate.Equal(*v.ClosingDate) {
		return nil
	}
	today, err := s.today(ctx)
	if err != nil {
		return err
	}
	if v.ClosingDate.Before(today) {
		fields["closing_date"] = "The closing date must be today or later for an open vacancy."
	}
	return nil
}

func (s *VacancyService) assignSlug(ctx context.Context, v *model.Vacancy, cur *model.Vacancy) error {
	exclude := uuid.Nil
	requested := strings.TrimSpace(v.Slug)
	if cur != nil {
		exclude = cur.ID
		if requested == "" {
			v.Slug = cur.Slug
			return nil
		}
	}
	slug, err := UniqueSlug(ctx, v.Title, requested, func(ctx context.Context, candidate string) (bool, error) {
		return s.store.SlugTaken(ctx, candidate, exclude)
	})
	if err != nil {
		return err
	}
	v.Slug = slug
	return nil
}

// PublicVacancy is a vacancy as the website sees it.
type PublicVacancy struct {
	model.Vacancy
	IsOpen bool
}

// PublicList returns one page of vacancies that are open today.
func (s *VacancyService) PublicList(ctx context.Context, q model.PublicVacancyQuery) (repository.Page[model.Vacancy], error) {
	today, err := s.today(ctx)
	if err != nil {
		return repository.Page[model.Vacancy]{}, err
	}
	f := model.PublicVacancyFilter{Q: q.Q, RankID: q.RankID, VesselTypeID: q.VesselTypeID, Department: q.Department, Location: q.Location, Today: today}
	return s.store.ListOpen(ctx, f, q.PerPage, (q.Page-1)*q.PerPage)
}

// PublicDetail returns a published vacancy by slug; closed and expired vacancies are returned with IsOpen false and drafts are not found.
func (s *VacancyService) PublicDetail(ctx context.Context, slug string) (PublicVacancy, error) {
	v, err := s.store.GetBySlug(ctx, slug)
	if errors.Is(err, repository.ErrNotFound) || (err == nil && v.Status == model.VacancyDraft) {
		return PublicVacancy{}, apperr.NotFound("Vacancy")
	}
	if err != nil {
		return PublicVacancy{}, err
	}
	today, err := s.today(ctx)
	if err != nil {
		return PublicVacancy{}, err
	}
	return PublicVacancy{Vacancy: v, IsOpen: VacancyAcceptsApplications(v, today)}, nil
}

// VacancyAcceptsApplications reports whether v is open and its closing date has not passed on today.
func VacancyAcceptsApplications(v model.Vacancy, today time.Time) bool {
	if v.Status != model.VacancyOpen {
		return false
	}
	return v.ClosingDate == nil || !v.ClosingDate.Before(today)
}

// PublicFilters returns the filter values used by vacancies that are open today.
func (s *VacancyService) PublicFilters(ctx context.Context) (model.VacancyFilterOptions, error) {
	today, err := s.today(ctx)
	if err != nil {
		return model.VacancyFilterOptions{}, err
	}
	return s.store.FilterOptions(ctx, today)
}
