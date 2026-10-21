package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// PageStore persists CMS pages.
type PageStore interface {
	List(ctx context.Context, q repository.ListQuery) (repository.Page[model.Page], error)
	Get(ctx context.Context, id uuid.UUID) (model.Page, error)
	SlugTaken(ctx context.Context, slug string, exclude uuid.UUID) (bool, error)
	Insert(ctx context.Context, p model.Page) (model.Page, error)
	Update(ctx context.Context, id uuid.UUID, p model.Page, redirectFrom string) (model.Page, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ReorderMenuOrder(ctx context.Context, ids []uuid.UUID) error
}

// PageLookup answers reference questions about page content.
type PageLookup interface {
	GalleryImagesExist(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error)
	Images(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.Image, error)
}

// PageDetail is a page together with the images its fields and blocks reference.
type PageDetail struct {
	Page  model.Page
	Media map[uuid.UUID]model.Image
}

// PageService applies the business rules of CMS pages.
type PageService struct {
	store  PageStore
	lookup PageLookup
	files  *PublicFileChecker
	html   HTMLSanitizer
	site   SiteChanger
	now    func() time.Time
}

// NewPageService builds a PageService.
func NewPageService(store PageStore, lookup PageLookup, files *PublicFileChecker, html HTMLSanitizer, site SiteChanger, now func() time.Time) *PageService {
	return &PageService{store: store, lookup: lookup, files: files, html: html, site: site, now: now}
}

// List returns one page of pages without blocks.
func (s *PageService) List(ctx context.Context, q repository.ListQuery) (repository.Page[model.Page], error) {
	return s.store.List(ctx, q)
}

// Get returns a page with its referenced images.
func (s *PageService) Get(ctx context.Context, id uuid.UUID) (PageDetail, error) {
	p, err := s.get(ctx, id)
	if err != nil {
		return PageDetail{}, err
	}
	return s.detail(ctx, p)
}

// Create validates and stores a new page.
func (s *PageService) Create(ctx context.Context, in model.Page) (PageDetail, error) {
	if err := s.prepareContent(ctx, &in, true); err != nil {
		return PageDetail{}, err
	}
	slug, err := s.uniqueSlug(ctx, in.Title, in.Slug, uuid.Nil)
	if err != nil {
		return PageDetail{}, err
	}
	in.Slug = slug
	if in.Status == model.StatusPublished {
		now := s.now().UTC()
		in.PublishedAt = &now
	}
	created, err := s.store.Insert(ctx, in)
	if err != nil {
		return PageDetail{}, mapPageErr(err)
	}
	if err := s.site.Changed(ctx, pageTags(created, false)...); err != nil {
		return PageDetail{}, err
	}
	return s.detail(ctx, created)
}

// Update validates and stores changes; system pages keep their slug, status and layout.
func (s *PageService) Update(ctx context.Context, id uuid.UUID, in model.Page) (PageDetail, error) {
	existing, err := s.get(ctx, id)
	if err != nil {
		return PageDetail{}, err
	}
	if existing.IsSystem {
		in.Slug, in.Status, in.Blocks, in.SystemKey = existing.Slug, model.StatusPublished, existing.Blocks, existing.SystemKey
	}
	if err := s.prepareContent(ctx, &in, !existing.IsSystem); err != nil {
		return PageDetail{}, err
	}
	redirectFrom := ""
	if !existing.IsSystem {
		requested := in.Slug
		if requested == "" {
			requested = existing.Slug
		}
		slug, err := s.uniqueSlug(ctx, in.Title, requested, id)
		if err != nil {
			return PageDetail{}, err
		}
		in.Slug = slug
		if slug != existing.Slug && existing.Status == model.StatusPublished && in.Status == model.StatusPublished {
			redirectFrom = "/" + existing.Slug
		}
	}
	in.PublishedAt = existing.PublishedAt
	if in.Status == model.StatusPublished && in.PublishedAt == nil {
		now := s.now().UTC()
		in.PublishedAt = &now
	}
	updated, err := s.store.Update(ctx, id, in, redirectFrom)
	if err != nil {
		return PageDetail{}, mapPageErr(err)
	}
	if err := s.site.Changed(ctx, pageTags(updated, redirectFrom != "")...); err != nil {
		return PageDetail{}, err
	}
	return s.detail(ctx, updated)
}

// Delete soft-deletes a page that is not a system page.
func (s *PageService) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return apperr.Conflict("system_page", "System pages cannot be deleted.")
	}
	if err := s.store.SoftDelete(ctx, id); err != nil {
		return mapPageErr(err)
	}
	return s.site.Changed(ctx, pageTags(existing, false)...)
}

