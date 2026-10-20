package admin

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/resource"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
)

// FileService is what the files controller needs.
type FileService interface {
	MaxUploadBytes(ctx context.Context) (int64, error)
	Upload(ctx context.Context, in service.UploadInput) (model.File, error)
	URL(ctx context.Context, id uuid.UUID) (service.PresignedURL, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// FileHandler serves uploads and presigned downloads.
type FileHandler struct {
	files FileService
}

// NewFileHandler returns a FileHandler.
func NewFileHandler(files FileService) *FileHandler {
	return &FileHandler{files: files}
}

const multipartOverhead = 64 << 10

// Upload handles POST /files with multipart fields file, context, owner_type and owner_id.
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	limit, err := h.files.MaxUploadBytes(r.Context())
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	in, err := readUpload(w, r, limit)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	f, err := h.files.Upload(r.Context(), in)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusCreated, resource.NewFile(f))
}

// URL handles GET /files/{id}/url.
func (h *FileHandler) URL(w http.ResponseWriter, r *http.Request) {
	id, ok := fileID(w, r)
	if !ok {
		return
	}
	link, err := h.files.URL(r.Context(), id)
	if err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.JSON(w, http.StatusOK, link)
}

// Delete handles DELETE /files/{id}.
func (h *FileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := fileID(w, r)
	if !ok {
		return
	}
	if err := h.files.Delete(r.Context(), id); err != nil {
		resource.Error(w, r, err)
		return
	}
	resource.NoContent(w)
}

func fileID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		resource.Error(w, r, apperr.NotFound("File"))
		return uuid.Nil, false
	}
	return id, true
}

func readUpload(w http.ResponseWriter, r *http.Request, limit int64) (service.UploadInput, error) {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "multipart/form-data" {
		return service.UploadInput{}, apperr.New(http.StatusUnsupportedMediaType, "unsupported_media_type", "The request must be multipart/form-data.")
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit+multipartOverhead)
	mr, err := r.MultipartReader()
	if err != nil {
		return service.UploadInput{}, apperr.ErrBadRequest.Wrap(err)
	}
	var in service.UploadInput
	fields := map[string]string{}
	gotFile := false
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return service.UploadInput{}, uploadReadError(err, limit)
		}
		if err := readPart(part, limit, &in, fields, &gotFile); err != nil {
			return service.UploadInput{}, err
		}
	}
	if !gotFile {
		return service.UploadInput{}, apperr.Validation(map[string]string{"file": "A file is required."})
	}
	return completeUpload(in, fields)
}

func readPart(part *multipart.Part, limit int64, in *service.UploadInput, fields map[string]string, gotFile *bool) error {
	defer func() { _ = part.Close() }()
	if part.FormName() == "file" && part.FileName() != "" {
		if *gotFile {
			return apperr.Validation(map[string]string{"file": "Only one file may be uploaded per request."})
		}
		data, err := io.ReadAll(io.LimitReader(part, limit+1))
		if err != nil {
			return uploadReadError(err, limit)
		}
		if int64(len(data)) > limit {
			return filecheck.TooLarge("The file is larger than the allowed size.")
		}
		in.Filename, in.Data, *gotFile = part.FileName(), data, true
		return nil
	}
	value, err := io.ReadAll(io.LimitReader(part, 256))
	if err != nil {
		return uploadReadError(err, limit)
	}
	fields[part.FormName()] = strings.TrimSpace(string(value))
	return nil
}

func completeUpload(in service.UploadInput, fields map[string]string) (service.UploadInput, error) {
	c, ok := filecheck.ParseContext(fields["context"])
	if !ok {
		return in, apperr.Validation(map[string]string{"context": "The upload context is invalid."})
	}
	in.Context = c
	in.OwnerType = fields["owner_type"]
	if raw := fields["owner_id"]; raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return in, apperr.Validation(map[string]string{"owner_id": "The owner ID is invalid."})
		}
		in.OwnerID = &id
	}
	if (in.OwnerType == "") != (in.OwnerID == nil) {
		return in, apperr.Validation(map[string]string{"owner_type": "owner_type and owner_id must be sent together."})
	}
	return in, nil
}

func uploadReadError(err error, limit int64) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return filecheck.TooLarge("The file may not be larger than " + formatMB(limit) + ".")
	}
	return apperr.ErrBadRequest.Wrap(err)
}

func formatMB(n int64) string {
	return strconv.FormatInt(n>>20, 10) + " MB"
}
