package filecheck

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"slices"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Limits that keep a workbook upload from expanding into a zip bomb.
const (
	MaxXLSXEntries      = 1000
	MaxXLSXUncompressed = 50 << 20
)

const invalidWorkbook = "The file is not a valid Excel workbook."

// CheckXLSX verifies the ZIP container holds a macro-free Excel workbook within the size limits and that it opens.
func CheckXLSX(data []byte) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(zr.File) > MaxXLSXEntries {
		return Rejected(invalidWorkbook)
	}
	var total uint64
	hasTypes, hasWorkbook := false, false
	for _, f := range zr.File {
		total += f.UncompressedSize64
		name := strings.ToLower(f.Name)
		if strings.HasSuffix(name, "vbaproject.bin") {
			return Rejected("Workbooks containing macros are not accepted.")
		}
		hasTypes = hasTypes || name == "[content_types].xml"
		hasWorkbook = hasWorkbook || name == "xl/workbook.xml"
	}
	if total > MaxXLSXUncompressed || !hasTypes || !hasWorkbook {
		return Rejected(invalidWorkbook)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: MaxXLSXUncompressed, UnzipXMLSizeLimit: MaxXLSXUncompressed})
	if err != nil {
		return Rejected(invalidWorkbook)
	}
	if err := f.Close(); err != nil {
		return Rejected(invalidWorkbook)
	}
	return nil
}

var storedXLSXBlockedDirs = []string{"xl/embeddings/", "xl/activex/", "xl/externallinks/"}

// CheckStoredXLSX checks a workbook that is kept and downloaded: no macros, embedded objects, ActiveX, external links or DDE formulas.
func CheckStoredXLSX(data []byte) error {
	if err := CheckXLSX(data); err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Rejected(invalidWorkbook)
	}
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		switch {
		case slices.ContainsFunc(storedXLSXBlockedDirs, func(dir string) bool { return strings.HasPrefix(name, dir) }):
			return Rejected("Workbooks with embedded objects, ActiveX controls or external links are not accepted.")
		case name == "[content_types].xml" && partContains(f, "macroenabled"):
			return Rejected("Workbooks containing macros are not accepted.")
		case strings.HasSuffix(name, ".rels") && partContains(f, "relationships/vbaproject", "relationships/oleobject"):
			return Rejected("Workbooks with macros or embedded objects are not accepted.")
		case strings.HasPrefix(name, "xl/worksheets/") && strings.HasSuffix(name, ".xml") && hasDDEFormula(f):
			return Rejected("Workbooks with DDE formulas are not accepted.")
		}
	}
	return nil
}

func partContains(f *zip.File, needles ...string) bool {
	rc, err := f.Open()
	if err != nil {
		return true
	}
	defer func() { _ = rc.Close() }()
	longest := 0
	for _, n := range needles {
		longest = max(longest, len(n))
	}
	buf := make([]byte, 64<<10)
	var tail []byte
	for {
		n, err := rc.Read(buf)
		window := bytes.ToLower(append(tail, buf[:n]...))
		for _, needle := range needles {
			if bytes.Contains(window, []byte(needle)) {
				return true
			}
		}
		if len(window) > longest {
			tail = append([]byte{}, window[len(window)-longest:]...)
		} else {
			tail = window
		}
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return true
		}
	}
}

func hasDDEFormula(f *zip.File) bool {
	rc, err := f.Open()
	if err != nil {
		return true
	}
	defer func() { _ = rc.Close() }()
	d := xml.NewDecoder(rc)
	var formula strings.Builder
	inFormula := false
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return true
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "f" {
				inFormula = true
				formula.Reset()
			}
		case xml.CharData:
			if inFormula && formula.Len() < 1<<16 {
				formula.Write(t)
			}
		case xml.EndElement:
			if t.Name.Local == "f" {
				inFormula = false
				if ddeOutsideQuotes(formula.String()) {
					return true
				}
			}
		}
	}
}

func ddeOutsideQuotes(formula string) bool {
	quoted := false
	for _, r := range formula {
		switch {
		case r == '"':
			quoted = !quoted
		case r == '|' && !quoted:
			return true
		}
	}
	return false
}
