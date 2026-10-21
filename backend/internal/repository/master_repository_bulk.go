package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// InsertAll stores several rows atomically at the end of the sort order, in the given order.
func (r *MasterRepository[T]) InsertAll(ctx context.Context, items []T) ([]T, error) {
	created := make([]T, 0, len(items))
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		for _, item := range items {
			values := r.spec.Values(item)
			ph := make([]string, len(values))
			for i := range values {
				ph[i] = "$" + strconv.Itoa(i+1)
			}
			sql := "INSERT INTO " + r.spec.Table + " (" + strings.Join(r.spec.Writable, ", ") + ", sort_order) VALUES (" +
				strings.Join(ph, ", ") + ", (SELECT coalesce(max(sort_order), 0) + 10 FROM " + r.spec.Table + " WHERE deleted_at IS NULL)) RETURNING " + r.spec.Select
			row, err := r.spec.Scan(tx.QueryRow(ctx, sql, values...))
			if err != nil {
				return fmt.Errorf("insert %s: %w", r.spec.Table, duplicate(err))
			}
			created = append(created, row)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}
