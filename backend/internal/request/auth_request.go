package request

import (
	"strings"
)

// Login is the body of POST /auth/login.
type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate implements Validator.
func (r *Login) Validate() map[string]string {
	f := map[string]string{}
	if strings.TrimSpace(r.Email) == "" {
		f["email"] = "The email is required."
	}
	if r.Password == "" {
		f["password"] = "The password is required."
	}
	return f
}

// Recovery is the body of POST /auth/recovery.
type Recovery struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"new_password"`
}

// Validate implements Validator.
func (r *Recovery) Validate() map[string]string {
	f := map[string]string{}
	if strings.TrimSpace(r.Email) == "" {
		f["email"] = "The email is required."
	}
	if strings.TrimSpace(r.Code) == "" {
		f["code"] = "The recovery code is required."
	}
	if r.NewPassword == "" {
		f["new_password"] = "The new password is required."
	}
	return f
}

// ChangePassword is the body of PUT /me/password.
type ChangePassword struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// Validate implements Validator.
func (r *ChangePassword) Validate() map[string]string {
	f := map[string]string{}
	if r.CurrentPassword == "" {
		f["current_password"] = "The current password is required."
	}
	if r.NewPassword == "" {
		f["new_password"] = "The new password is required."
	}
	return f
}

// UpdateProfile is the body of PUT /me.
type UpdateProfile struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
