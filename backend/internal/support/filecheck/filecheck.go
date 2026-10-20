// Package filecheck validates uploaded files by name, extension, magic bytes and structure (PRD 10.1, 10.2).
package filecheck

import (
	"archive/zip"
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/h2non/filetype"
	_ "golang.org/x/image/webp"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// Context names the place a file is uploaded from and decides its allowed types.
type Context string

// Upload contexts from PRD table 10.2.
const (
	ContextPhoto        Context = "photo"
	ContextCMSImage     Context = "cms_image"
	ContextLogo         Context = "logo"
	ContextFlag         Context = "flag"
	ContextCrewDocument Context = "crew_document"
	ContextCrewCV       Context = "crew_cv"
	ContextApplicant    Context = "applicant"
	ContextCSVImport    Context = "csv_import"
)

// Kind is the detected content family of a file.
type Kind string

// File kinds.
const (
	KindImage Kind = "image"
	KindPDF   Kind = "pdf"
	KindDOC   Kind = "doc"
	KindDOCX  Kind = "docx"
	KindSVG   Kind = "svg"
	KindCSV   Kind = "csv"
)

// MaxNameLength is the longest original file name accepted.
const MaxNameLength = 255

// MaxImagePixels guards against decompression bombs.
const MaxImagePixels = 50_000_000

var (
	imageExts    = []string{"jpg", "jpeg", "png", "webp"}
	documentExts = []string{"pdf", "jpg", "jpeg", "png", "webp"}
	officeExts   = []string{"pdf", "jpg", "jpeg", "png", "webp", "doc", "docx"}
)

var allowed = map[Context][]string{
	ContextPhoto:        imageExts,
	ContextCMSImage:     imageExts,
	ContextLogo:         append(slices.Clone(imageExts), "svg"),
	ContextFlag:         {"svg"},
	ContextCrewDocument: documentExts,
	ContextCrewCV:       officeExts,
	ContextApplicant:    officeExts,
	ContextCSVImport:    {"csv"},
}

var publicContexts = map[Context]bool{ContextCMSImage: true, ContextLogo: true, ContextFlag: true}

var knownExts = map[string]bool{
	"pdf": true, "jpg": true, "jpeg": true, "png": true, "webp": true, "gif": true, "svg": true,
	"doc": true, "docx": true, "docm": true, "xls": true, "xlsx": true, "csv": true, "txt": true, "zip": true, "rar": true,
}

var dangerousExts = map[string]bool{
	"exe": true, "com": true, "bat": true, "cmd": true, "scr": true, "pif": true, "msi": true, "dll": true,
	"js": true, "jse": true, "vbs": true, "vbe": true, "wsf": true, "ps1": true, "sh": true, "jar": true,
	"php": true, "phtml": true, "html": true, "htm": true, "hta": true, "lnk": true, "apk": true, "app": true,
}

// ParseContext returns the context named s.
func ParseContext(s string) (Context, bool) {
	c := Context(s)
	_, ok := allowed[c]
	return c, ok
}

// Allowed returns the extensions accepted in c.
func Allowed(c Context) []string { return slices.Clone(allowed[c]) }

// IsPublic reports whether files of c are published under the public prefix.
func IsPublic(c Context) bool { return publicContexts[c] }

// Error codes used by the file pipeline.
const (
	CodeInvalidName = "invalid_file_name"
	CodeInvalidType = "invalid_file_type"
	CodeTooLarge    = "file_too_large"
	CodeRejected    = "file_rejected"
)

// InvalidName builds an invalid_file_name error.
func InvalidName(msg string) *apperr.Error {
	return apperr.New(http.StatusUnprocessableEntity, CodeInvalidName, msg)
}

// InvalidType builds an invalid_file_type error.
func InvalidType(msg string) *apperr.Error {
	return apperr.New(http.StatusUnprocessableEntity, CodeInvalidType, msg)
}

// Rejected builds a file_rejected error.
func Rejected(msg string) *apperr.Error {
	return apperr.New(http.StatusUnprocessableEntity, CodeRejected, msg)
}

// TooLarge builds a file_too_large error.
func TooLarge(msg string) *apperr.Error {
	return apperr.New(http.StatusRequestEntityTooLarge, CodeTooLarge, msg)
}

// CheckName validates an original file name and returns its lowercase extension.
func CheckName(name string) (string, error) {
	if name == "" || len(name) > MaxNameLength {
		return "", InvalidName("The file name must be between 1 and 255 characters.")
	}
	if strings.ContainsAny(name, "/\\\x00") {
		return "", InvalidName("The file name may not contain path separators or null bytes.")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", InvalidName("The file name may not contain control characters.")
		}
	}
	if !utf8.ValidString(name) || path.Base(name) != name || strings.HasPrefix(name, ".") {
		return "", InvalidName("The file name is not valid.")
	}
	parts := strings.Split(strings.ToLower(name), ".")
	if len(parts) < 2 || parts[len(parts)-1] == "" {
		return "", InvalidName("The file name must have an extension.")
	}
	ext := parts[len(parts)-1]
	for _, inner := range parts[1 : len(parts)-1] {
		if dangerousExts[inner] || knownExts[inner] {
			return "", InvalidName("Files with a double extension are not accepted.")
		}
	}
	if dangerousExts[ext] {
		return "", InvalidType("This file type is not allowed.")
	}
	return ext, nil
}

