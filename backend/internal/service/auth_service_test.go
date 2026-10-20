package service

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/password"
)

type fakeAdmins struct {
	mu     sync.Mutex
	admins map[uuid.UUID]model.Admin
	codes  map[uuid.UUID]model.RecoveryCode
}

func newFakeAdmins() *fakeAdmins {
	return &fakeAdmins{admins: map[uuid.UUID]model.Admin{}, codes: map[uuid.UUID]model.RecoveryCode{}}
}

func (f *fakeAdmins) Count(context.Context) (int, error) { return len(f.admins), nil }

func (f *fakeAdmins) FindByEmail(_ context.Context, email string) (model.Admin, error) {
	for _, a := range f.admins {
		if strings.EqualFold(a.Email, email) {
			return a, nil
		}
	}
	return model.Admin{}, repository.ErrNotFound
}

func (f *fakeAdmins) FindByID(_ context.Context, id uuid.UUID) (model.Admin, error) {
	a, ok := f.admins[id]
	if !ok {
		return model.Admin{}, repository.ErrNotFound
	}
	return a, nil
}

func (f *fakeAdmins) CreateWithRecoveryCodes(_ context.Context, a model.Admin, hashes []string) (model.Admin, error) {
	a.ID = uuid.New()
	f.admins[a.ID] = a
	for _, h := range hashes {
		id := uuid.New()
		f.codes[id] = model.RecoveryCode{ID: id, AdminID: a.ID, CodeHash: h}
	}
	return a, nil
}

func (f *fakeAdmins) UpdatePassword(_ context.Context, id uuid.UUID, hash string, mustChange bool) error {
	a := f.admins[id]
	a.PasswordHash, a.MustChangePassword = hash, mustChange
	f.admins[id] = a
	return nil
}

func (f *fakeAdmins) UpdateProfile(_ context.Context, id uuid.UUID, name, email string) (model.Admin, error) {
	a := f.admins[id]
	a.Name, a.Email = name, email
	f.admins[id] = a
	return a, nil
}

