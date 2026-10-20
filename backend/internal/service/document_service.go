package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/gotenberg"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

// Related record types of generated documents.
const (
	RelatedContract  = "contract"
	RelatedQuotation = "quotation"
	RelatedCrew      = "crew"
)

const generationTimeout = 60 * time.Second

// CrewSource loads crew data for CVs.
type CrewSource interface {
	CrewProfile(ctx context.Context, publicID string) (repository.CrewProfile, error)
	ContractID(ctx context.Context, publicID string) (uuid.UUID, error)
	SEADataSource
}

// CrewTabs loads documents, certificates and sea service of a crew member.
type CrewTabs interface {
	ListDocuments(ctx context.Context, k repository.DocumentKind, publicID string) ([]DocumentView, error)
	SeaService(ctx context.Context, publicID string) (CrewSeaService, error)
}

// QuotationReader loads quotations.
type QuotationReader interface {
	Get(ctx context.Context, publicID string) (model.Quotation, error)
}

// TemplateResolver picks the template of a document.
type TemplateResolver interface {
	Resolve(ctx context.Context, kind string, id *uuid.UUID) (model.DocumentTemplate, error)
}

// HTMLRenderer converts HTML to PDF.
type HTMLRenderer interface {
	ConvertHTML(ctx context.Context, req gotenberg.HTMLRequest) ([]byte, error)
}

// PDFCompressor shrinks PDFs, returning the original when compression does not help.
type PDFCompressor interface {
	Compress(ctx context.Context, data []byte, preset string) ([]byte, error)
}

// GeneratedFiles stores generated PDFs and links them to their record.
type GeneratedFiles interface {
	StoreGenerated(ctx context.Context, entity, filename, mime, ext string, data []byte, ownerID *uuid.UUID) (model.File, error)
	Attach(ctx context.Context, id uuid.UUID, ownerType string, ownerID uuid.UUID) error
}

// GeneratedDocumentStore records generated documents.
type GeneratedDocumentStore interface {
	Create(ctx context.Context, g model.GeneratedDocument) (model.GeneratedDocument, error)
	List(ctx context.Context, relatedType string, relatedID *uuid.UUID, limit int) ([]model.GeneratedDocument, error)
}

// DocumentDeps groups the collaborators of DocumentService.
type DocumentDeps struct {
	Templates  TemplateResolver
	Sources    CrewSource
	Crews      CrewTabs
	Quotations QuotationReader
	Settings   SettingsReader
	Files      FileReader
	Renderer   HTMLRenderer
	Compressor PDFCompressor
	Storage    GeneratedFiles
	History    GeneratedDocumentStore
	Pool       Bounded
	Now        func() time.Time
}

// DocumentService merges templates with data and produces PDFs (PRD 6.6).
type DocumentService struct {
	d     DocumentDeps
	newID func() (string, error)
}

// NewDocumentService builds a DocumentService.
func NewDocumentService(d DocumentDeps) *DocumentService {
	return &DocumentService{d: d, newID: publicid.New}
}

// DocumentRequest selects what to render.
type DocumentRequest struct {
	Kind       string
	TemplateID *uuid.UUID
	RelatedID  string
	Hide       []string
}

// RenderedDocument is a merged template ready for preview or conversion.
type RenderedDocument struct {
	Template    model.DocumentTemplate
	Header      string
	Body        string
	Footer      string
	UnknownTags []string
	RelatedType string
	RelatedID   uuid.UUID
	Filename    string
}

// Render merges the template of req with its data.
func (s *DocumentService) Render(ctx context.Context, req DocumentRequest) (RenderedDocument, error) {
	if !validKind(req.Kind) {
		return RenderedDocument{}, apperr.Validation(map[string]string{"kind": "The kind must be sea, quotation or cv."})
	}
	for _, h := range req.Hide {
		if !slices.Contains(CVSections, h) {
			return RenderedDocument{}, apperr.Validation(map[string]string{"hide": "Unknown section: " + h + "."})
		}
	}
	tpl, err := s.d.Templates.Resolve(ctx, req.Kind, req.TemplateID)
	if err != nil {
		return RenderedDocument{}, err
	}
	f, err := s.format(ctx)
	if err != nil {
		return RenderedDocument{}, err
	}
	vars, relType, relID, name, err := s.vars(ctx, req, f)
	if err != nil {
		return RenderedDocument{}, err
	}
	company, err := s.d.Settings.Values(ctx, "company")
	if err != nil {
		return RenderedDocument{}, err
	}
	vars.Merge(CompanyVars(ctx, company, s.d.Files, f))
	header, u1 := RenderTemplate(tpl.HeaderHTML, vars)
	body, u2 := RenderTemplate(tpl.BodyHTML, vars)
	footer, u3 := RenderTemplate(tpl.FooterHTML, vars)
	unknown := append(append(u1, u2...), u3...)
	slices.Sort(unknown)
	return RenderedDocument{
		Template: tpl, Header: header, Body: body, Footer: footer, UnknownTags: slices.Compact(unknown),
		RelatedType: relType, RelatedID: relID, Filename: name,
	}, nil
}

