package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// DocumentTemplateRepository stores document builder templates.
type DocumentTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewDocumentTemplateRepository returns a DocumentTemplateRepository.
func NewDocumentTemplateRepository(pool *pgxpool.Pool) *DocumentTemplateRepository {
	return &DocumentTemplateRepository{pool: pool}
}

const templateColumns = `id, name, kind, paper_size, orientation, header_html, body_html, footer_html, is_default, created_at, updated_at`

func scanTemplate(row pgx.Row) (model.DocumentTemplate, error) {
	var t model.DocumentTemplate
	err := row.Scan(&t.ID, &t.Name, &t.Kind, &t.PaperSize, &t.Orientation, &t.HeaderHTML, &t.BodyHTML, &t.FooterHTML, &t.IsDefault, &t.CreatedAt, &t.UpdatedAt)
	return t, notFound(err)
}

// List returns active templates, optionally of one kind, defaults first.
func (r *DocumentTemplateRepository) List(ctx context.Context, kind string) ([]model.DocumentTemplate, error) {
	w := &Where{}
	w.Add("deleted_at IS NULL")
	if kind != "" {
		w.Add("kind = %s", kind)
	}
	rows, err := r.pool.Query(ctx, `SELECT `+templateColumns+` FROM document_templates`+w.SQL()+` ORDER BY kind, is_default DESC, name`, w.Args()...)
	if err != nil {
		return nil, fmt.Errorf("list document templates: %w", err)
	}
	return collect(rows, scanTemplate, "document templates")
}

// Get returns a template by ID.
func (r *DocumentTemplateRepository) Get(ctx context.Context, id uuid.UUID) (model.DocumentTemplate, error) {
	t, err := scanTemplate(r.pool.QueryRow(ctx, `SELECT `+templateColumns+` FROM document_templates WHERE id = $1 AND deleted_at IS NULL`, id))
	if err != nil {
		return t, fmt.Errorf("get document template: %w", err)
	}
	return t, nil
}

// Default returns the default template of a kind.
func (r *DocumentTemplateRepository) Default(ctx context.Context, kind string) (model.DocumentTemplate, error) {
	t, err := scanTemplate(r.pool.QueryRow(ctx, `SELECT `+templateColumns+` FROM document_templates
		WHERE kind = $1 AND is_default AND deleted_at IS NULL`, kind))
	if err != nil {
		return t, fmt.Errorf("get default template: %w", err)
	}
	return t, nil
}

// CountByKind returns the number of active templates of a kind.
func (r *DocumentTemplateRepository) CountByKind(ctx context.Context, kind string) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM document_templates WHERE kind = $1 AND deleted_at IS NULL`, kind).Scan(&n); err != nil {
		return 0, fmt.Errorf("count document templates: %w", err)
	}
	return n, nil
}

// Save inserts (zero ID) or updates a template; when it is default, the previous default of the kind is cleared in the same transaction.
func (r *DocumentTemplateRepository) Save(ctx context.Context, t model.DocumentTemplate) (model.DocumentTemplate, error) {
	var out model.DocumentTemplate
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if t.IsDefault {
			if _, err := tx.Exec(ctx, `UPDATE document_templates SET is_default = false
				WHERE kind = $1 AND is_default AND id <> $2 AND deleted_at IS NULL`, t.Kind, t.ID); err != nil {
				return fmt.Errorf("clear default template: %w", err)
			}
		}
		var err error
		if t.ID == uuid.Nil {
			out, err = scanTemplate(tx.QueryRow(ctx, `INSERT INTO document_templates (name, kind, paper_size, orientation, header_html,
				body_html, footer_html, is_default) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+templateColumns,
				t.Name, t.Kind, t.PaperSize, t.Orientation, t.HeaderHTML, t.BodyHTML, t.FooterHTML, t.IsDefault))
		} else {
			out, err = scanTemplate(tx.QueryRow(ctx, `UPDATE document_templates SET name = $2, paper_size = $3, orientation = $4,
				header_html = $5, body_html = $6, footer_html = $7, is_default = $8
				WHERE id = $1 AND deleted_at IS NULL RETURNING `+templateColumns,
				t.ID, t.Name, t.PaperSize, t.Orientation, t.HeaderHTML, t.BodyHTML, t.FooterHTML, t.IsDefault))
		}
		if err != nil {
			return fmt.Errorf("save document template: %w", err)
		}
		return nil
	})
	return out, err
}

// SoftDelete marks a template deleted.
func (r *DocumentTemplateRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `UPDATE document_templates SET deleted_at = now(), is_default = false WHERE id = $1 AND deleted_at IS NULL`, id)
	return affectedOne(tag, err, "delete document template")
}
