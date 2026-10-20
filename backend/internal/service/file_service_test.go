package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/job"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/gotenberg"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/imaging"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/pdf"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
)

type memFiles struct {
	rows    map[uuid.UUID]model.File
	deleted map[uuid.UUID]bool
}

func (m *memFiles) Create(_ context.Context, f model.File) (model.File, error) {
	f.ID = uuid.New()
	m.rows[f.ID] = f
	return f, nil
}

func (m *memFiles) Get(_ context.Context, id uuid.UUID) (model.File, error) {
	f, ok := m.rows[id]
	if !ok || m.deleted[id] {
		return model.File{}, repository.ErrNotFound
	}
	return f, nil
}

func (m *memFiles) SoftDelete(_ context.Context, id uuid.UUID) error {
	m.deleted[id] = true
	return nil
}

func (m *memFiles) AttachOwner(_ context.Context, id uuid.UUID, ownerType string, ownerID uuid.UUID) error {
	f, ok := m.rows[id]
	if !ok {
		return repository.ErrNotFound
	}
	f.OwnerType, f.OwnerID = ownerType, &ownerID
	m.rows[id] = f
	return nil
}

type memStorage struct {
	objects map[string][]byte
	types   map[string]string
}

func (m *memStorage) Put(_ context.Context, key string, data []byte, contentType string) error {
	m.objects[key], m.types[key] = data, contentType
	return nil
}

func (m *memStorage) Delete(_ context.Context, keys ...string) error {
	for _, k := range keys {
		delete(m.objects, k)
	}
	return nil
}

func (m *memStorage) PresignGet(_ context.Context, key, filename string, inline bool) (string, time.Duration, error) {
	return fmt.Sprintf("https://s3.test/%s?filename=%s&inline=%t", key, filename, inline), 5 * time.Minute, nil
}

type fakeImages struct{ calls []imaging.Options }

func (f *fakeImages) ToWebP(_ []byte, opts imaging.Options) (imaging.Result, error) {
	f.calls = append(f.calls, opts)
	res := imaging.Result{Data: []byte("RIFF....WEBPVP8 main"), Width: 1200, Height: 600}
	for _, w := range opts.Widths {
		if w < res.Width {
			res.Variants = append(res.Variants, imaging.Variant{Width: w, Height: w / 2, Data: []byte(fmt.Sprintf("RIFF-%d", w))})
		}
	}
	return res, nil
}

type fakePDF struct {
	compressed  []byte
	compressErr error
}

func (f *fakePDF) Inspect(_ context.Context, data []byte) error {
	if bytes.Contains(data, []byte("/JavaScript")) {
		return pdf.ErrActiveContent
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return pdf.ErrUnreadable
	}
	return nil
}

func (f *fakePDF) Compress(_ context.Context, data []byte, _ string) ([]byte, error) {
	if f.compressErr != nil {
		return nil, f.compressErr
	}
	if f.compressed != nil && len(f.compressed) < len(data) {
		return f.compressed, nil
	}
	return data, nil
}

type fakeOffice struct {
	err   error
	calls int
}

func (f *fakeOffice) ConvertOffice(context.Context, string, []byte) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return []byte("%PDF-1.7 converted from office"), nil
}

type staticSettings map[string]map[string]any

func (s staticSettings) Values(_ context.Context, group string) (map[string]any, error) {
	return s[group], nil
}

type fileFixture struct {
	svc     *FileService
	files   *memFiles
	storage *memStorage
	images  *fakeImages
	pdfs    *fakePDF
	office  *fakeOffice
}

func newFileFixture(t *testing.T) fileFixture {
	t.Helper()
	f := fileFixture{
		files:   &memFiles{rows: map[uuid.UUID]model.File{}, deleted: map[uuid.UUID]bool{}},
		storage: &memStorage{objects: map[string][]byte{}, types: map[string]string{}},
		images:  &fakeImages{},
		pdfs:    &fakePDF{},
		office:  &fakeOffice{},
	}
	settings := staticSettings{"uploads": {"max_file_size_mb": 1, "webp_quality": 80, "max_image_dimension": 2000, "pdf_preset": "ebook"}}
	clock := func() time.Time { return time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC) }
	f.svc = NewFileService(f.files, f.storage, f.images, f.pdfs, f.office, settings, job.NewPool(2), clock)
	return f
}

