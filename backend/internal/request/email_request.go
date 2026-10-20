package request

import (
	"github.com/google/uuid"
)

// SendEmail is the body of POST /emails/send.
type SendEmail struct {
	TemplateCode      string            `json:"template_code"`
	To                []string          `json:"to"`
	CC                []string          `json:"cc"`
	Subject           string            `json:"subject"`
	BodyHTML          string            `json:"body_html"`
	Variables         map[string]string `json:"variables"`
	RelatedType       string            `json:"related_type"`
	RelatedID         string            `json:"related_id"`
	AttachmentFileIDs []string          `json:"attachment_file_ids"`

	relatedID   *uuid.UUID
	attachments []uuid.UUID
}

// Validate implements Validator.
func (r *SendEmail) Validate() map[string]string {
	Trim(&r.TemplateCode, &r.RelatedType, &r.RelatedID)
	e := Errors{}
	r.relatedID = e.UUID("related_id", r.RelatedID)
	r.attachments = make([]uuid.UUID, 0, len(r.AttachmentFileIDs))
	for _, raw := range r.AttachmentFileIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			e["attachment_file_ids"] = "Contains an invalid file identifier."
			continue
		}
		r.attachments = append(r.attachments, id)
	}
	return e
}

// RelatedUUID returns the parsed related ID.
func (r *SendEmail) RelatedUUID() *uuid.UUID { return r.relatedID }

// Attachments returns the parsed attachment file IDs.
func (r *SendEmail) Attachments() []uuid.UUID { return r.attachments }

// EmailTemplate is the body of PUT /email-templates/{code}.
type EmailTemplate struct {
	Subject  string `json:"subject"`
	BodyHTML string `json:"body_html"`
}

// EmailPreview is the body of POST /email-templates/{code}/preview.
type EmailPreview struct {
	Subject   string            `json:"subject"`
	BodyHTML  string            `json:"body_html"`
	Variables map[string]string `json:"variables"`
}
