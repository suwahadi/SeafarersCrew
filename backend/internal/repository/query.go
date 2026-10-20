package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Where accumulates parameterized SQL conditions.
type Where struct {
	clauses []string
	args    []any
}

// Arg registers a value and returns its placeholder.
func (w *Where) Arg(v any) string {
	w.args = append(w.args, v)
	return "$" + strconv.Itoa(len(w.args))
}

// Add appends a condition whose %s verbs are replaced by placeholders for vals.
func (w *Where) Add(format string, vals ...any) {
	ph := make([]any, len(vals))
	for i, v := range vals {
		ph[i] = w.Arg(v)
	}
	w.clauses = append(w.clauses, fmt.Sprintf(format, ph...))
}

// SQL returns the WHERE clause, or an empty string when there are no conditions.
func (w *Where) SQL() string {
	if len(w.clauses) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w.clauses, " AND ")
}

// Args returns the accumulated arguments.
func (w *Where) Args() []any { return w.args }

// Order builds an ORDER BY clause from a whitelist of sortable columns, breaking ties on id.
func Order(sort string, desc bool, columns map[string]string, fallback string) string {
	return OrderWithTie(sort, desc, columns, fallback, "id")
}

// OrderWithTie is Order with an explicit tie-breaker column.
func OrderWithTie(sort string, desc bool, columns map[string]string, fallback, tie string) string {
	col, ok := columns[sort]
	if !ok {
		return " ORDER BY " + fallback
	}
	dir := " ASC"
	if desc {
		dir = " DESC"
	}
	return " ORDER BY " + col + dir + " NULLS LAST, " + tie + dir
}

// Page is one page of rows plus the total match count.
type Page[T any] struct {
	Items []T
	Total int
}

// LikePattern escapes s for use inside an ILIKE '%s%' pattern.
func LikePattern(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}

// Reorder sets sort_order on table rows to their position in ids.
func Reorder(ctx context.Context, db DBTX, table string, ids []uuid.UUID) error {
	if _, ok := reorderable[table]; !ok {
		return fmt.Errorf("reorder: table %q is not reorderable", table)
	}
	_, err := db.Exec(ctx, `UPDATE `+table+` t SET sort_order = o.pos * 10
		FROM unnest($1::uuid[]) WITH ORDINALITY AS o(id, pos) WHERE t.id = o.id`, ids)
	if err != nil {
		return fmt.Errorf("reorder %s: %w", table, err)
	}
	return nil
}

var reorderable = map[string]struct{}{
	"companies": {}, "vessels": {}, "ranks": {}, "vessel_types": {}, "currencies": {}, "crew_statuses": {},
	"allowance_types": {}, "document_types": {}, "certificate_types": {},
	"hero_slides": {}, "gallery_images": {}, "testimonials": {}, "client_logos": {}, "stats_items": {},
	"menu_items": {}, "languages": {},
}
