// Package pdf validates PDFs for active content and compresses them with Ghostscript.
package pdf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// ErrUnreadable is returned when a PDF cannot be parsed.
var ErrUnreadable = errors.New("pdf cannot be parsed")

// ErrActiveContent is returned when a PDF contains JavaScript, launch actions or embedded files.
var ErrActiveContent = errors.New("pdf contains active content or embedded files")

// Presets accepted by Ghostscript -dPDFSETTINGS.
var Presets = []string{"screen", "ebook", "printer", "prepress"}

var forbiddenKeys = []string{"JS", "JavaScript", "AA", "EmbeddedFiles", "EmbeddedFile", "EF", "RichMedia", "XFA"}

var forbiddenActions = []string{"JavaScript", "Launch", "ImportData", "SubmitForm", "GoToE", "RichMediaExecute"}

// Processor inspects and compresses PDFs.
type Processor struct {
	gsPath  string
	timeout time.Duration
}

// NewProcessor returns a Processor using the gs binary at gsPath.
func NewProcessor(gsPath string, timeout time.Duration) *Processor {
	return &Processor{gsPath: gsPath, timeout: timeout}
}

// Inspect parses data and rejects JavaScript, launch actions and embedded files.
func (p *Processor) Inspect(ctx context.Context, data []byte) error {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return ErrUnreadable
	}
	conf := model.NewStatelessConfiguration()
	conf.ValidationMode = model.ValidationRelaxed
	pctx, err := api.ReadContext(ctx, bytes.NewReader(data), conf)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	if err := pctx.EnsurePageCount(); err != nil || pctx.PageCount < 1 {
		return ErrUnreadable
	}
	for _, entry := range pctx.Table {
		if entry == nil || entry.Free || entry.Object == nil {
			continue
		}
		if hasActiveContent(entry.Object, 0) {
			return ErrActiveContent
		}
	}
	return nil
}

func hasActiveContent(obj types.Object, depth int) bool {
	if depth > 64 {
		return true
	}
	switch o := obj.(type) {
	case types.Dict:
		return dictActive(o, depth)
	case types.StreamDict:
		return dictActive(o.Dict, depth)
	case types.ObjectStreamDict:
		return dictActive(o.Dict, depth)
	case types.XRefStreamDict:
		return dictActive(o.Dict, depth)
	case types.Array:
		for _, v := range o {
			if hasActiveContent(v, depth+1) {
				return true
			}
		}
	}
	return false
}

func dictActive(d types.Dict, depth int) bool {
	for key, v := range d {
		if slices.Contains(forbiddenKeys, key) {
			return true
		}
		if key == "S" || key == "Type" || key == "Subtype" {
			if n, ok := v.(types.Name); ok && (slices.Contains(forbiddenActions, string(n)) || n == "EmbeddedFile") {
				return true
			}
		}
		if hasActiveContent(v, depth+1) {
			return true
		}
	}
	return false
}

// Compress rewrites data with Ghostscript using preset; the original is returned when the result is not smaller.
func (p *Processor) Compress(ctx context.Context, data []byte, preset string) ([]byte, error) {
	if !slices.Contains(Presets, preset) {
		preset = "ebook"
	}
	dir, err := os.MkdirTemp("", "pdf-")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()
	in, out := filepath.Join(dir, "in.pdf"), filepath.Join(dir, "out.pdf")
	if err := os.WriteFile(in, data, 0o600); err != nil {
		return nil, fmt.Errorf("write temp pdf: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, p.gsPath,
		"-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.6", "-dPDFSETTINGS=/"+preset,
		"-dNOPAUSE", "-dBATCH", "-dSAFER", "-dQUIET", "-dDetectDuplicateImages=true",
		"-sOutputFile="+out, in)
	if msg, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ghostscript: %w: %s", err, bytes.TrimSpace(msg))
	}
	compressed, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("read compressed pdf: %w", err)
	}
	if len(compressed) == 0 || len(compressed) >= len(data) {
		return data, nil
	}
	return compressed, nil
}