func (f *fakeAdmins) UnusedRecoveryCodes(_ context.Context, adminID uuid.UUID) ([]model.RecoveryCode, error) {
	var out []model.RecoveryCode
	for _, c := range f.codes {
		if c.AdminID == adminID && c.UsedAt == nil {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeAdmins) MarkRecoveryCodeUsed(_ context.Context, id uuid.UUID, at time.Time) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := f.codes[id]
	if c.UsedAt != nil {
		return false, nil
	}
	c.UsedAt = &at
	f.codes[id] = c
	return true, nil
}

func (f *fakeAdmins) ReplaceRecoveryCodes(_ context.Context, adminID uuid.UUID, hashes []string) error {
	for id, c := range f.codes {
		if c.AdminID == adminID {
			delete(f.codes, id)
		}
	}
	for _, h := range hashes {
		id := uuid.New()
		f.codes[id] = model.RecoveryCode{ID: id, AdminID: adminID, CodeHash: h}
	}
	return nil
}

type fakeSessions struct {
	sessions map[string]model.Session
	n        int
}

func newFakeSessions() *fakeSessions { return &fakeSessions{sessions: map[string]model.Session{}} }

func (f *fakeSessions) Create(_ context.Context, adminID uuid.UUID, ip, ua string) (string, error) {
	f.n++
	id := "sess-" + strings.Repeat("x", f.n)
	f.sessions[id] = model.Session{AdminID: adminID, IP: ip, UserAgent: ua}
	return id, nil
}

func (f *fakeSessions) Get(_ context.Context, id string) (model.Session, error) {
	s, ok := f.sessions[id]
	if !ok {
		return model.Session{}, repository.ErrNotFound
	}
	return s, nil
}

func (f *fakeSessions) Delete(_ context.Context, id string) error {
	delete(f.sessions, id)
	return nil
}

func (f *fakeSessions) DeleteAllForAdmin(_ context.Context, adminID uuid.UUID) error {
	for id, s := range f.sessions {
		if s.AdminID == adminID {
			delete(f.sessions, id)
		}
	}
	return nil
}

type fakeCounter struct {
	counts map[string]int64
}

func newFakeCounter() *fakeCounter { return &fakeCounter{counts: map[string]int64{}} }

func (f *fakeCounter) Hit(_ context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	f.counts[key]++
	return f.counts[key], window, nil
}

func (f *fakeCounter) Count(_ context.Context, key string) (int64, time.Duration, error) {
	return f.counts[key], 15 * time.Minute, nil
}

func (f *fakeCounter) Extend(context.Context, string, time.Duration) error { return nil }

func (f *fakeCounter) Reset(_ context.Context, key string) error {
	delete(f.counts, key)
	return nil
}

type authFixture struct {
	svc      *AuthService
	admins   *fakeAdmins
	sessions *fakeSessions
	counter  *fakeCounter
}

const testPassword = "Correct-Horse-42"

var client = Client{IP: "203.0.113.7", UserAgent: "test"}

func newAuthFixture(t *testing.T) authFixture {
	t.Helper()
	hasher := password.NewHasher(password.Params{Memory: 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32})
	f := authFixture{admins: newFakeAdmins(), sessions: newFakeSessions(), counter: newFakeCounter()}
	svc, err := NewAuthService(f.admins, f.sessions, f.counter, hasher, time.Now)
	require.NoError(t, err)
	f.svc = svc
	return f
}

func (f authFixture) createAdmin(t *testing.T) CreateAdminResult {
	t.Helper()
	res, err := f.svc.CreateAdmin(context.Background(), "Ambon", "admin@example.com", testPassword)
	require.NoError(t, err)
	return res
}

func requireAppErr(t *testing.T, err error, code string) apperr.Error {
	t.Helper()
	e, ok := apperr.As(err)
	require.True(t, ok, "expected apperr, got %v", err)
	assert.Equal(t, code, e.Code)
	return *e
}

func TestCreateAdminIssuesTenRecoveryCodes(t *testing.T) {
	f := newAuthFixture(t)
	res := f.createAdmin(t)
	assert.Len(t, res.RecoveryCodes, RecoveryCodeCount)
	assert.Empty(t, res.GeneratedPassword)
	assert.False(t, res.Admin.MustChangePassword)
	for _, c := range f.admins.codes {
		assert.True(t, strings.HasPrefix(c.CodeHash, "$argon2id$"), "codes must be stored hashed")
	}
}

func TestCreateAdminGeneratesPasswordAndForcesChangeWhenNoneGiven(t *testing.T) {
	f := newAuthFixture(t)
	res, err := f.svc.CreateAdmin(context.Background(), "Ambon", "admin@example.com", "")
	require.NoError(t, err)
	assert.NotEmpty(t, res.GeneratedPassword)
	assert.True(t, res.Admin.MustChangePassword)
}

func TestCreateAdminRejectsSecondAdmin(t *testing.T) {
	f := newAuthFixture(t)
	f.createAdmin(t)
	_, err := f.svc.CreateAdmin(context.Background(), "Other", "other@example.com", testPassword)
	requireAppErr(t, err, "admin_exists")
}

func TestCreateAdminValidatesInput(t *testing.T) {
	f := newAuthFixture(t)
	_, err := f.svc.CreateAdmin(context.Background(), "", "not-an-email", "short")
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "name")
	assert.Contains(t, e.Fields, "email")
	assert.Contains(t, e.Fields, "password")
}

func TestLoginSucceedsWithCaseInsensitiveEmail(t *testing.T) {
	f := newAuthFixture(t)
	f.createAdmin(t)
	res, err := f.svc.Login(context.Background(), " ADMIN@example.com ", testPassword, client, "")
	require.NoError(t, err)
	assert.NotEmpty(t, res.SessionID)
	assert.Contains(t, f.sessions.sessions, res.SessionID)
}

func TestLoginUsesGenericErrorForUnknownEmailAndWrongPassword(t *testing.T) {
	f := newAuthFixture(t)
	f.createAdmin(t)
	_, errWrongPass := f.svc.Login(context.Background(), "admin@example.com", "nope", client, "")
	_, errUnknown := f.svc.Login(context.Background(), "ghost@example.com", testPassword, client, "")
	a := requireAppErr(t, errWrongPass, "invalid_credentials")
	b := requireAppErr(t, errUnknown, "invalid_credentials")
	assert.Equal(t, a.Message, b.Message)
}

