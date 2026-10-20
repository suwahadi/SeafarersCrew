package gotenberg

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertOfficeSendsFileAndReturnsPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/forms/libreoffice/convert", r.URL.Path)
		require.NoError(t, r.ParseMultipartForm(1<<20))
		f, hdr, err := r.FormFile("files")
		require.NoError(t, err)
		body, _ := io.ReadAll(f)
		assert.Equal(t, "cv.docx", hdr.Filename)
		assert.Equal(t, "DOCX", string(body))
		_, _ = w.Write([]byte("%PDF-1.7 converted"))
	}))
	defer srv.Close()

	out, err := NewClient(srv.URL, time.Second).ConvertOffice(context.Background(), "cv.docx", []byte("DOCX"))
	require.NoError(t, err)
	assert.Equal(t, "%PDF-1.7 converted", string(out))
}

func TestConvertHTMLSendsPaperAndHeaderFooter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/forms/chromium/convert/html", r.URL.Path)
		require.NoError(t, r.ParseMultipartForm(1<<20))
		assert.Equal(t, "8.5", r.FormValue("paperWidth"))
		assert.Equal(t, "11", r.FormValue("paperHeight"))
		assert.Equal(t, "true", r.FormValue("landscape"))
		assert.Equal(t, "0.5", r.FormValue("marginTop"))
		names := []string{}
		for _, fh := range r.MultipartForm.File["files"] {
			names = append(names, fh.Filename)
		}
		assert.ElementsMatch(t, []string{"index.html", "header.html", "footer.html"}, names)
		_, _ = w.Write([]byte("%PDF-1.7"))
	}))
	defer srv.Close()

	_, err := NewClient(srv.URL, time.Second).ConvertHTML(context.Background(), HTMLRequest{
		Index: "<p>x</p>", Header: "<p>h</p>", Footer: "<p>f</p>", Paper: "letter", Landscape: true, MarginsIn: [4]float64{0.5, 0.5, 0.5, 0.5},
	})
	require.NoError(t, err)
}

func TestConvertReportsUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "LibreOffice crashed", http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, err := NewClient(srv.URL, time.Second).ConvertOffice(context.Background(), "a.doc", []byte("x"))
	assert.ErrorIs(t, err, ErrUnavailable)

	notPDF := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) }))
	defer notPDF.Close()
	_, err = NewClient(notPDF.URL, time.Second).ConvertOffice(context.Background(), "a.doc", []byte("x"))
	assert.ErrorIs(t, err, ErrUnavailable)

	_, err = NewClient("http://127.0.0.1:1", time.Second).ConvertOffice(context.Background(), "a.doc", []byte("x"))
	assert.ErrorIs(t, err, ErrUnavailable)
}