func testJPEG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 16, 16)), nil))
	return b.Bytes()
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 16, 16))))
	return b.Bytes()
}

func testDOCX(t *testing.T, macros bool) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	files := []string{"[Content_Types].xml", "word/document.xml"}
	if macros {
		files = append(files, "word/vbaProject.bin")
	}
	for _, name := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, _ = w.Write([]byte("<x/>"))
	}
	require.NoError(t, zw.Close())
	return b.Bytes()
}

var (
	cleanPDFBytes = []byte("%PDF-1.7\n1 0 obj << /Type /Catalog >> endobj\n%%EOF")
	jsPDFBytes    = []byte("%PDF-1.7\n1 0 obj << /OpenAction << /S /JavaScript /JS (x) >> >> endobj\n%%EOF")
	exeBytes      = append([]byte("MZ\x90\x00\x03\x00\x00\x00\x04\x00\x00\x00\xff\xff"), make([]byte, 64)...)
	storedKey     = regexp.MustCompile(`^(public/)?[a-z_]+/2026/10/[a-z0-9]{12}\.(webp|pdf|svg|csv)$`)
)

func requireFileErr(t *testing.T, err error, code string) {
	t.Helper()
	requireAppErr(t, err, code)
}

func TestUploadAllowedInputPerContextAndTransformation(t *testing.T) {
	cases := []struct {
		name     string
		ctx      filecheck.Context
		filename string
		data     func(t *testing.T) []byte
		mime     string
		ext      string
		public   bool
	}{
		{"photo jpg to webp", filecheck.ContextPhoto, "me.jpg", testJPEG, "image/webp", "webp", false},
		{"photo png to webp", filecheck.ContextPhoto, "me.png", testPNG, "image/webp", "webp", false},
		{"cms image to public webp", filecheck.ContextCMSImage, "hero.jpeg", testJPEG, "image/webp", "webp", true},
		{"logo svg sanitized", filecheck.ContextLogo, "logo.svg", func(*testing.T) []byte {
			return []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="x()"><circle r="1"/></svg>`)
		}, "image/svg+xml", "svg", true},
		{"flag svg", filecheck.ContextFlag, "id.svg", func(*testing.T) []byte { return []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`) }, "image/svg+xml", "svg", true},
		{"crew document pdf", filecheck.ContextCrewDocument, "passport.pdf", func(*testing.T) []byte { return cleanPDFBytes }, "application/pdf", "pdf", false},
		{"crew document image", filecheck.ContextCrewDocument, "passport.jpg", testJPEG, "image/webp", "webp", false},
		{"crew cv docx to pdf", filecheck.ContextCrewCV, "cv.docx", func(t *testing.T) []byte { return testDOCX(t, false) }, "application/pdf", "pdf", false},
		{"applicant doc to pdf", filecheck.ContextApplicant, "cv.doc", func(*testing.T) []byte {
			return append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, make([]byte, 600)...)
		}, "application/pdf", "pdf", false},
		{"csv import", filecheck.ContextCSVImport, "crew.csv", func(*testing.T) []byte { return []byte("full_name\nJohn\n") }, "text/csv", "csv", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFileFixture(t)
			data := tc.data(t)
			got, err := f.svc.Upload(context.Background(), UploadInput{Context: tc.ctx, Filename: tc.filename, Data: data})
			require.NoError(t, err)
			assert.Equal(t, tc.mime, got.MimeType)
			assert.Equal(t, tc.public, got.IsPublic)
			assert.Equal(t, tc.filename, got.OriginalName, "original name is kept only as metadata")
			assert.Regexp(t, storedKey, got.StorageKey)
			assert.True(t, strings.HasSuffix(got.StorageKey, "."+tc.ext))
			assert.Regexp(t, `^[a-z0-9]{12}\.`+tc.ext+`$`, path.Base(got.StorageKey), "stored under a random name, never the original")
			assert.Equal(t, int64(len(data)), got.SizeOriginal)
			assert.Equal(t, int64(len(f.storage.objects[got.StorageKey])), got.SizeStored)
			assert.Equal(t, tc.mime, f.storage.types[got.StorageKey])
			assert.Len(t, got.SHA256, 64)
			assert.Equal(t, model.FileReady, got.Status)
		})
	}
}

