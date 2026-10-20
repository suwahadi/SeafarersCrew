// Package service holds business rules and orchestrates repositories.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// Authentication limits from PRD Section 5.
const (
	MaxLoginFailures      = 5
	LoginFailureWindow    = 15 * time.Minute
	RecoveryCodeCount     = 10
	MinPasswordLength     = 12
	MaxPasswordLength     = 128
	generatedPasswordSize = 18
)

// AdminStore is the persistence the auth service needs.
type AdminStore interface {
	Count(ctx context.Context) (int, error)
	FindByEmail(ctx context.Context, email string) (model.Admin, error)
	FindByID(ctx context.Context, id uuid.UUID) (model.Admin, error)
	CreateWithRecoveryCodes(ctx context.Context, a model.Admin, codeHashes []string) (model.Admin, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, hash string, mustChange bool) error
	UpdateProfile(ctx context.Context, id uuid.UUID, name, email string) (model.Admin, error)
	UnusedRecoveryCodes(ctx context.Context, adminID uuid.UUID) ([]model.RecoveryCode, error)
	MarkRecoveryCodeUsed(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)
	ReplaceRecoveryCodes(ctx context.Context, adminID uuid.UUID, codeHashes []string) error
}

// SessionStore persists server-side sessions.
type SessionStore interface {
	Create(ctx context.Context, adminID uuid.UUID, ip, userAgent string) (string, error)
	Get(ctx context.Context, rawID string) (model.Session, error)
	Delete(ctx context.Context, rawID string) error
	DeleteAllForAdmin(ctx context.Context, adminID uuid.UUID) error
}

// Counter is a fixed-window counter used for rate limiting.
type Counter interface {
	Hit(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
	Count(ctx context.Context, key string) (int64, time.Duration, error)
	Extend(ctx context.Context, key string, window time.Duration) error
	Reset(ctx context.Context, key string) error
}

// SecretHasher hashes and verifies passwords and recovery codes.
type SecretHasher interface {
	Hash(secret string) (string, error)
	Verify(secret, encoded string) (bool, error)
}

// AuthService implements login, sessions, password changes and recovery.
type AuthService struct {
	admins    AdminStore
	sessions  SessionStore
	counter   Counter
	hasher    SecretHasher
	now       func() time.Time
	dummyHash string
}

// NewAuthService builds an AuthService.
func NewAuthService(admins AdminStore, sessions SessionStore, counter Counter, hasher SecretHasher, now func() time.Time) (*AuthService, error) {
	dummy, err := hasher.Hash("timing-equalizer")
	if err != nil {
		return nil, fmt.Errorf("prepare dummy hash: %w", err)
	}
	return &AuthService{admins: admins, sessions: sessions, counter: counter, hasher: hasher, now: now, dummyHash: dummy}, nil
}

// Client identifies the caller for sessions and rate limits.
type Client struct {
	IP        string
	UserAgent string
}

// LoginResult carries the new session and the authenticated admin.
type LoginResult struct {
	SessionID string
	Admin     model.Admin
}

// Login verifies credentials, enforces the failure lockout and opens a fresh session.
func (s *AuthService) Login(ctx context.Context, email, password string, client Client, previousSessionID string) (LoginResult, error) {
	key := "login:" + client.IP
	if err := s.ensureNotLocked(ctx, key); err != nil {
		return LoginResult{}, err
	}
	admin, ok, err := s.checkCredentials(ctx, email, password)
	if err != nil {
		return LoginResult{}, err
	}
	if !ok {
		return LoginResult{}, s.recordFailure(ctx, key)
	}
	if err := s.counter.Reset(ctx, key); err != nil {
		return LoginResult{}, err
	}
	sid, err := s.rotateSession(ctx, admin.ID, client, previousSessionID)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{SessionID: sid, Admin: admin}, nil
}

// Logout deletes the session.
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.sessions.Delete(ctx, sessionID)
}

// Authenticate resolves a session ID to its admin.
func (s *AuthService) Authenticate(ctx context.Context, sessionID string) (model.Admin, error) {
	if sessionID == "" {
		return model.Admin{}, apperr.ErrUnauthenticated
	}
	sess, err := s.sessions.Get(ctx, sessionID)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Admin{}, apperr.ErrUnauthenticated
	}
	if err != nil {
		return model.Admin{}, err
	}
	admin, err := s.admins.FindByID(ctx, sess.AdminID)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Admin{}, apperr.ErrUnauthenticated
	}
	if err != nil {
		return model.Admin{}, err
	}
	return admin, nil
}

