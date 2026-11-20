package service

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/xuri/excelize/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
)

// CVPersonal is the personal block of an RVOS CV as written in the file.
type CVPersonal struct {
	FullName, Rank, PlaceOfBirth, MaritalStatus, Nationality, Religion string
	Email, PPESize, HomePhone, MobilePhone, Address                    string
	BloodGroup, DateOfJoin                                             string
	DateOfBirth                                                        *time.Time
	NameRow                                                            int
}

// CVKin is the next-of-kin block of an RVOS CV.
type CVKin struct{ Name, Relationship, Address, Phone string }

// CVDocument is one travelling-document row.
type CVDocument struct {
	Row                         int
	Label, Number, PlaceOfIssue string
	Issued, Expires             *time.Time
}

// CVCertificate is one dated certificate row from the competency or training block.
type CVCertificate struct {
	Row             int
	Text            string
	Competency      bool
	Issued, Expires *time.Time
}

// CVSeaService is one shipboard-experience row.
type CVSeaService struct {
	Row                                            int
	Vessel, Rank, Flag, VesselType, GRTHP, Company string
	From, To                                       *time.Time
}

// CVCellError points at a cell that could not be read.
type CVCellError struct {
	Section string `json:"section"`
	Row     int    `json:"row"`
	Message string `json:"message"`
}

// CVWorkbook is everything read from an RVOS CV.
type CVWorkbook struct {
	Personal       CVPersonal
	NextOfKin      CVKin
	Father, Mother string
	Documents      []CVDocument
	Certificates   []CVCertificate
	SeaService     []CVSeaService
	Errors         []CVCellError
	Missing        []string
}

const cvDateError = "The date could not be read."

const (
	cvHeaderKin         = "NEXT OF KIN"
	cvHeaderTravel      = "TRAVELLING DOCUMENT"
	cvHeaderCompetency  = "CERTIFICATE OF COMPETENCY"
	cvHeaderTraining    = "SAFETY AND SKILL TRAINING CERTIFICATES"
	cvHeaderVessel      = "NAME OF VESSEL"
	cvColNumber         = "NUMBER"
	cvColIssued         = "DATE OF ISSUE"
	cvColExpires        = "DATE OF EXPIRY"
	cvColPlace          = "PLACE OF ISSUE"
	cvColCapacity       = "CAPACITY"
	cvColRank           = "RANK"
	cvColFlag           = "FLAG / REGISTRY"
	cvColVesselType     = "VESSEL TYPE"
	cvColGRT            = "GRT / HP"
	cvColCompany        = "COMPANY"
	cvColFrom           = "FROM"
	cvColTo             = "TO"
	cvLabelName         = "NAME"
	cvLabelHomeAddress  = "HOME ADDRESS"
	cvSectionPersonal   = "personal"
	cvSectionDocuments  = "documents"
	cvSectionCerts      = "certificates"
	cvSectionSeaService = "sea_service"
)

var cvPersonalLabels = []string{
	cvLabelName, "RANK", "DATE OF BIRTH", "PLACE OF BIRTH", "MARITAL STATUS", "NATIONALITY", "RELIGION", "BLOOD GROUP",
	"EMAIL", "PPE SIZE", "HOME TEL NO", "MOBILE NO", cvLabelHomeAddress, "DATE OF JOIN",
}

var cvKinLabels = []string{cvLabelName, "RELATIONSHIP", "ADDRESS", "TEL NO", "FATHER'S NAME", "MOTHER'S NAME"}

var (
	cvTextDate    = regexp.MustCompile(`^(\d{1,2})[./-](\d{1,2})[./-](\d{4})$`)
	cvDateLayouts = []string{"2-Jan-2006", "02-Jan-2006", "2 Jan 2006", "2 January 2006", "2006-01-02"}
	cvNoDate      = map[string]bool{"": true, "UNLIMITED": true, "PERMANENT": true, "UNLIMITEDVALIDITY": true}
)

func cvLayoutError() error {
	return apperr.Validation(map[string]string{"file": "This file does not look like an RVOS CV."})
}

