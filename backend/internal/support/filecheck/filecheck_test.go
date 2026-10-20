package filecheck

import (
	"archive/zip"
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

func code(t *testing.T, err error) string {
	t.Helper()
	e, ok := apperr.As(err)
	require.True(t, ok, "expected apperr, got %v", err)
	return e.Code
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(1, 1, color.RGBA{R: 200, A: 255})
	var b bytes.Buffer
	require.NoError(t, jpeg.Encode(&b, img, nil))
	return b.Bytes()
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewGray(image.Rect(0, 0, 4, 4))))
	return b.Bytes()
}

func docxBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for name, body := range files {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return b.Bytes()
}

var (
	minimalPDF = []byte("%PDF-1.4\n1 0 obj << /Type /Catalog >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")
	oleHeader  = append([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, make([]byte, 600)...)
	exeBytes   = append([]byte("MZ\x90\x00\x03\x00\x00\x00\x04\x00\x00\x00\xff\xff"), make([]byte, 100)...)
)

func TestCheckName(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantExt string
		code    string
	}{
		{"simple pdf", "cv.pdf", "pdf", ""},
		{"uppercase extension", "PHOTO.JPG", "jpg", ""},
		{"dots in base name", "john.doe.cv.v2.pdf", "pdf", ""},
		{"double extension exe", "cv.pdf.exe", "", CodeInvalidName},
		{"double known extension", "photo.jpg.png", "", CodeInvalidName},
		{"hidden script inner", "invoice.js.pdf", "", CodeInvalidName},
		{"executable", "setup.exe", "", CodeInvalidType},
		{"null byte", "cv\x00.pdf", "", CodeInvalidName},
		{"path traversal", "../../etc/passwd.pdf", "", CodeInvalidName},
		{"windows path", `C:\\temp\\cv.pdf`, "", CodeInvalidName},
		{"too long", string(bytes.Repeat([]byte("a"), 252)) + ".pdf", "", CodeInvalidName},
		{"no extension", "README", "", CodeInvalidName},
		{"trailing dot", "cv.", "", CodeInvalidName},
		{"hidden file", ".htaccess", "", CodeInvalidName},
		{"control character", "cv\n.pdf", "", CodeInvalidName},
		{"empty", "", "", CodeInvalidName},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ext, err := CheckName(tc.in)
			if tc.code == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.wantExt, ext)
				return
			}
			assert.Equal(t, tc.code, code(t, err))
		})
	}
}

func TestAllowedExtensionsPerContextMatchPRDTable10_2(t *testing.T) {
	assert.ElementsMatch(t, []string{"jpg", "jpeg", "png", "webp"}, Allowed(ContextPhoto))
	assert.ElementsMatch(t, []string{"jpg", "jpeg", "png", "webp"}, Allowed(ContextCMSImage))
	assert.ElementsMatch(t, []string{"jpg", "jpeg", "png", "webp", "svg"}, Allowed(ContextLogo))
	assert.ElementsMatch(t, []string{"svg"}, Allowed(ContextFlag))
	assert.ElementsMatch(t, []string{"pdf", "jpg", "jpeg", "png", "webp"}, Allowed(ContextCrewDocument))
	assert.ElementsMatch(t, []string{"pdf", "jpg", "jpeg", "png", "webp", "doc", "docx"}, Allowed(ContextCrewCV))
	assert.ElementsMatch(t, []string{"pdf", "jpg", "jpeg", "png", "webp", "doc", "docx"}, Allowed(ContextApplicant))
	assert.ElementsMatch(t, []string{"csv"}, Allowed(ContextCSVImport))

	assert.Error(t, CheckExtension(ContextPhoto, "pdf"))
	assert.Error(t, CheckExtension(ContextCrewDocument, "docx"))
	assert.Error(t, CheckExtension(ContextPhoto, "svg"), "SVG only for logos and flags")
	assert.NoError(t, CheckExtension(ContextLogo, "svg"))
	assert.NoError(t, CheckExtension(ContextApplicant, "doc"))

	_, ok := ParseContext("unknown")
	assert.False(t, ok)
	assert.True(t, IsPublic(ContextCMSImage))
	assert.False(t, IsPublic(ContextCrewDocument))
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		ext  string
		kind Kind
		code string
	}{
		{"jpeg", jpegBytes(t), "jpg", KindImage, ""},
		{"jpeg as jpeg ext", jpegBytes(t), "jpeg", KindImage, ""},
		{"png", pngBytes(t), "png", KindImage, ""},
		{"pdf", minimalPDF, "pdf", KindPDF, ""},
		{"doc", oleHeader, "doc", KindDOC, ""},
		{"docx", docxBytes(t, map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": "<w:document/>"}), "docx", KindDOCX, ""},
		{"svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), "svg", KindSVG, ""},
		{"csv", []byte("name,rank\nJohn,AB\n"), "csv", KindCSV, ""},
		{"renamed executable as pdf", exeBytes, "pdf", "", CodeInvalidType},
		{"png renamed to jpg", pngBytes(t), "jpg", "", CodeInvalidType},
		{"pdf renamed to docx", minimalPDF, "docx", "", CodeInvalidType},
		{"html renamed to svg", []byte("<html><script>alert(1)</script></html>"), "svg", "", CodeInvalidType},
		{"binary renamed to csv", exeBytes, "csv", "", CodeInvalidType},
		{"zip without word renamed to doc", docxBytes(t, map[string]string{"a.txt": "x"}), "doc", "", CodeInvalidType},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, err := Detect(tc.data, tc.ext)
			if tc.code == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.kind, kind)
				return
			}
			assert.Equal(t, tc.code, code(t, err))
		})
	}
}

