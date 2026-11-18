package filecheck

import (
	"archive/zip"
	"bytes"
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