func TestLoginBlocksSixthAttemptAfterFiveFailures(t *testing.T) {
	f := newAuthFixture(t)
	f.createAdmin(t)
	ctx := context.Background()
	for i := 0; i < MaxLoginFailures; i++ {
		_, err := f.svc.Login(ctx, "admin@example.com", "wrong", client, "")
		requireAppErr(t, err, "invalid_credentials")
	}
	// The correct password is refused while the IP is locked out.
	_, err := f.svc.Login(ctx, "admin@example.com", testPassword, client, "")
	e := requireAppErr(t, err, "too_many_attempts")
	assert.Equal(t, http.StatusTooManyRequests, e.Status)

	_, err = f.svc.Login(ctx, "admin@example.com", testPassword, Client{IP: "198.51.100.1"}, "")
	assert.NoError(t, err, "lockout is per IP")
}

func TestSuccessfulLoginResetsFailureCounter(t *testing.T) {
	f := newAuthFixture(t)
	f.createAdmin(t)
	ctx := context.Background()
	for i := 0; i < MaxLoginFailures-1; i++ {
		_, _ = f.svc.Login(ctx, "admin@example.com", "wrong", client, "")
	}
	_, err := f.svc.Login(ctx, "admin@example.com", testPassword, client, "")
	require.NoError(t, err)
	assert.Zero(t, f.counter.counts["login:"+client.IP])
}

func TestLoginRotatesPreviousSession(t *testing.T) {
	f := newAuthFixture(t)
	f.createAdmin(t)
	ctx := context.Background()
	first, err := f.svc.Login(ctx, "admin@example.com", testPassword, client, "")
	require.NoError(t, err)
	second, err := f.svc.Login(ctx, "admin@example.com", testPassword, client, first.SessionID)
	require.NoError(t, err)
	assert.NotEqual(t, first.SessionID, second.SessionID)
	assert.NotContains(t, f.sessions.sessions, first.SessionID)
}

func TestAuthenticate(t *testing.T) {
	f := newAuthFixture(t)
	f.createAdmin(t)
	ctx := context.Background()
	res, err := f.svc.Login(ctx, "admin@example.com", testPassword, client, "")
	require.NoError(t, err)

	admin, err := f.svc.Authenticate(ctx, res.SessionID)
	require.NoError(t, err)
	assert.Equal(t, res.Admin.ID, admin.ID)

	_, err = f.svc.Authenticate(ctx, "")
	requireAppErr(t, err, "unauthenticated")
	_, err = f.svc.Authenticate(ctx, "missing")
	requireAppErr(t, err, "unauthenticated")
}

func TestLogoutInvalidatesSession(t *testing.T) {
	f := newAuthFixture(t)
	f.createAdmin(t)
	ctx := context.Background()
	res, err := f.svc.Login(ctx, "admin@example.com", testPassword, client, "")
	require.NoError(t, err)
	require.NoError(t, f.svc.Logout(ctx, res.SessionID))
	_, err = f.svc.Authenticate(ctx, res.SessionID)
	requireAppErr(t, err, "unauthenticated")
}

func TestChangePasswordRevokesAllSessionsAndIssuesNewOne(t *testing.T) {
	f := newAuthFixture(t)
	created := f.createAdmin(t)
	ctx := context.Background()
	a, _ := f.svc.Login(ctx, "admin@example.com", testPassword, client, "")
	b, _ := f.svc.Login(ctx, "admin@example.com", testPassword, client, "")

	sid, err := f.svc.ChangePassword(ctx, created.Admin.ID, testPassword, "Brand-New-Password-7", client)
	require.NoError(t, err)
	assert.NotContains(t, f.sessions.sessions, a.SessionID)
	assert.NotContains(t, f.sessions.sessions, b.SessionID)
	assert.Contains(t, f.sessions.sessions, sid)

	_, err = f.svc.Login(ctx, "admin@example.com", "Brand-New-Password-7", client, "")
	assert.NoError(t, err)
}

