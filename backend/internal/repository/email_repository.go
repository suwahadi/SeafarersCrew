package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// EmailRepository stores email templates and email logs.
type EmailRepository struct {
	pool *pgxpool.Pool
}

// NewEmailRepository returns an EmailRepository.
func NewEmailRepository(pool *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{pool: pool}
}

const emailTemplateColumns = `id, code, name, subject, body_html, updated_at`

func scanEmailTemplate(row pgx.Row) (model.EmailTemplate, error) {
	var t model.EmailTemplate
	err := row.Scan(&t.ID, &t.Code, &t.Name, &t.Subject, &t.BodyHTML, &t.UpdatedAt)
	return t, notFound(err)
}

// ListTemplates returns all email templates ordered by name.
func (r *EmailRepository) ListTemplates(ctx context.Context) ([]model.EmailTemplate, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+emailTemplateColumns+` FROM email_templates ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list email templates: %w", err)
	}
	return collect(rows, scanEmailTemplate, "email templates")
}

// GetTemplate returns the template with code.
func (r *EmailRepository) GetTemplate(ctx context.Context, code string) (model.EmailTemplate, error) {
	t, err := scanEmailTemplate(r.pool.QueryRow(ctx, `SELECT `+emailTemplateColumns+` FROM email_templates WHERE code = $1`, code))
	if err != nil {
		return t, fmt.Errorf("get email template: %w", err)
	}
	return t, nil
}

// UpdateTemplate stores a new subject and body for code.
func (r *EmailRepository) UpdateTemplate(ctx context.Context, code, subject, body string) (model.EmailTemplate, error) {
	t, err := scanEmailTemplate(r.pool.QueryRow(ctx,
		`UPDATE email_templates SET subject = $2, body_html = $3 WHERE code = $1 RETURNING `+emailTemplateColumns, code, subject, body))
	if err != nil {
		return t, fmt.Errorf("update email template: %w", err)
	}
	return t, nil
}

const emailLogColumns = `id, coalesce(template_code, ''), to_addresses, cc_addresses, subject, coalesce(related_type, ''), related_id,
	attachment_file_ids, coalesce(brevo_message_id, ''), status, attempts, coalesce(error, ''), sent_at, created_at, updated_at`

func scanEmailLog(row pgx.Row) (model.EmailLog, error) {
	var l model.EmailLog
	err := row.Scan(&l.ID, &l.TemplateCode, &l.To, &l.CC, &l.Subject, &l.RelatedType, &l.RelatedID, &l.AttachmentFileIDs,
		&l.BrevoMessageID, &l.Status, &l.Attempts, &l.Error, &l.SentAt, &l.CreatedAt, &l.UpdatedAt)
	return l, notFound(err)
}

func emailNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CreateLog inserts a queued email log.
func (r *EmailRepository) CreateLog(ctx context.Context, l model.EmailLog) (model.EmailLog, error) {
	cc := l.CC
	if cc == nil {
		cc = []string{}
	}
	files := l.AttachmentFileIDs
	if files == nil {
		files = []uuid.UUID{}
	}
	out, err := scanEmailLog(r.pool.QueryRow(ctx, `INSERT INTO email_logs (template_code, to_addresses, cc_addresses, subject,
		related_type, related_id, attachment_file_ids, status) VALUES ($1, $2, $3, $4, $5, $6, $7, 'queued') RETURNING `+emailLogColumns,
		emailNullString(l.TemplateCode), l.To, cc, l.Subject, emailNullString(l.RelatedType), l.RelatedID, files))
	if err != nil {
		return out, fmt.Errorf("create email log: %w", err)
	}
	return out, nil
}

// GetLog returns an email log by ID.
func (r *EmailRepository) GetLog(ctx context.Context, id uuid.UUID) (model.EmailLog, error) {
	l, err := scanEmailLog(r.pool.QueryRow(ctx, `SELECT `+emailLogColumns+` FROM email_logs WHERE id = $1`, id))
	if err != nil {
		return l, fmt.Errorf("get email log: %w", err)
	}
	return l, nil
}

// ListLogs returns a page of email logs, newest first.
func (r *EmailRepository) ListLogs(ctx context.Context, f model.EmailLogFilter, limit, offset int) (Page[model.EmailLog], error) {
	w := &Where{}
	if f.RelatedType != "" {
		w.Add("related_type = %s", f.RelatedType)
	}
	if f.RelatedID != nil {
		w.Add("related_id = %s", *f.RelatedID)
	}
	if f.Status != "" {
		w.Add("status = %s", f.Status)
	}
	var page Page[model.EmailLog]
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM email_logs`+w.SQL(), w.Args()...).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("count email logs: %w", err)
	}
	lim, off := w.Arg(limit), w.Arg(offset)
	rows, err := r.pool.Query(ctx, `SELECT `+emailLogColumns+` FROM email_logs`+w.SQL()+` ORDER BY created_at DESC, id DESC LIMIT `+lim+` OFFSET `+off, w.Args()...)
	if err != nil {
		return page, fmt.Errorf("list email logs: %w", err)
	}
	items, err := collect(rows, scanEmailLog, "email logs")
	if err != nil {
		return page, err
	}
	page.Items = items
	return page, nil
}

// MarkSent records a successful delivery attempt.
func (r *EmailRepository) MarkSent(ctx context.Context, id uuid.UUID, messageID string, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `UPDATE email_logs SET status = 'sent', brevo_message_id = $2, attempts = attempts + 1,
		error = NULL, sent_at = $3 WHERE id = $1`, id, messageID, at)
	return affectedOne(tag, err, "mark email sent")
}

// MarkFailed records a terminal failed attempt.
func (r *EmailRepository) MarkFailed(ctx context.Context, id uuid.UUID, msg string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE email_logs SET status = 'failed', error = $2, attempts = attempts + 1 WHERE id = $1`, id, msg)
	return affectedOne(tag, err, "mark email failed")
}

// MarkRetrying records a failed attempt that will be retried.
func (r *EmailRepository) MarkRetrying(ctx context.Context, id uuid.UUID, msg string) error {
	tag, err := r.pool.Exec(ctx, `UPDATE email_logs SET status = 'queued', error = $2, attempts = attempts + 1 WHERE id = $1`, id, msg)
	return affectedOne(tag, err, "mark email retrying")
}

// RequeueFailed moves a failed log back to queued and reports whether it was failed.
func (r *EmailRepository) RequeueFailed(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `UPDATE email_logs SET status = 'queued', error = NULL WHERE id = $1 AND status = 'failed'`, id)
	if err != nil {
		return false, fmt.Errorf("requeue email: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// LatestJobPayload returns the payload of the newest send job of a log, which holds the rendered body.
func (r *EmailRepository) LatestJobPayload(ctx context.Context, kind string, logID uuid.UUID) (json.RawMessage, error) {
	var payload json.RawMessage
	err := r.pool.QueryRow(ctx, `SELECT payload FROM jobs WHERE kind = $1 AND payload->>'log_id' = $2 ORDER BY created_at DESC LIMIT 1`,
		kind, logID.String()).Scan(&payload)
	if err != nil {
		return nil, fmt.Errorf("load email job payload: %w", notFound(err))
	}
	return payload, nil
}