func TestUploadJPGIsStoredAsWebPWithRandomTwelveCharName(t *testing.T) {
	f := newFileFixture(t)
	owner := uuid.New()
	got, err := f.svc.Upload(context.Background(), UploadInput{Context: filecheck.ContextPhoto, Filename: "IMG_2041.JPG", Data: testJPEG(t), OwnerType: "crews", OwnerID: &owner})
	require.NoError(t, err)
	assert.Regexp(t, `^crews/2026/10/[a-z0-9]{12}\.webp$`, got.StorageKey)
	require.Len(t, f.images.calls, 1)
	assert.Equal(t, imaging.Options{MaxDimension: 2000, Quality: 80}, f.images.calls[0])
	assert.Equal(t, &owner, got.OwnerID)

	other, err := f.svc.Upload(context.Background(), UploadInput{Context: filecheck.ContextPhoto, Filename: "IMG_2041.JPG", Data: testJPEG(t)})
	require.NoError(t, err)
	assert.NotEqual(t, got.StorageKey, other.StorageKey)
}

func TestUploadCMSImageStoresResponsiveVariants(t *testing.T) {
	f := newFileFixture(t)
	got, err := f.svc.Upload(context.Background(), UploadInput{Context: filecheck.ContextCMSImage, Filename: "hero.jpg", Data: testJPEG(t)})
	require.NoError(t, err)
	require.Len(t, got.Variants, 2)
	for _, v := range got.Variants {
		assert.True(t, strings.HasPrefix(v.StorageKey, "public/cms_image/2026/10/"))
		assert.Contains(t, f.storage.objects, v.StorageKey)
	}
	assert.Equal(t, []int{480, 960}, []int{got.Variants[0].Width, got.Variants[1].Width})
}

func TestUploadRejections(t *testing.T) {
	cases := []struct {
		name     string
		ctx      filecheck.Context
		filename string
		data     func(t *testing.T) []byte
		code     string
	}{
		{"renamed executable as pdf", filecheck.ContextApplicant, "cv.pdf", func(*testing.T) []byte { return exeBytes }, filecheck.CodeInvalidType},
		{"double extension", filecheck.ContextApplicant, "cv.pdf.exe", func(*testing.T) []byte { return exeBytes }, filecheck.CodeInvalidName},
		{"docx with macros", filecheck.ContextCrewCV, "cv.docx", func(t *testing.T) []byte { return testDOCX(t, true) }, filecheck.CodeRejected},
		{"pdf with javascript", filecheck.ContextCrewDocument, "passport.pdf", func(*testing.T) []byte { return jsPDFBytes }, filecheck.CodeRejected},
		{"docx not allowed for documents", filecheck.ContextCrewDocument, "passport.docx", func(t *testing.T) []byte { return testDOCX(t, false) }, filecheck.CodeInvalidType},
		{"svg not allowed for photos", filecheck.ContextPhoto, "me.svg", func(*testing.T) []byte { return []byte(`<svg/>`) }, filecheck.CodeInvalidType},
		{"corrupt jpeg", filecheck.ContextPhoto, "me.jpg", func(t *testing.T) []byte { d := testJPEG(t); return d[:len(d)/2] }, filecheck.CodeRejected},
		{"too large", filecheck.ContextCrewDocument, "big.pdf", func(*testing.T) []byte { return append(cleanPDFBytes, make([]byte, 2<<20)...) }, filecheck.CodeTooLarge},
		{"empty", filecheck.ContextCrewDocument, "empty.pdf", func(*testing.T) []byte { return nil }, filecheck.CodeRejected},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFileFixture(t)
			_, err := f.svc.Upload(context.Background(), UploadInput{Context: tc.ctx, Filename: tc.filename, Data: tc.data(t)})
			requireFileErr(t, err, tc.code)
			assert.Empty(t, f.storage.objects, "nothing is stored for rejected files")
			assert.Empty(t, f.files.rows)
		})
	}
}

