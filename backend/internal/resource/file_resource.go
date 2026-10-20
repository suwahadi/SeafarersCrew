package resource

import (
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// FileVariant is the public shape of an image variant.
type FileVariant struct {
	Width  int   `json:"width"`
	Height int   `json:"height"`
	Size   int64 `json:"size"`
}

// File is the public shape of a stored file; storage keys are never exposed.
type File struct {
	ID           string        `json:"id"`
	OriginalName string        `json:"original_name"`
	MimeType     string        `json:"mime_type"`
	SizeOriginal int64         `json:"size_original"`
	SizeStored   int64         `json:"size_stored"`
	Status       string        `json:"status"`
	IsPublic     bool          `json:"is_public"`
	Variants     []FileVariant `json:"variants"`
	CreatedAt    time.Time     `json:"created_at"`
}

// NewFile maps a model.File.
func NewFile(f model.File) File {
	variants := make([]FileVariant, 0, len(f.Variants))
	for _, v := range f.Variants {
		variants = append(variants, FileVariant{Width: v.Width, Height: v.Height, Size: v.Size})
	}
	return File{
		ID:           f.ID.String(),
		OriginalName: f.OriginalName,
		MimeType:     f.MimeType,
		SizeOriginal: f.SizeOriginal,
		SizeStored:   f.SizeStored,
		Status:       f.Status,
		IsPublic:     f.IsPublic,
		Variants:     variants,
		CreatedAt:    f.CreatedAt,
	}
}
