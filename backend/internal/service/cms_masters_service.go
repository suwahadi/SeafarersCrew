package service

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// MaxBulkGalleryItems bounds one bulk gallery request.
const MaxBulkGalleryItems = 50

// GalleryStore persists gallery images, including atomic bulk inserts.
type GalleryStore interface {
	MasterStore[model.GalleryImage]
	InsertAll(ctx context.Context, items []model.GalleryImage) ([]model.GalleryImage, error)
}

// CMSStores groups the stores of the simple CMS lists.
type CMSStores struct {
	HeroSlides   MasterStore[model.HeroSlide]
	Gallery      GalleryStore
	Testimonials MasterStore[model.Testimonial]
	ClientLogos  MasterStore[model.ClientLogo]
	Stats        MasterStore[model.StatsItem]
	Languages    MasterStore[model.Language]
}

// CMS groups the services of the simple CMS lists.
type CMS struct {
	HeroSlides   *MasterService[model.HeroSlide]
	Gallery      *GalleryService
	Testimonials *MasterService[model.Testimonial]
	ClientLogos  *MasterService[model.ClientLogo]
	Stats        *MasterService[model.StatsItem]
	Languages    *MasterService[model.Language]
}

// NewCMS builds the services of the simple CMS lists; every write announces its cache tags to site.
func NewCMS(stores CMSStores, files *PublicFileChecker, site SiteChanger) CMS {
	home := changedHook(site, model.TagHome)
	galleryChanged := changedHook(site, model.TagGallery, model.TagHome, model.TagPages)
	return CMS{
		HeroSlides:   NewMasterService(stores.HeroSlides, heroSlideRules(files), home),
		Gallery:      NewGalleryService(stores.Gallery, files, galleryChanged),
		Testimonials: NewMasterService(stores.Testimonials, testimonialRules(files), home),
		ClientLogos:  NewMasterService(stores.ClientLogos, clientLogoRules(files), home),
		Stats:        NewMasterService(stores.Stats, MasterRules[model.StatsItem]{Noun: "Stats item"}, home),
		Languages:    NewMasterService(stores.Languages, languageRules(files), changedHook(site, model.TagSite)),
	}
}

func heroSlideRules(files *PublicFileChecker) MasterRules[model.HeroSlide] {
	return MasterRules[model.HeroSlide]{
		Noun: "Hero slide",
		Prepare: func(ctx context.Context, s *model.HeroSlide, _ *model.HeroSlide) error {
			_, err := files.Check(ctx,
				FileRef{Field: "image_file_id", ID: &s.ImageFileID, Kind: ImageFile},
				FileRef{Field: "mobile_image_file_id", ID: s.MobileImageFileID, Kind: ImageFile})
			return err
		},
	}
}

func testimonialRules(files *PublicFileChecker) MasterRules[model.Testimonial] {
	return MasterRules[model.Testimonial]{
		Noun: "Testimonial",
		Prepare: func(ctx context.Context, t *model.Testimonial, _ *model.Testimonial) error {
			_, err := files.Check(ctx, FileRef{Field: "photo_file_id", ID: t.PhotoFileID, Kind: ImageFile})
			return err
		},
	}
}

func clientLogoRules(files *PublicFileChecker) MasterRules[model.ClientLogo] {
	return MasterRules[model.ClientLogo]{
		Noun: "Client logo",
		Prepare: func(ctx context.Context, c *model.ClientLogo, _ *model.ClientLogo) error {
			_, err := files.Check(ctx, FileRef{Field: "logo_file_id", ID: &c.LogoFileID, Kind: LogoFile})
			return err
		},
	}
}

func languageRules(files *PublicFileChecker) MasterRules[model.Language] {
	return MasterRules[model.Language]{
		Noun: "Language",
		Prepare: func(ctx context.Context, l *model.Language, existing *model.Language) error {
			if existing != nil && existing.IsSource {
				fields := map[string]string{}
				if !l.IsActive {
					fields["is_active"] = "The source language cannot be deactivated."
				}
				if !strings.EqualFold(l.Code, existing.Code) {
					fields["code"] = "The code of the source language cannot be changed."
				}
				if len(fields) > 0 {
					return apperr.Validation(fields)
				}
			}
			_, err := files.Check(ctx, FileRef{Field: "flag_file_id", ID: l.FlagFileID, Kind: FlagFile})
			return err
		},
		Unique: []UniqueField[model.Language]{
			{Field: "code", Message: "The code is already used by another language.", Value: func(l model.Language) string { return l.Code }},
		},
		CanDelete: func(l model.Language) error {
			if l.IsSource {
				return apperr.Conflict("source_language", "The source language cannot be deleted.")
			}
			return nil
		},
	}
}

// GalleryService manages gallery images, including bulk creation.
type GalleryService struct {
	*MasterService[model.GalleryImage]
	store    GalleryStore
	files    *PublicFileChecker
	onChange func(ctx context.Context) error
}

// NewGalleryService builds a GalleryService; onChange runs after every successful write.
func NewGalleryService(store GalleryStore, files *PublicFileChecker, onChange func(ctx context.Context) error) *GalleryService {
	rules := MasterRules[model.GalleryImage]{
		Noun: "Gallery image",
		Prepare: func(ctx context.Context, g *model.GalleryImage, _ *model.GalleryImage) error {
			_, err := files.Check(ctx, FileRef{Field: "image_file_id", ID: &g.ImageFileID, Kind: ImageFile})
			return err
		},
	}
	return &GalleryService{MasterService: NewMasterService(store, rules, onChange), store: store, files: files, onChange: onChange}
}

// BulkCreate stores many gallery images atomically in the given order; a missing alt text falls back to the caption, then the file name.
func (s *GalleryService) BulkCreate(ctx context.Context, items []model.GalleryImage) ([]model.GalleryImage, error) {
	if len(items) == 0 || len(items) > MaxBulkGalleryItems {
		return nil, apperr.Validation(map[string]string{"items": fmt.Sprintf("Between 1 and %d images are required.", MaxBulkGalleryItems)})
	}
	refs := make([]FileRef, len(items))
	for i := range items {
		refs[i] = FileRef{Field: fmt.Sprintf("items.%d.image_file_id", i), ID: &items[i].ImageFileID, Kind: ImageFile}
	}
	infos, err := s.files.Check(ctx, refs...)
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	for i := range items {
		if items[i].AltText == "" {
			items[i].AltText = items[i].Caption
		}
		if items[i].AltText == "" {
			items[i].AltText = humanizeFileName(infos[items[i].ImageFileID].OriginalName)
		}
		if items[i].AltText == "" {
			fields[fmt.Sprintf("items.%d.alt_text", i)] = "The alt text is required."
		}
	}
	if len(fields) > 0 {
		return nil, apperr.Validation(fields)
	}
	created, err := s.store.InsertAll(ctx, items)
	if err != nil {
		return nil, err
	}
	return created, s.onChange(ctx)
}

func humanizeFileName(name string) string {
	base := strings.TrimSuffix(path.Base(name), path.Ext(name))
	base = strings.NewReplacer("-", " ", "_", " ").Replace(base)
	return strings.Join(strings.Fields(base), " ")
}
