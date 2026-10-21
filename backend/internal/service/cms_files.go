package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// PublicFileKind names the kind of public file a CMS field accepts.
type PublicFileKind string

// Kinds of public files.
const (
	ImageFile PublicFileKind = "image"
	LogoFile  PublicFileKind = "logo"
	FlagFile  PublicFileKind = "flag"
)

var kindMimes = map[PublicFileKind][]string{
	ImageFile: {"image/webp"},
	LogoFile:  {"image/webp", "image/svg+xml"},
	FlagFile:  {"image/svg+xml"},
}

var kindMessages = map[PublicFileKind]string{
	ImageFile: "The file must be an uploaded CMS image.",
	LogoFile:  "The file must be an uploaded logo image.",
	FlagFile:  "The file must be an uploaded SVG flag.",
}

// PublicFileLookup finds ready public files by ID.
type PublicFileLookup interface {
	PublicFiles(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]model.PublicFileInfo, error)
}

// FileRef is one file reference to validate; a nil ID is skipped.
type FileRef struct {
	Field string
	ID    *uuid.UUID
	Kind  PublicFileKind
}

// PublicFileChecker validates that referenced files exist, are ready, public and of the right type.
type PublicFileChecker struct {
	lookup PublicFileLookup
}

// NewPublicFileChecker builds a PublicFileChecker.
func NewPublicFileChecker(lookup PublicFileLookup) *PublicFileChecker {
	return &PublicFileChecker{lookup: lookup}
}

// Check validates refs and returns the info of every referenced file, or a validation error keyed by field.
func (c *PublicFileChecker) Check(ctx context.Context, refs ...FileRef) (map[uuid.UUID]model.PublicFileInfo, error) {
	ids := make([]uuid.UUID, 0, len(refs))
	for _, r := range refs {
		if r.ID != nil && !slices.Contains(ids, *r.ID) {
			ids = append(ids, *r.ID)
		}
	}
	found, err := c.lookup.PublicFiles(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("check files: %w", err)
	}
	fields := map[string]string{}
	for _, r := range refs {
		if r.ID == nil {
			continue
		}
		info, ok := found[*r.ID]
		switch {
		case !ok:
			fields[r.Field] = "The file does not exist or is not a ready public file."
		case !slices.Contains(kindMimes[r.Kind], info.MimeType):
			fields[r.Field] = kindMessages[r.Kind]
		}
	}
	if len(fields) > 0 {
		return nil, apperr.Validation(fields)
	}
	return found, nil
}

// SiteChanger announces that public content changed.
type SiteChanger interface {
	Changed(ctx context.Context, tags ...string) error
}

func changedHook(site SiteChanger, tags ...string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return site.Changed(ctx, tags...)
	}
}