func TestCheckImageRejectsCorruptImages(t *testing.T) {
	good := jpegBytes(t)
	require.NoError(t, CheckImage(good))
	require.NoError(t, CheckImage(pngBytes(t)))

	truncated := good[:len(good)/2]
	assert.Equal(t, CodeRejected, code(t, CheckImage(truncated)))
	assert.Equal(t, CodeRejected, code(t, CheckImage([]byte("\xff\xd8\xff\xe0 not really a jpeg"))))
}

func TestCheckDOCX(t *testing.T) {
	valid := docxBytes(t, map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": "<w:document/>"})
	require.NoError(t, CheckDOCX(valid))

	withMacros := docxBytes(t, map[string]string{"word/document.xml": "<w:document/>", "word/vbaProject.bin": "binary"})
	assert.Equal(t, CodeRejected, code(t, CheckDOCX(withMacros)))

	macroContentType := docxBytes(t, map[string]string{
		"[Content_Types].xml": `<Types><Override ContentType="application/vnd.ms-word.document.macroEnabled.main+xml"/></Types>`,
		"word/document.xml":   "<w:document/>",
	})
	assert.Equal(t, CodeRejected, code(t, CheckDOCX(macroContentType)))

	notWord := docxBytes(t, map[string]string{"xl/workbook.xml": "<workbook/>"})
	assert.Equal(t, CodeInvalidType, code(t, CheckDOCX(notWord)))
	assert.Equal(t, CodeInvalidType, code(t, CheckDOCX([]byte("PK garbage"))))
}

func TestCheckCSV(t *testing.T) {
	require.NoError(t, CheckCSV([]byte("\xEF\xBB\xBFname,rank\nJosé,AB\n")))
	assert.Error(t, CheckCSV([]byte{0xff, 0xfe, 0x00, 0x41}))
	assert.Error(t, CheckCSV([]byte("a,b\x00c")))
}

func TestSanitizeSVG(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		contains   []string
		notContain []string
	}{
		{
			name:       "removes script elements",
			in:         `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><circle r="5"/></svg>`,
			contains:   []string{`<circle r="5">`},
			notContain: []string{"script", "alert"},
		},
		{
			name:       "removes event handlers",
			in:         `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect onclick="x()" width="2"/></svg>`,
			contains:   []string{`width="2"`},
			notContain: []string{"onload", "onclick", "alert"},
		},
		{
			name:       "removes foreignObject subtree",
			in:         `<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><body><iframe src="https://evil"/></body></foreignObject><g/></svg>`,
			contains:   []string{"<g>"},
			notContain: []string{"foreignObject", "iframe", "evil"},
		},
		{
			name:       "drops external references and keeps fragments",
			in:         `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><use xlink:href="#icon"/><use href="https://evil/x.svg#a"/><image href="javascript:alert(1)"/></svg>`,
			contains:   []string{`xlink:href="#icon"`, `xmlns:xlink="http://www.w3.org/1999/xlink"`},
			notContain: []string{"evil", "javascript"},
		},
		{
			name:       "drops unsafe css",
			in:         `<svg xmlns="http://www.w3.org/2000/svg"><style>@import url(https://evil/a.css);</style><rect style="fill:url(https://evil/p)"/><rect style="fill:url(#grad)"/></svg>`,
			contains:   []string{`style="fill:url(#grad)"`},
			notContain: []string{"evil", "@import"},
		},
		{
			name:       "removes animation that can rewrite links",
			in:         `<svg xmlns="http://www.w3.org/2000/svg"><a><set attributeName="href" to="javascript:alert(1)"/><text>hi</text></a></svg>`,
			contains:   []string{"<text>hi</text>"},
			notContain: []string{"set", "javascript"},
		},
		{
			name:     "keeps safe styles",
			in:       `<svg xmlns="http://www.w3.org/2000/svg"><style>.a{fill:#0B2545}</style></svg>`,
			contains: []string{"<style>.a{fill:#0B2545}</style>"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := SanitizeSVG([]byte(tc.in))
			require.NoError(t, err)
			for _, c := range tc.contains {
				assert.Contains(t, string(out), c)
			}
			for _, c := range tc.notContain {
				assert.NotContains(t, string(out), c)
			}
		})
	}
}

func TestSanitizeSVGRejectsNonSVG(t *testing.T) {
	_, err := SanitizeSVG([]byte(`<html><body/></html>`))
	assert.Equal(t, CodeInvalidType, code(t, err))
	_, err = SanitizeSVG([]byte(`<svg><g></svg>`))
	assert.Equal(t, CodeRejected, code(t, err))
	_, err = SanitizeSVG([]byte(`<!DOCTYPE svg [<!ENTITY x "boom">]><svg>&x;</svg>`))
	assert.Equal(t, CodeRejected, code(t, err))
}
