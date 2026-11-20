package filecheck

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"path"
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
	seen := make(map[string]bool, len(zr.File))
	for _, f := range zr.File {
		key := strings.ToLower(path.Clean("/" + strings.ReplaceAll(f.Name, "\\", "/")))
		if seen[key] {
			return Rejected(invalidWorkbook)
		}
		seen[key] = true
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

var relationshipRoots = []string{
	"http://schemas.openxmlformats.org/officedocument/2006/relationships/",
	"http://schemas.openxmlformats.org/package/2006/relationships/",
	"http://purl.oclc.org/ooxml/officedocument/relationships/",
	"http://schemas.microsoft.com/office/",
}

var allowedRelationships = map[string]bool{
	"officedocument": true, "extended-properties": true, "core-properties": true, "custom-properties": true, "thumbnail": true,
	"worksheet": true, "chartsheet": true, "theme": true, "styles": true, "sharedstrings": true, "calcchain": true,
	"drawing": true, "vmldrawing": true, "image": true, "chart": true, "chartusershapes": true, "chartstyle": true, "chartcolorstyle": true,
	"diagramdata": true, "diagramlayout": true, "diagramquickstyle": true, "diagramcolors": true, "diagramdrawing": true,
	"comments": true, "threadedcomment": true, "person": true, "hyperlink": true, "printersettings": true,
	"table": true, "querytable": true, "connections": true, "pivottable": true, "pivotcachedefinition": true, "pivotcacherecords": true,
	"slicer": true, "slicercache": true, "timeline": true, "timelinecache": true, "customxml": true, "customxmlprops": true,
	"sheetmetadata": true, "volatiledependencies": true, "classificationlabels": true, "featurepropertybag": true,
	"rdrichvalue": true, "rdrichvaluestructure": true, "rdrichvaluetypes": true, "richvaluerel": true, "rdarray": true,
	"rdsupportingpropertybag": true, "rdsupportingpropertybagstructure": true,
}

var deniedContentTypes = []string{"macroenabled", "vbaproject", "oleobject", "activex", "externallink", "macrosheet", "dialogsheet"}

const activeWorkbook = "Workbooks with macros, embedded objects, ActiveX controls, external links or DDE formulas are not accepted."

type xlsxContentTypes struct {
	defaults  map[string]string
	overrides map[string]string
}

func (c xlsxContentTypes) of(name string) string {
	if ct, ok := c.overrides[name]; ok {
		return ct
	}
	return c.defaults[strings.TrimPrefix(path.Ext(name), ".")]
}

func deniedContentType(ct string) bool {
	return slices.ContainsFunc(deniedContentTypes, func(d string) bool { return strings.Contains(ct, d) })
}

// CheckStoredXLSX checks a workbook that is kept and downloaded: no macros, embedded objects, ActiveX, external links or DDE formulas wherever they sit.
func CheckStoredXLSX(data []byte) error {
	if err := CheckXLSX(data); err != nil {
		return err
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Rejected(invalidWorkbook)
	}
	types, denied, err := readXLSXContentTypes(zr)
	if err != nil {
		return Rejected(invalidWorkbook)
	}
	if denied {
		return Rejected(activeWorkbook)
	}
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		ct := types.of(name)
		if slices.ContainsFunc(storedXLSXBlockedDirs, func(dir string) bool { return strings.HasPrefix(name, dir) }) || deniedContentType(ct) {
			return Rejected(activeWorkbook)
		}
		var active bool
		switch {
		case strings.HasSuffix(name, ".rels") || strings.HasSuffix(ct, "relationships+xml"):
			active, err = activeRelationship(f)
		case strings.HasSuffix(name, ".xml") || strings.HasSuffix(ct, "+xml") || strings.HasSuffix(ct, "/xml"):
			active, err = formulaWithDDE(f)
		}
		if err != nil {
			return Rejected(invalidWorkbook)
		}
		if active {
			return Rejected(activeWorkbook)
		}
	}
	return nil
}

func walkXML(f *zip.File, visit func(tok xml.Token) bool) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	d := xml.NewDecoder(rc)
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if !visit(tok) {
			return nil
		}
	}
}

func xmlAttr(se xml.StartElement, name string) string {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func readXLSXContentTypes(zr *zip.Reader) (xlsxContentTypes, bool, error) {
	types := xlsxContentTypes{defaults: map[string]string{}, overrides: map[string]string{}}
	denied := false
	for _, f := range zr.File {
		if !strings.EqualFold(f.Name, "[Content_Types].xml") {
			continue
		}
		err := walkXML(f, func(tok xml.Token) bool {
			se, ok := tok.(xml.StartElement)
			if !ok {
				return true
			}
			ct := strings.ToLower(xmlAttr(se, "ContentType"))
			switch se.Name.Local {
			case "Default":
				types.defaults[strings.ToLower(xmlAttr(se, "Extension"))] = ct
			case "Override":
				types.overrides[strings.ToLower(strings.TrimPrefix(xmlAttr(se, "PartName"), "/"))] = ct
			}
			denied = denied || deniedContentType(ct)
			return true
		})
		return types, denied, err
	}
	return types, false, errors.New("no content types")
}

func activeRelationship(f *zip.File) (bool, error) {
	active := false
	err := walkXML(f, func(tok xml.Token) bool {
		se, ok := tok.(xml.StartElement)
		if !ok || se.Name.Local != "Relationship" {
			return true
		}
		typ := strings.ToLower(strings.TrimSpace(xmlAttr(se, "Type")))
		kind := typ[strings.LastIndex(typ, "/")+1:]
		known := allowedRelationships[kind] && slices.ContainsFunc(relationshipRoots, func(root string) bool { return strings.HasPrefix(typ, root) })
		external := strings.EqualFold(strings.TrimSpace(xmlAttr(se, "TargetMode")), "External")
		active = !known || (external && kind != "hyperlink")
		return !active
	})
	return active, err
}

func formulaElement(local string) bool {
	return local == "f" || local == "definedName" || strings.HasPrefix(local, "formula")
}

func formulaWithDDE(f *zip.File) (bool, error) {
	found, depth := false, 0
	err := walkXML(f, func(tok xml.Token) bool {
		switch t := tok.(type) {
		case xml.StartElement:
			if formulaElement(t.Name.Local) {
				depth++
			}
		case xml.EndElement:
			if formulaElement(t.Name.Local) && depth > 0 {
				depth--
			}
		case xml.CharData:
			found = depth > 0 && bytes.IndexByte(t, '|') >= 0
		}
		return !found
	})
	return found, err
}
