package service

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/secret"
)

type memSettings struct {
	rows map[string]model.Setting
}

func (m *memSettings) ListGroup(_ context.Context, group string) ([]model.Setting, error) {
	var out []model.Setting
	for _, r := range m.rows {
		if r.Group == group {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *memSettings) Upsert(_ context.Context, rows []model.Setting) error {
	for _, r := range rows {
		m.rows[r.Group+"."+r.Key] = r
	}
	return nil
}

func (m *memSettings) InsertMissing(_ context.Context, rows []model.Setting) error {
	for _, r := range rows {
		if _, ok := m.rows[r.Group+"."+r.Key]; !ok {
			m.rows[r.Group+"."+r.Key] = r
		}
	}
	return nil
}

type memCache struct {
	data map[string][]byte
}

func newMemCache() *memCache { return &memCache{data: map[string][]byte{}} }

func (c *memCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	v, ok := c.data[key]
	return v, ok, nil
}

func (c *memCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	c.data[key] = value
	return nil
}

func (c *memCache) Delete(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(c.data, k)
	}
	return nil
}

func newSettingsFixture(t *testing.T) (*SettingsService, *memSettings, *[]string) {
	t.Helper()
	box, err := secret.NewBox(bytes.Repeat([]byte{1}, 32))
	require.NoError(t, err)
	store := &memSettings{rows: map[string]model.Setting{}}
	var changed []string
	svc := NewSettingsService(store, newMemCache(), box, func(_ context.Context, g string) { changed = append(changed, g) })
	return svc, store, &changed
}

func raw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func TestSettingGroupsHaveUniqueKeysAndValidDefaults(t *testing.T) {
	seen := map[string]bool{}
	for _, g := range SettingGroups {
		assert.False(t, seen[g.Name], "duplicate group %s", g.Name)
		seen[g.Name] = true
		keys := map[string]bool{}
		for _, f := range g.Fields {
			assert.False(t, keys[f.Key], "duplicate key %s.%s", g.Name, f.Key)
			keys[f.Key] = true
			_, err := f.Normalize(raw(f.Default))
			assert.NoError(t, err, "default of %s.%s must validate", g.Name, f.Key)
		}
	}
}

func TestDefaultsMatchPRD(t *testing.T) {
	svc, _, _ := newSettingsFixture(t)
	ctx := context.Background()
	require.NoError(t, svc.SeedDefaults(ctx, nil))

	general, err := svc.Values(ctx, "general")
	require.NoError(t, err)
	assert.Equal(t, "Asia/Jakarta", general["timezone"])
	assert.Equal(t, "DD-MMM-YYYY", general["date_format"])

	uploads, err := svc.Values(ctx, "uploads")
	require.NoError(t, err)
	assert.Equal(t, 10, uploads["max_file_size_mb"])
	assert.Equal(t, 80, uploads["webp_quality"])
	assert.Equal(t, 2000, uploads["max_image_dimension"])

	alerts, err := svc.Values(ctx, "alerts")
	require.NoError(t, err)
	assert.Equal(t, []int{30, 60, 90}, []int{alerts["expiry_threshold_1"].(int), alerts["expiry_threshold_2"].(int), alerts["expiry_threshold_3"].(int)})

	branding, err := svc.Values(ctx, "branding")
	require.NoError(t, err)
	assert.Equal(t, "#0B2545", branding["primary_color"])
}

func TestSecretsAreEncryptedAtRestAndMaskedInAdminView(t *testing.T) {
	svc, store, _ := newSettingsFixture(t)
	ctx := context.Background()
	_, err := svc.Update(ctx, "email", map[string]json.RawMessage{"brevo_api_key": raw("xkeysib-0123456789abcdef")})
	require.NoError(t, err)

	stored := store.rows["email.brevo_api_key"]
	assert.True(t, stored.IsSecret)
	assert.NotContains(t, string(stored.Value), "xkeysib")

	view, err := svc.AdminView(ctx, "email")
	require.NoError(t, err)
	assert.Equal(t, SecretView{Configured: true, Masked: "••••••••cdef"}, view["brevo_api_key"])

	values, err := svc.Values(ctx, "email")
	require.NoError(t, err)
	assert.Equal(t, "xkeysib-0123456789abcdef", values["brevo_api_key"])
}

func TestEmptySecretKeepsExistingValue(t *testing.T) {
	svc, _, _ := newSettingsFixture(t)
	ctx := context.Background()
	_, err := svc.Update(ctx, "security", map[string]json.RawMessage{"turnstile_secret_key": raw("0x4AAA-secret")})
	require.NoError(t, err)
	_, err = svc.Update(ctx, "security", map[string]json.RawMessage{"turnstile_secret_key": raw(""), "turnstile_site_key": raw("0x4AAA-site")})
	require.NoError(t, err)

	values, err := svc.Values(ctx, "security")
	require.NoError(t, err)
	assert.Equal(t, "0x4AAA-secret", values["turnstile_secret_key"])
	assert.Equal(t, "0x4AAA-site", values["turnstile_site_key"])
}

func TestUpdateRejectsInvalidValues(t *testing.T) {
	svc, _, changed := newSettingsFixture(t)
	ctx := context.Background()
	cases := []struct {
		group string
		key   string
		value any
	}{
		{"general", "timezone", "Mars/Olympus"},
		{"general", "date_format", "YYYY"},
		{"general", "unknown_key", "x"},
		{"branding", "primary_color", "navy"},
		{"company", "email", "not-an-email"},
		{"contact", "whatsapp_number", "08123"},
		{"contact", "linkedin_url", "javascript:alert(1)"},
		{"uploads", "webp_quality", 101},
		{"uploads", "max_file_size_mb", 2.5},
		{"backup", "run_time", "25:00"},
		{"email", "admin_notification_emails", []string{"ok@example.com", "bad"}},
		{"website", "home_sections", []string{"hero", "unknown"}},
		{"general", "maintenance_mode", "yes"},
	}
	for _, tc := range cases {
		t.Run(tc.group+"."+tc.key, func(t *testing.T) {
			_, err := svc.Update(ctx, tc.group, map[string]json.RawMessage{tc.key: raw(tc.value)})
			e := requireAppErr(t, err, "validation_failed")
			assert.Contains(t, e.Fields, tc.key)
		})
	}
	assert.Empty(t, *changed, "failed updates must not trigger revalidation")
}

func TestUpdateUnknownGroup(t *testing.T) {
	svc, _, _ := newSettingsFixture(t)
	_, err := svc.Update(context.Background(), "nope", nil)
	requireAppErr(t, err, "not_found")
}

func TestAlertThresholdsMustIncrease(t *testing.T) {
	svc, _, _ := newSettingsFixture(t)
	ctx := context.Background()
	_, err := svc.Update(ctx, "alerts", map[string]json.RawMessage{
		"expiry_threshold_1": raw(30), "expiry_threshold_2": raw(30), "expiry_threshold_3": raw(90),
	})
	requireAppErr(t, err, "validation_failed")

	_, err = svc.Update(ctx, "alerts", map[string]json.RawMessage{
		"expiry_threshold_1": raw(14), "expiry_threshold_2": raw(45), "expiry_threshold_3": raw(120),
	})
	assert.NoError(t, err)
}

func TestUpdateInvalidatesCacheAndNotifies(t *testing.T) {
	svc, _, changed := newSettingsFixture(t)
	ctx := context.Background()
	require.NoError(t, svc.SeedDefaults(ctx, nil))
	before, err := svc.Values(ctx, "general")
	require.NoError(t, err)
	assert.Equal(t, "Seafarers Crew", before["site_name"])

	_, err = svc.Update(ctx, "general", map[string]json.RawMessage{"site_name": raw("  Ocean Crew  ")})
	require.NoError(t, err)
	after, err := svc.Values(ctx, "general")
	require.NoError(t, err)
	assert.Equal(t, "Ocean Crew", after["site_name"])
	assert.Equal(t, []string{"general"}, *changed)
}

func TestSeedDefaultsDoesNotOverwriteExistingValues(t *testing.T) {
	svc, _, _ := newSettingsFixture(t)
	ctx := context.Background()
	_, err := svc.Update(ctx, "general", map[string]json.RawMessage{"site_name": raw("Ocean Crew")})
	require.NoError(t, err)
	require.NoError(t, svc.SeedDefaults(ctx, map[string]map[string]any{"storage": {"bucket": "seafarers"}}))

	general, _ := svc.Values(ctx, "general")
	assert.Equal(t, "Ocean Crew", general["site_name"])
	storage, _ := svc.Values(ctx, "storage")
	assert.Equal(t, "seafarers", storage["bucket"])
}