func normalizeCVLabel(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func cvLabelIndex(labels []string) map[string]string {
	idx := make(map[string]string, len(labels))
	for _, l := range labels {
		idx[normalizeCVLabel(l)] = l
	}
	return idx
}

type cvGrid struct {
	rows     [][]string
	date1904 bool
	errors   []CVCellError
}

type cvValue struct {
	text     string
	row, col int
}

func (g *cvGrid) cell(r, c int) string {
	if r < 0 || c < 0 || r >= len(g.rows) || c >= len(g.rows[r]) {
		return ""
	}
	return strings.TrimSpace(g.rows[r][c])
}

func (g *cvGrid) find(match func(label string) bool) (int, int, bool) {
	for r, row := range g.rows {
		for c := range row {
			if match(normalizeCVLabel(row[c])) {
				return r, c, true
			}
		}
	}
	return -1, -1, false
}

func (g *cvGrid) findLabel(label string) (int, int, bool) {
	want := normalizeCVLabel(label)
	return g.find(func(s string) bool { return s == want })
}

func (g *cvGrid) column(rows []int, label string) int {
	want := normalizeCVLabel(label)
	for _, r := range rows {
		if r < 0 || r >= len(g.rows) {
			continue
		}
		for c := range g.rows[r] {
			if normalizeCVLabel(g.rows[r][c]) == want {
				return c
			}
		}
	}
	return -1
}

func (g *cvGrid) labelValues(from, to int, labels []string) map[string]cvValue {
	out := map[string]cvValue{}
	idx := cvLabelIndex(labels)
	for r := max(from, 0); r < min(to, len(g.rows)); r++ {
		row := g.rows[r]
		for c := 0; c < len(row); c++ {
			label, ok := idx[normalizeCVLabel(row[c])]
			if !ok {
				continue
			}
			if _, seen := out[label]; seen {
				continue
			}
			v := cvValue{row: r, col: c}
			for k := c + 1; k < len(row); k++ {
				if _, isLabel := idx[normalizeCVLabel(row[k])]; isLabel {
					break
				}
				if text := strings.TrimSpace(row[k]); text != "" {
					v.text = text
					break
				}
			}
			out[label] = v
		}
	}
	return out
}

func (g *cvGrid) rowHasLabel(r int, labels []string) bool {
	if r < 0 || r >= len(g.rows) {
		return false
	}
	idx := cvLabelIndex(labels)
	for _, s := range g.rows[r] {
		if _, ok := idx[normalizeCVLabel(s)]; ok {
			return true
		}
	}
	return false
}

func (g *cvGrid) tableEnd(header int, headers []int) int {
	end := g.sectionEnd(header, headers)
	for r := header + 1; r < end; r++ {
		if g.rowHasLabel(r, []string{cvColIssued, cvColExpires}) {
			return r
		}
	}
	return end
}

func (g *cvGrid) firstText(r int) string {
	if r < 0 || r >= len(g.rows) {
		return ""
	}
	for _, s := range g.rows[r] {
		if text := strings.TrimSpace(s); text != "" {
			return text
		}
	}
	return ""
}

func (g *cvGrid) date(raw, section string, row int) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "-" || cvNoDate[normalizeCVLabel(raw)] {
		return nil
	}
	if serial, err := strconv.ParseFloat(raw, 64); err == nil {
		if serial >= 3653 && serial <= 2958465 {
			if t, err := excelize.ExcelDateToTime(serial, g.date1904); err == nil {
				d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
				return &d
			}
		}
	} else if m := cvTextDate.FindStringSubmatch(raw); m != nil {
		day, _ := strconv.Atoi(m[1])
		month, _ := strconv.Atoi(m[2])
		year, _ := strconv.Atoi(m[3])
		d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if d.Day() == day && int(d.Month()) == month {
			return &d
		}
	} else {
		for _, layout := range cvDateLayouts {
			if t, err := time.Parse(layout, raw); err == nil {
				return &t
			}
		}
	}
	g.errors = append(g.errors, CVCellError{Section: section, Row: row + 1, Message: cvDateError})
	return nil
}

func (g *cvGrid) sectionEnd(start int, headers []int) int {
	end := len(g.rows)
	for _, h := range headers {
		if h > start && h < end {
			end = h
		}
	}
	return end
}

const (
	cvMaxRows = 500
	cvMaxCols = 100
)

func cvHasText(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return true
		}
	}
	return false
}