func (s *DocumentService) vars(ctx context.Context, req DocumentRequest, f docFormat) (Vars, string, uuid.UUID, string, error) {
	id := strings.ToUpper(strings.TrimSpace(req.RelatedID))
	if !publicid.Valid(id) {
		return Vars{}, "", uuid.Nil, "", apperr.Validation(map[string]string{"related_id": "A valid record ID is required."})
	}
	switch req.Kind {
	case model.DocumentSEA:
		rec, err := s.d.Sources.SEA(ctx, id)
		if errors.Is(err, repository.ErrNotFound) {
			return Vars{}, "", uuid.Nil, "", apperr.NotFound("Contract")
		}
		if err != nil {
			return Vars{}, "", uuid.Nil, "", err
		}
		return SEAVars(rec, f), RelatedContract, rec.ContractID, "SEA-" + rec.ContractPublicID + "-" + rec.Crew.FullName, nil
	case model.DocumentQuotation:
		q, err := s.d.Quotations.Get(ctx, id)
		if err != nil {
			return Vars{}, "", uuid.Nil, "", err
		}
		return QuotationVars(q, f), RelatedQuotation, q.ID, "Quotation-" + q.PublicID + "-" + q.ClientName, nil
	default:
		data, err := s.cvData(ctx, id)
		if err != nil {
			return Vars{}, "", uuid.Nil, "", err
		}
		hide := map[string]bool{}
		for _, h := range req.Hide {
			hide[h] = true
		}
		return CVVars(data, f, hide), RelatedCrew, data.Profile.ID, "CV-" + data.Profile.PublicID + "-" + data.Profile.FullName, nil
	}
}

func (s *DocumentService) cvData(ctx context.Context, publicID string) (CVData, error) {
	p, err := s.d.Sources.CrewProfile(ctx, publicID)
	if errors.Is(err, repository.ErrNotFound) {
		return CVData{}, apperr.NotFound("Crew")
	}
	if err != nil {
		return CVData{}, err
	}
	data := CVData{Profile: p}
	if data.Documents, err = s.d.Crews.ListDocuments(ctx, repository.TravelDocuments, publicID); err != nil {
		return CVData{}, err
	}
	if data.Certificates, err = s.d.Crews.ListDocuments(ctx, repository.Certificates, publicID); err != nil {
		return CVData{}, err
	}
	if data.SeaService, err = s.d.Crews.SeaService(ctx, publicID); err != nil {
		return CVData{}, err
	}
	if p.PhotoFileID != nil && s.d.Files != nil {
		photo, mime, err := s.d.Files.Content(ctx, *p.PhotoFileID)
		if err != nil {
			logctx.From(ctx).WarnContext(ctx, "crew photo not embedded", "error", err)
		} else {
			data.Photo, data.PhotoMime = photo, mime
		}
	}
	return data, nil
}

func (s *DocumentService) format(ctx context.Context) (docFormat, error) {
	loc, _, err := LoadDisplaySettings(ctx, s.d.Settings)
	if err != nil {
		return docFormat{}, err
	}
	general, err := s.d.Settings.Values(ctx, "general")
	if err != nil {
		return docFormat{}, err
	}
	pattern, _ := general["date_format"].(string)
	if pattern == "" {
		pattern = datetime.DefaultDisplayFormat
	}
	return docFormat{pattern: pattern, today: datetime.Today(s.d.Now(), loc)}, nil
}

// Preview is a merged template as one HTML page for the browser.
type Preview struct {
	HTML        string   `json:"html"`
	UnknownTags []string `json:"unknown_tags"`
}

// Preview renders req as a standalone HTML page.
func (s *DocumentService) Preview(ctx context.Context, req DocumentRequest) (Preview, error) {
	r, err := s.Render(ctx, req)
	if err != nil {
		return Preview{}, err
	}
	page := htmlPage(r.Template, `<div class="doc-header">`+r.Header+`</div><main>`+r.Body+`</main><div class="doc-footer">`+r.Footer+`</div>`)
	return Preview{HTML: page, UnknownTags: nonNil(r.UnknownTags)}, nil
}

