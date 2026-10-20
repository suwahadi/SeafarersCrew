package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Job statuses.
const (
	JobQueued    = "queued"
	JobRunning   = "running"
	JobSucceeded = "succeeded"
	JobFailed    = "failed"
)

// Job is a background task tracked in the jobs table.
type Job struct {
	ID         uuid.UUID
	Kind       string
	Status     string
	Payload    json.RawMessage
	Result     json.RawMessage
	Error      string
	Attempts   int
	StartedAt  *time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
}
