package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type stubBackups struct {
	started int
	err     error
	items   []model.Backup
}

func (s *stubBackups) Start(context.Context) (model.Backup, model.Job, error) {
	s.started++
	if s.err != nil {
		return model.Backup{}, model.Job{}, s.err
	}
	return model.Backup{}, model.Job{ID: uuid.New(), Kind: "backup.run", Status: "queued", CreatedAt: time.Now()}, nil
}

func (s *stubBackups) List(context.Context, int, int) (repository.Page[model.Backup], error) {
	return repository.Page[model.Backup]{Items: s.items, Total: len(s.items)}, nil
}

func TestBackupRunAnswersWithAnAcceptedJob(t *testing.T) {
	s := &stubBackups{}
	rec := httptest.NewRecorder()
	NewBackupHandler(s).Run(rec, httptest.NewRequest(http.MethodPost, "/backups/run", nil))
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Contains(t, rec.Body.String(), `"kind":"backup.run"`)
	assert.Equal(t, 1, s.started)
}

func TestBackupRunReportsInvalidSettings(t *testing.T) {
	s := &stubBackups{err: apperr.Validation(map[string]string{"prefix": "Backups cannot be stored in the public folder."})}
	rec := httptest.NewRecorder()
	NewBackupHandler(s).Run(rec, httptest.NewRequest(http.MethodPost, "/backups/run", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "prefix")
}

func TestBackupIndexListsRunsWithFileNameOnly(t *testing.T) {
	finished := time.Date(2026, 10, 21, 2, 3, 0, 0, time.UTC)
	s := &stubBackups{items: []model.Backup{{ID: uuid.New(), Status: model.BackupSucceeded, StorageKey: "backups/seafarers-20261021-020000.dump", SizeBytes: 2048, StartedAt: finished.Add(-3 * time.Minute), FinishedAt: &finished}}}
	rec := httptest.NewRecorder()
	NewBackupHandler(s).Index(rec, httptest.NewRequest(http.MethodGet, "/backups", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"file_name":"seafarers-20261021-020000.dump"`)
	assert.NotContains(t, rec.Body.String(), "backups/seafarers")
	assert.Contains(t, rec.Body.String(), `"total":1`)
}