// Generate renders req to a compressed PDF, stores it privately and records it in the history.
func (s *DocumentService) Generate(ctx context.Context, req DocumentRequest) (model.GeneratedDocument, error) {
	r, err := s.Render(ctx, req)
	if err != nil {
		return model.GeneratedDocument{}, err
	}
	var pdf []byte
	err = s.d.Pool.Do(ctx, func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, generationTimeout)
		defer cancel()
		pdf, err = s.d.Renderer.ConvertHTML(ctx, s.htmlRequest(r))
		if err != nil {
			return err
		}
		pdf = s.compress(ctx, pdf)
		return nil
	})
	if errors.Is(err, gotenberg.ErrUnavailable) {
		return model.GeneratedDocument{}, apperr.New(503, "pdf_unavailable", "The PDF service is unavailable; try again shortly.").Wrap(err)
	}
	if err != nil {
		return model.GeneratedDocument{}, err
	}
	file, err := s.d.Storage.StoreGenerated(ctx, "generated_documents", safeDocumentName(r.Filename)+".pdf", "application/pdf", "pdf", pdf, nil)
	if err != nil {
		return model.GeneratedDocument{}, err
	}
	doc, err := s.record(ctx, model.GeneratedDocument{Kind: r.Template.Kind, TemplateID: &r.Template.ID, RelatedType: r.RelatedType, RelatedID: r.RelatedID, FileID: file.ID})
	if err != nil {
		return model.GeneratedDocument{}, err
	}
	if err := s.d.Storage.Attach(ctx, file.ID, "generated_documents", doc.ID); err != nil {
		return model.GeneratedDocument{}, err
	}
	return doc, nil
}

func (s *DocumentService) record(ctx context.Context, g model.GeneratedDocument) (model.GeneratedDocument, error) {
	for range publicid.MaxAttempts {
		pid, err := s.newID()
		if err != nil {
			return g, err
		}
		g.PublicID = pid
		out, err := s.d.History.Create(ctx, g)
		if postgres.IsUniqueViolation(err, "generated_documents_public_id_key") {
			continue
		}
		return out, err
	}
	return g, fmt.Errorf("record generated document: no unique public id after %d attempts", publicid.MaxAttempts)
}

func (s *DocumentService) compress(ctx context.Context, pdf []byte) []byte {
	preset := "ebook"
	if uploads, err := s.d.Settings.Values(ctx, "uploads"); err == nil {
		if p, ok := uploads["pdf_preset"].(string); ok && p != "" {
			preset = p
		}
	}
	out, err := s.d.Compressor.Compress(ctx, pdf, preset)
	if err != nil {
		logctx.From(ctx).WarnContext(ctx, "generated pdf not compressed", "error", err)
		return pdf
	}
	return out
}

func (s *DocumentService) htmlRequest(r RenderedDocument) gotenberg.HTMLRequest {
	margins := [4]float64{0.6, 0.6, 0.6, 0.6}
	req := gotenberg.HTMLRequest{Index: htmlPage(r.Template, "<main>"+r.Body+"</main>"), Paper: r.Template.PaperSize, Landscape: r.Template.Orientation == "landscape"}
	if strings.TrimSpace(r.Header) != "" {
		req.Header = marginPage(r.Header)
		margins[0] = 1.2
	}
	if strings.TrimSpace(r.Footer) != "" {
		req.Footer = marginPage(r.Footer)
		margins[2] = 0.9
	}
	req.MarginsIn = margins
	return req
}

// History lists generated documents, optionally for one record addressed by its public ID.
func (s *DocumentService) History(ctx context.Context, relatedType, relatedPublicID string) ([]model.GeneratedDocument, error) {
	if relatedType == "" {
		return s.d.History.List(ctx, "", nil, 200)
	}
	id, err := s.resolveRelated(ctx, relatedType, strings.ToUpper(strings.TrimSpace(relatedPublicID)))
	if err != nil {
		return nil, err
	}
	return s.d.History.List(ctx, relatedType, id, 200)
}

