package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/gotenberg"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/imaging"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/pdf"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/logctx"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// FileStore persists file rows.
type FileStore interface {
	Create(ctx context.Context, f model.File) (model.File, error)
	Get(ctx context.Context, id uuid.UUID) (model.File, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	AttachOwner(ctx context.Context, id uuid.UUID, ownerType string, ownerID uuid.UUID) error
}

// ObjectStorage stores file bytes privately.
type ObjectStorage interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Delete(ctx context.Context, keys ...string) error
	PresignGet(ctx context.Context, key, filename string, inline bool) (string, time.Duration, error)
}

// ImageConverter converts images to WebP.
type ImageConverter interface {
	ToWebP(data []byte, opts imaging.Options) (imaging.Result, error)
}

// PDFProcessor validates and compresses PDFs.
type PDFProcessor interface {
	Inspect(ctx context.Context, data []byte) error
	Compress(ctx context.Context, data []byte, preset string) ([]byte, error)
}

// OfficeConverter converts DOC and DOCX to PDF.
type OfficeConverter interface {
	ConvertOffice(ctx context.Context, filename string, data []byte) ([]byte, error)
}

// Bounded runs CPU-heavy work with limited concurrency.
type Bounded interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

// FileService runs the upload pipeline of PRD Section 10.
type FileService struct {
	files    FileStore
	storage  ObjectStorage
	images   ImageConverter
	pdfs     PDFProcessor
	office   OfficeConverter
	settings SettingsReader
	pool     Bounded
	now      func() time.Time
}

// NewFileService builds a FileService.
func NewFileService(files FileStore, storage ObjectStorage, images ImageConverter, pdfs PDFProcessor, office OfficeConverter, settings SettingsReader, pool Bounded, now func() time.Time) *FileService {
	return &FileService{files: files, storage: storage, images: images, pdfs: pdfs, office: office, settings: settings, pool: pool, now: now}
}

// UploadInput is one file to process.
type UploadInput struct {
	Context   filecheck.Context
	Filename  string
	Data      []byte
	OwnerType string
	OwnerID   *uuid.UUID
}

type uploadSettings struct {
	maxBytes     int64
	webpQuality  int
	maxDimension int
	pdfPreset    string
}

var ownerTypePattern = regexp.MustCompile(`^[a-z][a-z_]{1,39}$`)

// MaxUploadBytes returns the per-file size limit from Settings.
func (s *FileService) MaxUploadBytes(ctx context.Context) (int64, error) {
	cfg, err := s.uploadSettings(ctx)
	if err != nil {
		return 0, err
	}
	return cfg.maxBytes, nil
}

func (s *FileService) uploadSettings(ctx context.Context) (uploadSettings, error) {
	v, err := s.settings.Values(ctx, "uploads")
	if err != nil {
		return uploadSettings{}, err
	}
	preset, _ := v["pdf_preset"].(string)
	return uploadSettings{
		maxBytes:     int64(uploadInt(v["max_file_size_mb"], 10)) << 20,
		webpQuality:  uploadInt(v["webp_quality"], 80),
		maxDimension: uploadInt(v["max_image_dimension"], 2000),
		pdfPreset:    preset,
	}, nil
}

func uploadInt(v any, fallback int) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	}
	return fallback
}

type processed struct {
	data     []byte
	mime     string
	ext      string
	variants []imaging.Variant
	width    int
	height   int
}

// Upload validates, transforms, renames and stores a file and records it.
func (s *FileService) Upload(ctx context.Context, in UploadInput) (model.File, error) {
	cfg, err := s.uploadSettings(ctx)
	if err != nil {
		return model.File{}, err
	}
	if int64(len(in.Data)) > cfg.maxBytes {
		return model.File{}, filecheck.TooLarge(fmt.Sprintf("The file may not be larger than %d MB.", cfg.maxBytes>>20))
	}
	if len(in.Data) == 0 {
		return model.File{}, filecheck.Rejected("The file is empty.")
	}
	if in.OwnerType != "" && !ownerTypePattern.MatchString(in.OwnerType) {
		return model.File{}, apperr.Validation(map[string]string{"owner_type": "The owner type is invalid."})
	}
	ext, err := filecheck.CheckName(in.Filename)
	if err != nil {
		return model.File{}, err
	}
	if err := filecheck.CheckExtension(in.Context, ext); err != nil {
		return model.File{}, err
	}
	kind, err := filecheck.Detect(in.Data, ext)
	if err != nil {
		return model.File{}, err
	}
	var out processed
	err = s.pool.Do(ctx, func(ctx context.Context) error {
		var perr error
		out, perr = s.process(ctx, in, kind, cfg)
		return perr
	})
	if err != nil {
		return model.File{}, err
	}
	return s.store(ctx, in, out)
}