// ChangePassword verifies the current password, stores the new one, revokes all sessions and opens a new one.
func (s *AuthService) ChangePassword(ctx context.Context, adminID uuid.UUID, current, next string, client Client) (string, error) {
	admin, err := s.admins.FindByID(ctx, adminID)
	if err != nil {
		return "", fmt.Errorf("load admin: %w", err)
	}
	ok, err := s.hasher.Verify(current, admin.PasswordHash)
	if err != nil {
		return "", fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return "", apperr.Validation(map[string]string{"current_password": "The current password is incorrect."})
	}
	if msg := validateNewPassword(next); msg != "" {
		return "", apperr.Validation(map[string]string{"new_password": msg})
	}
	if current == next {
		return "", apperr.Validation(map[string]string{"new_password": "The new password must differ from the current password."})
	}
	return s.setPasswordAndRelogin(ctx, adminID, next, client)
}

// Recover signs in with a one-time recovery code and sets a new password.
func (s *AuthService) Recover(ctx context.Context, email, code, next string, client Client) (LoginResult, error) {
	key := "recovery:" + client.IP
	if err := s.ensureNotLocked(ctx, key); err != nil {
		return LoginResult{}, err
	}
	if msg := validateNewPassword(next); msg != "" {
		return LoginResult{}, apperr.Validation(map[string]string{"new_password": msg})
	}
	admin, err := s.admins.FindByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, repository.ErrNotFound) {
		return LoginResult{}, s.recordFailure(ctx, key)
	}
	if err != nil {
		return LoginResult{}, err
	}
	used, err := s.consumeRecoveryCode(ctx, admin.ID, code)
	if err != nil {
		return LoginResult{}, err
	}
	if !used {
		return LoginResult{}, s.recordFailure(ctx, key)
	}
	if err := s.counter.Reset(ctx, key); err != nil {
		return LoginResult{}, err
	}
	sid, err := s.setPasswordAndRelogin(ctx, admin.ID, next, client)
	if err != nil {
		return LoginResult{}, err
	}
	admin.MustChangePassword = false
	return LoginResult{SessionID: sid, Admin: admin}, nil
}

// CreateAdminResult carries one-time values printed by the CLI.
type CreateAdminResult struct {
	Admin             model.Admin
	GeneratedPassword string
	RecoveryCodes     []string
}

// CreateAdmin creates the single administrator; it fails when one already exists.
func (s *AuthService) CreateAdmin(ctx context.Context, name, email, password string) (CreateAdminResult, error) {
	n, err := s.admins.Count(ctx)
	if err != nil {
		return CreateAdminResult{}, err
	}
	if n > 0 {
		return CreateAdminResult{}, apperr.Conflict("admin_exists", "An administrator account already exists.")
	}
	fields := map[string]string{}
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" {
		fields["name"] = "The name is required."
	}
	if !validEmail(email) {
		fields["email"] = "A valid email is required."
	}
	generated := ""
	if password == "" {
		if generated, err = token.URLSafe(generatedPasswordSize); err != nil {
			return CreateAdminResult{}, err
		}
		password = generated
	} else if msg := validateNewPassword(password); msg != "" {
		fields["password"] = msg
	}
	if len(fields) > 0 {
		return CreateAdminResult{}, apperr.Validation(fields)
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return CreateAdminResult{}, fmt.Errorf("hash password: %w", err)
	}
	codes, hashes, err := s.newRecoveryCodes()
	if err != nil {
		return CreateAdminResult{}, err
	}
	admin, err := s.admins.CreateWithRecoveryCodes(ctx, model.Admin{Name: name, Email: email, PasswordHash: hash, MustChangePassword: generated != ""}, hashes)
	if err != nil {
		return CreateAdminResult{}, err
	}
	return CreateAdminResult{Admin: admin, GeneratedPassword: generated, RecoveryCodes: codes}, nil
}

// ResetPassword sets a temporary password, forces a change at next login and revokes all sessions.
func (s *AuthService) ResetPassword(ctx context.Context, email string) (string, error) {
	admin, err := s.admins.FindByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, repository.ErrNotFound) {
		return "", apperr.NotFound("Administrator")
	}
	if err != nil {
		return "", err
	}
	temp, err := token.URLSafe(generatedPasswordSize)
	if err != nil {
		return "", err
	}
	hash, err := s.hasher.Hash(temp)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	if err := s.admins.UpdatePassword(ctx, admin.ID, hash, true); err != nil {
		return "", err
	}
	if err := s.sessions.DeleteAllForAdmin(ctx, admin.ID); err != nil {
		return "", err
	}
	return temp, nil
}

// RegenerateRecoveryCodes replaces the admin's recovery codes and returns the new plain codes once.
func (s *AuthService) RegenerateRecoveryCodes(ctx context.Context, adminID uuid.UUID) ([]string, error) {
	codes, hashes, err := s.newRecoveryCodes()
	if err != nil {
		return nil, err
	}
	if err := s.admins.ReplaceRecoveryCodes(ctx, adminID, hashes); err != nil {
		return nil, err
	}
	return codes, nil
}

