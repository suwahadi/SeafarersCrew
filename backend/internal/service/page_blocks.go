package service

import (
	"context"
	"fmt"
	"maps"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type galleryRef struct {
	field string
	id    uuid.UUID
}

func (s *PageService) prepareBlocks(ctx context.Context, blocks []model.Block) ([]FileRef, map[string]string, error) {
	var refs []FileRef
	var galleries []galleryRef
	for i, b := range blocks {
		prefix := fmt.Sprintf("blocks.%d.settings.", i)
		switch v := b.Settings.(type) {
		case *model.RichTextSettings:
			v.HTML = s.html.Sanitize(v.HTML)
		case *model.ImageTextSettings:
			v.TextHTML = s.html.Sanitize(v.TextHTML)
			refs = append(refs, FileRef{Field: prefix + "image_file_id", ID: v.ImageFileID, Kind: ImageFile})
		case *model.GallerySettings:
			for j, id := range v.ImageIDs {
				galleries = append(galleries, galleryRef{field: fmt.Sprintf("%simage_ids.%d", prefix, j), id: id})
			}
		case *model.CTASettings:
			refs = append(refs, FileRef{Field: prefix + "background_image_file_id", ID: v.BackgroundImageFileID, Kind: ImageFile})
		case *model.FAQSettings:
			for j := range v.Items {
				v.Items[j].AnswerHTML = s.html.Sanitize(v.Items[j].AnswerHTML)
			}
		case *model.CardGridSettings:
			for j, c := range v.Cards {
				refs = append(refs, FileRef{Field: fmt.Sprintf("%scards.%d.image_file_id", prefix, j), ID: c.ImageFileID, Kind: ImageFile})
			}
		}
	}
	missing, err := s.missingGalleryRefs(ctx, galleries)
	if err != nil {
		return nil, nil, err
	}
	return refs, missing, nil
}

func (s *PageService) missingGalleryRefs(ctx context.Context, refs []galleryRef) (map[string]string, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(refs))
	for i, r := range refs {
		ids[i] = r.id
	}
	found, err := s.lookup.GalleryImagesExist(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("check gallery images: %w", err)
	}
	missing := map[string]string{}
	for _, r := range refs {
		if !found[r.id] {
			missing[r.field] = "The gallery image does not exist."
		}
	}
	return missing, nil
}

func mergeValidation(fields map[string]string, err error) error {
	if err == nil && len(fields) == 0 {
		return nil
	}
	merged := map[string]string{}
	if e, ok := apperr.As(err); ok && e.Code == "validation_failed" {
		maps.Copy(merged, e.Fields)
	} else if err != nil {
		return err
	}
	maps.Copy(merged, fields)
	return apperr.Validation(merged)
}