// Reorder sets the menu order of pages to the order of ids.
func (s *PageService) Reorder(ctx context.Context, ids []uuid.UUID) error {
	if err := validateReorderIDs(ids); err != nil {
		return err
	}
	if err := s.store.ReorderMenuOrder(ctx, ids); err != nil {
		return err
	}
	return s.site.Changed(ctx, model.TagPages, model.TagSite)
}

func (s *PageService) get(ctx context.Context, id uuid.UUID) (model.Page, error) {
	p, err := s.store.Get(ctx, id)
	return p, mapPageErr(err)
}

func (s *PageService) uniqueSlug(ctx context.Context, title, requested string, exclude uuid.UUID) (string, error) {
	return UniqueSlug(ctx, title, requested, func(ctx context.Context, slug string) (bool, error) {
		if IsReservedSlug(slug) {
			return true, nil
		}
		return s.store.SlugTaken(ctx, slug, exclude)
	})
}

func (s *PageService) prepareContent(ctx context.Context, p *model.Page, withBlocks bool) error {
	p.IntroHTML = s.html.Sanitize(p.IntroHTML)
	refs := []FileRef{{Field: "og_image_file_id", ID: p.OGImageFileID, Kind: ImageFile}}
	var missing map[string]string
	if withBlocks {
		blockRefs, galleryMissing, err := s.prepareBlocks(ctx, p.Blocks)
		if err != nil {
			return err
		}
		refs, missing = append(refs, blockRefs...), galleryMissing
	}
	_, err := s.files.Check(ctx, refs...)
	return mergeValidation(missing, err)
}

func (s *PageService) detail(ctx context.Context, p model.Page) (PageDetail, error) {
	ids := pageImageIDs(p)
	media, err := s.lookup.Images(ctx, ids)
	if err != nil {
		return PageDetail{}, err
	}
	return PageDetail{Page: p, Media: media}, nil
}

func pageImageIDs(p model.Page) []uuid.UUID {
	var ids []uuid.UUID
	add := func(id *uuid.UUID) {
		if id != nil && !slices.Contains(ids, *id) {
			ids = append(ids, *id)
		}
	}
	add(p.OGImageFileID)
	for _, b := range p.Blocks {
		switch v := b.Settings.(type) {
		case *model.ImageTextSettings:
			add(v.ImageFileID)
		case *model.CTASettings:
			add(v.BackgroundImageFileID)
		case *model.CardGridSettings:
			for _, c := range v.Cards {
				add(c.ImageFileID)
			}
		}
	}
	return ids
}

func pageTags(p model.Page, redirected bool) []string {
	tags := []string{model.TagPages, model.TagSite, model.TagSitemap}
	if p.SystemKey != nil && *p.SystemKey == model.SystemHome {
		tags = append(tags, model.TagHome)
	}
	if redirected {
		tags = append(tags, model.TagRedirects)
	}
	return tags
}

func mapPageErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, repository.ErrNotFound):
		return apperr.NotFound("Page")
	case errors.Is(err, repository.ErrDuplicate):
		return apperr.Conflict("duplicate", "A page with the same slug already exists.").Wrap(err)
	}
	return err
}

func validateReorderIDs(ids []uuid.UUID) error {
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
	return nil
}

// IsReservedSlug reports whether slug collides with a route of the website or the API.
func IsReservedSlug(slug string) bool {
	switch slug {
	case "api", "admin", "sitemap", "robots", "favicon", "career", "contact", "gallery", "news", "home":
		return true
	}
	return false
}
