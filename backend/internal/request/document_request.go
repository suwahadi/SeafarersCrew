package request

import (
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// MaxTemplateHTML limits each HTML part of a template.
const MaxTemplateHTML = 2 << 20

// DocumentTemplate is the body of POST /document-templates and PUT /document-templates/{id}.
type DocumentTemplate struct {
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	PaperSize   string `json:"paper_size"`
	Orientation string `json:"orientation"`
	HeaderHTML  string `json:"header_html"`
	BodyHTML    string `json:"body_html"`
	FooterHTML  string `json:"footer_html"`
	IsDefault   bool   `json:"is_default"`
}

// Validate implements Validator.
func (r *DocumentTemplate) Validate() map[string]string {
	Trim(&r.Name, &r.Kind, &r.PaperSize, &r.Orientation)
	if r.PaperSize == "" {
		r.PaperSize = "a4"
	}
	if r.Orientation == "" {
		r.Orientation = "portrait"
	}
	e := Errors{}
	e.Required("name", r.Name, 120)
	e.Required("kind", r.Kind, 20)
	e.OneOf("kind", r.Kind, model.DocumentSEA, model.DocumentQuotation, model.DocumentCV)
	e.OneOf("paper_size", r.PaperSize, "a4", "letter")
	e.OneOf("orientation", r.Orientation, "portrait", "landscape")
	for field, v := range map[string]string{"header_html": r.HeaderHTML, "body_html": r.BodyHTML, "footer_html": r.FooterHTML} {
		if len(v) > MaxTemplateHTML {
			e[field] = "This content is too large."
		}
	}
	return e
}

// Model maps the request onto a template.
func (r *DocumentTemplate) Model() model.DocumentTemplate {
	return model.DocumentTemplate{Name: r.Name, Kind: r.Kind, PaperSize: r.PaperSize, Orientation: r.Orientation,
		HeaderHTML: r.HeaderHTML, BodyHTML: r.BodyHTML, FooterHTML: r.FooterHTML, IsDefault: r.IsDefault}
}

// DocumentRender is the body of POST /documents/preview and POST /documents/generate.
type DocumentRender struct {
	Kind       string   `json:"kind"`
	TemplateID string   `json:"template_id"`
	RelatedID  string   `json:"related_id"`
	Hide       []string `json:"hide"`

	templateID *uuid.UUID
}

// Validate implements Validator.
func (r *DocumentRender) Validate() map[string]string {
	Trim(&r.Kind, &r.TemplateID, &r.RelatedID)
	e := Errors{}
	e.Required("kind", r.Kind, 20)
	e.OneOf("kind", r.Kind, model.DocumentSEA, model.DocumentQuotation, model.DocumentCV)
	e.Required("related_id", r.RelatedID, 20)
	r.templateID = e.UUID("template_id", r.TemplateID)
	if len(r.Hide) > 10 {
		e["hide"] = "Too many sections."
	}
	return e
}

// Parsed returns the template ID chosen, if any.
func (r *DocumentRender) Parsed() *uuid.UUID { return r.templateID }