// CheckExtension verifies ext is allowed in c.
func CheckExtension(c Context, ext string) error {
	if !slices.Contains(allowed[c], ext) {
		return InvalidType("This file type is not allowed here. Allowed: " + strings.Join(allowed[c], ", ") + ".")
	}
	return nil
}

var sniffed = map[string][]string{
	"jpg":  {"image/jpeg"},
	"jpeg": {"image/jpeg"},
	"png":  {"image/png"},
	"webp": {"image/webp"},
	"pdf":  {"application/pdf"},
	"doc":  {"application/octet-stream"},
	"docx": {"application/zip"},
	"svg":  {"text/xml; charset=utf-8", "text/plain; charset=utf-8"},
	"csv":  {"text/plain; charset=utf-8"},
}

var magic = map[string][]string{
	"jpg":  {"jpg"},
	"jpeg": {"jpg"},
	"png":  {"png"},
	"webp": {"webp"},
	"pdf":  {"pdf"},
	"docx": {"docx", "zip"},
}

// Detect checks the magic bytes against the claimed extension and returns the content kind.
func Detect(data []byte, ext string) (Kind, error) {
	head := data[:min(len(data), 512)]
	sniff := http.DetectContentType(head)
	if want, ok := sniffed[ext]; !ok || !slices.Contains(want, sniff) {
		return "", InvalidType("The file content does not match its extension.")
	}
	if kinds, ok := magic[ext]; ok {
		t, err := filetype.Match(data)
		if err != nil || !slices.Contains(kinds, t.Extension) {
			return "", InvalidType("The file content does not match its extension.")
		}
	}
	switch ext {
	case "jpg", "jpeg", "png", "webp":
		return KindImage, nil
	case "pdf":
		return KindPDF, nil
	case "doc":
		if !bytes.HasPrefix(data, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
			return "", InvalidType("The file content does not match its extension.")
		}
		return KindDOC, nil
	case "docx":
		return KindDOCX, nil
	case "svg":
		return KindSVG, nil
	default:
		return KindCSV, nil
	}
}

// CheckImage fully decodes an image so corrupt or truncated files fail.
func CheckImage(data []byte) error {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Rejected("The image could not be read.")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > MaxImagePixels {
		return Rejected("The image dimensions are not supported.")
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return Rejected("The image could not be read.")
	}
	return nil
}

// CheckDOCX verifies the ZIP container holds a Word document without macros.
func CheckDOCX(data []byte) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return InvalidType("The document is not a valid DOCX file.")
	}
	hasDocument := false
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		if strings.HasSuffix(name, "vbaproject.bin") || strings.HasSuffix(name, "vbadata.xml") {
			return Rejected("Documents containing macros are not accepted.")
		}
		if name == "[content_types].xml" && containsMacroContentType(f) {
			return Rejected("Documents containing macros are not accepted.")
		}
		if name == "word/document.xml" {
			hasDocument = true
		}
	}
	if !hasDocument {
		return InvalidType("The document is not a valid DOCX file.")
	}
	return nil
}

func containsMacroContentType(f *zip.File) bool {
	rc, err := f.Open()
	if err != nil {
		return true
	}
	defer func() { _ = rc.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(io.LimitReader(rc, 1<<20)); err != nil {
		return true
	}
	return bytes.Contains(bytes.ToLower(buf.Bytes()), []byte("macroenabled"))
}

// CheckCSV verifies the content is UTF-8 text without null bytes.
func CheckCSV(data []byte) error {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return InvalidType("The CSV file must be UTF-8 text.")
	}
	return nil
}