func readCVRows(f *excelize.File, sheet string) ([][]string, bool, error) {
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = rows.Close() }()
	var out [][]string
	for r := 0; rows.Next(); r++ {
		cols, err := rows.Columns(excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, false, err
		}
		if r >= cvMaxRows {
			if cvHasText(cols) {
				return nil, true, nil
			}
			continue
		}
		if len(cols) > cvMaxCols {
			if cvHasText(cols[cvMaxCols:]) {
				return nil, true, nil
			}
			cols = cols[:cvMaxCols]
		}
		out = append(out, cols)
	}
	return out, false, rows.Error()
}

// ParseCrewCV reads an RVOS CV workbook by its labels.
func ParseCrewCV(data []byte) (CVWorkbook, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit: filecheck.MaxXLSXUncompressed, UnzipXMLSizeLimit: filecheck.MaxXLSXUncompressed,
	})
	if err != nil {
		return CVWorkbook{}, cvLayoutError()
	}
	defer func() { _ = f.Close() }()
	date1904 := false
	if props, err := f.GetWorkbookProps(); err == nil && props.Date1904 != nil {
		date1904 = *props.Date1904
	}
	var personal, sea *cvGrid
	for _, sheet := range f.GetSheetList() {
		if visible, err := f.GetSheetVisible(sheet); err != nil || !visible {
			continue
		}
		rows, tooBig, err := readCVRows(f, sheet)
		if tooBig {
			return CVWorkbook{}, cvLayoutError()
		}
		if err != nil {
			continue
		}
		g := &cvGrid{rows: rows, date1904: date1904}
		if _, _, ok := g.findLabel(cvLabelName); ok && personal == nil {
			personal = g
			continue
		}
		if _, _, ok := g.findLabel(cvHeaderVessel); ok && sea == nil {
			sea = g
		}
	}
	if personal == nil {
		return CVWorkbook{}, cvLayoutError()
	}
	wb, ok := readCVPersonalSheet(personal)
	if !ok {
		return CVWorkbook{}, cvLayoutError()
	}
	if sea != nil {
		wb.SeaService = readCVSeaService(sea)
		personal.errors = append(personal.errors, sea.errors...)
	} else {
		wb.Missing = append(wb.Missing, "shipboard experience")
	}
	wb.Errors = personal.errors
	return wb, nil
}

func readCVPersonalSheet(g *cvGrid) (CVWorkbook, bool) {
	kinRow, _, hasKin := g.find(func(s string) bool { return strings.HasPrefix(s, normalizeCVLabel(cvHeaderKin)) })
	travelRow, travelCol, hasTravel := g.findLabel(cvHeaderTravel)
	cocRow, cocCol, hasCoc := g.findLabel(cvHeaderCompetency)
	trainRow, trainCol, hasTrain := g.findLabel(cvHeaderTraining)
	if !hasTravel && !hasTrain {
		return CVWorkbook{}, false
	}
	var wb CVWorkbook
	for _, s := range []struct {
		name  string
		found bool
	}{{"next of kin", hasKin}, {"travelling documents", hasTravel}, {"certificate of competency", hasCoc}, {"safety and skill training certificates", hasTrain}} {
		if !s.found {
			wb.Missing = append(wb.Missing, s.name)
		}
	}
	headers := []int{}
	for _, h := range []struct {
		row int
		ok  bool
	}{{kinRow, hasKin}, {travelRow, hasTravel}, {cocRow, hasCoc}, {trainRow, hasTrain}} {
		if h.ok {
			headers = append(headers, h.row)
		}
	}
	personalEnd := g.sectionEnd(-1, headers)
	values := g.labelValues(0, personalEnd, cvPersonalLabels)
	if _, ok := values[cvLabelName]; !ok {
		return CVWorkbook{}, false
	}
	text := func(label string) string { return values[label].text }
	p := CVPersonal{
		FullName: text(cvLabelName), Rank: text("RANK"), PlaceOfBirth: text("PLACE OF BIRTH"), MaritalStatus: text("MARITAL STATUS"),
		Nationality: text("NATIONALITY"), Religion: text("RELIGION"), Email: text("EMAIL"), PPESize: text("PPE SIZE"),
		HomePhone: text("HOME TEL NO"), MobilePhone: text("MOBILE NO"), BloodGroup: text("BLOOD GROUP"),
		NameRow: values[cvLabelName].row + 1,
	}
	if join := text("DATE OF JOIN"); strings.IndexFunc(join, unicode.IsDigit) >= 0 {
		p.DateOfJoin = join
	}
	if dob, ok := values["DATE OF BIRTH"]; ok && dob.text != "" {
		p.DateOfBirth = g.date(dob.text, cvSectionPersonal, dob.row)
	}
	if addr, ok := values[cvLabelHomeAddress]; ok {
		p.Address = addr.text
		next := addr.row + 1
		if next < personalEnd && !g.rowHasLabel(next, cvPersonalLabels) {
			if more := g.firstText(next); more != "" {
				p.Address = strings.TrimPrefix(p.Address+", "+more, ", ")
			}
		}
	}
	wb.Personal = p
	if hasKin {
		kin := g.labelValues(kinRow+1, g.sectionEnd(kinRow, headers), cvKinLabels)
		wb.NextOfKin = CVKin{Name: kin[cvLabelName].text, Relationship: kin["RELATIONSHIP"].text, Address: kin["ADDRESS"].text, Phone: kin["TEL NO"].text}
		wb.Father, wb.Mother = kin["FATHER'S NAME"].text, kin["MOTHER'S NAME"].text
	}
	if hasTravel {
		wb.Documents = readCVDocuments(g, travelRow, travelCol, g.tableEnd(travelRow, headers))
	}
	if hasCoc {
		wb.Certificates = append(wb.Certificates, readCVCertificates(g, cocRow, cocCol, g.tableEnd(cocRow, headers), true)...)
	}
	if hasTrain {
		wb.Certificates = append(wb.Certificates, readCVCertificates(g, trainRow, trainCol, g.tableEnd(trainRow, headers), false)...)
	}
	return wb, true
}

