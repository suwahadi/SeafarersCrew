package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// ComplianceMatrixRepository stores the compliance matrix.
type ComplianceMatrixRepository struct {
	pool *pgxpool.Pool
}

// NewComplianceMatrixRepository returns a ComplianceMatrixRepository.
func NewComplianceMatrixRepository(pool *pgxpool.Pool) *ComplianceMatrixRepository {
	return &ComplianceMatrixRepository{pool: pool}
}

const activeScope = `
	JOIN ranks r ON r.id = c.rank_id AND r.deleted_at IS NULL AND r.is_active
	LEFT JOIN document_types d ON d.id = c.document_type_id
	LEFT JOIN certificate_types t ON t.id = c.certificate_type_id
	WHERE (d.id IS NULL OR (d.deleted_at IS NULL AND d.is_active))
	  AND (t.id IS NULL OR (t.deleted_at IS NULL AND t.is_active))`

// ActiveCells returns the cells whose rank and type are active.
func (r *ComplianceMatrixRepository) ActiveCells(ctx context.Context) ([]model.ComplianceCell, error) {
	rows, err := r.pool.Query(ctx, `SELECT c.rank_id, c.document_type_id, c.certificate_type_id FROM compliance_requirements c`+activeScope+` ORDER BY r.sort_order, c.created_at`)
	if err != nil {
		return nil, fmt.Errorf("list compliance cells: %w", err)
	}
	cells, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.ComplianceCell, error) {
		var c model.ComplianceCell
		err := row.Scan(&c.RankID, &c.DocumentTypeID, &c.CertificateTypeID)
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan compliance cells: %w", err)
	}
	return cells, nil
}

// ReplaceActive deletes the cells of active ranks and types and stores cells in one transaction.
func (r *ComplianceMatrixRepository) ReplaceActive(ctx context.Context, cells []model.ComplianceCell) error {
	return InTx(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM compliance_requirements WHERE id IN (SELECT c.id FROM compliance_requirements c`+activeScope+`)`); err != nil {
			return fmt.Errorf("clear compliance matrix: %w", err)
		}
		rows := make([][]any, len(cells))
		for i, c := range cells {
			kind := "document"
			if c.CertificateTypeID != nil {
				kind = "certificate"
			}
			rows[i] = []any{c.RankID, kind, c.DocumentTypeID, c.CertificateTypeID}
		}
		if _, err := tx.CopyFrom(ctx, pgx.Identifier{"compliance_requirements"},
			[]string{"rank_id", "requirement_type", "document_type_id", "certificate_type_id"}, pgx.CopyFromRows(rows)); err != nil {
			return fmt.Errorf("insert compliance matrix: %w", err)
		}
		return nil
	})
}
