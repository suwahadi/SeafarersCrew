package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

const maxEmbeddedFile = 5 << 20

// FileGetter loads file metadata.
type FileGetter interface {
	Get(ctx context.Context, id uuid.UUID) (model.File, error)
}

// ObjectGetter reads stored objects.
type ObjectGetter interface {
	Get(ctx context.Context, key string) ([]byte, error)
}

// StoredFileReader reads ready files from storage for embedding in documents.
type StoredFileReader struct {
	files   FileGetter
	objects ObjectGetter
}

// NewStoredFileReader returns a StoredFileReader.
func NewStoredFileReader(files FileGetter, objects ObjectGetter) *StoredFileReader {
	return &StoredFileReader{files: files, objects: objects}
}

// Content returns the bytes and MIME type of a ready file no larger than 5 MB.
func (r *StoredFileReader) Content(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	f, err := r.files.Get(ctx, id)
	if err != nil {
		return nil, "", fmt.Errorf("load file metadata: %w", err)
	}
	if f.Status != model.FileReady || f.SizeStored > maxEmbeddedFile {
		return nil, "", fmt.Errorf("file %s cannot be embedded", id)
	}
	data, err := r.objects.Get(ctx, f.StorageKey)
	if err != nil {
		return nil, "", fmt.Errorf("read file: %w", err)
	}
	return data, f.MimeType, nil
}
