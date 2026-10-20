package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/gotenberg"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type fakeSources struct {
	sea     repository.SEARecord
	profile repository.CrewProfile
}

func (f fakeSources) SEA(_ context.Context, pid string) (repository.SEARecord, error) {
	if pid != f.sea.ContractPublicID {
		return repository.SEARecord{}, repository.ErrNotFound
	}
	return f.sea, nil
}

func (f fakeSources) CrewProfile(_ context.Context, pid string) (repository.CrewProfile, error) {
	if pid != f.profile.PublicID {
		return repository.CrewProfile{}, repository.ErrNotFound
	}
	return f.profile, nil
}

func (f fakeSources) ContractID(_ context.Context, pid string) (uuid.UUID, error) {
	if pid != f.sea.ContractPublicID {
		return uuid.Nil, repository.ErrNotFound
	}
	return f.sea.ContractID, nil
}

type fakeTabs struct {
	docs, certs []DocumentView
	sea         CrewSeaService
}

func (f fakeTabs) ListDocuments(_ context.Context, k repository.DocumentKind, _ string) ([]DocumentView, error) {
	if k.Table == "certificates" {
		return f.certs, nil
	}
	return f.docs, nil
}

func (f fakeTabs) SeaService(context.Context, string) (CrewSeaService, error) { return f.sea, nil }

type fakeQuotationReader struct{ q model.Quotation }

func (f fakeQuotationReader) Get(_ context.Context, pid string) (model.Quotation, error) {
	if pid != f.q.PublicID {
		return model.Quotation{}, errors.New("not found")
	}
	return f.q, nil
}

type fakeRenderer struct {
	got gotenberg.HTMLRequest
	err error
}

func (f *fakeRenderer) ConvertHTML(_ context.Context, req gotenberg.HTMLRequest) ([]byte, error) {
	f.got = req
	if f.err != nil {
		return nil, f.err
	}
	return []byte("%PDF-1.7 rendered " + strings.Repeat("x", 100)), nil
}

type shrinkingCompressor struct{ preset string }

func (s *shrinkingCompressor) Compress(_ context.Context, data []byte, preset string) ([]byte, error) {
	s.preset = preset
	return data[:20], nil
}

type fakeGeneratedFiles struct {
	stored   map[uuid.UUID][]byte
	names    map[uuid.UUID]string
	attached map[uuid.UUID]uuid.UUID
}

func (f *fakeGeneratedFiles) StoreGenerated(_ context.Context, entity, filename, mime, ext string, data []byte, _ *uuid.UUID) (model.File, error) {
	id := uuid.New()
	f.stored[id] = data
	f.names[id] = entity + "|" + filename + "|" + mime + "|" + ext
	return model.File{ID: id}, nil
}

func (f *fakeGeneratedFiles) Attach(_ context.Context, id uuid.UUID, _ string, owner uuid.UUID) error {
	f.attached[id] = owner
	return nil
}

type memHistory struct{ rows []model.GeneratedDocument }

func (m *memHistory) Create(_ context.Context, g model.GeneratedDocument) (model.GeneratedDocument, error) {
	g.ID = uuid.New()
	m.rows = append(m.rows, g)
	return g, nil
}

func (m *memHistory) List(_ context.Context, relatedType string, relatedID *uuid.UUID, _ int) ([]model.GeneratedDocument, error) {
	var out []model.GeneratedDocument
	for _, r := range m.rows {
		if (relatedType == "" || r.RelatedType == relatedType) && (relatedID == nil || r.RelatedID == *relatedID) {
			out = append(out, r)
		}
	}
	return out, nil
}

type fakeFileReader map[uuid.UUID][]byte

func (f fakeFileReader) Content(_ context.Context, id uuid.UUID) ([]byte, string, error) {
	data, ok := f[id]
	if !ok {
		return nil, "", errors.New("missing")
	}
	return data, "image/webp", nil
}

type docFixture struct {
	svc      *DocumentService
	render   *fakeRenderer
	compress *shrinkingCompressor
	files    *fakeGeneratedFiles
	history  *memHistory
	sea      repository.SEARecord
	profile  repository.CrewProfile
	quote    model.Quotation
}

