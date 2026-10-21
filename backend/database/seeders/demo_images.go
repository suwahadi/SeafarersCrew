package seeders

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/seafarerscrew/backend/database/seeders/demoimage"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
)

const (
	demoHeroCount    = 3
	demoGalleryCount = 12
	demoNewsCount    = 6
	demoFilePrefix   = "demo-"
)

func heroFileName(i int) string    { return fmt.Sprintf("%shero-%02d.png", demoFilePrefix, i+1) }
func galleryFileName(i int) string { return fmt.Sprintf("%sgallery-%02d.png", demoFilePrefix, i+1) }
func newsFileName(i int) string    { return fmt.Sprintf("%snews-%02d.png", demoFilePrefix, i+1) }
func logoFileName(i int) string    { return fmt.Sprintf("%slogo-%02d.png", demoFilePrefix, i+1) }

const aboutFileName = demoFilePrefix + "about.png"

type demoImages struct {
	hero    []uuid.UUID
	gallery []uuid.UUID
	news    []uuid.UUID
	logos   []uuid.UUID
	about   uuid.UUID
}

func seedDemoImages(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	scene := 0
	next := func(name string) error {
		data, err := demoimage.Scene(scene)
		scene++
		if err != nil {
			return err
		}
		return d.uploadImage(ctx, filecheck.ContextCMSImage, name, data)
	}
	for i := range demoHeroCount {
		if err := next(heroFileName(i)); err != nil {
			return err
		}
	}
	for i := range demoGalleryCount {
		if err := next(galleryFileName(i)); err != nil {
			return err
		}
	}
	for i := range demoNewsCount {
		if err := next(newsFileName(i)); err != nil {
			return err
		}
	}
	if err := next(aboutFileName); err != nil {
		return err
	}
	for i, name := range demoClientNames {
		data, err := demoimage.Logo(i, name)
		if err != nil {
			return err
		}
		if err := d.uploadImage(ctx, filecheck.ContextLogo, logoFileName(i), data); err != nil {
			return err
		}
	}
	return nil
}

func (d *demoRun) uploadImage(ctx context.Context, kind filecheck.Context, name string, data []byte) error {
	if _, err := d.Files.Upload(ctx, service.UploadInput{Context: kind, Filename: name, Data: data}); err != nil {
		return fmt.Errorf("upload %s: %w", name, err)
	}
	return nil
}

func (d *demoRun) loadImages(ctx context.Context) (demoImages, error) {
	rows, err := d.Pool.Query(ctx, `SELECT original_name, id FROM files WHERE original_name LIKE $1 AND is_public AND deleted_at IS NULL`, demoFilePrefix+"%")
	if err != nil {
		return demoImages{}, fmt.Errorf("load demo images: %w", err)
	}
	type row struct {
		name string
		id   uuid.UUID
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.name, &x.id)
		return x, err
	})
	if err != nil {
		return demoImages{}, fmt.Errorf("scan demo images: %w", err)
	}
	byName := make(map[string]uuid.UUID, len(items))
	for _, it := range items {
		byName[it.name] = it.id
	}
	var missing error
	lookup := func(name string) uuid.UUID {
		id, ok := byName[name]
		if !ok && missing == nil {
			missing = fmt.Errorf("demo image %s not found", name)
		}
		return id
	}
	var out demoImages
	for i := range demoHeroCount {
		out.hero = append(out.hero, lookup(heroFileName(i)))
	}
	for i := range demoGalleryCount {
		out.gallery = append(out.gallery, lookup(galleryFileName(i)))
	}
	for i := range demoNewsCount {
		out.news = append(out.news, lookup(newsFileName(i)))
	}
	for i := range demoClientNames {
		out.logos = append(out.logos, lookup(logoFileName(i)))
	}
	out.about = lookup(aboutFileName)
	return out, missing
}
