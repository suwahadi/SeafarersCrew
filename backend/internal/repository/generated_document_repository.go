package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// GeneratedDocumentRepository stores the history of generated PDFs.
type GeneratedDocumentRepository struct {
	pool *pgxpool.Pool
}

// NewGeneratedDocumentRepository returns a GeneratedDocumentRepository.
func NewGeneratedDocumentRepository(pool *pgxpool.Pool) *GeneratedDocumentRepository {
	return &GeneratedDocumentRepository{pool: pool}
}

const generatedSelect = `SELECT g.id, g.public_id, g.kind, g.template_id, coalesce(t.name, ''), g.related_type, g.related_id,
	g.file_id, f.size_stored, g.created_at
FROM generated_documents g
JOIN files f ON f.id = g.file_id
LEFT JOIN document_templates t ON t.id = g.template_id`

func scanGenerated(row pgx.Row) (model.GeneratedDocument, error) {
	var g model.GeneratedDocument
	err := row.Scan(&g.ID, &g.PublicID, &g.Kind, &g.TemplateID, &g.TemplateName, &g.RelatedType, &g.RelatedID, &g.FileID, &g.FileSize, &g.CreatedAt)
	return g, notFound(err)
}

// Create records a generated document.
func (r *GeneratedDocumentRepository) Create(ctx context.Context, g model.GeneratedDocument) (model.GeneratedDocument, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `INSERT INTO generated_documents (public_id, kind, template_id, related_type, related_id, file_id)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, g.PublicID, g.Kind, g.TemplateID, g.RelatedType, g.RelatedID, g.FileID).Scan(&id)
	if err != nil {
		return g, fmt.Errorf("insert generated document: %w", err)
	}
	out, err := scanGenerated(r.pool.QueryRow(ctx, generatedSelect+` WHERE g.id = $1`, id))
	if err != nil {
		return out, fmt.Errorf("load generated document: %w", err)
	}
	return out, nil
}

// List returns generated documents, newest first, optionally for one related record.
func (r *GeneratedDocumentRepository) List(ctx context.Context, relatedType string, relatedID *uuid.UUID, limit int) ([]model.GeneratedDocument, error) {
	w := &Where{}
	w.Add("f.deleted_at IS NULL")
	if relatedType != "" {
		w.Add("g.related_type = %s", relatedType)
	}
	if relatedID != nil {
		w.Add("g.related_id = %s", *relatedID)
	}
	lim := w.Arg(limit)
	rows, err := r.pool.Query(ctx, generatedSelect+w.SQL()+` ORDER BY g.created_at DESC, g.id DESC LIMIT `+lim, w.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list generated documents: %w", err)
	}
	return collect(rows, scanGenerated, "generated documents")
}