func newDocFixture(t *testing.T) docFixture {
	t.Helper()
	store := newMemTemplates()
	templates := NewDocumentTemplateService(store)
	for _, k := range []string{model.DocumentSEA, model.DocumentQuotation, model.DocumentCV} {
		tp := tpl(k, k, true)
		tp.HeaderHTML = "<p>{company_logo}{company_name}</p>"
		tp.FooterHTML = "<p>{today}</p>"
		switch k {
		case model.DocumentSEA:
			tp.BodyHTML = "<p>{crew_name} {rank} {basic_salary} {sign_on_date}</p>{allowance_table}<p>{total_monthly_pay} {oops}</p>"
		case model.DocumentQuotation:
			tp.BodyHTML = "{pricing_table}<p>{grand_total}</p>{terms}"
		default:
			tp.BodyHTML = "{photo}<p>{crew_name} {passport_no} {mobile_phone}</p>{document_table}{certificate_table}{experience_table}{sea_service_summary}"
		}
		_, err := templates.Save(context.Background(), tp)
		require.NoError(t, err)
	}
	photo, logo := uuid.New(), uuid.New()
	dob := date(1985, 5, 20)
	profile := repository.CrewProfile{ID: uuid.New(), PublicID: "CREWAAAA2", FullName: "Budi <b>Santoso</b>", DateOfBirth: &dob,
		PassportNo: "X1234567", MobilePhone: "+6281234567890", PhotoFileID: &photo}
	leave := dec("2.5")
	sea := repository.SEARecord{ContractID: uuid.New(), ContractPublicID: "KNTRAAAA2", Crew: profile, Rank: "Master",
		SignOnDate: date(2026, 11, 1), PlannedSignOff: date(2027, 5, 1), DurationMonths: 6, Currency: "USD", CurrencyDecimal: 2,
		BasicSalary: dec("8000"), LeaveDays: &leave, Allowances: []repository.AllowanceRow{
			{Name: "Overtime", Basis: "fixed_month", Value: dec("1200"), MonthlyAmount: dec("1200")},
			{Name: "Bonus", Basis: "percent_basic", Value: dec("10"), MonthlyAmount: dec("800")},
		}}
	quote := model.Quotation{ID: uuid.New(), PublicID: "QTEAAAAA2", ClientName: "Alpha", CurrencyCode: "IDR", CurrencyDecimal: 0,
		QuotationDate: date(2026, 10, 1), ValidUntil: date(2026, 10, 31), Subtotal: dec("4500000"), GrandTotal: dec("4995000"),
		Terms: "Line one\n\nLine <two>", Lines: []model.QuotationLine{lumpLine("Mobilisation", "4500000")}}
	f := docFixture{
		render: &fakeRenderer{}, compress: &shrinkingCompressor{}, history: &memHistory{},
		files: &fakeGeneratedFiles{stored: map[uuid.UUID][]byte{}, names: map[uuid.UUID]string{}, attached: map[uuid.UUID]uuid.UUID{}},
		sea:   sea, profile: profile, quote: quote,
	}
	settings := staticSettings{
		"general": {"timezone": "Asia/Jakarta", "date_format": "DD-MMM-YYYY"},
		"alerts":  {},
		"company": {"legal_name": "PT Samudra", "logo_light_file_id": logo.String()},
		"uploads": {"pdf_preset": "printer"},
	}
	f.svc = NewDocumentService(DocumentDeps{
		Templates: templates, Sources: fakeSources{sea: sea, profile: profile},
		Crews: fakeTabs{
			docs: []DocumentView{{CrewDocument: model.CrewDocument{TypeName: "Passport", Number: "X1234567"}}},
			sea:  CrewSeaService{ByRank: []RankSeaService{{RankName: "Master", Total: NewSpan(400)}}, Total: NewSpan(400)},
		},
		Quotations: fakeQuotationReader{q: quote}, Settings: settings,
		Files:    fakeFileReader{photo: []byte("photo-bytes"), logo: []byte("logo-bytes")},
		Renderer: f.render, Compressor: f.compress, Storage: f.files, History: f.history, Pool: job.NewPool(1),
		Now: func() time.Time { return time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC) },
	})
	return f
}

func TestPreviewMergesSEAData(t *testing.T) {
	f := newDocFixture(t)
	p, err := f.svc.Preview(context.Background(), DocumentRequest{Kind: model.DocumentSEA, RelatedID: "kntraaaa2"})
	require.NoError(t, err)
	assert.Contains(t, p.HTML, "Budi &lt;b&gt;Santoso&lt;/b&gt; Master USD 8,000.00 01-Nov-2026")
	assert.Contains(t, p.HTML, "<td>Bonus</td><td>Percentage of basic salary</td><td>10.00%</td><td>USD 800.00</td>")
	assert.Contains(t, p.HTML, "USD 10,000.00", "total monthly pay = basic + allowances")
	assert.Contains(t, p.HTML, "PT Samudra")
	assert.Contains(t, p.HTML, `src="data:image/webp;base64,`)
	assert.Contains(t, p.HTML, "20-Oct-2026")
	assert.Equal(t, []string{"oops"}, p.UnknownTags)
}

func TestPreviewQuotationUsesCurrencyDecimalsAndEscapesTerms(t *testing.T) {
	f := newDocFixture(t)
	p, err := f.svc.Preview(context.Background(), DocumentRequest{Kind: model.DocumentQuotation, RelatedID: "QTEAAAAA2"})
	require.NoError(t, err)
	assert.Contains(t, p.HTML, "IDR 4,995,000")
	assert.Contains(t, p.HTML, "<td>Mobilisation</td><td>IDR 4,500,000</td>")
	assert.Contains(t, p.HTML, "<p>Line one</p><p>Line &lt;two&gt;</p>")
	assert.Empty(t, p.UnknownTags)
}

