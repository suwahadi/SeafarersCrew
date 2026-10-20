package model

import (
	"time"

	"github.com/google/uuid"
)

// Email log statuses.
const (
	EmailQueued = "queued"
	EmailSent   = "sent"
	EmailFailed = "failed"
)

// EmailTemplate is an editable email with {snake_case} tags.
type EmailTemplate struct {
	ID        uuid.UUID
	Code      string
	Name      string
	Subject   string
	BodyHTML  string
	UpdatedAt time.Time
}

// EmailLog records one dispatched email.
type EmailLog struct {
	ID                uuid.UUID
	TemplateCode      string
	To                []string
	CC                []string
	Subject           string
	RelatedType       string
	RelatedID         *uuid.UUID
	AttachmentFileIDs []uuid.UUID
	BrevoMessageID    string
	Status            string
	Attempts          int
	Error             string
	SentAt            *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// EmailLogFilter narrows the email log list.
type EmailLogFilter struct {
	RelatedType string
	RelatedID   *uuid.UUID
	Status      string
}