func (s *FileService) process(ctx context.Context, in UploadInput, kind filecheck.Kind, cfg uploadSettings) (processed, error) {
	switch kind {
	case filecheck.KindImage:
		return s.processImage(in, cfg)
	case filecheck.KindPDF:
		data, err := s.processPDF(ctx, in.Data, cfg.pdfPreset)
		return processed{data: data, mime: "application/pdf", ext: "pdf"}, err
	case filecheck.KindDOC, filecheck.KindDOCX:
		return s.processOffice(ctx, in, kind, cfg)
	case filecheck.KindSVG:
		data, err := filecheck.SanitizeSVG(in.Data)
		return processed{data: data, mime: "image/svg+xml", ext: "svg"}, err
	default:
		if err := filecheck.CheckCSV(in.Data); err != nil {
			return processed{}, err
		}
		return processed{data: in.Data, mime: "text/csv", ext: "csv"}, nil
	}
}

func (s *FileService) processImage(in UploadInput, cfg uploadSettings) (processed, error) {
	if err := filecheck.CheckImage(in.Data); err != nil {
		return processed{}, err
	}
	opts := imaging.Options{MaxDimension: cfg.maxDimension, Quality: cfg.webpQuality}
	if in.Context == filecheck.ContextCMSImage {
		opts.Widths = imaging.ResponsiveWidths
	}
	res, err := s.images.ToWebP(in.Data, opts)
	if err != nil {
		return processed{}, filecheck.Rejected("The image could not be processed.").Wrap(err)
	}
	return processed{data: res.Data, mime: "image/webp", ext: "webp", variants: res.Variants, width: res.Width, height: res.Height}, nil
}

func (s *FileService) processPDF(ctx context.Context, data []byte, preset string) ([]byte, error) {
	if err := s.pdfs.Inspect(ctx, data); err != nil {
		if errors.Is(err, pdf.ErrActiveContent) {
			return nil, filecheck.Rejected("PDF files with JavaScript, actions or embedded files are not accepted.")
		}
		return nil, filecheck.Rejected("The PDF file could not be read.").Wrap(err)
	}
	compressed, err := s.pdfs.Compress(ctx, data, preset)
	if err != nil {
		logctx.From(ctx).WarnContext(ctx, "pdf compression skipped", "error", err)
		return data, nil
	}
	return compressed, nil
}

func (s *FileService) processOffice(ctx context.Context, in UploadInput, kind filecheck.Kind, cfg uploadSettings) (processed, error) {
	if kind == filecheck.KindDOCX {
		if err := filecheck.CheckDOCX(in.Data); err != nil {
			return processed{}, err
		}
	}
	converted, err := s.office.ConvertOffice(ctx, "document."+string(kind), in.Data)
	if err != nil {
		if errors.Is(err, gotenberg.ErrUnavailable) {
			return processed{}, apperr.New(http.StatusServiceUnavailable, "conversion_unavailable", "The document could not be converted right now. Please try again.").Wrap(err)
		}
		return processed{}, err
	}
	data, err := s.processPDF(ctx, converted, cfg.pdfPreset)
	return processed{data: data, mime: "application/pdf", ext: "pdf"}, err
}

