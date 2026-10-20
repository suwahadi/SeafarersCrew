package resource

import (
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// EmailTemplate is an email template with the tags it may use.
type EmailTemplate struct {
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Subject       string    `json:"subject"`
	BodyHTML      string    `json:"body_html,omitempty"`
	AvailableTags []string  `json:"available_tags,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewEmailTemplate maps a template; tags is nil for list views.
func NewEmailTemplate(t model.EmailTemplate, withBody bool, tags []string) EmailTemplate {
	out := EmailTemplate{Code: t.Code, Name: t.Name, Subject: t.Subject, AvailableTags: tags, UpdatedAt: t.UpdatedAt}
	if withBody {
		out.BodyHTML = t.BodyHTML
	}
	return out
}

// EmailLog is one dispatched email.
type EmailLog struct {
	ID                uuid.UUID   `json:"id"`
	TemplateCode      string      `json:"template_code"`
	To                []string    `json:"to"`
	CC                []string    `json:"cc"`
	Subject           string      `json:"subject"`
	RelatedType       string      `json:"related_type"`
	RelatedID         *uuid.UUID  `json:"related_id"`
	AttachmentFileIDs []uuid.UUID `json:"attachment_file_ids"`
	BrevoMessageID    string      `json:"brevo_message_id"`
	Status            string      `json:"status"`
	Attempts          int         `json:"attempts"`
	Error             string      `json:"error"`
	SentAt            *time.Time  `json:"sent_at"`
	CreatedAt         time.Time   `json:"created_at"`
}

// NewEmailLog maps an email log.
func NewEmailLog(l model.EmailLog) EmailLog {
	return EmailLog{ID: l.ID, TemplateCode: l.TemplateCode, To: l.To, CC: l.CC, Subject: l.Subject, RelatedType: l.RelatedType,
		RelatedID: l.RelatedID, AttachmentFileIDs: l.AttachmentFileIDs, BrevoMessageID: l.BrevoMessageID, Status: l.Status,
		Attempts: l.Attempts, Error: l.Error, SentAt: l.SentAt, CreatedAt: l.CreatedAt}
}

// QueuedEmail is returned when an email is accepted for delivery.
type QueuedEmail struct {
	Log   EmailLog  `json:"log"`
	JobID uuid.UUID `json:"job_id"`
}
