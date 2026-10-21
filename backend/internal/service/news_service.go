package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// NewsStore persists news articles.
type NewsStore interface {
	List(ctx context.Context, q repository.ListQuery) (repository.Page[model.News], error)
	Get(ctx context.Context, id uuid.UUID) (model.News, error)
	SlugTaken(ctx context.Context, slug string, exclude uuid.UUID) (bool, error)
	Insert(ctx context.Context, n model.News) (model.News, error)
	Update(ctx context.Context, id uuid.UUID, n model.News, redirectFrom string) (model.News, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// NewsService applies the business rules of news articles.
type NewsService struct {
	store NewsStore
	files *PublicFileChecker
	html  HTMLSanitizer
	site  SiteChanger
	now   func() time.Time
}

// NewNewsService builds a NewsService.
func NewNewsService(store NewsStore, files *PublicFileChecker, html HTMLSanitizer, site SiteChanger, now func() time.Time) *NewsService {
	return &NewsService{store: store, files: files, html: html, site: site, now: now}
}

// List returns one page of articles without bodies.
func (s *NewsService) List(ctx context.Context, q repository.ListQuery) (repository.Page[model.News], error) {
	return s.store.List(ctx, q)
}

// Get returns one article.
func (s *NewsService) Get(ctx context.Context, id uuid.UUID) (model.News, error) {
	n, err := s.store.Get(ctx, id)
	return n, mapNewsErr(err)
}

// Create validates and stores a new article.
func (s *NewsService) Create(ctx context.Context, in model.News) (model.News, error) {
	if err := s.prepare(ctx, &in); err != nil {
		return model.News{}, err
	}
	slug, err := s.uniqueSlug(ctx, in.Title, in.Slug, uuid.Nil)
	if err != nil {
		return model.News{}, err
	}
	in.Slug = slug
	in.PublishedAt = s.publishedAt(in, nil)
	created, err := s.store.Insert(ctx, in)
	if err != nil {
		return model.News{}, mapNewsErr(err)
	}
	return created, s.site.Changed(ctx, newsTags(false)...)
}

// Update validates and stores changes; changing the slug of a visible article keeps the old URL redirecting.
func (s *NewsService) Update(ctx context.Context, id uuid.UUID, in model.News) (model.News, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return model.News{}, err
	}
	if err := s.prepare(ctx, &in); err != nil {
		return model.News{}, err
	}
	requested := in.Slug
	if requested == "" {
		requested = existing.Slug
	}
	slug, err := s.uniqueSlug(ctx, in.Title, requested, id)
	if err != nil {
		return model.News{}, err
	}
	in.Slug = slug
	in.PublishedAt = s.publishedAt(in, &existing)
	redirectFrom := ""
	if slug != existing.Slug && existing.Live(s.now()) {
		redirectFrom = "/news/" + existing.Slug
	}
	updated, err := s.store.Update(ctx, id, in, redirectFrom)
	if err != nil {
		return model.News{}, mapNewsErr(err)
	}
	return updated, s.site.Changed(ctx, newsTags(redirectFrom != "")...)
}

// Delete soft-deletes an article.
func (s *NewsService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.store.SoftDelete(ctx, id); err != nil {
		return mapNewsErr(err)
	}
	return s.site.Changed(ctx, newsTags(false)...)
}

func (s *NewsService) prepare(ctx context.Context, n *model.News) error {
	n.BodyHTML = s.html.Sanitize(n.BodyHTML)
	_, err := s.files.Check(ctx, FileRef{Field: "thumbnail_file_id", ID: n.ThumbnailFileID, Kind: ImageFile})
	return err
}

func (s *NewsService) uniqueSlug(ctx context.Context, title, requested string, exclude uuid.UUID) (string, error) {
	return UniqueSlug(ctx, title, requested, func(ctx context.Context, slug string) (bool, error) {
		return s.store.SlugTaken(ctx, slug, exclude)
	})
}

func (s *NewsService) publishedAt(in model.News, existing *model.News) *time.Time {
	if in.PublishedAt != nil {
		t := in.PublishedAt.UTC()
		return &t
	}
	if in.Status != model.StatusPublished {
		return nil
	}
	if existing != nil && existing.PublishedAt != nil {
		return existing.PublishedAt
	}
	now := s.now().UTC()
	return &now
}

func newsTags(redirected bool) []string {
	tags := []string{model.TagNews, model.TagHome, model.TagSitemap}
	if redirected {
		tags = append(tags, model.TagRedirects)
	}
	return tags
}

func mapNewsErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return apperr.NotFound("Article")
	case errors.Is(err, repository.ErrDuplicate):
		return apperr.Conflict("duplicate", "An article with the same slug already exists.").Wrap(err)
	}
	return err
}
