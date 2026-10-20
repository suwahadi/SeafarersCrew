package admin

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
)

type stubFiles struct {
	got service.UploadInput
}

func (s *stubFiles) MaxUploadBytes(context.Context) (int64, error) { return 1 << 20, nil }

func (s *stubFiles) Upload(_ context.Context, in service.UploadInput) (model.File, error) {
	s.got = in
	if strings.HasSuffix(in.Filename, ".exe") {
		return model.File{}, filecheck.InvalidName("Files with a double extension are not accepted.")
	}
	return model.File{ID: uuid.New(), StorageKey: "crews/2026/10/abcdefghijkl.webp", OriginalName: in.Filename, MimeType: "image/webp", Status: model.FileReady}, nil
}

func (s *stubFiles) URL(context.Context, uuid.UUID) (service.PresignedURL, error) {
	return service.PresignedURL{URL: "https://s3.test/x", ExpiresAt: time.Now()}, nil
}

func (s *stubFiles) Delete(_ context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return apperr.NotFound("File")
	}
	return nil
}

func multipartBody(t *testing.T, fields map[string]string, filename string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	for k, v := range fields {
		require.NoError(t, mw.WriteField(k, v))
	}
	if filename != "" {
		w, err := mw.CreateFormFile("file", filename)
		require.NoError(t, err)
		_, _ = w.Write(data)
	}
	require.NoError(t, mw.Close())
	return &b, mw.FormDataContentType()
}

func TestUploadHandlerParsesMultipart(t *testing.T) {
	stub := &stubFiles{}
	owner := uuid.New()
	body, ct := multipartBody(t, map[string]string{"context": "photo", "owner_type": "crews", "owner_id": owner.String()}, "me.jpg", []byte("jpegdata"))
	req := httptest.NewRequest(http.MethodPost, "/files", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	NewFileHandler(stub).Upload(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, filecheck.ContextPhoto, stub.got.Context)
	assert.Equal(t, "me.jpg", stub.got.Filename)
	assert.Equal(t, "jpegdata", string(stub.got.Data))
	assert.Equal(t, &owner, stub.got.OwnerID)
	assert.NotContains(t, rec.Body.String(), "abcdefghijkl", "storage keys are never exposed")
}

func TestUploadHandlerErrors(t *testing.T) {
	cases := []struct {
		name   string
		fields map[string]string
		file   string
		data   []byte
		ct     string
		status int
		code   string
	}{
		{"not multipart", nil, "", nil, "application/json", http.StatusUnsupportedMediaType, "unsupported_media_type"},
		{"missing file", map[string]string{"context": "photo"}, "", nil, "", http.StatusUnprocessableEntity, "validation_failed"},
		{"unknown context", map[string]string{"context": "evil"}, "a.jpg", []byte("x"), "", http.StatusUnprocessableEntity, "validation_failed"},
		{"owner id without type", map[string]string{"context": "photo", "owner_id": uuid.NewString()}, "a.jpg", []byte("x"), "", http.StatusUnprocessableEntity, "validation_failed"},
		{"too large", map[string]string{"context": "photo"}, "a.jpg", make([]byte, 2<<20), "", http.StatusRequestEntityTooLarge, filecheck.CodeTooLarge},
		{"service rejection", map[string]string{"context": "applicant"}, "cv.pdf.exe", []byte("MZ"), "", http.StatusUnprocessableEntity, filecheck.CodeInvalidName},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, ct := multipartBody(t, tc.fields, tc.file, tc.data)
			if tc.ct != "" {
				ct = tc.ct
			}
			req := httptest.NewRequest(http.MethodPost, "/files", body)
			req.Header.Set("Content-Type", ct)
			rec := httptest.NewRecorder()
			NewFileHandler(&stubFiles{}).Upload(rec, req)
			assert.Equal(t, tc.status, rec.Code, rec.Body.String())
			assert.Contains(t, rec.Body.String(), `"code":"`+tc.code+`"`)
		})
	}
}

func TestFileURLAndDelete(t *testing.T) {
	r := chi.NewRouter()
	h := NewFileHandler(&stubFiles{})
	r.Get("/files/{id}/url", h.URL)
	r.Delete("/files/{id}", h.Delete)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/"+uuid.NewString()+"/url", nil))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "https://s3.test/x")

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/files/not-a-uuid/url", nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/files/"+uuid.Nil.String(), nil))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/files/"+uuid.NewString(), nil))
	assert.Equal(t, http.StatusNoContent, rec.Code)
}
