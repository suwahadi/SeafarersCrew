package repository

import (
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// NewHeroSlideRepository returns the hero slides repository.
func NewHeroSlideRepository(pool *pgxpool.Pool) *MasterRepository[model.HeroSlide] {
	return NewMasterRepository(pool, MasterSpec[model.HeroSlide]{
		Table: "hero_slides",
		Select: baseColumns + ", image_file_id, mobile_image_file_id, heading, subheading, button_label, button_link, " +
			ImageJSON("image_file_id") + ", " + ImageJSON("mobile_image_file_id"),
		Writable: []string{"image_file_id", "mobile_image_file_id", "heading", "subheading", "button_label", "button_link", "is_active"},
		Values: func(s model.HeroSlide) []any {
			return []any{s.ImageFileID, s.MobileImageFileID, s.Heading, s.Subheading, s.ButtonLabel, s.ButtonLink, s.IsActive}
		},
		Scan: func(row pgx.Row) (model.HeroSlide, error) {
			var s model.HeroSlide
			err := scanWith(row, &s.MasterBase, &s.ImageFileID, &s.MobileImageFileID, &s.Heading, &s.Subheading, &s.ButtonLabel, &s.ButtonLink, &s.Image, &s.MobileImage)
			return s, err
		},
		Search:       []string{"heading", "subheading"},
		Sorts:        sorts(map[string]string{"heading": "heading"}),
		Filters:      filters(nil),
		DefaultOrder: "sort_order, created_at",
	})
}

var galleryColumns = baseColumns + ", image_file_id, caption, alt_text, show_on_home, " + ImageJSON("image_file_id")

func scanGalleryImage(row pgx.Row) (model.GalleryImage, error) {
	var g model.GalleryImage
	err := scanWith(row, &g.MasterBase, &g.ImageFileID, &g.Caption, &g.AltText, &g.ShowOnHome, &g.Image)
	return g, err
}

// NewGalleryImageRepository returns the gallery images repository.
func NewGalleryImageRepository(pool *pgxpool.Pool) *MasterRepository[model.GalleryImage] {
	return NewMasterRepository(pool, MasterSpec[model.GalleryImage]{
		Table:    "gallery_images",
		Select:   galleryColumns,
		Writable: []string{"image_file_id", "caption", "alt_text", "show_on_home", "is_active"},
		Values: func(g model.GalleryImage) []any {
			return []any{g.ImageFileID, g.Caption, g.AltText, g.ShowOnHome, g.IsActive}
		},
		Scan:         scanGalleryImage,
		Search:       []string{"caption", "alt_text"},
		Sorts:        sorts(map[string]string{"caption": "caption", "show_on_home": "show_on_home"}),
		Filters:      filters(map[string]string{"show_on_home": "show_on_home"}),
		DefaultOrder: "sort_order, created_at",
	})
}

// NewTestimonialRepository returns the testimonials repository.
func NewTestimonialRepository(pool *pgxpool.Pool) *MasterRepository[model.Testimonial] {
	return NewMasterRepository(pool, MasterSpec[model.Testimonial]{
		Table:    "testimonials",
		Select:   baseColumns + ", name, position, photo_file_id, quote, rating, " + ImageJSON("photo_file_id"),
		Writable: []string{"name", "position", "photo_file_id", "quote", "rating", "is_active"},
		Values: func(t model.Testimonial) []any {
			return []any{t.Name, t.Position, t.PhotoFileID, t.Quote, t.Rating, t.IsActive}
		},
		Scan: func(row pgx.Row) (model.Testimonial, error) {
			var t model.Testimonial
			err := scanWith(row, &t.MasterBase, &t.Name, &t.Position, &t.PhotoFileID, &t.Quote, &t.Rating, &t.Photo)
			return t, err
		},
		Search:       []string{"name", "position", "quote"},
		Sorts:        sorts(map[string]string{"name": "name", "rating": "rating"}),
		Filters:      filters(map[string]string{"rating": "rating"}),
		DefaultOrder: "sort_order, created_at",
	})
}

// NewClientLogoRepository returns the client logos repository.
func NewClientLogoRepository(pool *pgxpool.Pool) *MasterRepository[model.ClientLogo] {
	return NewMasterRepository(pool, MasterSpec[model.ClientLogo]{
		Table:    "client_logos",
		Select:   baseColumns + ", name, logo_file_id, url, " + ImageJSON("logo_file_id"),
		Writable: []string{"name", "logo_file_id", "url", "is_active"},
		Values: func(c model.ClientLogo) []any {
			return []any{c.Name, c.LogoFileID, c.URL, c.IsActive}
		},
		Scan: func(row pgx.Row) (model.ClientLogo, error) {
			var c model.ClientLogo
			err := scanWith(row, &c.MasterBase, &c.Name, &c.LogoFileID, &c.URL, &c.Logo)
			return c, err
		},
		Search:       []string{"name", "url"},
		Sorts:        sorts(map[string]string{"name": "name"}),
		Filters:      filters(nil),
		DefaultOrder: "sort_order, created_at",
	})
}

// NewStatsItemRepository returns the stats items repository.
func NewStatsItemRepository(pool *pgxpool.Pool) *MasterRepository[model.StatsItem] {
	return NewMasterRepository(pool, MasterSpec[model.StatsItem]{
		Table:    "stats_items",
		Select:   baseColumns + ", number, suffix, label",
		Writable: []string{"number", "suffix", "label", "is_active"},
		Values: func(s model.StatsItem) []any {
			return []any{s.Number, s.Suffix, s.Label, s.IsActive}
		},
		Scan: func(row pgx.Row) (model.StatsItem, error) {
			var s model.StatsItem
			err := scanWith(row, &s.MasterBase, &s.Number, &s.Suffix, &s.Label)
			return s, err
		},
		Search:       []string{"label", "suffix"},
		Sorts:        sorts(map[string]string{"label": "label", "number": "number"}),
		Filters:      filters(nil),
		DefaultOrder: "sort_order, created_at",
	})
}

// NewLanguageRepository returns the languages repository.
func NewLanguageRepository(pool *pgxpool.Pool) *MasterRepository[model.Language] {
	return NewMasterRepository(pool, MasterSpec[model.Language]{
		Table:    "languages",
		Select:   baseColumns + ", name, code, flag_file_id, flag_code, is_source, " + ImageJSON("flag_file_id"),
		Writable: []string{"name", "code", "flag_file_id", "flag_code", "is_active"},
		Values: func(l model.Language) []any {
			return []any{l.Name, l.Code, l.FlagFileID, l.FlagCode, l.IsActive}
		},
		Scan: func(row pgx.Row) (model.Language, error) {
			var l model.Language
			err := scanWith(row, &l.MasterBase, &l.Name, &l.Code, &l.FlagFileID, &l.FlagCode, &l.IsSource, &l.Flag)
			return l, err
		},
		Search:       []string{"name", "code"},
		Sorts:        sorts(map[string]string{"name": "name", "code": "code"}),
		Filters:      filters(nil),
		DefaultOrder: "sort_order, name",
		Unique:       map[string]string{"code": "lower(code) = lower($1) AND deleted_at IS NULL"},
	})
}