func readCVDocuments(g *cvGrid, header, nameCol, end int) []CVDocument {
	rows := []int{header}
	number, issued, expires, place := g.column(rows, cvColNumber), g.column(rows, cvColIssued), g.column(rows, cvColExpires), g.column(rows, cvColPlace)
	var out []CVDocument
	for r := header + 1; r < end; r++ {
		num := g.cell(r, number)
		if num == "" {
			continue
		}
		out = append(out, CVDocument{
			Row: r + 1, Label: g.cell(r, nameCol), Number: num, PlaceOfIssue: g.cell(r, place),
			Issued: g.date(g.cell(r, issued), cvSectionDocuments, r), Expires: g.date(g.cell(r, expires), cvSectionDocuments, r),
		})
	}
	return out
}

func readCVCertificates(g *cvGrid, header, nameCol, end int, competency bool) []CVCertificate {
	rows := []int{header}
	issued, expires, capacity := g.column(rows, cvColIssued), g.column(rows, cvColExpires), g.column(rows, cvColCapacity)
	var out []CVCertificate
	for r := header + 1; r < end; r++ {
		name, issue, expiry := g.cell(r, nameCol), g.cell(r, issued), g.cell(r, expires)
		if name == "" || (issue == "" && expiry == "") {
			continue
		}
		if c := g.cell(r, capacity); competency && c != "" {
			name += " – " + c
		}
		out = append(out, CVCertificate{
			Row: r + 1, Text: name, Competency: competency,
			Issued: g.date(issue, cvSectionCerts, r), Expires: g.date(expiry, cvSectionCerts, r),
		})
	}
	return out
}

func readCVSeaService(g *cvGrid) []CVSeaService {
	header, vesselCol, _ := g.findLabel(cvHeaderVessel)
	both := []int{header, header + 1}
	head := []int{header}
	rank, flag, vtype, grt, company := g.column(head, cvColRank), g.column(head, cvColFlag), g.column(head, cvColVesselType), g.column(head, cvColGRT), g.column(head, cvColCompany)
	from, to := g.column(both, cvColFrom), g.column(both, cvColTo)
	var out []CVSeaService
	for r := header + 1; r < len(g.rows); r++ {
		vessel := g.cell(r, vesselCol)
		if vessel == "" {
			continue
		}
		out = append(out, CVSeaService{
			Row: r + 1, Vessel: vessel, Rank: g.cell(r, rank), Flag: g.cell(r, flag), VesselType: g.cell(r, vtype),
			GRTHP: g.cell(r, grt), Company: g.cell(r, company),
			From: g.date(g.cell(r, from), cvSectionSeaService, r), To: g.date(g.cell(r, to), cvSectionSeaService, r),
		})
	}
	return out
}
