package model

import (
	"time"

	"github.com/google/uuid"
)

// File statuses.
const (
	FileProcessing = "processing"
	FileReady      = "ready"
	FileFailed     = "failed"
)

// FileVariant is a resized copy of an image file.
type FileVariant struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	StorageKey string `json:"storage_key"`
	Size       int64  `json:"size"`
}

// File is a stored upload recorded in the files table.
type File struct {
	ID           uuid.UUID
	StorageKey   string
	OriginalName string
	MimeType     string
	SizeOriginal int64
	SizeStored   int64
	SHA256       string
	OwnerType    string
	OwnerID      *uuid.UUID
	IsPublic     bool
	Variants     []FileVariant
	Width        int
	Height       int
	Status       string
	CreatedAt    time.Time
}

// Image is the stored-file data needed to render a public image.
type Image struct {
	ID         uuid.UUID     `json:"id"`
	StorageKey string        `json:"storage_key"`
	MimeType   string        `json:"mime_type"`
	Width      int           `json:"width"`
	Height     int           `json:"height"`
	Variants   []FileVariant `json:"variants"`
}
