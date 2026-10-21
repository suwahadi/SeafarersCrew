package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// ContactRepository stores contact form messages.
type ContactRepository struct {
	pool *pgxpool.Pool
}

// NewContactRepository returns a ContactRepository.
func NewContactRepository(pool *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{pool: pool}
}

const contactColumns = `id, name, email, phone, subject, message, coalesce(host(ip_address), ''), is_read, is_archived, created_at, updated_at`

func scanContact(row pgx.Row) (model.ContactMessage, error) {
	var m model.ContactMessage
	err := row.Scan(&m.ID, &m.Name, &m.Email, &m.Phone, &m.Subject, &m.Message, &m.IPAddress, &m.IsRead, &m.IsArchived, &m.CreatedAt, &m.UpdatedAt)
	return m, notFound(err)
}

// Create stores a message.
func (r *ContactRepository) Create(ctx context.Context, m model.ContactMessage) (model.ContactMessage, error) {
	created, err := scanContact(r.pool.QueryRow(ctx, `INSERT INTO contact_messages (name, email, phone, subject, message, ip_address)
		VALUES ($1, $2, $3, $4, $5, nullif($6, '')::inet) RETURNING `+contactColumns,
		m.Name, m.Email, m.Phone, m.Subject, m.Message, m.IPAddress))
	if err != nil {
		return created, fmt.Errorf("insert contact message: %w", err)
	}
	return created, nil
}

// Get returns one message.
func (r *ContactRepository) Get(ctx context.Context, id uuid.UUID) (model.ContactMessage, error) {
	m, err := scanContact(r.pool.QueryRow(ctx, `SELECT `+contactColumns+` FROM contact_messages WHERE id = $1`, id))
	if err != nil {
		return m, fmt.Errorf("get contact message: %w", err)
	}
	return m, nil
}

// SetFlags updates the read and archived flags that are given.
func (r *ContactRepository) SetFlags(ctx context.Context, id uuid.UUID, isRead, isArchived *bool) (model.ContactMessage, error) {
	m, err := scanContact(r.pool.QueryRow(ctx, `UPDATE contact_messages SET is_read = coalesce($2, is_read), is_archived = coalesce($3, is_archived)
		WHERE id = $1 RETURNING `+contactColumns, id, isRead, isArchived))
	if err != nil {
		return m, fmt.Errorf("update contact message: %w", err)
	}
	return m, nil
}

func contactWhere(f model.ContactFilter) *Where {
	w := &Where{}
	switch f.Status {
	case model.ContactUnread:
		w.Add(`NOT is_read AND NOT is_archived`)
	case model.ContactRead:
		w.Add(`is_read AND NOT is_archived`)
	case model.ContactArchived:
		w.Add(`is_archived`)
	}
	if f.Q != "" {
		w.Add(`(name ILIKE %[1]s OR email ILIKE %[1]s OR subject ILIKE %[1]s)`, LikePattern(f.Q))
	}
	return w
}

// List returns one page of messages, newest first.
func (r *ContactRepository) List(ctx context.Context, f model.ContactFilter, limit, offset int) (Page[model.ContactMessage], error) {
	w := contactWhere(f)
	var page Page[model.ContactMessage]
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM contact_messages`+w.SQL(), w.Args()...).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("count contact messages: %w", err)
	}
	lim, off := w.Arg(limit), w.Arg(offset)
	rows, err := r.pool.Query(ctx, `SELECT `+contactColumns+` FROM contact_messages`+w.SQL()+
		` ORDER BY created_at DESC, id DESC LIMIT `+lim+` OFFSET `+off, w.Args()...)
	if err != nil {
		return page, fmt.Errorf("list contact messages: %w", err)
	}
	page.Items, err = collect(rows, scanContact, "contact messages")
	return page, err
}

// UnreadCount returns the number of unread, unarchived messages.
func (r *ContactRepository) UnreadCount(ctx context.Context) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM contact_messages WHERE NOT is_read AND NOT is_archived`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count unread contact messages: %w", err)
	}
	return n, nil
}