func TestChangePasswordValidation(t *testing.T) {
	f := newAuthFixture(t)
	created := f.createAdmin(t)
	ctx := context.Background()
	cases := []struct {
		name    string
		current string
		next    string
		field   string
	}{
		{"wrong current password", "wrong", "Another-Password-1", "current_password"},
		{"too short", testPassword, "short", "new_password"},
		{"same as current", testPassword, testPassword, "new_password"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := f.svc.ChangePassword(ctx, created.Admin.ID, tc.current, tc.next, client)
			e := requireAppErr(t, err, "validation_failed")
			assert.Contains(t, e.Fields, tc.field)
		})
	}
}

func TestRecoveryCodeRestoresAccessOnlyOnce(t *testing.T) {
	f := newAuthFixture(t)
	created := f.createAdmin(t)
	ctx := context.Background()
	code := strings.ToLower(created.RecoveryCodes[3])

	res, err := f.svc.Recover(ctx, "admin@example.com", code, "Recovered-Password-1", client)
	require.NoError(t, err)
	assert.NotEmpty(t, res.SessionID)

	_, err = f.svc.Recover(ctx, "admin@example.com", code, "Recovered-Password-2", client)
	requireAppErr(t, err, "invalid_credentials")

	_, err = f.svc.Login(ctx, "admin@example.com", "Recovered-Password-1", client, "")
	assert.NoError(t, err)
}

func TestRecoveryAttemptsAreRateLimited(t *testing.T) {
	f := newAuthFixture(t)
	created := f.createAdmin(t)
	ctx := context.Background()
	for i := 0; i < MaxLoginFailures; i++ {
		_, err := f.svc.Recover(ctx, "admin@example.com", "AAAAA-AAAAA", "Recovered-Password-1", client)
		requireAppErr(t, err, "invalid_credentials")
	}
	_, err := f.svc.Recover(ctx, "admin@example.com", created.RecoveryCodes[0], "Recovered-Password-1", client)
	requireAppErr(t, err, "too_many_attempts")
}

func TestResetPasswordForcesChangeAndRevokesSessions(t *testing.T) {
	f := newAuthFixture(t)
	created := f.createAdmin(t)
	ctx := context.Background()
	login, _ := f.svc.Login(ctx, "admin@example.com", testPassword, client, "")

	temp, err := f.svc.ResetPassword(ctx, "admin@example.com")
	require.NoError(t, err)
	assert.NotContains(t, f.sessions.sessions, login.SessionID)
	assert.True(t, f.admins.admins[created.Admin.ID].MustChangePassword)

	res, err := f.svc.Login(ctx, "admin@example.com", temp, client, "")
	require.NoError(t, err)
	assert.True(t, res.Admin.MustChangePassword)

	_, err = f.svc.ResetPassword(ctx, "ghost@example.com")
	requireAppErr(t, err, "not_found")
}

func TestRegenerateRecoveryCodesInvalidatesOldSet(t *testing.T) {
	f := newAuthFixture(t)
	created := f.createAdmin(t)
	ctx := context.Background()
	fresh, err := f.svc.RegenerateRecoveryCodes(ctx, created.Admin.ID)
	require.NoError(t, err)
	assert.Len(t, fresh, RecoveryCodeCount)

	_, err = f.svc.Recover(ctx, "admin@example.com", created.RecoveryCodes[0], "Recovered-Password-1", client)
	requireAppErr(t, err, "invalid_credentials")
	_, err = f.svc.Recover(ctx, "admin@example.com", fresh[0], "Recovered-Password-1", client)
	assert.NoError(t, err)
}

func TestUpdateProfile(t *testing.T) {
	f := newAuthFixture(t)
	created := f.createAdmin(t)
	ctx := context.Background()

	updated, err := f.svc.UpdateProfile(ctx, created.Admin.ID, " Captain Ambon ", "captain@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Captain Ambon", updated.Name)
	assert.Equal(t, "captain@example.com", updated.Email)

	_, err = f.svc.UpdateProfile(ctx, created.Admin.ID, "", "invalid")
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields, "name")
	assert.Contains(t, e.Fields, "email")
}

func TestPasswordLengthLimits(t *testing.T) {
	f := newAuthFixture(t)
	created := f.createAdmin(t)
	_, err := f.svc.ChangePassword(context.Background(), created.Admin.ID, testPassword, strings.Repeat("x", MaxPasswordLength+1), client)
	e := requireAppErr(t, err, "validation_failed")
	assert.Contains(t, e.Fields["new_password"], "may not exceed")
}
