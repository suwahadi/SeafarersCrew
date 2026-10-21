package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
)

// ContactMaxMessage limits the contact form message.
const ContactMaxMessage = 2000

// ContactStore persists contact messages.
type ContactStore interface {
	Create(ctx context.Context, m model.ContactMessage) (model.ContactMessage, error)
	Get(ctx context.Context, id uuid.UUID) (model.ContactMessage, error)
	SetFlags(ctx context.Context, id uuid.UUID, isRead, isArchived *bool) (model.ContactMessage, error)
	List(ctx context.Context, f model.ContactFilter, limit, offset int) (repository.Page[model.ContactMessage], error)
	UnreadCount(ctx context.Context) (int, error)
}

// ContactInput is the raw content of the public contact form.
type ContactInput struct {
	Name           string
	Email          string
	Phone          string
	Subject        string
	Message        string
	Website        string
	TurnstileToken string
	IP             string
}

// ContactService receives contact messages and runs the inbox (PRD 7.10, 8.3).
type ContactService struct {
	store     ContactStore
	guard     SubmissionGuard
	mailer    SubmissionMailer
	dashboard CacheInvalidator
	adminURL  string
}

// NewContactService builds a ContactService.
func NewContactService(store ContactStore, guard SubmissionGuard, mailer SubmissionMailer, dashboard CacheInvalidator, adminURL string) *ContactService {
	return &ContactService{store: store, guard: guard, mailer: mailer, dashboard: dashboard, adminURL: strings.TrimRight(adminURL, "/")}
}

// Submit stores a contact message and notifies the admin. A filled honeypot field is accepted silently without storing anything.
func (s *ContactService) Submit(ctx context.Context, in ContactInput) error {
	if strings.TrimSpace(in.Website) != "" {
		return nil
	}
	if err := s.guard.Verify(ctx, in.TurnstileToken, in.IP); err != nil {
		return err
	}
	m, err := validateContact(in)
	if err != nil {
		return err
	}
	created, err := s.store.Create(ctx, m)
	if err != nil {
		return err
	}
	s.announce(ctx, created)
	return nil
}

func validateContact(in ContactInput) (model.ContactMessage, error) {
	e := request.Errors{}
	m := model.ContactMessage{
		Name:      plainText(in.Name, false),
		Email:     strings.ToLower(strings.TrimSpace(in.Email)),
		Phone:     strings.TrimSpace(in.Phone),
		Subject:   plainText(in.Subject, false),
		Message:   plainText(in.Message, true),
		IPAddress: cleanIP(in.IP),
	}
	e.Required("name", m.Name, 120)
	e.Required("email", m.Email, 254)
	e.Email("email", m.Email)
	e.Phone("phone", m.Phone)
	e.Required("subject", m.Subject, 200)
	e.Required("message", m.Message, ContactMaxMessage)
	if len(e) > 0 {
		return model.ContactMessage{}, apperr.Validation(e)
	}
	return m, nil
}

func (s *ContactService) announce(ctx context.Context, m model.ContactMessage) {
	ctx = context.WithoutCancel(ctx)
	log := logctx.From(ctx)
	if err := s.dashboard.Invalidate(ctx); err != nil {
		log.WarnContext(ctx, "dashboard cache not invalidated after contact message", "error", err)
	}
	vars := map[string]string{
		"subject": m.Subject, "sender_name": m.Name, "sender_email": m.Email, "sender_phone": m.Phone,
		"message": m.Message, "admin_url": s.adminURL + "/contact-messages",
	}
	if err := s.mailer.Notify(ctx, "admin_new_contact", vars, "contact_message", &m.ID); err != nil {
		log.WarnContext(ctx, "admin contact notification not queued", "message_id", m.ID, "error", err)
	}
}

// List returns one page of messages; the status filter is all, unread, read or archived.
func (s *ContactService) List(ctx context.Context, f model.ContactFilter, page, perPage int) (repository.Page[model.ContactMessage], error) {
	return s.store.List(ctx, f, perPage, (page-1)*perPage)
}

// Get returns one message without changing its flags.
func (s *ContactService) Get(ctx context.Context, id uuid.UUID) (model.ContactMessage, error) {
	m, err := s.store.Get(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return m, apperr.NotFound("Message")
	}
	return m, err
}

// SetFlags changes the read and archived flags that are given.
func (s *ContactService) SetFlags(ctx context.Context, id uuid.UUID, isRead, isArchived *bool) (model.ContactMessage, error) {
	if isRead == nil && isArchived == nil {
		return model.ContactMessage{}, apperr.Validation(map[string]string{"is_read": "Send is_read, is_archived or both."})
	}
	m, err := s.store.SetFlags(ctx, id, isRead, isArchived)
	if errors.Is(err, repository.ErrNotFound) {
		return m, apperr.NotFound("Message")
	}
	return m, err
}

// UnreadCount returns the number of unread, unarchived messages.
func (s *ContactService) UnreadCount(ctx context.Context) (int, error) {
	return s.store.UnreadCount(ctx)
}
