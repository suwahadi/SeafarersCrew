package admin

import (
	"context"
	"net/http"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
)

// BackupService is what the backup controller needs.
type BackupService interface {
	Start(ctx context.Context) (model.Backup, model.Job, error)
	List(ctx context.Context, page, perPage int) (repository.Page[model.Backup], error)
}

// BackupHandler serves backup history and manual backups.
type BackupHandler struct {
	backups BackupService
}

// NewBackupHandler builds a BackupHandler.
func NewBackupHandler(backups BackupService) *BackupHandler {
	return &BackupHandler{backups: backups}
}

// Index handles GET /backups.
func (h *BackupHandler) Index(w http.ResponseWriter, r *http.Request) {
	p := request.ParseList(r.URL.Query())
	page, err := h.backups.List(r.Context(), p.Page, p.PerPage)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	items := make([]resource.Backup, len(page.Items))
	for i, b := range page.Items {
		items[i] = resource.NewBackup(b)
	}
	resource.List(w, items, resource.Meta{Page: p.Page, PerPage: p.PerPage, Total: page.Total})
}

// Run handles POST /backups/run.
func (h *BackupHandler) Run(w http.ResponseWriter, r *http.Request) {
	_, j, err := h.backups.Start(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	Accepted(w, j)
}
