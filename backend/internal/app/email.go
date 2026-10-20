package app

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/handler/admin"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/brevo"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/s3"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/turnstile"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

const integrationTimeout = 30 * time.Second

func (a *App) wireEmail() {
	a.brevo = brevo.New(brevo.DefaultBaseURL, integrationTimeout)
	a.Email = service.NewEmailService(
		repository.NewEmailRepository(a.DB),
		a.brevo,
		emailAttachments{files: a.Files, storage: a.Storage},
		a.Settings,
		a.Jobs,
		time.Now,
	)
	a.Jobs.Register(service.EmailSendJob, a.Email.Handle)
}

func (a *App) emailHandler() *admin.EmailHandler {
	tester := service.NewEmailIntegrationTester(a.brevo, a.Storage, turnstile.New(turnstile.DefaultURL, integrationTimeout), a.Settings)
	return admin.NewEmailHandler(a.Email, tester)
}

type emailAttachments struct {
	files   *service.FileService
	storage *s3.Storage
}

func (e emailAttachments) Load(ctx context.Context, id uuid.UUID) (brevo.Attachment, error) {
	f, err := e.files.Get(ctx, id)
	if err != nil {
		return brevo.Attachment{}, err
	}
	data, err := e.storage.Get(ctx, f.StorageKey)
	if err != nil {
		return brevo.Attachment{}, err
	}
	base := strings.TrimSuffix(f.OriginalName, path.Ext(f.OriginalName))
	if base == "" {
		base = "attachment"
	}
	return brevo.Attachment{Name: base + path.Ext(f.StorageKey), Content: data}, nil
}
