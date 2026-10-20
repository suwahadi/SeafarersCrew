// Package s3 stores private objects in AWS S3 or any S3-compatible service configured in Settings.
package s3

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrNotConfigured is returned when storage settings are incomplete.
var ErrNotConfigured = errors.New("object storage is not configured")

// Config is the storage configuration read from Settings.
type Config struct {
	Provider        string
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UsePathStyle    bool
	PresignMinutes  int
}

// ConfigSource loads the current storage configuration.
type ConfigSource func(ctx context.Context) (Config, error)

// Storage puts, deletes and presigns objects, rebuilding the client when Settings change.
type Storage struct {
	source ConfigSource
	mu     sync.Mutex
	hash   string
	client *minio.Client
	cfg    Config
}

// New returns a Storage reading configuration from source.
func New(source ConfigSource) *Storage {
	return &Storage{source: source}
}

func (s *Storage) current(ctx context.Context) (*minio.Client, Config, error) {
	cfg, err := s.source(ctx)
	if err != nil {
		return nil, Config{}, fmt.Errorf("load storage settings: %w", err)
	}
	if cfg.Bucket == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return nil, Config{}, ErrNotConfigured
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, Config{}, fmt.Errorf("hash storage settings: %w", err)
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil && s.hash == hash {
		return s.client, s.cfg, nil
	}
	client, err := newClient(cfg)
	if err != nil {
		return nil, Config{}, err
	}
	s.client, s.hash, s.cfg = client, hash, cfg
	return client, cfg, nil
}

func newClient(cfg Config) (*minio.Client, error) {
	host, secure := "s3.amazonaws.com", true
	if cfg.Endpoint != "" {
		u, err := url.Parse(cfg.Endpoint)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("invalid storage endpoint %q", cfg.Endpoint)
		}
		host, secure = u.Host, u.Scheme == "https"
	} else if cfg.Region != "" && cfg.Region != "us-east-1" {
		host = "s3." + cfg.Region + ".amazonaws.com"
	}
	lookup := minio.BucketLookupDNS
	if cfg.UsePathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure:       secure,
		Region:       cfg.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, fmt.Errorf("create storage client: %w", err)
	}
	return client, nil
}

// Put uploads data as a private object.
func (s *Storage) Put(ctx context.Context, key string, data []byte, contentType string) error {
	client, cfg, err := s.current(ctx)
	if err != nil {
		return err
	}
	_, err = client.PutObject(ctx, cfg.Bucket, key, bytes.NewReader(data), int64(len(data)), minio.PutObjectOptions{
		ContentType:  contentType,
		CacheControl: "private, max-age=0, no-store",
	})
	if err != nil {
		return fmt.Errorf("put object %s: %w", key, err)
	}
	return nil
}

// Get downloads an object.
func (s *Storage) Get(ctx context.Context, key string) ([]byte, error) {
	client, cfg, err := s.current(ctx)
	if err != nil {
		return nil, err
	}
	obj, err := client.GetObject(ctx, cfg.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object %s: %w", key, err)
	}
	defer func() { _ = obj.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(obj); err != nil {
		return nil, fmt.Errorf("read object %s: %w", key, err)
	}
	return buf.Bytes(), nil
}

// Delete removes objects; missing objects are not an error.
func (s *Storage) Delete(ctx context.Context, keys ...string) error {
	client, cfg, err := s.current(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := client.RemoveObject(ctx, cfg.Bucket, key, minio.RemoveObjectOptions{}); err != nil {
			return fmt.Errorf("delete object %s: %w", key, err)
		}
	}
	return nil
}

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// Stat returns object metadata.
func (s *Storage) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	client, cfg, err := s.current(ctx)
	if err != nil {
		return ObjectInfo{}, err
	}
	info, err := client.StatObject(ctx, cfg.Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("stat object %s: %w", key, err)
	}
	return ObjectInfo{Size: info.Size, ContentType: info.ContentType}, nil
}

// PresignGet returns a short-lived download URL and its lifetime.
func (s *Storage) PresignGet(ctx context.Context, key, filename string, inline bool) (string, time.Duration, error) {
	client, cfg, err := s.current(ctx)
	if err != nil {
		return "", 0, err
	}
	lifetime := time.Duration(cfg.PresignMinutes) * time.Minute
	if lifetime <= 0 {
		lifetime = 5 * time.Minute
	}
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	params := url.Values{}
	params.Set("response-content-disposition", mime.FormatMediaType(disposition, map[string]string{"filename": safeFilename(filename)}))
	u, err := client.PresignedGetObject(ctx, cfg.Bucket, key, lifetime, params)
	if err != nil {
		return "", 0, fmt.Errorf("presign object %s: %w", key, err)
	}
	return u.String(), lifetime, nil
}

// Test verifies the bucket exists and that an object can be written and deleted.
func (s *Storage) Test(ctx context.Context) error {
	client, cfg, err := s.current(ctx)
	if err != nil {
		return err
	}
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", cfg.Bucket)
	}
	key := fmt.Sprintf("connection-test/%d.txt", time.Now().UnixNano())
	if err := s.Put(ctx, key, []byte("ok"), "text/plain"); err != nil {
		return err
	}
	return s.Delete(ctx, key)
}

func safeFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == '"' || r == '\\' || r == '/' {
			return '_'
		}
		return r
	}, name)
	if name == "" {
		return "download"
	}
	return name
}

// Ping checks that the configured bucket is reachable.
func (s *Storage) Ping(ctx context.Context) error {
	client, cfg, err := s.current(ctx)
	if err != nil {
		return err
	}
	if _, err := client.BucketExists(ctx, cfg.Bucket); err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	return nil
}
