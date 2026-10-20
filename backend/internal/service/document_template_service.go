package service

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// TemplateStore persists document templates.
type TemplateStore interface {
	List(ctx context.Context, kind string) ([]model.DocumentTemplate, error)
	Get(ctx context.Context, id uuid.UUID) (model.DocumentTemplate, error)
	Default(ctx context.Context, kind string) (model.DocumentTemplate, error)
	CountByKind(ctx context.Context, kind string) (int, error)
	Save(ctx context.Context, t model.DocumentTemplate) (model.DocumentTemplate, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// DocumentTemplateService manages document builder templates.
type DocumentTemplateService struct {
	store    TemplateStore
	sanitize *bluemonday.Policy
}

// NewDocumentTemplateService builds a DocumentTemplateService.
func NewDocumentTemplateService(store TemplateStore) *DocumentTemplateService {
	return &DocumentTemplateService{store: store, sanitize: NewTemplatePolicy()}
}

// NewTemplatePolicy returns the HTML policy for template and rich text content: UGC markup, tables, data URI images and a limited style whitelist.
func NewTemplatePolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowDataURIImages()
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*( [a-zA-Z][a-zA-Z0-9-]*)*$`)).Globally()
	p.AllowAttrs("colspan", "rowspan").Matching(regexp.MustCompile(`^[0-9]{1,2}$`)).OnElements("td", "th")
	p.AllowAttrs("width", "height").Matching(regexp.MustCompile(`^[0-9]{1,4}(px|%)?$`)).OnElements("img", "table", "td", "th", "col")
	length := regexp.MustCompile(`^-?[0-9]{1,4}(\.[0-9]{1,2})?(px|pt|mm|cm|%|em|rem)?$`)
	p.AllowStyles("text-align").MatchingEnum("left", "center", "right", "justify").Globally()
	p.AllowStyles("vertical-align").MatchingEnum("top", "middle", "bottom").Globally()
	p.AllowStyles("font-weight").MatchingEnum("normal", "bold", "600", "700").Globally()
	p.AllowStyles("font-style").MatchingEnum("normal", "italic").Globally()
	p.AllowStyles("text-decoration").MatchingEnum("none", "underline", "line-through").Globally()
	p.AllowStyles("width", "height", "font-size", "line-height", "margin-top", "margin-bottom", "margin-left", "margin-right",
		"padding", "padding-top", "padding-bottom", "padding-left", "padding-right").Matching(length).Globally()
	p.AllowStyles("color", "background-color").Matching(regexp.MustCompile(`^#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?$`)).Globally()
	p.AllowStyles("border-collapse").MatchingEnum("collapse", "separate").Globally()
	p.AllowStyles("page-break-before", "page-break-after", "break-before", "break-after").MatchingEnum("always", "auto", "page", "avoid").Globally()
	p.AllowStyles("page-break-inside", "break-inside").MatchingEnum("auto", "avoid").Globally()
	return p
}

// SanitizeHTML cleans HTML with the template policy.
func (s *DocumentTemplateService) SanitizeHTML(in string) string {
	return s.sanitize.Sanitize(in)
}

// TemplateResult is a saved template with warnings about unknown tags.
type TemplateResult struct {
	Template    model.DocumentTemplate
	UnknownTags []string
}

// List returns templates, optionally of one kind.
func (s *DocumentTemplateService) List(ctx context.Context, kind string) ([]model.DocumentTemplate, error) {
	if kind != "" && !validKind(kind) {
		return nil, apperr.Validation(map[string]string{"kind": "The kind must be sea, quotation or cv."})
	}
	return s.store.List(ctx, kind)
}

// Get returns a template.
func (s *DocumentTemplateService) Get(ctx context.Context, id uuid.UUID) (TemplateResult, error) {
	t, err := s.store.Get(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return TemplateResult{}, apperr.NotFound("Template")
	}
	if err != nil {
		return TemplateResult{}, err
	}
	return TemplateResult{Template: t, UnknownTags: templateUnknownTags(t)}, nil
}

// Save validates, sanitizes and stores a template (zero ID creates); the first template of a kind becomes its default.
func (s *DocumentTemplateService) Save(ctx context.Context, t model.DocumentTemplate) (TemplateResult, error) {
	if t.ID != uuid.Nil {
		cur, err := s.store.Get(ctx, t.ID)
		if errors.Is(err, repository.ErrNotFound) {
			return TemplateResult{}, apperr.NotFound("Template")
		}
		if err != nil {
			return TemplateResult{}, err
		}
		t.Kind = cur.Kind
		if cur.IsDefault && !t.IsDefault {
			return TemplateResult{}, apperr.Validation(map[string]string{"is_default": "Make another template the default instead of unsetting this one."})
		}
	} else {
		n, err := s.store.CountByKind(ctx, t.Kind)
		if err != nil {
			return TemplateResult{}, err
		}
		if n == 0 {
			t.IsDefault = true
		}
	}
	t.HeaderHTML = s.sanitize.Sanitize(t.HeaderHTML)
	t.BodyHTML = s.sanitize.Sanitize(t.BodyHTML)
	t.FooterHTML = s.sanitize.Sanitize(t.FooterHTML)
	if strings.TrimSpace(t.BodyHTML) == "" {
		return TemplateResult{}, apperr.Validation(map[string]string{"body_html": "The body is required."})
	}
	saved, err := s.store.Save(ctx, t)
	if errors.Is(err, repository.ErrNotFound) {
		return TemplateResult{}, apperr.NotFound("Template")
	}
	if err != nil {
		return TemplateResult{}, err
	}
	return TemplateResult{Template: saved, UnknownTags: templateUnknownTags(saved)}, nil
}

// Delete soft-deletes a template that is neither the default nor the last of its kind.
func (s *DocumentTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	t, err := s.store.Get(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound("Template")
	}
	if err != nil {
		return err
	}
	if t.IsDefault {
		return apperr.Conflict("default_template", "The default template cannot be deleted; make another template the default first.")
	}
	n, err := s.store.CountByKind(ctx, t.Kind)
	if err != nil {
		return err
	}
	if n <= 1 {
		return apperr.Conflict("last_template", "The last template of a kind cannot be deleted.")
	}
	return s.store.SoftDelete(ctx, id)
}

// Resolve returns the template with id, or the default of kind when id is nil.
func (s *DocumentTemplateService) Resolve(ctx context.Context, kind string, id *uuid.UUID) (model.DocumentTemplate, error) {
	var (
		t   model.DocumentTemplate
		err error
	)
	if id != nil {
		t, err = s.store.Get(ctx, *id)
	} else {
		t, err = s.store.Default(ctx, kind)
	}
	if errors.Is(err, repository.ErrNotFound) {
		return t, apperr.NotFound("Template")
	}
	if err != nil {
		return t, err
	}
	if t.Kind != kind {
		return t, apperr.Validation(map[string]string{"template_id": "The template is for another document kind."})
	}
	return t, nil
}

func templateUnknownTags(t model.DocumentTemplate) []string {
	return UnknownTags(t.Kind, t.HeaderHTML+t.BodyHTML+t.FooterHTML)
}

func validKind(kind string) bool {
	return kind == model.DocumentSEA || kind == model.DocumentQuotation || kind == model.DocumentCV
}
