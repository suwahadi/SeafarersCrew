package pdf

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildPDF assembles a PDF with a correct xref table from object bodies (object 1 is the catalog).
func buildPDF(objects ...string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return b.Bytes()
}

func stream(content string) string {
	return fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)
}

const (
	pages = "<< /Type /Pages /Kids [3 0 R] /Count 1 >>"
	page  = "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>"
	font  = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"
)

func cleanPDF() []byte {
	return buildPDF("<< /Type /Catalog /Pages 2 0 R >>", pages, page, stream("BT /F1 24 Tf 72 720 Td (Crew CV) Tj ET"), font)
}

func newProcessor() *Processor {
	return NewProcessor("gs", 30*time.Second)
}

func TestInspectAcceptsCleanPDF(t *testing.T) {
	require.NoError(t, newProcessor().Inspect(context.Background(), cleanPDF()))
}

func TestInspectRejectsActiveContent(t *testing.T) {
	body := stream("BT /F1 24 Tf 72 720 Td (x) Tj ET")
	cases := []struct {
		name    string
		catalog string
		extra   []string
	}{
		{"open action javascript", "<< /Type /Catalog /Pages 2 0 R /OpenAction << /S /JavaScript /JS (app.alert(1)) >> >>", nil},
		{"document level javascript", "<< /Type /Catalog /Pages 2 0 R /Names << /JavaScript 6 0 R >> >>", []string{"<< /Names [(a) 7 0 R] >>", "<< /S /JavaScript /JS (x) >>"}},
		{"embedded files", "<< /Type /Catalog /Pages 2 0 R /Names << /EmbeddedFiles 6 0 R >> >>", []string{"<< /Names [(a.exe) 7 0 R] >>", "<< /Type /Filespec /F (a.exe) >>"}},
		{"launch action", "<< /Type /Catalog /Pages 2 0 R /OpenAction << /S /Launch /F (cmd.exe) >> >>", nil},
		{"additional actions", "<< /Type /Catalog /Pages 2 0 R /AA << /WC << /S /Named /N /Print >> >> >>", nil},
		{"hex escaped javascript name", "<< /Type /Catalog /Pages 2 0 R /OpenAction << /S /J#61vaScript /J#53 (x) >> >>", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			objs := append([]string{tc.catalog, pages, page, body, font}, tc.extra...)
			err := newProcessor().Inspect(context.Background(), buildPDF(objs...))
			assert.ErrorIs(t, err, ErrActiveContent)
		})
	}
}

func TestInspectRejectsUnparseableFiles(t *testing.T) {
	p := newProcessor()
	assert.ErrorIs(t, p.Inspect(context.Background(), []byte("MZ\x90\x00 not a pdf")), ErrUnreadable)
	assert.ErrorIs(t, p.Inspect(context.Background(), []byte("%PDF-1.4\ngarbage only")), ErrUnreadable)
}

func requireGhostscript(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gs"); err != nil {
		t.Skip("ghostscript not installed")
	}
}

func TestCompressKeepsOriginalWhenNotSmaller(t *testing.T) {
	requireGhostscript(t)
	in := cleanPDF()
	out, err := newProcessor().Compress(context.Background(), in, "ebook")
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestCompressShrinksUncompressedContent(t *testing.T) {
	requireGhostscript(t)
	content := strings.Repeat("BT /F1 12 Tf 72 720 Td (Seafarer employment agreement clause text) Tj ET\n", 3000)
	in := buildPDF("<< /Type /Catalog /Pages 2 0 R >>", pages, page, stream(content), font)
	out, err := newProcessor().Compress(context.Background(), in, "ebook")
	require.NoError(t, err)
	assert.Less(t, len(out), len(in))
	assert.True(t, bytes.HasPrefix(out, []byte("%PDF-")))
	require.NoError(t, newProcessor().Inspect(context.Background(), out), "compressed output stays a valid PDF")
}

func TestCompressHonoursTimeout(t *testing.T) {
	requireGhostscript(t)
	p := NewProcessor("gs", time.Nanosecond)
	_, err := p.Compress(context.Background(), cleanPDF(), "ebook")
	assert.Error(t, err)
}
