package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// SessionIDBytes is the entropy of a session ID.
const SessionIDBytes = 32

// SessionRepository keeps sessions in Redis under sess:<sha256(id)>.
type SessionRepository struct {
	rdb      *goredis.Client
	idle     time.Duration
	absolute time.Duration
	now      func() time.Time
}

// NewSessionRepository returns a SessionRepository with idle and absolute lifetimes.
func NewSessionRepository(rdb *goredis.Client, idle, absolute time.Duration, now func() time.Time) *SessionRepository {
	return &SessionRepository{rdb: rdb, idle: idle, absolute: absolute, now: now}
}

func sessionKey(rawID string) string { return "sess:" + token.SHA256Hex(rawID) }

func adminSessionsKey(adminID uuid.UUID) string { return "admin_sessions:" + adminID.String() }

// Create stores a new session and returns the raw ID for the cookie.
func (r *SessionRepository) Create(ctx context.Context, adminID uuid.UUID, ip, userAgent string) (string, error) {
	rawID, err := token.URLSafe(SessionIDBytes)
	if err != nil {
		return "", err
	}
	now := r.now().UTC()
	s := model.Session{AdminID: adminID, IP: ip, UserAgent: userAgent, CreatedAt: now, LastSeen: now}
	payload, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encode session: %w", err)
	}
	key := sessionKey(rawID)
	pipe := r.rdb.TxPipeline()
	pipe.Set(ctx, key, payload, r.idle)
	pipe.SAdd(ctx, adminSessionsKey(adminID), key)
	pipe.Expire(ctx, adminSessionsKey(adminID), r.absolute)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("store session: %w", err)
	}
	return rawID, nil
}

// Get loads a session, refreshes its idle TTL and enforces the absolute lifetime.
func (r *SessionRepository) Get(ctx context.Context, rawID string) (model.Session, error) {
	key := sessionKey(rawID)
	payload, err := r.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return model.Session{}, ErrNotFound
	}
	if err != nil {
		return model.Session{}, fmt.Errorf("load session: %w", err)
	}
	var s model.Session
	if err := json.Unmarshal(payload, &s); err != nil {
		return model.Session{}, fmt.Errorf("decode session: %w", err)
	}
	now := r.now().UTC()
	remaining := s.CreatedAt.Add(r.absolute).Sub(now)
	if remaining <= 0 {
		if err := r.Delete(ctx, rawID); err != nil {
			return model.Session{}, err
		}
		return model.Session{}, ErrNotFound
	}
	s.LastSeen = now
	updated, err := json.Marshal(s)
	if err != nil {
		return model.Session{}, fmt.Errorf("encode session: %w", err)
	}
	if err := r.rdb.Set(ctx, key, updated, min(r.idle, remaining)).Err(); err != nil {
		return model.Session{}, fmt.Errorf("refresh session: %w", err)
	}
	return s, nil
}

// Delete removes one session.
func (r *SessionRepository) Delete(ctx context.Context, rawID string) error {
	key := sessionKey(rawID)
	payload, err := r.rdb.GetDel(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	var s model.Session
	if json.Unmarshal(payload, &s) == nil {
		if err := r.rdb.SRem(ctx, adminSessionsKey(s.AdminID), key).Err(); err != nil {
			return fmt.Errorf("unindex session: %w", err)
		}
	}
	return nil
}

// DeleteAllForAdmin removes every session of the admin.
func (r *SessionRepository) DeleteAllForAdmin(ctx context.Context, adminID uuid.UUID) error {
	indexKey := adminSessionsKey(adminID)
	keys, err := r.rdb.SMembers(ctx, indexKey).Result()
	if err != nil {
		return fmt.Errorf("list admin sessions: %w", err)
	}
	if err := r.rdb.Del(ctx, append(keys, indexKey)...).Err(); err != nil {
		return fmt.Errorf("delete admin sessions: %w", err)
	}
	return nil
}