func TestCVHidesRequestedSectionsAndNeverShowsBankData(t *testing.T) {
	f := newDocFixture(t)
	full, err := f.svc.Preview(context.Background(), DocumentRequest{Kind: model.DocumentCV, RelatedID: "CREWAAAA2"})
	require.NoError(t, err)
	assert.Contains(t, full.HTML, `class="crew-photo"`)
	assert.Contains(t, full.HTML, "X1234567")
	assert.Contains(t, full.HTML, "1 y 1 m 5 d")

	hidden, err := f.svc.Preview(context.Background(), DocumentRequest{Kind: model.DocumentCV, RelatedID: "CREWAAAA2",
		Hide: []string{CVHidePhoto, CVHidePersonal, CVHideDocuments}})
	require.NoError(t, err)
	assert.NotContains(t, hidden.HTML, `class="crew-photo"`)
	assert.NotContains(t, hidden.HTML, "X1234567")
	assert.NotContains(t, hidden.HTML, "+6281234567890")
	assert.Contains(t, hidden.HTML, "Budi")
	assert.NotContains(t, strings.ToLower(hidden.HTML), "bank")

	_, err = f.svc.Preview(context.Background(), DocumentRequest{Kind: model.DocumentCV, RelatedID: "CREWAAAA2", Hide: []string{"bank"}})
	requireAppErr(t, err, "validation_failed")
}

func TestPreviewValidatesRequest(t *testing.T) {
	f := newDocFixture(t)
	_, err := f.svc.Preview(context.Background(), DocumentRequest{Kind: "invoice", RelatedID: "KNTRAAAA2"})
	requireAppErr(t, err, "validation_failed")
	_, err = f.svc.Preview(context.Background(), DocumentRequest{Kind: model.DocumentSEA, RelatedID: "bad id"})
	requireAppErr(t, err, "validation_failed")
	_, err = f.svc.Preview(context.Background(), DocumentRequest{Kind: model.DocumentSEA, RelatedID: "ZZZZZZZZ2"})
	requireAppErr(t, err, "not_found")
}

func TestGenerateStoresCompressedPDFAndRecordsHistory(t *testing.T) {
	f := newDocFixture(t)
	ctx := context.Background()
	doc, err := f.svc.Generate(ctx, DocumentRequest{Kind: model.DocumentSEA, RelatedID: "KNTRAAAA2"})
	require.NoError(t, err)

	assert.Regexp(t, `^[2-9A-HJKMNP-Z]{9}$`, doc.PublicID)
	assert.Equal(t, RelatedContract, doc.RelatedType)
	assert.Equal(t, f.sea.ContractID, doc.RelatedID)
	assert.Equal(t, "printer", f.compress.preset)
	require.Len(t, f.files.stored, 1)
	assert.Len(t, f.files.stored[doc.FileID], 20, "the compressed PDF is stored")
	assert.Equal(t, "generated_documents|SEA-KNTRAAAA2-Budi-b-Santoso-b.pdf|application/pdf|pdf", f.files.names[doc.FileID])
	assert.Equal(t, doc.ID, f.files.attached[doc.FileID])

	assert.Equal(t, "a4", f.render.got.Paper)
	assert.Contains(t, f.render.got.Index, "Budi &lt;b&gt;Santoso")
	assert.Contains(t, f.render.got.Header, "PT Samudra")
	assert.Equal(t, 1.2, f.render.got.MarginsIn[0], "room for the header")

	history, err := f.svc.History(ctx, RelatedContract, "KNTRAAAA2")
	require.NoError(t, err)
	assert.Len(t, history, 1)
	_, err = f.svc.History(ctx, "invoice", "KNTRAAAA2")
	requireAppErr(t, err, "validation_failed")
}

func TestGenerateReportsUnavailablePDFService(t *testing.T) {
	f := newDocFixture(t)
	f.render.err = gotenberg.ErrUnavailable
	_, err := f.svc.Generate(context.Background(), DocumentRequest{Kind: model.DocumentQuotation, RelatedID: "QTEAAAAA2"})
	e := requireAppErr(t, err, "pdf_unavailable")
	assert.Equal(t, 503, e.Status)
	assert.Empty(t, f.history.rows)
}

func TestSafeDocumentName(t *testing.T) {
	assert.Equal(t, "CV-ABC-Jos-Mar-a", safeDocumentName("CV-ABC-José María"))
	assert.Equal(t, "document", safeDocumentName("///"))
	assert.Len(t, safeDocumentName(strings.Repeat("a", 300)), 120)
}
