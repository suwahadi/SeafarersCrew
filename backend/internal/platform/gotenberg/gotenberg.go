// Package gotenberg converts office documents and HTML to PDF through a Gotenberg 8 server.
package gotenberg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ErrUnavailable is returned when Gotenberg cannot be reached or fails.
var ErrUnavailable = errors.New("gotenberg unavailable")

const maxResponse = 200 << 20

// Client calls the Gotenberg HTTP API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a Client for baseURL with a per-request timeout.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

// ConvertOffice converts a DOC or DOCX file to PDF with LibreOffice.
func (c *Client) ConvertOffice(ctx context.Context, filename string, data []byte) ([]byte, error) {
	return c.post(ctx, "/forms/libreoffice/convert", nil, []part{{name: "files", filename: filename, data: data}})
}

// Paper sizes in inches.
var paperSizes = map[string][2]float64{
	"a4":     {8.27, 11.7},
	"letter": {8.5, 11},
}

// HTMLRequest describes an HTML to PDF conversion.
type HTMLRequest struct {
	Index     string
	Header    string
	Footer    string
	Paper     string
	Landscape bool
	MarginsIn [4]float64
}

// ConvertHTML renders HTML to PDF with Chromium.
func (c *Client) ConvertHTML(ctx context.Context, req HTMLRequest) ([]byte, error) {
	size, ok := paperSizes[req.Paper]
	if !ok {
		size = paperSizes["a4"]
	}
	fields := map[string]string{
		"paperWidth":        formatFloat(size[0]),
		"paperHeight":       formatFloat(size[1]),
		"marginTop":         formatFloat(req.MarginsIn[0]),
		"marginRight":       formatFloat(req.MarginsIn[1]),
		"marginBottom":      formatFloat(req.MarginsIn[2]),
		"marginLeft":        formatFloat(req.MarginsIn[3]),
		"landscape":         strconv.FormatBool(req.Landscape),
		"printBackground":   "true",
		"preferCssPageSize": "false",
	}
	parts := []part{{name: "files", filename: "index.html", data: []byte(req.Index)}}
	if req.Header != "" {
		parts = append(parts, part{name: "files", filename: "header.html", data: []byte(req.Header)})
	}
	if req.Footer != "" {
		parts = append(parts, part{name: "files", filename: "footer.html", data: []byte(req.Footer)})
	}
	return c.post(ctx, "/forms/chromium/convert/html", fields, parts)
}

// Ping checks the Gotenberg health endpoint.
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("build gotenberg health request: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: health status %d", ErrUnavailable, res.StatusCode)
	}
	return nil
}

type part struct {
	name     string
	filename string
	data     []byte
}

func (c *Client) post(ctx context.Context, path string, fields map[string]string, parts []part) ([]byte, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, p := range parts {
		w, err := mw.CreateFormFile(p.name, p.filename)
		if err != nil {
			return nil, fmt.Errorf("build gotenberg form: %w", err)
		}
		if _, err := w.Write(p.data); err != nil {
			return nil, fmt.Errorf("build gotenberg form: %w", err)
		}
	}
	for k, v := range fields {
		if v == "" {
			continue
		}
		if err := mw.WriteField(k, v); err != nil {
			return nil, fmt.Errorf("build gotenberg form: %w", err)
		}
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("build gotenberg form: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &body)
	if err != nil {
		return nil, fmt.Errorf("build gotenberg request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = res.Body.Close() }()
	out, err := io.ReadAll(io.LimitReader(res.Body, maxResponse))
	if err != nil {
		return nil, fmt.Errorf("%w: read response: %w", ErrUnavailable, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d: %s", ErrUnavailable, res.StatusCode, bytes.TrimSpace(out[:min(len(out), 300)]))
	}
	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		return nil, fmt.Errorf("%w: response is not a PDF", ErrUnavailable)
	}
	return out, nil
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
