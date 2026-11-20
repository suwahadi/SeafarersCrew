package filecheck

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

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
	assert.ElementsMatch(t, []string{"pdf", "jpg", "jpeg", "png", "webp", "doc", "docx", "xlsx"}, Allowed(ContextCrewCV))
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

func xlsxBytes(t *testing.T) []byte {
	t.Helper()
	f := excelize.NewFile()
	require.NoError(t, f.SetCellValue("Sheet1", "A1", "Employee ID"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	return buf.Bytes()
}

func zipWithEntries(t *testing.T, n int) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for _, name := range []string{"[Content_Types].xml", "xl/workbook.xml"} {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte("<x/>"))
		require.NoError(t, err)
	}
	for i := range n {
		_, err := zw.Create(fmt.Sprintf("xl/extra%d.xml", i))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return b.Bytes()
}

func zipDeclaringSize(t *testing.T, size uint64) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for _, name := range []string{"[Content_Types].xml", "xl/workbook.xml"} {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write([]byte("<x/>"))
		require.NoError(t, err)
	}
	w, err := zw.CreateRaw(&zip.FileHeader{Name: "xl/worksheets/sheet1.xml", Method: zip.Store, CompressedSize64: 4, UncompressedSize64: size})
	require.NoError(t, err)
	_, err = w.Write([]byte("<x/>"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return b.Bytes()
}

func TestCheckXLSX(t *testing.T) {
	require.NoError(t, CheckXLSX(xlsxBytes(t)))

	cases := map[string][]byte{
		"word document":            docxBytes(t, map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": "<w:document/>"}),
		"no content types":         docxBytes(t, map[string]string{"xl/workbook.xml": "<workbook/>"}),
		"plain text":               []byte("Employee ID,IDR\n900101,1000\n"),
		"too many entries":         zipWithEntries(t, MaxXLSXEntries),
		"declared size over limit": zipDeclaringSize(t, MaxXLSXUncompressed+1),
		"macros": docxBytes(t, map[string]string{
			"[Content_Types].xml": "<Types/>", "xl/workbook.xml": "<workbook/>", "xl/vbaProject.bin": "binary",
		}),
	}
	for name, data := range cases {
		err := CheckXLSX(data)
		require.Error(t, err, name)
		assert.Equal(t, CodeRejected, code(t, err), name)
	}
}

func TestDetect_XLSX(t *testing.T) {
	kind, err := Detect(xlsxBytes(t), "xlsx")
	require.NoError(t, err)
	assert.Equal(t, KindXLSX, kind)

	_, err = Detect([]byte("Employee ID,IDR\n"), "xlsx")
	assert.Equal(t, CodeInvalidType, code(t, err))
	assert.ElementsMatch(t, []string{"xlsx"}, Allowed(ContextPayrollImport))
	_, ok := ParseContext("payroll_import")
	assert.True(t, ok)
}

func xlsxWith(t *testing.T, change func(parts map[string]string)) []byte {
	t.Helper()
	base := xlsxBytes(t)
	zr, err := zip.NewReader(bytes.NewReader(base), int64(len(base)))
	require.NoError(t, err)
	parts := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		b, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		parts[f.Name] = string(b)
	}
	change(parts)
	return docxBytes(t, parts)
}

func addRelationship(p map[string]string, part, relType, target string) {
	rel := `<Relationship Id="rIdX" Type="` + relType + `" Target="` + target + `"/>`
	if body, ok := p[part]; ok {
		p[part] = strings.Replace(body, "</Relationships>", rel+"</Relationships>", 1)
		return
	}
	p[part] = `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rel + `</Relationships>`
}

func TestCheckStoredXLSXRejectsActiveContent(t *testing.T) {
	cases := map[string]func(p map[string]string){
		"embedded OLE object": func(p map[string]string) { p["xl/embeddings/oleObject1.bin"] = "ole" },
		"ActiveX control":     func(p map[string]string) { p["xl/activeX/activeX1.xml"] = "<ax/>" },
		"external link":       func(p map[string]string) { p["xl/externalLinks/externalLink1.xml"] = "<externalLink/>" },
		"macro content type": func(p map[string]string) {
			p["[Content_Types].xml"] = strings.Replace(p["[Content_Types].xml"], "</Types>",
				`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.ms-excel.sheet.macroEnabled.main+xml"/></Types>`, 1)
		},
		"vbaProject under another name": func(p map[string]string) {
			addRelationship(p, "xl/_rels/workbook.xml.rels", "http://schemas.microsoft.com/office/2006/relationships/vbaProject", "data.bin")
			p["xl/data.bin"] = "code"
		},
		"OLE object relationship": func(p map[string]string) {
			addRelationship(p, "xl/worksheets/_rels/sheet1.xml.rels", "http://schemas.openxmlformats.org/officeDocument/2006/relationships/oleObject", "../objects/o.bin")
		},
		"DDE formula": func(p map[string]string) {
			p["xl/worksheets/sheet1.xml"] = strings.Replace(p["xl/worksheets/sheet1.xml"], "</sheetData>", `<row r="5"><c r="A5"><f>cmd|' /C calc'!A0</f></c></row></sheetData>`, 1)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			data := xlsxWith(t, change)
			require.Error(t, CheckStoredXLSX(data))
		})
	}

	require.NoError(t, CheckStoredXLSX(xlsxBytes(t)))
	require.NoError(t, CheckStoredXLSX(xlsxWith(t, func(p map[string]string) {
		p["xl/media/image1.png"] = "png"
		p["xl/worksheets/sheet1.xml"] = strings.Replace(p["xl/worksheets/sheet1.xml"], "</sheetData>", `<row r="5"><c r="A5"><f>SUM(A1:A2)</f></c></row></sheetData>`, 1)
	})), "pictures and ordinary formulas are fine")
}

