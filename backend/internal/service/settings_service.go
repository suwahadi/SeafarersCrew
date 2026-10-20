package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/secret"
)

// SettingStore persists settings rows.
type SettingStore interface {
	ListGroup(ctx context.Context, group string) ([]model.Setting, error)
	Upsert(ctx context.Context, settings []model.Setting) error
	InsertMissing(ctx context.Context, settings []model.Setting) error
}

// Cache stores small JSON payloads with a TTL.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// Cipher encrypts secret setting values.
type Cipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// SettingsChanged is called after a group is saved, e.g. to revalidate the public site.
type SettingsChanged func(ctx context.Context, group string)

// SettingsService reads and writes the Settings panel.
type SettingsService struct {
	store     SettingStore
	cache     Cache
	cipher    Cipher
	onChanged SettingsChanged
}

const settingsCacheTTL = time.Hour

// NewSettingsService builds a SettingsService.
func NewSettingsService(store SettingStore, cache Cache, cipher Cipher, onChanged SettingsChanged) *SettingsService {
	return &SettingsService{store: store, cache: cache, cipher: cipher, onChanged: onChanged}
}

// SecretView is how a secret value is exposed to the admin UI.
type SecretView struct {
	Configured bool   `json:"configured"`
	Masked     string `json:"masked"`
}

// AdminView returns a group with secrets masked, for the admin API.
func (s *SettingsService) AdminView(ctx context.Context, group string) (map[string]any, error) {
	g, values, err := s.load(ctx, group)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(g.Fields))
	for _, f := range g.Fields {
		v := values[f.Key]
		if f.Secret {
			plain, _ := v.(string)
			out[f.Key] = SecretView{Configured: plain != "", Masked: secret.Mask(plain)}
			continue
		}
		out[f.Key] = v
	}
	return out, nil
}

// Values returns a group with decrypted secrets, for internal use only.
func (s *SettingsService) Values(ctx context.Context, group string) (map[string]any, error) {
	_, values, err := s.load(ctx, group)
	return values, err
}

// Update validates input against the registry and stores it; secrets left empty keep their current value.
func (s *SettingsService) Update(ctx context.Context, group string, input map[string]json.RawMessage) (map[string]any, error) {
	g, ok := FindSettingGroup(group)
	if !ok {
		return nil, apperr.NotFound("Settings group")
	}
	fields := map[string]string{}
	rows := make([]model.Setting, 0, len(input))
	for key, raw := range input {
		f, ok := g.Field(key)
		if !ok {
			fields[key] = "Unknown setting."
			continue
		}
		row, skip, err := s.prepareRow(g.Name, f, raw)
		if err != nil {
			fields[key] = "The value " + err.Error() + "."
			continue
		}
		if !skip {
			rows = append(rows, row)
		}
	}
	if len(fields) > 0 {
		return nil, apperr.Validation(fields)
	}
	if err := s.validateGroup(g.Name, input); err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		if err := s.store.Upsert(ctx, rows); err != nil {
			return nil, err
		}
		if err := s.cache.Delete(ctx, cacheKey(g.Name)); err != nil {
			return nil, err
		}
		if s.onChanged != nil {
			s.onChanged(ctx, g.Name)
		}
	}
	return s.AdminView(ctx, g.Name)
}

// SeedDefaults inserts default values for every missing key.
func (s *SettingsService) SeedDefaults(ctx context.Context, overrides map[string]map[string]any) error {
	var rows []model.Setting
	for _, g := range SettingGroups {
		for _, f := range g.Fields {
			value := f.Default
			if v, ok := overrides[g.Name][f.Key]; ok {
				value = v
			}
			row, err := s.encode(g.Name, f, value)
			if err != nil {
				return err
			}
			rows = append(rows, row)
		}
	}
	if err := s.store.InsertMissing(ctx, rows); err != nil {
		return err
	}
	keys := make([]string, 0, len(SettingGroups))
	for _, g := range SettingGroups {
		keys = append(keys, cacheKey(g.Name))
	}
	return s.cache.Delete(ctx, keys...)
}