func TestUploadRejectsInvalidOwnerType(t *testing.T) {
	f := newFileFixture(t)
	_, err := f.svc.Upload(context.Background(), UploadInput{Context: filecheck.ContextPhoto, Filename: "a.jpg", Data: testJPEG(t), OwnerType: "../crews"})
	requireFileErr(t, err, "validation_failed")
}

func TestUploadPDFUsesCompressedOutputOnlyWhenSmaller(t *testing.T) {
	f := newFileFixture(t)
	f.pdfs.compressed = []byte("%PDF-x")
	got, err := f.svc.Upload(context.Background(), UploadInput{Context: filecheck.ContextCrewDocument, Filename: "a.pdf", Data: cleanPDFBytes})
	require.NoError(t, err)
	assert.Equal(t, int64(6), got.SizeStored)

	f = newFileFixture(t)
	f.pdfs.compressErr = errors.New("gs crashed")
	got, err = f.svc.Upload(context.Background(), UploadInput{Context: filecheck.ContextCrewDocument, Filename: "a.pdf", Data: cleanPDFBytes})
	require.NoError(t, err, "compression failure keeps the validated original")
	assert.Equal(t, int64(len(cleanPDFBytes)), got.SizeStored)
}

func TestUploadOfficeConversionUnavailable(t *testing.T) {
	f := newFileFixture(t)
	f.office.err = fmt.Errorf("%w: connection refused", gotenberg.ErrUnavailable)
	_, err := f.svc.Upload(context.Background(), UploadInput{Context: filecheck.ContextCrewCV, Filename: "cv.docx", Data: testDOCX(t, false)})
	e := requireAppErr(t, err, "conversion_unavailable")
	assert.Equal(t, http.StatusServiceUnavailable, e.Status)
}

func TestURLDeleteAndAttach(t *testing.T) {
	f := newFileFixture(t)
	ctx := context.Background()
	got, err := f.svc.Upload(ctx, UploadInput{Context: filecheck.ContextCrewCV, Filename: "John CV.docx", Data: testDOCX(t, false)})
	require.NoError(t, err)

	link, err := f.svc.URL(ctx, got.ID)
	require.NoError(t, err)
	assert.Contains(t, link.URL, "filename=John CV.pdf")
	assert.Contains(t, link.URL, "inline=false")
	assert.Equal(t, time.Date(2026, 10, 20, 9, 5, 0, 0, time.UTC), link.ExpiresAt)

	owner := uuid.New()
	require.NoError(t, f.svc.Attach(ctx, got.ID, "crews", owner))
	assert.Equal(t, "crews", f.files.rows[got.ID].OwnerType)

	require.NoError(t, f.svc.Delete(ctx, got.ID))
	assert.Empty(t, f.storage.objects)
	_, err = f.svc.URL(ctx, got.ID)
	requireFileErr(t, err, "not_found")
	requireFileErr(t, f.svc.Attach(ctx, uuid.New(), "crews", owner), "not_found")
}

func TestImageURLIsInline(t *testing.T) {
	f := newFileFixture(t)
	got, err := f.svc.Upload(context.Background(), UploadInput{Context: filecheck.ContextPhoto, Filename: "me.png", Data: testPNG(t)})
	require.NoError(t, err)
	link, err := f.svc.URL(context.Background(), got.ID)
	require.NoError(t, err)
	assert.Contains(t, link.URL, "inline=true")
	assert.Contains(t, link.URL, "filename=me.webp")
}
