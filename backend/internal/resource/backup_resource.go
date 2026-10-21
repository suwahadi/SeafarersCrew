package resource

import (
	"path"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// Backup is a database backup run as shown to administrators.
type Backup struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	FileName   string     `json:"file_name"`
	SizeBytes  int64      `json:"size_bytes"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Error      string     `json:"error"`
}

// NewBackup maps a backup run.
func NewBackup(b model.Backup) Backup {
	name := ""
	if b.StorageKey != "" {
		name = path.Base(b.StorageKey)
	}
	return Backup{ID: b.ID.String(), Status: b.Status, FileName: name, SizeBytes: b.SizeBytes, StartedAt: b.StartedAt, FinishedAt: b.FinishedAt, Error: b.Error}
}