func (s *FileService) store(ctx context.Context, in UploadInput, out processed) (model.File, error) {
	name, err := token.FileName()
	if err != nil {
		return model.File{}, err
	}
	entity := in.OwnerType
	if entity == "" {
		entity = string(in.Context)
	}
	public := filecheck.IsPublic(in.Context)
	now := s.now().UTC()
	dir := fmt.Sprintf("%s/%04d/%02d", entity, now.Year(), int(now.Month()))
	if public {
		dir = "public/" + dir
	}
	key := dir + "/" + name + "." + out.ext
	written := []string{key}
	if err := s.storage.Put(ctx, key, out.data, out.mime); err != nil {
		return model.File{}, err
	}
	variants := make([]model.FileVariant, 0, len(out.variants))
	for _, v := range out.variants {
		vkey := fmt.Sprintf("%s/%s-%d.%s", dir, name, v.Width, out.ext)
		if err := s.storage.Put(ctx, vkey, v.Data, out.mime); err != nil {
			s.cleanup(ctx, written)
			return model.File{}, err
		}
		written = append(written, vkey)
		variants = append(variants, model.FileVariant{Width: v.Width, Height: v.Height, StorageKey: vkey, Size: int64(len(v.Data))})
	}
	sum := sha256.Sum256(out.data)
	f, err := s.files.Create(ctx, model.File{
		StorageKey:   key,
		OriginalName: in.Filename,
		MimeType:     out.mime,
		SizeOriginal: int64(len(in.Data)),
		SizeStored:   int64(len(out.data)),
		SHA256:       hex.EncodeToString(sum[:]),
		OwnerType:    in.OwnerType,
		OwnerID:      in.OwnerID,
		IsPublic:     public,
		Variants:     variants,
		Width:        out.width,
		Height:       out.height,
		Status:       model.FileReady,
	})
	if err != nil {
		s.cleanup(ctx, written)
		return model.File{}, err
	}
	return f, nil
}

// StoreGenerated saves server-generated content such as exports and PDFs without upload validation.
func (s *FileService) StoreGenerated(ctx context.Context, entity, filename, mime, ext string, data []byte, ownerID *uuid.UUID) (model.File, error) {
	if !ownerTypePattern.MatchString(entity) {
		return model.File{}, fmt.Errorf("store generated file: invalid entity %q", entity)
	}
	return s.store(ctx, UploadInput{Filename: filename, Data: data, OwnerType: entity, OwnerID: ownerID}, processed{data: data, mime: mime, ext: ext})
}

func (s *FileService) cleanup(ctx context.Context, keys []string) {
	if err := s.storage.Delete(context.WithoutCancel(ctx), keys...); err != nil {
		logctx.From(ctx).WarnContext(ctx, "orphaned objects not removed", "error", err)
	}
}

// PresignedURL is a short-lived download link.
type PresignedURL struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// URL returns a presigned download link for a file.
func (s *FileService) URL(ctx context.Context, id uuid.UUID) (PresignedURL, error) {
	f, err := s.get(ctx, id)
	if err != nil {
		return PresignedURL{}, err
	}
	inline := f.MimeType == "image/webp"
	u, lifetime, err := s.storage.PresignGet(ctx, f.StorageKey, DownloadName(f), inline)
	if err != nil {
		return PresignedURL{}, err
	}
	return PresignedURL{URL: u, ExpiresAt: s.now().UTC().Add(lifetime)}, nil
}

// Get returns a stored file.
func (s *FileService) Get(ctx context.Context, id uuid.UUID) (model.File, error) {
	return s.get(ctx, id)
}

// Attach links a file to the record that owns it.
func (s *FileService) Attach(ctx context.Context, id uuid.UUID, ownerType string, ownerID uuid.UUID) error {
	err := s.files.AttachOwner(ctx, id, ownerType, ownerID)
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound("File")
	}
	return err
}

// Delete removes the stored objects and soft-deletes the row.
func (s *FileService) Delete(ctx context.Context, id uuid.UUID) error {
	f, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	keys := []string{f.StorageKey}
	for _, v := range f.Variants {
		keys = append(keys, v.StorageKey)
	}
	if err := s.storage.Delete(ctx, keys...); err != nil {
		return err
	}
	return s.files.SoftDelete(ctx, id)
}

func (s *FileService) get(ctx context.Context, id uuid.UUID) (model.File, error) {
	f, err := s.files.Get(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return f, apperr.NotFound("File")
	}
	return f, err
}

// DownloadName is the original base name with the stored extension, e.g. cv.docx becomes cv.pdf.
func DownloadName(f model.File) string {
	base := strings.TrimSuffix(f.OriginalName, path.Ext(f.OriginalName))
	return base + path.Ext(f.StorageKey)
}
