package model

import (
	"time"

	"github.com/google/uuid"
)

// Backup statuses.
const (
	BackupRunning   = "running"
	BackupSucceeded = "succeeded"
	BackupFailed    = "failed"
)

// Backup records one database dump uploaded to object storage.
type Backup struct {
	ID         uuid.UUID
	StorageKey string
	SizeBytes  int64
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
	Error      string
}