func utf16LE(s string) string {
	units := utf16.Encode([]rune(s))
	out := []byte{0xFF, 0xFE}
	for _, u := range units {
		out = append(out, byte(u), byte(u>>8))
	}
	return string(out)
}

const relsNS = `xmlns="http://schemas.openxmlformats.org/package/2006/relationships"`

func TestCheckStoredXLSXRejectsDisguisedActiveContent(t *testing.T) {
	cases := map[string]func(p map[string]string){
		"relationship type written with an XML entity": func(p map[string]string) {
			p["xl/worksheets/_rels/sheet1.xml.rels"] = `<?xml version="1.0" encoding="UTF-8"?><Relationships ` + relsNS + `><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships&#47;oleObject" Target="../media/o.bin"/></Relationships>`
			p["xl/media/o.bin"] = "ole"
		},
		"relationships part in UTF-16": func(p map[string]string) {
			p["xl/worksheets/_rels/sheet1.xml.rels"] = utf16LE(`<?xml version="1.0" encoding="UTF-16"?><Relationships ` + relsNS + `><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/oleObject" Target="../media/o.bin"/></Relationships>`)
		},
		"external link stored under another folder": func(p map[string]string) {
			addRelationship(p, "xl/_rels/workbook.xml.rels", "http://schemas.openxmlformats.org/officeDocument/2006/relationships/externalLink", "links/l1.xml")
			p["xl/links/l1.xml"] = `<externalLink xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"/>`
		},
		"DDE in a defined name": func(p map[string]string) {
			p["xl/workbook.xml"] = strings.Replace(p["xl/workbook.xml"], "</workbook>", `<definedNames><definedName name="x">cmd|' /C calc'!A0</definedName></definedNames></workbook>`, 1)
		},
		"DDE in a sheet stored under another folder": func(p map[string]string) {
			p["xl/other/s.xml"] = `<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1"><f>'cmd'|' /C calc'!A0</f></c></row></sheetData></worksheet>`
		},
		"image loaded from outside the file": func(p map[string]string) {
			p["xl/drawings/_rels/drawing1.xml.rels"] = `<Relationships ` + relsNS + `><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" TargetMode="External" Target="http://example.invalid/x.png"/></Relationships>`
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, CheckStoredXLSX(xlsxWith(t, change)))
		})
	}
	require.NoError(t, CheckStoredXLSX(xlsxWith(t, func(p map[string]string) {
		p["xl/worksheets/_rels/sheet1.xml.rels"] = `<Relationships ` + relsNS + `><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" TargetMode="External" Target="mailto:crew@example.com"/></Relationships>`
	})), "an ordinary hyperlink is fine")
}

type zipEntry struct{ name, body string }

func zipOf(t *testing.T, entries []zipEntry) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for _, e := range entries {
		w, err := zw.Create(e.name)
		require.NoError(t, err)
		_, err = w.Write([]byte(e.body))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return b.Bytes()
}

func xlsxEntries(t *testing.T) []zipEntry {
	t.Helper()
	base := xlsxBytes(t)
	zr, err := zip.NewReader(bytes.NewReader(base), int64(len(base)))
	require.NoError(t, err)
	var out []zipEntry
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		b, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		out = append(out, zipEntry{f.Name, string(b)})
	}
	return out
}

