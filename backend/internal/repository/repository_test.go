package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func TestMigrationsRollBackAndReapplyCleanly(t *testing.T) {
	for range 4 {
		require.NoError(t, postgres.Migrate(testDSN, "down"))
	}
	var n int
	require.NoError(t, testDB.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name <> 'schema_migrations'`).Scan(&n))
	assert.Zero(t, n)
	require.NoError(t, postgres.Migrate(testDSN, "up"))
	require.NoError(t, testDB.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name <> 'schema_migrations'`).Scan(&n))
	assert.Equal(t, 46, n)
}

func TestPrimaryKeysAreUUIDv7(t *testing.T) {
	var id uuid.UUID
	require.NoError(t, testDB.QueryRow(context.Background(), `SELECT uuid_generate_v7()`).Scan(&id))
	assert.Equal(t, uuid.Version(7), id.Version())
	assert.Equal(t, uuid.RFC4122, id.Variant())

	var a, b uuid.UUID
	require.NoError(t, testDB.QueryRow(context.Background(), `SELECT uuid_generate_v7()`).Scan(&a))
	time.Sleep(2 * time.Millisecond)
	require.NoError(t, testDB.QueryRow(context.Background(), `SELECT uuid_generate_v7()`).Scan(&b))
	assert.Less(t, a.String(), b.String(), "UUID v7 must sort by creation time")
}

func TestAdminRepositoryCreatesAdminWithRecoveryCodes(t *testing.T) {
	truncate(t, "admins")
	ctx := context.Background()
	repo := repository.NewAdminRepository(testDB)

	created, err := repo.CreateWithRecoveryCodes(ctx, model.Admin{Name: "Ambon", Email: "Admin@Example.com", PasswordHash: "h"}, []string{"c1", "c2", "c3"})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, created.ID)

	found, err := repo.FindByEmail(ctx, "admin@example.COM")
	require.NoError(t, err)
	assert.Equal(t, created.ID, found.ID)

	codes, err := repo.UnusedRecoveryCodes(ctx, created.ID)
	require.NoError(t, err)
	assert.Len(t, codes, 3)

	_, err = repo.CreateWithRecoveryCodes(ctx, model.Admin{Name: "Dup", Email: "admin@example.com", PasswordHash: "h"}, nil)
	assert.True(t, postgres.IsUniqueViolation(err, "admins_email_key"), "email must be unique case-insensitively")

	_, err = repo.FindByEmail(ctx, "ghost@example.com")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestRecoveryCodeCanBeMarkedUsedOnlyOnce(t *testing.T) {
	truncate(t, "admins")
	ctx := context.Background()
	repo := repository.NewAdminRepository(testDB)
	admin, err := repo.CreateWithRecoveryCodes(ctx, model.Admin{Name: "A", Email: "a@example.com", PasswordHash: "h"}, []string{"c1"})
	require.NoError(t, err)
	codes, err := repo.UnusedRecoveryCodes(ctx, admin.ID)
	require.NoError(t, err)

	first, err := repo.MarkRecoveryCodeUsed(ctx, codes[0].ID, time.Now())
	require.NoError(t, err)
	second, err := repo.MarkRecoveryCodeUsed(ctx, codes[0].ID, time.Now())
	require.NoError(t, err)
	assert.True(t, first)
	assert.False(t, second)

	remaining, err := repo.UnusedRecoveryCodes(ctx, admin.ID)
	require.NoError(t, err)
	assert.Empty(t, remaining)

	require.NoError(t, repo.ReplaceRecoveryCodes(ctx, admin.ID, []string{"n1", "n2"}))
	remaining, err = repo.UnusedRecoveryCodes(ctx, admin.ID)
	require.NoError(t, err)
	assert.Len(t, remaining, 2)
}

