package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func TestEmailTemplatesReadAndUpdate(t *testing.T) {
	truncate(t, "email_templates")
	ctx := context.Background()
	_, err := testDB.Exec(ctx, `INSERT INTO email_templates (code, name, subject, body_html) VALUES ('send_cv', 'Send CV', 'CV', '<p>x</p>')`)
	require.NoError(t, err)
	repo := repository.NewEmailRepository(testDB)

	list, err := repo.ListTemplates(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)

	updated, err := repo.UpdateTemplate(ctx, "send_cv", "Crew CV {crew_name}", "<p>y</p>")
	require.NoError(t, err)
	assert.Equal(t, "Crew CV {crew_name}", updated.Subject)

	_, err = repo.GetTemplate(ctx, "missing")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestEmailLogsLifecycleAndFilters(t *testing.T) {
	truncate(t, "email_logs", "jobs")
	ctx := context.Background()
	repo := repository.NewEmailRepository(testDB)
	crew, contract, file := uuid.New(), uuid.New(), uuid.New()

	a, err := repo.CreateLog(ctx, model.EmailLog{TemplateCode: "send_cv", To: []string{"a@example.com"}, Subject: "CV", RelatedType: "crew", RelatedID: &crew, AttachmentFileIDs: []uuid.UUID{file}})
	require.NoError(t, err)
	assert.Equal(t, model.EmailQueued, a.Status)
	assert.Equal(t, []string{}, a.CC)
	assert.Equal(t, []uuid.UUID{file}, a.AttachmentFileIDs)

	b, err := repo.CreateLog(ctx, model.EmailLog{To: []string{"b@example.com"}, CC: []string{"c@example.com"}, Subject: "Contract", RelatedType: "contract", RelatedID: &contract})
	require.NoError(t, err)
	assert.Empty(t, b.TemplateCode)

	require.NoError(t, repo.MarkRetrying(ctx, a.ID, "503"))
	require.NoError(t, repo.MarkSent(ctx, a.ID, "<id@brevo>", time.Now()))
	require.NoError(t, repo.MarkFailed(ctx, b.ID, "400 invalid"))

	sent, err := repo.GetLog(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, model.EmailSent, sent.Status)
	assert.Equal(t, 2, sent.Attempts)
	assert.Empty(t, sent.Error)
	assert.NotNil(t, sent.SentAt)

	page, err := repo.ListLogs(ctx, model.EmailLogFilter{RelatedType: "crew", RelatedID: &crew}, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	assert.Equal(t, a.ID, page.Items[0].ID)

	page, err = repo.ListLogs(ctx, model.EmailLogFilter{Status: model.EmailFailed}, 10, 0)
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	assert.Equal(t, "400 invalid", page.Items[0].Error)

	page, err = repo.ListLogs(ctx, model.EmailLogFilter{}, 1, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, b.ID, page.Items[0].ID, "newest first")

	ok, err := repo.RequeueFailed(ctx, a.ID)
	require.NoError(t, err)
	assert.False(t, ok, "a sent email cannot be requeued")
	ok, err = repo.RequeueFailed(ctx, b.ID)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestLatestJobPayloadFindsNewestSendJob(t *testing.T) {
	truncate(t, "jobs")
	ctx := context.Background()
	repo := repository.NewEmailRepository(testDB)
	jobs := repository.NewJobRepository(testDB)
	logID := uuid.New()
	_, err := jobs.Create(ctx, "email_send", []byte(`{"log_id":"`+logID.String()+`","body_html":"old"}`))
	require.NoError(t, err)
	time.Sleep(2 * time.Millisecond)
	_, err = jobs.Create(ctx, "email_send", []byte(`{"log_id":"`+logID.String()+`","body_html":"new"}`))
	require.NoError(t, err)

	payload, err := repo.LatestJobPayload(ctx, "email_send", logID)
	require.NoError(t, err)
	assert.JSONEq(t, `{"log_id":"`+logID.String()+`","body_html":"new"}`, string(payload))

	_, err = repo.LatestJobPayload(ctx, "email_send", uuid.New())
	assert.ErrorIs(t, err, repository.ErrNotFound)
}
