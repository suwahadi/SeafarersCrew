package resource

import (
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// Admin is the public shape of the administrator.
type Admin struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Email              string    `json:"email"`
	MustChangePassword bool      `json:"must_change_password"`
	PasswordChangedAt  time.Time `json:"password_changed_at"`
}

// Session is returned by login, recovery and /me.
type Session struct {
	Admin     Admin  `json:"admin"`
	CSRFToken string `json:"csrf_token"`
}

// NewAdmin maps a model.Admin.
func NewAdmin(a model.Admin) Admin {
	return Admin{
		ID:                 a.ID.String(),
		Name:               a.Name,
		Email:              a.Email,
		MustChangePassword: a.MustChangePassword,
		PasswordChangedAt:  a.PasswordChangedAt.UTC(),
	}
}

// RecoveryCodes is returned once when codes are regenerated.
type RecoveryCodes struct {
	Codes []string `json:"codes"`
}
