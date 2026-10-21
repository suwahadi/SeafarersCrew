package app_test

import (
	"context"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/suwahadi/seafarerscrew/backend/database/seeders"
	"github.com/suwahadi/seafarerscrew/backend/internal/app"
	"github.com/suwahadi/seafarerscrew/backend/internal/config"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

const seedBucket = "seafarers"

func startSeedApp(t *testing.T) *app.App {
	t.Helper()
	if os.Getenv("SKIP_CONTAINER_TESTS") == "1" {
		t.Skip("SKIP_CONTAINER_TESTS=1")
	}
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("seafarers_seed"), tcpostgres.WithUsername("test"), tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)))
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(pg) })
	rc, err := tcredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(rc) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	require.NoError(t, postgres.Migrate(dsn, "up"))
	redisURL, err := rc.ConnectionString(ctx)
	require.NoError(t, err)

	backend := s3mem.New()
	require.NoError(t, backend.CreateBucket(seedBucket))
	s3srv := httptest.NewServer(gofakes3.New(backend).Server())
	t.Cleanup(s3srv.Close)

	cfg := config.Config{
		Env: config.EnvLocal, AppURL: "http://localhost:3000", AdminURL: "http://localhost:5173", APIURL: "http://localhost:8080",
		EncryptionKey: []byte(strings.Repeat("k", 32)), DatabaseURL: dsn, RedisURL: redisURL, GotenbergURL: "http://localhost:3001",
		RevalidateSecret: base64.StdEncoding.EncodeToString([]byte(strings.Repeat("s", 32))), HTTPAddr: ":0",
		S3Bootstrap: config.S3Bootstrap{Endpoint: s3srv.URL, Bucket: seedBucket, Region: "us-east-1", AccessKeyID: "key", SecretAccessKey: "secret", UsePathStyle: true},
	}
	a, err := app.New(ctx, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	require.NoError(t, err)
	t.Cleanup(a.Close)
	return a
}

func scalar[T any](t *testing.T, a *app.App, sql string, args ...any) T {
	t.Helper()
	var v T
	require.NoError(t, a.DB.QueryRow(context.Background(), sql, args...).Scan(&v), sql)
	return v
}

func tableCounts(t *testing.T, a *app.App) map[string]int {
	t.Helper()
	tables := []string{"companies", "vessels", "crews", "travel_documents", "certificates", "crew_bank_accounts", "crew_next_of_kins", "experiences",
		"contracts", "contract_allowances", "relief_plans", "quotations", "quotation_lines", "hero_slides", "gallery_images", "testimonials",
		"client_logos", "stats_items", "news", "pages", "menu_items", "vacancies", "applicants", "applicant_stage_logs", "contact_messages", "files", "settings"}
	out := map[string]int{}
	for _, table := range tables {
		out[table] = scalar[int](t, a, "SELECT count(*) FROM "+table)
	}
	return out
}