func TestCheckXLSXRejectsDuplicateEntries(t *testing.T) {
	macroTypes := `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Override PartName="/xl/workbook.xml" ContentType="application/vnd.ms-excel.sheet.macroEnabled.main+xml"/></Types>`
	for name, extra := range map[string]zipEntry{
		"same name":         {"[Content_Types].xml", macroTypes},
		"different case":    {"[content_types].XML", macroTypes},
		"backslash in path": {`xl\workbook.xml`, "<workbook/>"},
	} {
		t.Run(name, func(t *testing.T) {
			data := zipOf(t, append(xlsxEntries(t), extra))
			require.Error(t, CheckXLSX(data))
			require.Error(t, CheckStoredXLSX(data))
		})
	}
}

func TestCheckStoredXLSXAllowsOnlyKnownRelationships(t *testing.T) {
	cases := map[string]func(p map[string]string){
		"dangerous type padded with spaces": func(p map[string]string) {
			p["xl/worksheets/_rels/sheet1.xml.rels"] = `<Relationships ` + relsNS + `><Relationship Id="rId1" Type=" http://schemas.openxmlformats.org/officeDocument/2006/relationships/oleObject " Target="../media/o.bin"/></Relationships>`
		},
		"unknown relationship type": func(p map[string]string) {
			addRelationship(p, "xl/_rels/workbook.xml.rels", "http://example.invalid/relationships/anything", "anything.bin")
		},
		"add-in web extension": func(p map[string]string) {
			addRelationship(p, "_rels/.rels", "http://schemas.microsoft.com/office/2011/relationships/webextensiontaskpanes", "xl/webextensions/taskpanes.xml")
		},
		"DDE in a data validation formula": func(p map[string]string) {
			p["xl/worksheets/sheet1.xml"] = strings.Replace(p["xl/worksheets/sheet1.xml"], "</worksheet>",
				`<dataValidations count="1"><dataValidation type="list" sqref="B2"><formula1>cmd|' /C calc'!A0</formula1></dataValidation></dataValidations></worksheet>`, 1)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, CheckStoredXLSX(xlsxWith(t, change)))
		})
	}
	require.NoError(t, CheckStoredXLSX(xlsxWith(t, func(p map[string]string) {
		addRelationship(p, "_rels/.rels", "http://schemas.microsoft.com/office/2020/02/relationships/classificationlabels", "docMetadata/LabelInfo.xml")
		p["docMetadata/LabelInfo.xml"] = `<clbl:labelList xmlns:clbl="http://schemas.microsoft.com/office/2020/mipLabelMetadata"/>`
		p["xl/worksheets/_rels/sheet1.xml.rels"] = `<Relationships ` + relsNS + `><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/drawing" Target="../drawings/drawing1.xml"/></Relationships>`
		p["xl/drawings/_rels/drawing1.xml.rels"] = `<Relationships ` + relsNS + `><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/></Relationships>`
		p["xl/media/image1.png"] = "png"
	})), "sensitivity labels, drawings and pictures are ordinary content")
}

func TestCheckStoredXLSXRejectsDataConnectionsAndShadowedAttributes(t *testing.T) {
	cases := map[string]func(p map[string]string){
		"data connection": func(p map[string]string) {
			addRelationship(p, "xl/_rels/workbook.xml.rels", "http://schemas.openxmlformats.org/officeDocument/2006/relationships/connections", "data.xml")
			p["xl/data.xml"] = `<connections xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><connection id="1" name="q" type="4" refreshOnLoad="1"><webPr url="http://example.invalid/data"/></connection></connections>`
		},
		"query table": func(p map[string]string) {
			p["xl/worksheets/_rels/sheet1.xml.rels"] = `<Relationships ` + relsNS + `><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/queryTable" Target="../tables/q1.xml"/></Relationships>`
			p["xl/tables/q1.xml"] = `<queryTable xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" name="q" connectionId="1"/>`
		},
		"connections part at the usual path": func(p map[string]string) {
			p["xl/connections.xml"] = `<connections xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"/>`
		},
		"namespaced Type next to a safe one": func(p map[string]string) {
			p["xl/worksheets/_rels/sheet1.xml.rels"] = `<Relationships ` + relsNS + ` xmlns:x="urn:x"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" x:Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/oleObject" Target="../media/o.bin"/></Relationships>`
		},
		"relationships outside the package namespace": func(p map[string]string) {
			p["xl/worksheets/_rels/sheet1.xml.rels"] = `<Relationships xmlns="urn:not-opc"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/i.png"/></Relationships>`
		},
		"namespaced content type override": func(p map[string]string) {
			p["[Content_Types].xml"] = strings.Replace(p["[Content_Types].xml"], "</Types>",
				`<Override xmlns:x="urn:x" PartName="/xl/workbook.xml" x:ContentType="application/vnd.ms-excel.sheet.macroEnabled.main+xml" ContentType="application/xml"/></Types>`, 1)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, CheckStoredXLSX(xlsxWith(t, change)))
		})
	}
}
