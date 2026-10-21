package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type memContacts struct {
	rows []model.ContactMessage
}

func (m *memContacts) Create(_ context.Context, c model.ContactMessage) (model.ContactMessage, error) {
	c.ID, c.CreatedAt = uuid.New(), time.Now()
	m.rows = append(m.rows, c)
	return c, nil
}

func (m *memContacts) Get(_ context.Context, id uuid.UUID) (model.ContactMessage, error) {
	for _, c := range m.rows {
		if c.ID == id {
			return c, nil
		}
	}
	return model.ContactMessage{}, repository.ErrNotFound
}

func (m *memContacts) SetFlags(_ context.Context, id uuid.UUID, isRead, isArchived *bool) (model.ContactMessage, error) {
	for i, c := range m.rows {
		if c.ID != id {
			continue
		}
		if isRead != nil {
			c.IsRead = *isRead
		}
		if isArchived != nil {
			c.IsArchived = *isArchived
		}
		m.rows[i] = c
		return c, nil
	}
	return model.ContactMessage{}, repository.ErrNotFound
}

func (m *memContacts) List(context.Context, model.ContactFilter, int, int) (repository.Page[model.ContactMessage], error) {
	return repository.Page[model.ContactMessage]{Items: m.rows, Total: len(m.rows)}, nil
}

func (m *memContacts) UnreadCount(context.Context) (int, error) {
	n := 0
	for _, c := range m.rows {
		if !c.IsRead && !c.IsArchived {
			n++
		}
	}
	return n, nil
}

type contactFixture struct {
	svc       *ContactService
	store     *memContacts
	guard     *fakeGuard
	mailer    *fakeMailer
	dashboard *countingInvalidator
}

func newContactFixture() *contactFixture {
	f := &contactFixture{store: &memContacts{}, guard: &fakeGuard{}, mailer: &fakeMailer{}, dashboard: &countingInvalidator{}}
	f.svc = NewContactService(f.store, f.guard, f.mailer, f.dashboard, "https://admin.example.com/")
	return f
}

func validContact() ContactInput {
	return ContactInput{Name: " John Doe ", Email: "John@Example.com", Phone: "+6281234567890", Subject: "Crew request", Message: "Do you have a 2nd Engineer?",
		TurnstileToken: "tok", IP: "203.0.113.9"}
}

func TestSubmitContactStoresMessageAndNotifiesAdmin(t *testing.T) {
	f := newContactFixture()

	err := f.svc.Submit(context.Background(), validContact())

	require.NoError(t, err)
	require.Len(t, f.store.rows, 1)
	m := f.store.rows[0]
	assert.Equal(t, "John Doe", m.Name)
	assert.Equal(t, "john@example.com", m.Email)
	assert.Equal(t, "203.0.113.9", m.IPAddress)
	assert.False(t, m.IsRead)
	require.Len(t, f.mailer.notified, 1)
	assert.Equal(t, "admin_new_contact", f.mailer.notified[0].code)
	assert.Equal(t, "contact_message", f.mailer.notified[0].relatedType)
	assert.Equal(t, "https://admin.example.com/contact-messages", f.mailer.notified[0].vars["admin_url"])
	assert.Equal(t, "Crew request", f.mailer.notified[0].vars["subject"])
	assert.Equal(t, 1, f.dashboard.calls)
}

func TestSubmitContactDropsHoneypotSubmissionsSilently(t *testing.T) {
	f := newContactFixture()
	in := validContact()
	in.Website = "http://spam.example"

	err := f.svc.Submit(context.Background(), in)

	require.NoError(t, err)
	assert.Zero(t, f.guard.called)
	assert.Empty(t, f.store.rows)
	assert.Empty(t, f.mailer.notified)
}

func TestSubmitContactVerifiesTurnstileBeforeValidating(t *testing.T) {
	f := newContactFixture()
	f.guard.err = ErrTurnstileFailed

	err := f.svc.Submit(context.Background(), ContactInput{})

	requireAppError(t, err, 422, "turnstile_failed")
	assert.Empty(t, f.store.rows)
}

func TestSubmitContactValidatesFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ContactInput)
		field  string
	}{
		{"missing name", func(in *ContactInput) { in.Name = "" }, "name"},
		{"invalid email", func(in *ContactInput) { in.Email = "john@" }, "email"},
		{"phone not E.164", func(in *ContactInput) { in.Phone = "12345" }, "phone"},
		{"missing subject", func(in *ContactInput) { in.Subject = " " }, "subject"},
		{"subject too long", func(in *ContactInput) { in.Subject = strings.Repeat("a", 201) }, "subject"},
		{"missing message", func(in *ContactInput) { in.Message = "" }, "message"},
		{"message above 2000 characters", func(in *ContactInput) { in.Message = strings.Repeat("é", 2001) }, "message"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newContactFixture()
			in := validContact()
			tc.mutate(&in)

			err := f.svc.Submit(context.Background(), in)

			fields := appErrorFields(t, err, 422, "validation_failed")
			assert.Contains(t, fields, tc.field)
			assert.Empty(t, f.store.rows)
		})
	}
}

func TestSubmitContactAcceptsMissingPhoneAndExactlyMaxMessage(t *testing.T) {
	f := newContactFixture()
	in := validContact()
	in.Phone = ""
	in.Message = strings.Repeat("é", ContactMaxMessage)

	require.NoError(t, f.svc.Submit(context.Background(), in))
}

func TestSubmitContactSucceedsWhenNotificationFails(t *testing.T) {
	f := newContactFixture()
	f.mailer.notifyErr = errors.New("queue down")

	require.NoError(t, f.svc.Submit(context.Background(), validContact()))
	assert.Len(t, f.store.rows, 1)
}

func TestContactInboxFlagsAreIndependentAndViewingMarksNothing(t *testing.T) {
	f := newContactFixture()
	require.NoError(t, f.svc.Submit(context.Background(), validContact()))
	id := f.store.rows[0].ID
	yes := true

	viewed, err := f.svc.Get(context.Background(), id)
	require.NoError(t, err)
	assert.False(t, viewed.IsRead)

	archived, err := f.svc.SetFlags(context.Background(), id, nil, &yes)
	require.NoError(t, err)
	assert.True(t, archived.IsArchived)
	assert.False(t, archived.IsRead, "archiving does not mark a message read")
	n, err := f.svc.UnreadCount(context.Background())
	require.NoError(t, err)
	assert.Zero(t, n, "archived messages do not count as unread")

	_, err = f.svc.SetFlags(context.Background(), id, nil, nil)
	fields := appErrorFields(t, err, 422, "validation_failed")
	assert.Contains(t, fields, "is_read")

	_, err = f.svc.SetFlags(context.Background(), uuid.New(), &yes, nil)
	requireAppError(t, err, 404, "not_found")
	_, err = f.svc.Get(context.Background(), uuid.New())
	requireAppError(t, err, 404, "not_found")
}