func (s *DocumentService) resolveRelated(ctx context.Context, relatedType, publicID string) (*uuid.UUID, error) {
	if publicID == "" {
		return nil, nil
	}
	if !publicid.Valid(publicID) {
		return nil, apperr.Validation(map[string]string{"related_id": "A valid record ID is required."})
	}
	switch relatedType {
	case RelatedContract:
		id, err := s.d.Sources.ContractID(ctx, publicID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperr.NotFound("Contract")
		}
		return &id, err
	case RelatedQuotation:
		q, err := s.d.Quotations.Get(ctx, publicID)
		return &q.ID, err
	case RelatedCrew:
		p, err := s.d.Sources.CrewProfile(ctx, publicID)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperr.NotFound("Crew")
		}
		return &p.ID, err
	}
	return nil, apperr.Validation(map[string]string{"related_type": "The related type must be contract, quotation or crew."})
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeDocumentName(name string) string {
	name = strings.Trim(unsafeName.ReplaceAllString(name, "-"), "-.")
	if len(name) > 120 {
		name = name[:120]
	}
	if name == "" {
		return "document"
	}
	return name
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

const printCSS = `*{box-sizing:border-box}
body{font-family:"Helvetica Neue",Arial,"Liberation Sans",sans-serif;font-size:10.5pt;line-height:1.45;color:#111827;margin:0}
h1,h2,h3,h4{color:#0B2545;margin:0.8em 0 0.4em;line-height:1.25}h1{font-size:18pt}h2{font-size:14pt}h3{font-size:12pt}h4{font-size:11pt}
p{margin:0 0 0.6em}table{width:100%;border-collapse:collapse;margin:0.4em 0 0.9em;page-break-inside:auto}
tr{page-break-inside:avoid}th,td{border:1px solid #CBD5E1;padding:4px 6px;text-align:left;vertical-align:top;font-size:9.5pt}
thead th{background:#E2E8F0;color:#0B2545}tfoot th{background:#F1F5F9}td.empty{color:#64748B;text-align:center}
table.no-border td,table.no-border th{border:none;padding:2px 6px 2px 0}
.page-break{page-break-after:always;break-after:page}.text-center{text-align:center}.text-right{text-align:right}
.crew-photo{width:35mm;height:45mm;object-fit:cover;border:1px solid #CBD5E1}.company-logo{max-height:18mm;max-width:60mm}
.signature{margin-top:2.5em}.small{font-size:8.5pt}.muted{color:#64748B}
.doc-header,.doc-footer{color:#334155;font-size:9pt}.doc-header{border-bottom:1px solid #CBD5E1;margin-bottom:12px}
.doc-footer{border-top:1px solid #CBD5E1;margin-top:16px}`

func htmlPage(t model.DocumentTemplate, inner string) string {
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>` + t.Name + `</title><style>` + printCSS +
		`</style></head><body>` + inner + `</body></html>`
}

func marginPage(inner string) string {
	return `<!doctype html><html><head><meta charset="utf-8"><style>body{font-family:"Helvetica Neue",Arial,sans-serif;font-size:8pt;color:#334155;margin:0;padding:0 0.6in;width:100%;box-sizing:border-box}
img{max-height:14mm}p{margin:0}table{width:100%;border-collapse:collapse}td{vertical-align:middle;padding:0}
.text-right{text-align:right}.text-center{text-align:center}.small{font-size:7.5pt}.muted{color:#64748B}</style></head><body>` + inner + `</body></html>`
}

var emailDocumentKinds = map[string]string{
	RelatedContract:  model.DocumentSEA,
	RelatedQuotation: model.DocumentQuotation,
	RelatedCrew:      model.DocumentCV,
}

// EmailContext returns the internal ID and plain-text template variables of a contract, quotation or crew record.
func (s *DocumentService) EmailContext(ctx context.Context, relatedType, publicID string) (uuid.UUID, map[string]string, error) {
	kind, ok := emailDocumentKinds[relatedType]
	if !ok {
		return uuid.Nil, nil, apperr.Validation(map[string]string{"related_type": "The related type must be contract, quotation or crew."})
	}
	f, err := s.format(ctx)
	if err != nil {
		return uuid.Nil, nil, err
	}
	v, _, id, _, err := s.vars(ctx, DocumentRequest{Kind: kind, RelatedID: publicID}, f)
	if err != nil {
		return uuid.Nil, nil, err
	}
	company, err := s.d.Settings.Values(ctx, "company")
	if err != nil {
		return uuid.Nil, nil, err
	}
	out := make(map[string]string, len(v.Text)+4)
	for k, val := range CompanyVars(ctx, company, s.d.Files, f).Text {
		out[k] = val
	}
	for k, val := range v.Text {
		out[k] = val
	}
	return id, out, nil
}