// UpdateProfile changes the admin name and email.
func (s *AuthService) UpdateProfile(ctx context.Context, adminID uuid.UUID, name, email string) (model.Admin, error) {
	fields := map[string]string{}
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" || utf8.RuneCountInString(name) > 120 {
		fields["name"] = "The name is required and may not exceed 120 characters."
	}
	if !validEmail(email) {
		fields["email"] = "A valid email is required."
	}
	if len(fields) > 0 {
		return model.Admin{}, apperr.Validation(fields)
	}
	return s.admins.UpdateProfile(ctx, adminID, name, email)
}

func (s *AuthService) checkCredentials(ctx context.Context, email, password string) (model.Admin, bool, error) {
	admin, err := s.admins.FindByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, repository.ErrNotFound) {
		_, _ = s.hasher.Verify(password, s.dummyHash)
		return model.Admin{}, false, nil
	}
	if err != nil {
		return model.Admin{}, false, err
	}
	ok, err := s.hasher.Verify(password, admin.PasswordHash)
	if err != nil {
		return model.Admin{}, false, fmt.Errorf("verify password: %w", err)
	}
	return admin, ok, nil
}

func (s *AuthService) ensureNotLocked(ctx context.Context, key string) error {
	n, retry, err := s.counter.Count(ctx, key)
	if err != nil {
		return err
	}
	if n >= MaxLoginFailures {
		return lockedError(retry)
	}
	return nil
}

func (s *AuthService) recordFailure(ctx context.Context, key string) error {
	n, _, err := s.counter.Hit(ctx, key, LoginFailureWindow)
	if err != nil {
		return err
	}
	if n >= MaxLoginFailures {
		if err := s.counter.Extend(ctx, key, LoginFailureWindow); err != nil {
			return err
		}
	}
	return apperr.ErrInvalidCredentials
}

func (s *AuthService) rotateSession(ctx context.Context, adminID uuid.UUID, client Client, previous string) (string, error) {
	if previous != "" {
		if err := s.sessions.Delete(ctx, previous); err != nil {
			return "", err
		}
	}
	return s.sessions.Create(ctx, adminID, client.IP, client.UserAgent)
}

func (s *AuthService) setPasswordAndRelogin(ctx context.Context, adminID uuid.UUID, next string, client Client) (string, error) {
	hash, err := s.hasher.Hash(next)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	if err := s.admins.UpdatePassword(ctx, adminID, hash, false); err != nil {
		return "", err
	}
	if err := s.sessions.DeleteAllForAdmin(ctx, adminID); err != nil {
		return "", err
	}
	return s.sessions.Create(ctx, adminID, client.IP, client.UserAgent)
}

func (s *AuthService) consumeRecoveryCode(ctx context.Context, adminID uuid.UUID, code string) (bool, error) {
	normalized := token.NormalizeRecoveryCode(code)
	if normalized == "" {
		return false, nil
	}
	codes, err := s.admins.UnusedRecoveryCodes(ctx, adminID)
	if err != nil {
		return false, err
	}
	for _, c := range codes {
		ok, err := s.hasher.Verify(normalized, c.CodeHash)
		if err != nil {
			return false, fmt.Errorf("verify recovery code: %w", err)
		}
		if ok {
			return s.admins.MarkRecoveryCodeUsed(ctx, c.ID, s.now())
		}
	}
	return false, nil
}

func (s *AuthService) newRecoveryCodes() ([]string, []string, error) {
	codes := make([]string, RecoveryCodeCount)
	hashes := make([]string, RecoveryCodeCount)
	for i := range codes {
		code, err := token.RecoveryCode()
		if err != nil {
			return nil, nil, err
		}
		hash, err := s.hasher.Hash(token.NormalizeRecoveryCode(code))
		if err != nil {
			return nil, nil, fmt.Errorf("hash recovery code: %w", err)
		}
		codes[i], hashes[i] = code, hash
	}
	return codes, hashes, nil
}

func lockedError(retry time.Duration) error {
	minutes := int(retry.Round(time.Minute) / time.Minute)
	if minutes < 1 {
		minutes = 1
	}
	return apperr.New(http.StatusTooManyRequests, apperr.ErrTooManyAttempts.Code,
		fmt.Sprintf("Too many failed attempts. Try again in %d minute(s).", minutes))
}

func validateNewPassword(p string) string {
	n := utf8.RuneCountInString(p)
	if n < MinPasswordLength {
		return fmt.Sprintf("The password must be at least %d characters.", MinPasswordLength)
	}
	if n > MaxPasswordLength {
		return fmt.Sprintf("The password may not exceed %d characters.", MaxPasswordLength)
	}
	return ""
}

func validEmail(email string) bool {
	at := strings.LastIndexByte(email, '@')
	return at > 0 && at < len(email)-3 && strings.Contains(email[at:], ".") && !strings.ContainsAny(email, " \t\r\n<>") && len(email) <= 254
}