func (s *SettingsService) prepareRow(group string, f SettingField, raw json.RawMessage) (model.Setting, bool, error) {
	value, err := f.Normalize(raw)
	if err != nil {
		return model.Setting{}, false, err
	}
	if f.Secret && value == "" {
		return model.Setting{}, true, nil
	}
	row, err := s.encode(group, f, value)
	return row, false, err
}

func (s *SettingsService) encode(group string, f SettingField, value any) (model.Setting, error) {
	if f.Secret {
		plain, _ := value.(string)
		if plain != "" {
			enc, err := s.cipher.Encrypt(plain)
			if err != nil {
				return model.Setting{}, fmt.Errorf("encrypt %s.%s: %w", group, f.Key, err)
			}
			value = enc
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return model.Setting{}, fmt.Errorf("encode %s.%s: %w", group, f.Key, err)
	}
	return model.Setting{Group: group, Key: f.Key, Value: raw, IsSecret: f.Secret}, nil
}

func (s *SettingsService) validateGroup(group string, input map[string]json.RawMessage) error {
	if group != "alerts" {
		return nil
	}
	var t [3]int
	for i, key := range []string{"expiry_threshold_1", "expiry_threshold_2", "expiry_threshold_3"} {
		raw, ok := input[key]
		if !ok {
			return apperr.Validation(map[string]string{key: "All three thresholds must be sent together."})
		}
		if err := json.Unmarshal(raw, &t[i]); err != nil {
			return apperr.Validation(map[string]string{key: "The value must be a whole number."})
		}
	}
	if t[0] >= t[1] || t[1] >= t[2] {
		return apperr.Validation(map[string]string{"expiry_threshold_2": "Thresholds must be strictly increasing."})
	}
	return nil
}

func (s *SettingsService) load(ctx context.Context, group string) (SettingGroup, map[string]any, error) {
	g, ok := FindSettingGroup(group)
	if !ok {
		return SettingGroup{}, nil, apperr.NotFound("Settings group")
	}
	rows, err := s.cachedRows(ctx, g.Name)
	if err != nil {
		return g, nil, err
	}
	stored := make(map[string]model.Setting, len(rows))
	for _, r := range rows {
		stored[r.Key] = r
	}
	values := make(map[string]any, len(g.Fields))
	for _, f := range g.Fields {
		row, ok := stored[f.Key]
		if !ok {
			values[f.Key] = f.Default
			continue
		}
		v, err := s.decode(f, row)
		if err != nil {
			return g, nil, fmt.Errorf("decode %s.%s: %w", g.Name, f.Key, err)
		}
		values[f.Key] = v
	}
	return g, values, nil
}

func (s *SettingsService) decode(f SettingField, row model.Setting) (any, error) {
	var v any
	if err := json.Unmarshal(row.Value, &v); err != nil {
		return nil, err
	}
	if f.Kind == KindInt {
		if n, ok := v.(float64); ok {
			return int(n), nil
		}
	}
	if list, ok := v.([]any); ok {
		out := make([]string, 0, len(list))
		for _, it := range list {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out, nil
	}
	if !row.IsSecret {
		return v, nil
	}
	enc, _ := v.(string)
	if enc == "" {
		return "", nil
	}
	plain, err := s.cipher.Decrypt(enc)
	if errors.Is(err, secret.ErrInvalidCiphertext) {
		return "", fmt.Errorf("secret cannot be decrypted; APP_ENCRYPTION_KEY may have changed: %w", err)
	}
	return plain, err
}

func (s *SettingsService) cachedRows(ctx context.Context, group string) ([]model.Setting, error) {
	raw, ok, err := s.cache.Get(ctx, cacheKey(group))
	if err != nil {
		return nil, err
	}
	if ok {
		var rows []model.Setting
		if json.Unmarshal(raw, &rows) == nil {
			return rows, nil
		}
	}
	rows, err := s.store.ListGroup(ctx, group)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("encode settings cache: %w", err)
	}
	if err := s.cache.Set(ctx, cacheKey(group), payload, settingsCacheTTL); err != nil {
		return nil, err
	}
	return rows, nil
}

func cacheKey(group string) string { return "settings:" + group }
