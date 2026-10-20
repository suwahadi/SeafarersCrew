// Package model holds domain structs shared by services and repositories.
package model

import (
	"time"

	"github.com/google/uuid"
)

// Admin is the single administrator account.
type Admin struct {
	ID                 uuid.UUID
	Name               string
	Email              string
	PasswordHash       string
	PasswordChangedAt  time.Time
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// RecoveryCode is a hashed one-time recovery code.
type RecoveryCode struct {
	ID       uuid.UUID
	AdminID  uuid.UUID
	CodeHash string
	UsedAt   *time.Time
}

// Session is the server-side session stored in Redis.
type Session struct {
	AdminID   uuid.UUID `json:"admin_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	LastSeen  time.Time `json:"last_seen"`
}
