package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// ErrDuplicate is returned when an insert or update violates a unique index.
var ErrDuplicate = errors.New("duplicate value")

// ListQuery is a page request with equality filters keyed by public filter names.
type ListQuery struct {
	Page    int
	PerPage int
	Sort    string
	Desc    bool
	Q       string
	Filters map[string]any
}

// UsageQuery counts active references to a record; $1 is the record ID.
type UsageQuery struct {
	Label string
	SQL   string
}

// MasterSpec describes how one master table is read and written.
type MasterSpec[T any] struct {
	Table        string
	Select       string
	Writable     []string
	Values       func(T) []any
	Scan         func(row pgx.Row) (T, error)
	Search       []string
	Sorts        map[string]string
	Filters      map[string]string
	DefaultOrder string
	Unique       map[string]string
	Usages       []UsageQuery
}

// MasterRepository provides CRUD for one master table described by a MasterSpec.
type MasterRepository[T any] struct {
	pool *pgxpool.Pool
	spec MasterSpec[T]
}

// NewMasterRepository returns a MasterRepository.
func NewMasterRepository[T any](pool *pgxpool.Pool, spec MasterSpec[T]) *MasterRepository[T] {
	return &MasterRepository[T]{pool: pool, spec: spec}
}

func (r *MasterRepository[T]) where(q ListQuery) *Where {
	w := &Where{}
	w.Add("deleted_at IS NULL")
	if q.Q != "" && len(r.spec.Search) > 0 {
		parts := make([]string, len(r.spec.Search))
		for i, col := range r.spec.Search {
			parts[i] = col + ` ILIKE %[1]s`
		}
		w.Add("("+strings.Join(parts, " OR ")+")", LikePattern(q.Q))
	}
	for key, val := range q.Filters {
		if col, ok := r.spec.Filters[key]; ok {
			w.Add(col+" = %s", val)
		}
	}
	return w
}

// List returns one page of non-deleted rows.
func (r *MasterRepository[T]) List(ctx context.Context, q ListQuery) (Page[T], error) {
	w := r.where(q)
	from := " FROM " + r.spec.Table + w.SQL()
	order := Order(q.Sort, q.Desc, r.spec.Sorts, r.spec.DefaultOrder)
	limit := " LIMIT " + strconv.Itoa(q.PerPage) + " OFFSET " + strconv.Itoa((q.Page-1)*q.PerPage)

	var page Page[T]
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		rows, err := r.pool.Query(gctx, "SELECT "+r.spec.Select+from+order+limit, w.Args()...)
		if err != nil {
			return fmt.Errorf("list %s: %w", r.spec.Table, err)
		}
		items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (T, error) { return r.spec.Scan(row) })
		if err != nil {
			return fmt.Errorf("scan %s: %w", r.spec.Table, err)
		}
		page.Items = items
		return nil
	})
	g.Go(func() error {
		if err := r.pool.QueryRow(gctx, "SELECT count(*)"+from, w.Args()...).Scan(&page.Total); err != nil {
			return fmt.Errorf("count %s: %w", r.spec.Table, err)
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return Page[T]{}, err
	}
	return page, nil
}

// Active returns every active, non-deleted row in display order.
func (r *MasterRepository[T]) Active(ctx context.Context) ([]T, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+r.spec.Select+" FROM "+r.spec.Table+" WHERE deleted_at IS NULL AND is_active ORDER BY "+r.spec.DefaultOrder)
	if err != nil {
		return nil, fmt.Errorf("list active %s: %w", r.spec.Table, err)
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (T, error) { return r.spec.Scan(row) })
	if err != nil {
		return nil, fmt.Errorf("scan active %s: %w", r.spec.Table, err)
	}
	return items, nil
}

// Get returns a non-deleted row by ID.
func (r *MasterRepository[T]) Get(ctx context.Context, id uuid.UUID) (T, error) {
	item, err := r.spec.Scan(r.pool.QueryRow(ctx, "SELECT "+r.spec.Select+" FROM "+r.spec.Table+" WHERE id = $1 AND deleted_at IS NULL", id))
	if err != nil {
		return item, fmt.Errorf("get %s: %w", r.spec.Table, notFound(err))
	}
	return item, nil
}