func TestOnlyOneActiveContractPerCrewIsAllowed(t *testing.T) {
	truncate(t, "crews", "vessels", "companies", "ranks", "crew_statuses", "currencies")
	ctx := context.Background()
	var statusID, rankID, companyID, vesselID, crewID uuid.UUID
	require.NoError(t, testDB.QueryRow(ctx, `INSERT INTO crew_statuses (code, name, is_system) VALUES ('available', 'Available', true) RETURNING id`).Scan(&statusID))
	require.NoError(t, testDB.QueryRow(ctx, `INSERT INTO ranks (name, short_code, department) VALUES ('Master', 'MST', 'deck') RETURNING id`).Scan(&rankID))
	require.NoError(t, testDB.QueryRow(ctx, `INSERT INTO companies (code, name, type) VALUES ('ACME', 'Acme', 'both') RETURNING id`).Scan(&companyID))
	require.NoError(t, testDB.QueryRow(ctx, `INSERT INTO vessels (name, imo_number) VALUES ('MV Test', '9074729') RETURNING id`).Scan(&vesselID))
	require.NoError(t, testDB.QueryRow(ctx, `INSERT INTO crews (public_id, full_name, crew_status_id) VALUES ('ABCDEFGH2', 'John Doe', $1) RETURNING id`, statusID).Scan(&crewID))
	_, err := testDB.Exec(ctx, `INSERT INTO currencies (code, decimals) VALUES ('USD', 2)`)
	require.NoError(t, err)

	insert := func(publicID, status string) error {
		_, err := testDB.Exec(ctx, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id,
			sign_on_date, duration_months, planned_sign_off_date, currency_code, basic_salary, status)
			VALUES ($1, $2, $3, $4, $5, $5, '2026-01-01', 6, '2026-07-01', 'USD', 5000, $6)`,
			publicID, crewID, vesselID, rankID, companyID, status)
		return err
	}
	require.NoError(t, insert("CONTRACT2", "active"))
	require.NoError(t, insert("CONTRACT3", "completed"))
	err = insert("CONTRACT4", "active")
	assert.True(t, postgres.IsUniqueViolation(err, "contracts_one_active_per_crew"), "got %v", err)
}

func TestSessionRepositoryEnforcesIdleAndAbsoluteLifetimes(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testRedis.FlushDB(ctx).Err())
	now := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	repo := repository.NewSessionRepository(testRedis, 8*time.Hour, 7*24*time.Hour, clock)
	adminID := uuid.New()

	sid, err := repo.Create(ctx, adminID, "203.0.113.7", "agent")
	require.NoError(t, err)
	assert.Len(t, sid, 43, "32 random bytes in base64url")

	keys, err := testRedis.Keys(ctx, "sess:*").Result()
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.NotContains(t, keys[0], sid, "raw session ID must not be stored")
	ttl, err := testRedis.TTL(ctx, keys[0]).Result()
	require.NoError(t, err)
	assert.InDelta(t, (8 * time.Hour).Seconds(), ttl.Seconds(), 5)

	s, err := repo.Get(ctx, sid)
	require.NoError(t, err)
	assert.Equal(t, adminID, s.AdminID)

	now = now.Add(7*24*time.Hour + time.Minute)
	_, err = repo.Get(ctx, sid)
	assert.ErrorIs(t, err, repository.ErrNotFound, "absolute lifetime exceeded")
}

func TestSessionRepositoryDeletesAllSessionsOfAdmin(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testRedis.FlushDB(ctx).Err())
	repo := repository.NewSessionRepository(testRedis, time.Hour, 24*time.Hour, time.Now)
	adminID := uuid.New()
	a, err := repo.Create(ctx, adminID, "ip", "ua")
	require.NoError(t, err)
	b, err := repo.Create(ctx, adminID, "ip", "ua")
	require.NoError(t, err)
	other, err := repo.Create(ctx, uuid.New(), "ip", "ua")
	require.NoError(t, err)

	require.NoError(t, repo.DeleteAllForAdmin(ctx, adminID))
	_, err = repo.Get(ctx, a)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	_, err = repo.Get(ctx, b)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	_, err = repo.Get(ctx, other)
	assert.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, other))
	_, err = repo.Get(ctx, other)
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestLimiterRepositoryCountsWithinWindow(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, testRedis.FlushDB(ctx).Err())
	repo := repository.NewLimiterRepository(testRedis)
	for i := int64(1); i <= 3; i++ {
		n, ttl, err := repo.Hit(ctx, "login:1.2.3.4", 15*time.Minute)
		require.NoError(t, err)
		assert.Equal(t, i, n)
		assert.InDelta(t, (15 * time.Minute).Seconds(), ttl.Seconds(), 2)
	}
	n, _, err := repo.Count(ctx, "login:1.2.3.4")
	require.NoError(t, err)
	assert.Equal(t, int64(3), n)

	require.NoError(t, repo.Reset(ctx, "login:1.2.3.4"))
	n, _, err = repo.Count(ctx, "login:1.2.3.4")
	require.NoError(t, err)
	assert.Zero(t, n)
}

func TestSettingRepositoryUpsertAndInsertMissing(t *testing.T) {
	truncate(t, "settings")
	ctx := context.Background()
	repo := repository.NewSettingRepository(testDB)
	require.NoError(t, repo.InsertMissing(ctx, []model.Setting{{Group: "general", Key: "site_name", Value: []byte(`"Default"`)}}))
	require.NoError(t, repo.Upsert(ctx, []model.Setting{{Group: "general", Key: "site_name", Value: []byte(`"Custom"`)}}))
	require.NoError(t, repo.InsertMissing(ctx, []model.Setting{{Group: "general", Key: "site_name", Value: []byte(`"Default"`)}}))

	rows, err := repo.ListGroup(ctx, "general")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.JSONEq(t, `"Custom"`, string(rows[0].Value))
}