func TestDemoSeedLoadsTheDataSetOfPRD163OnceAndOnlyOnce(t *testing.T) {
	a := startSeedApp(t)
	ctx := context.Background()

	reference, err := a.Seed(ctx, app.SeedOptions{})
	require.NoError(t, err)
	assert.Empty(t, reference)
	assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM crews`), "without --demo no demo data is loaded")
	assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM files`))
	assert.Equal(t, 5, scalar[int](t, a, `SELECT count(*) FROM pages WHERE NOT is_system`))

	steps, err := a.Seed(ctx, app.SeedOptions{Demo: true})
	require.NoError(t, err)
	require.NotEmpty(t, steps)
	for _, s := range steps {
		assert.False(t, s.Skipped, s.Name)
	}

	t.Run("master data and crew", func(t *testing.T) {
		assert.Equal(t, 5, scalar[int](t, a, `SELECT count(*) FROM companies`))
		assert.Equal(t, 10, scalar[int](t, a, `SELECT count(*) FROM vessels`))
		assert.Equal(t, 60, scalar[int](t, a, `SELECT count(*) FROM crews`))
		assert.Equal(t, 120, scalar[int](t, a, `SELECT count(*) FROM experiences`))
		imos := scalar[[]string](t, a, `SELECT array_agg(imo_number::text) FROM vessels`)
		for _, imo := range imos {
			assert.True(t, service.ValidIMO(imo), "IMO %s has a wrong check digit", imo)
		}
		assert.Equal(t, 60, scalar[int](t, a, `SELECT count(DISTINCT full_name) FROM crews`))
		assert.Positive(t, scalar[int](t, a, `SELECT count(*) FROM crews WHERE nationality <> 'Indonesian'`))
	})

	t.Run("documents and certificates spread over every expiry bucket", func(t *testing.T) {
		rows, err := a.DB.Query(ctx, `WITH d AS (
			SELECT expired_date e FROM travel_documents WHERE expired_date IS NOT NULL UNION ALL SELECT expired_date FROM certificates WHERE expired_date IS NOT NULL)
			SELECT CASE WHEN e < current_date THEN 'expired' WHEN e <= current_date + 30 THEN 'd30' WHEN e <= current_date + 60 THEN 'd60'
				WHEN e <= current_date + 90 THEN 'd90' ELSE 'valid' END, count(*) FROM d GROUP BY 1`)
		require.NoError(t, err)
		type bucket struct {
			name string
			n    int
		}
		buckets, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (bucket, error) {
			var b bucket
			return b, r.Scan(&b.name, &b.n)
		})
		require.NoError(t, err)
		byName := map[string]int{}
		total := 0
		for _, b := range buckets {
			byName[b.name] = b.n
			total += b.n
		}
		for _, name := range []string{"expired", "d30", "d60", "d90", "valid"} {
			assert.Positive(t, byName[name], "bucket %s is empty", name)
		}
		assert.Greater(t, byName["valid"]*2, total, "most documents are valid")
		assert.Positive(t, scalar[int](t, a, `SELECT count(*) FROM travel_documents WHERE document_type_id IN (SELECT id FROM document_types WHERE code = 'passport')`))
		assert.Positive(t, scalar[int](t, a, `SELECT count(*) FROM certificates WHERE certificate_type_id IN (SELECT id FROM certificate_types WHERE code = 'bst')`))
	})

	t.Run("contracts keep crew statuses consistent", func(t *testing.T) {
		assert.Equal(t, 15, scalar[int](t, a, `SELECT count(*) FROM contracts WHERE status = 'active'`))
		assert.Equal(t, 15, scalar[int](t, a, `SELECT count(DISTINCT crew_id) FROM contracts WHERE status = 'active'`))
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM contracts k JOIN crews c ON c.id = k.crew_id JOIN crew_statuses s ON s.id = c.crew_status_id
			WHERE k.status = 'active' AND s.code <> 'on_board'`), "crew with an active contract must be On Board")
		assert.Equal(t, 15, scalar[int](t, a, `SELECT count(*) FROM crews c JOIN crew_statuses s ON s.id = c.crew_status_id WHERE s.code = 'on_board'`))
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM crews c JOIN crew_statuses s ON s.id = c.crew_status_id
			WHERE s.code = 'on_board' AND NOT EXISTS (SELECT 1 FROM contracts k WHERE k.crew_id = c.id AND k.status = 'active')`), "no On Board crew without a contract")
		assert.Positive(t, scalar[int](t, a, `SELECT count(*) FROM contract_allowances`))
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM contracts k WHERE k.status = 'active' AND NOT EXISTS (SELECT 1 FROM contract_allowances x WHERE x.contract_id = k.id)`),
			"every active contract has allowances")
		assert.Equal(t, 4, scalar[int](t, a, `SELECT count(*) FROM experiences WHERE source = 'contract'`))
		assert.Equal(t, 5, scalar[int](t, a, `SELECT count(*) FROM relief_plans WHERE status IN ('planned', 'confirmed')`))
		assert.Equal(t, 5, scalar[int](t, a, `SELECT count(*) FROM relief_plans`))
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM relief_plans p JOIN contracts k ON k.id = p.contract_id WHERE k.status <> 'active'`))
	})

	t.Run("quotations total through the quotation rules", func(t *testing.T) {
		assert.Equal(t, 3, scalar[int](t, a, `SELECT count(*) FROM quotations`))
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM quotations q WHERE NOT EXISTS (SELECT 1 FROM quotation_lines l WHERE l.quotation_id = q.id)`))
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM quotations WHERE grand_total <> subtotal - discount_amount + tax_amount
			OR abs(tax_amount - (subtotal - discount_amount) * tax_percent / 100) > 0.5
			OR subtotal <> (SELECT coalesce(sum(amount), 0) FROM quotation_lines l WHERE l.quotation_id = quotations.id)`))
	})

	t.Run("website content", func(t *testing.T) {
		assert.Equal(t, 3, scalar[int](t, a, `SELECT count(*) FROM hero_slides`))
		assert.Equal(t, 12, scalar[int](t, a, `SELECT count(*) FROM gallery_images`))
		assert.Equal(t, 6, scalar[int](t, a, `SELECT count(*) FROM testimonials`))
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM testimonials WHERE rating NOT BETWEEN 4 AND 5`))
		assert.Equal(t, 8, scalar[int](t, a, `SELECT count(*) FROM client_logos`))
		assert.Equal(t, 4, scalar[int](t, a, `SELECT count(*) FROM stats_items`))
		assert.Equal(t, 6, scalar[int](t, a, `SELECT count(*) FROM news`))
		assert.Equal(t, 1, scalar[int](t, a, `SELECT count(*) FROM news WHERE published_at > now()`), "one article is scheduled")
		assert.GreaterOrEqual(t, scalar[int](t, a, `SELECT count(*) FROM news WHERE published_at < now() - interval '60 days'`), 2, "some articles are old")
	})

	t.Run("images go through the upload pipeline into public storage", func(t *testing.T) {
		assert.Equal(t, 30, scalar[int](t, a, `SELECT count(*) FROM files`))
		assert.Equal(t, 30, scalar[int](t, a, `SELECT count(*) FROM files WHERE is_public AND mime_type = 'image/webp' AND status = 'ready'`))
		key := scalar[string](t, a, `SELECT storage_key FROM files WHERE original_name = 'demo-hero-01.png'`)
		assert.True(t, strings.HasPrefix(key, "public/cms_image/"), key)
		assert.True(t, strings.HasSuffix(key, ".webp"), key)
		_, err := a.Storage.Stat(ctx, key)
		require.NoError(t, err)
		variantKeys := scalar[[]string](t, a, `SELECT array_agg(v->>'storage_key') FROM files f, jsonb_array_elements(f.variants) v WHERE f.original_name = 'demo-hero-01.png'`)
		assert.GreaterOrEqual(t, len(variantKeys), 2, "responsive variants are generated")
		for _, k := range variantKeys {
			_, err := a.Storage.Stat(ctx, k)
			require.NoError(t, err, k)
		}
		logoKey := scalar[string](t, a, `SELECT storage_key FROM files WHERE original_name = 'demo-logo-01.png'`)
		assert.True(t, strings.HasPrefix(logoKey, "public/logo/"), logoKey)
		_, err = a.Storage.Stat(ctx, logoKey)
		require.NoError(t, err)
	})

	t.Run("company pages and the about image", func(t *testing.T) {
		for _, slug := range []string{"about-us", "services", "terms-and-conditions", "privacy-policy", "faq"} {
			assert.Equal(t, 1, scalar[int](t, a, `SELECT count(*) FROM pages WHERE slug = $1 AND status = 'published'`, slug), slug)
		}
		assert.Equal(t, 1, scalar[int](t, a, `SELECT count(*) FROM pages p, jsonb_array_elements(p.blocks) b WHERE p.slug = 'about-us' AND b->>'type' = 'image_text'`))
	})

	t.Run("career data", func(t *testing.T) {
		assert.Equal(t, 6, scalar[int](t, a, `SELECT count(*) FROM vacancies`))
		assert.Equal(t, 4, scalar[int](t, a, `SELECT count(*) FROM vacancies WHERE status = 'open' AND closing_date >= current_date`))
		assert.Equal(t, 2, scalar[int](t, a, `SELECT count(*) FROM vacancies WHERE status = 'closed'`))
		assert.Equal(t, 15, scalar[int](t, a, `SELECT count(*) FROM applicants`))
		rows, err := a.DB.Query(ctx, `SELECT stage, count(*) FROM applicants GROUP BY stage`)
		require.NoError(t, err)
		type stage struct {
			name string
			n    int
		}
		stages, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (stage, error) {
			var s stage
			return s, r.Scan(&s.name, &s.n)
		})
		require.NoError(t, err)
		got := map[string]int{}
		for _, s := range stages {
			got[s.name] = s.n
		}
		assert.Equal(t, map[string]int{model.StageNew: 4, model.StageScreening: 3, model.StageInterview: 3, model.StageAccepted: 2, model.StageRejected: 3}, got)
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM applicants a WHERE a.stage <> (SELECT l.to_stage FROM applicant_stage_logs l WHERE l.applicant_id = a.id ORDER BY l.created_at DESC, l.id DESC LIMIT 1)`),
			"the last stage log matches the current stage")
		assert.Zero(t, scalar[int](t, a, `SELECT count(*) FROM applicants a WHERE NOT EXISTS (SELECT 1 FROM applicant_stage_logs l WHERE l.applicant_id = a.id AND l.from_stage IS NULL AND l.to_stage = 'new')`))
		assert.Equal(t, 5, scalar[int](t, a, `SELECT count(*) FROM contact_messages`))
		assert.Positive(t, scalar[int](t, a, `SELECT count(*) FROM contact_messages WHERE NOT is_read AND NOT is_archived`))
		assert.Positive(t, scalar[int](t, a, `SELECT count(*) FROM contact_messages WHERE is_read AND NOT is_archived`))
		assert.Positive(t, scalar[int](t, a, `SELECT count(*) FROM contact_messages WHERE is_archived`))
	})

	before := tableCounts(t, a)
	again, err := a.Seed(ctx, app.SeedOptions{Demo: true})
	require.NoError(t, err)
	for _, s := range again {
		assert.True(t, s.Skipped, "step %s ran twice", s.Name)
	}
	assert.Equal(t, before, tableCounts(t, a), "a second run adds nothing")

	t.Run("production refuses demo data unless forced", func(t *testing.T) {
		a.Config.Env = config.EnvProduction
		_, err := a.Seed(ctx, app.SeedOptions{Demo: true})
		require.ErrorIs(t, err, seeders.ErrDemoInProduction)
		_, err = a.Seed(ctx, app.SeedOptions{Demo: true, Force: true})
		require.NoError(t, err)
		_, err = a.Seed(ctx, app.SeedOptions{})
		require.NoError(t, err)
		assert.Equal(t, before, tableCounts(t, a))
	})
}