// Insert stores a new row at the end of the sort order.
func (r *MasterRepository[T]) Insert(ctx context.Context, item T) (T, error) {
	values := r.spec.Values(item)
	ph := make([]string, len(values))
	for i := range values {
		ph[i] = "$" + strconv.Itoa(i+1)
	}
	sql := "INSERT INTO " + r.spec.Table + " (" + strings.Join(r.spec.Writable, ", ") + ", sort_order) VALUES (" +
		strings.Join(ph, ", ") + ", (SELECT coalesce(max(sort_order), 0) + 10 FROM " + r.spec.Table + " WHERE deleted_at IS NULL)) RETURNING " + r.spec.Select
	created, err := r.spec.Scan(r.pool.QueryRow(ctx, sql, values...))
	if err != nil {
		return created, fmt.Errorf("insert %s: %w", r.spec.Table, duplicate(err))
	}
	return created, nil
}

// Update overwrites the writable columns of a non-deleted row.
func (r *MasterRepository[T]) Update(ctx context.Context, id uuid.UUID, item T) (T, error) {
	values := r.spec.Values(item)
	sets := make([]string, len(r.spec.Writable))
	for i, col := range r.spec.Writable {
		sets[i] = col + " = $" + strconv.Itoa(i+2)
	}
	sql := "UPDATE " + r.spec.Table + " SET " + strings.Join(sets, ", ") + " WHERE id = $1 AND deleted_at IS NULL RETURNING " + r.spec.Select
	updated, err := r.spec.Scan(r.pool.QueryRow(ctx, sql, append([]any{id}, values...)...))
	if err != nil {
		return updated, fmt.Errorf("update %s: %w", r.spec.Table, duplicate(notFound(err)))
	}
	return updated, nil
}

// SoftDelete marks a row deleted.
func (r *MasterRepository[T]) SoftDelete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "UPDATE "+r.spec.Table+" SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("delete %s: %w", r.spec.Table, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Reorder sets the display order to the order of ids.
func (r *MasterRepository[T]) Reorder(ctx context.Context, ids []uuid.UUID) error {
	return Reorder(ctx, r.pool, r.spec.Table, ids)
}

// Taken reports whether another row already uses value in a unique field.
func (r *MasterRepository[T]) Taken(ctx context.Context, field, value string, exclude uuid.UUID) (bool, error) {
	cond, ok := r.spec.Unique[field]
	if !ok {
		return false, fmt.Errorf("taken: %s.%s is not a unique field", r.spec.Table, field)
	}
	var taken bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM "+r.spec.Table+" WHERE "+cond+" AND id <> $2)", value, exclude).Scan(&taken)
	if err != nil {
		return false, fmt.Errorf("check unique %s.%s: %w", r.spec.Table, field, err)
	}
	return taken, nil
}

// Usages counts active records referencing id, omitting zero counts.
func (r *MasterRepository[T]) Usages(ctx context.Context, id uuid.UUID) ([]model.Usage, error) {
	if len(r.spec.Usages) == 0 {
		return nil, nil
	}
	parts := make([]string, len(r.spec.Usages))
	counts := make([]int, len(r.spec.Usages))
	dest := make([]any, len(r.spec.Usages))
	for i, u := range r.spec.Usages {
		parts[i] = "(" + u.SQL + ")"
		dest[i] = &counts[i]
	}
	if err := r.pool.QueryRow(ctx, "SELECT "+strings.Join(parts, ", "), id).Scan(dest...); err != nil {
		return nil, fmt.Errorf("count usages of %s: %w", r.spec.Table, err)
	}
	var out []model.Usage
	for i, u := range r.spec.Usages {
		if counts[i] > 0 {
			out = append(out, model.Usage{Label: u.Label, Count: counts[i]})
		}
	}
	return out, nil
}

func duplicate(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %s", ErrDuplicate, pgErr.ConstraintName)
	}
	return err
}
