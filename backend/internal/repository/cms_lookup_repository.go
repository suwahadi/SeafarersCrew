package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// CMSLookupRepository answers reference questions needed when saving or rendering CMS content.
type CMSLookupRepository struct {
	pool *pgxpool.Pool
}

// NewCMSLookupRepository returns a CMSLookupRepository.
func NewCMSLookupRepository(pool *pgxpool.Pool) *CMSLookupRepository {
	return &CMSLookupRepository{pool: pool}
}

// PublicFiles returns the ready, public, non-deleted files among ids.
func (r *CMSLookupRepository) PublicFiles(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.PublicFileInfo, error) {
	out := map[uuid.UUID]model.PublicFileInfo{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT id, mime_type, original_name FROM files
		WHERE id = ANY($1) AND deleted_at IS NULL AND is_public AND status = 'ready'`, ids)
	if err != nil {
		return nil, fmt.Errorf("look up public files: %w", err)
	}
	infos, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.PublicFileInfo, error) {
		var f model.PublicFileInfo
		err := row.Scan(&f.ID, &f.MimeType, &f.OriginalName)
		return f, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan public files: %w", err)
	}
	for _, f := range infos {
		out[f.ID] = f
	}
	return out, nil
}

// Images returns the public image data of the given file IDs; unknown or private files are omitted.
func (r *CMSLookupRepository) Images(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.Image, error) {
	out := map[uuid.UUID]model.Image{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT json_build_object('id', f.id, 'storage_key', f.storage_key, 'mime_type', f.mime_type,
		'width', coalesce(f.width, 0), 'height', coalesce(f.height, 0), 'variants', f.variants)
		FROM files f WHERE f.id = ANY($1) AND f.deleted_at IS NULL AND f.is_public`, ids)
	if err != nil {
		return nil, fmt.Errorf("look up images: %w", err)
	}
	images, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.Image, error) {
		var raw []byte
		if err := row.Scan(&raw); err != nil {
			return model.Image{}, err
		}
		var img model.Image
		return img, json.Unmarshal(raw, &img)
	})
	if err != nil {
		return nil, fmt.Errorf("scan images: %w", err)
	}
	for _, img := range images {
		out[img.ID] = img
	}
	return out, nil
}

// GalleryImagesExist returns which of ids are live gallery images.
func (r *CMSLookupRepository) GalleryImagesExist(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	return r.existing(ctx, `SELECT id FROM gallery_images WHERE id = ANY($1) AND deleted_at IS NULL`, ids)
}

// PagesExist returns which of ids are live pages.
func (r *CMSLookupRepository) PagesExist(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	return r.existing(ctx, `SELECT id FROM pages WHERE id = ANY($1) AND deleted_at IS NULL`, ids)
}

func (r *CMSLookupRepository) existing(ctx context.Context, sql string, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, sql, ids)
	if err != nil {
		return nil, fmt.Errorf("look up records: %w", err)
	}
	found, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("scan records: %w", err)
	}
	for _, id := range found {
		out[id] = true
	}
	return out, nil
}
