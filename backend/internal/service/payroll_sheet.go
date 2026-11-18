package service

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	"image/png"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"golang.org/x/image/webp"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// PayrollSheetHead is the letterhead and title of an exported payroll workbook.
type PayrollSheetHead struct {
	Company, Address, Phone, Email, Title, Range string
	Logo                                         []byte
}

const (
	payrollSheetName = "Payroll"
	payrollFirstRow  = 10
	payrollLogoPx    = 90.0
	usdNumFmt        = `_(* #,##0.00_);_(* \(#,##0.00\);_(* "-"??_);_(@_)`
	idrNumFmt        = `_-* #,##0_-;\-* #,##0_-;_-* "-"_-;_-@_-`
	payrollDateFmt   = `[$-409]d\-mmm\-yyyy;@`
)

var payrollSheetHeader = []string{"No", "Name", "Employee ID", "Rank", "Vessel", "BANK", "NAME ACCOUNT", "ACCOUNT NO", "USD", "IDR", "DATE", "REMARK"}

var payrollColumnWidths = map[string]float64{
	"A": 9.9, "B": 9.7, "C": 30.7, "D": 30.6, "E": 14.7, "F": 63.3, "G": 42.7, "H": 48.3, "I": 30.6, "J": 32.4, "K": 27.1, "L": 21.9, "M": 36.1,
}

// FormatPayrollRange renders a period range the way the bank sheet shows it, e.g. "25 Aug - 24 Sep".
func FormatPayrollRange(start, end time.Time) string {
	return start.Format("2 Jan") + " - " + end.Format("2 Jan")
}

type payrollStyles struct {
	letterhead, title, header, no, text, account, usd, idr, date, remark, totalUSD, totalIDR int
}

func newPayrollStyles(f *excelize.File) (payrollStyles, error) {
	border := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1},
	}
	font := func(size float64) *excelize.Font { return &excelize.Font{Family: "Calibri", Size: size, Bold: true} }
	center := &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true}
	format := func(s string) *string { return &s }
	specs := []*excelize.Style{
		{Font: font(16), Alignment: &excelize.Alignment{Horizontal: "left"}},
		{Font: font(26), Alignment: &excelize.Alignment{Horizontal: "center"}},
		{Font: font(18), Alignment: center, Border: border},
		{Font: font(12), Alignment: center, Border: border},
		{Font: font(16), Alignment: &excelize.Alignment{Vertical: "center"}, Border: border},
		{Font: font(12), Alignment: center, Border: border, NumFmt: 49},
		{Font: font(26), Border: border, CustomNumFmt: format(usdNumFmt)},
		{Font: font(20), Border: border, CustomNumFmt: format(idrNumFmt)},
		{Font: font(20), Border: border, CustomNumFmt: format(payrollDateFmt)},
		{Font: font(12), Alignment: center, Border: border},
		{Font: font(26), CustomNumFmt: format(usdNumFmt)},
		{Font: font(20), CustomNumFmt: format(idrNumFmt)},
	}
	ids := make([]int, len(specs))
	for i, s := range specs {
		id, err := f.NewStyle(s)
		if err != nil {
			return payrollStyles{}, fmt.Errorf("payroll sheet style: %w", err)
		}
		ids[i] = id
	}
	return payrollStyles{ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6], ids[7], ids[8], ids[9], ids[10], ids[11]}, nil
}

type sheetWriter struct {
	f   *excelize.File
	err error
}

func (w *sheetWriter) do(err error) {
	if w.err == nil && err != nil {
		w.err = err
	}
}

func (w *sheetWriter) text(axis, v string, style int) {
	w.do(w.f.SetCellStr(payrollSheetName, axis, SafeCell(v)))
	w.do(w.f.SetCellStyle(payrollSheetName, axis, axis, style))
}

func (w *sheetWriter) number(axis, v string, style int) {
	if v != "" {
		w.do(w.f.SetCellDefault(payrollSheetName, axis, v))
	}
	w.do(w.f.SetCellStyle(payrollSheetName, axis, axis, style))
}

// WritePayrollWorkbook renders lines in the bank-transfer layout the crewing team uses.
func WritePayrollWorkbook(head PayrollSheetHead, lines []model.PayrollLine) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	if err := f.SetSheetName("Sheet1", payrollSheetName); err != nil {
		return nil, fmt.Errorf("name payroll sheet: %w", err)
	}
	st, err := newPayrollStyles(f)
	if err != nil {
		return nil, err
	}
	w := &sheetWriter{f: f}
	for col, width := range payrollColumnWidths {
		w.do(f.SetColWidth(payrollSheetName, col, col, width))
	}
	w.letterhead(head, st)
	w.do(f.SetRowHeight(payrollSheetName, 8, 55.5))
	for i, h := range payrollSheetHeader {
		col, _ := excelize.ColumnNumberToName(i + 2)
		w.text(col+"8", h, st.header)
		w.do(f.MergeCell(payrollSheetName, col+"8", col+"9"))
		w.do(f.SetCellStyle(payrollSheetName, col+"9", col+"9", st.header))
	}
	for i, l := range lines {
		w.line(payrollFirstRow+i, i+1, l, st)
	}
	if len(lines) > 0 {
		last, total := strconv.Itoa(payrollFirstRow+len(lines)-1), strconv.Itoa(payrollFirstRow+len(lines))
		w.do(f.SetCellFormula(payrollSheetName, "J"+total, "SUM(J10:J"+last+")"))
		w.do(f.SetCellStyle(payrollSheetName, "J"+total, "J"+total, st.totalUSD))
		w.do(f.SetCellFormula(payrollSheetName, "K"+total, "SUM(K10:K"+last+")"))
		w.do(f.SetCellStyle(payrollSheetName, "K"+total, "K"+total, st.totalIDR))
	}
	if w.err != nil {
		return nil, fmt.Errorf("write payroll sheet: %w", w.err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("encode payroll workbook: %w", err)
	}
	return buf.Bytes(), nil
}

func (w *sheetWriter) letterhead(head PayrollSheetHead, st payrollStyles) {
	lines := []string{head.Company, head.Address}
	if head.Phone != "" {
		lines = append(lines, "Tlp. "+head.Phone)
	}
	if head.Email != "" {
		lines = append(lines, "Email : "+head.Email)
	}
	for i, v := range lines {
		if v != "" {
			w.text("C"+strconv.Itoa(2+i), v, st.letterhead)
		}
	}
	w.text("F3", strings.ToUpper(head.Title), st.title)
	w.text("F5", head.Range, st.title)
	if pic, ok := excelLogo(head.Logo); ok {
		w.do(w.f.AddPictureFromBytes(payrollSheetName, "B2", pic))
	}
}

func (w *sheetWriter) line(row, no int, l model.PayrollLine, st payrollStyles) {
	r := strconv.Itoa(row)
	w.number("B"+r, strconv.Itoa(no), st.no)
	w.text("C"+r, l.CrewName, st.text)
	w.text("D"+r, l.EmployeeNo, st.text)
	w.text("E"+r, l.RankCode, st.text)
	w.text("F"+r, l.VesselName, st.text)
	w.text("G"+r, l.BankName, st.text)
	w.text("H"+r, l.AccountName, st.text)
	w.text("I"+r, l.AccountNumber, st.account)
	usd := ""
	if l.AmountUSD != nil {
		usd = l.AmountUSD.String()
	}
	w.number("J"+r, usd, st.usd)
	w.number("K"+r, l.AmountIDR.String(), st.idr)
	w.do(w.f.SetCellValue(payrollSheetName, "L"+r, l.TransferDate))
	w.do(w.f.SetCellStyle(payrollSheetName, "L"+r, "L"+r, st.date))
	w.text("M"+r, l.Remark, st.remark)
}

func excelLogo(data []byte) (*excelize.Picture, bool) {
	if len(data) == 0 {
		return nil, false
	}
	if bytes.Contains(data[:min(len(data), 1024)], []byte("<svg")) {
		return &excelize.Picture{Extension: ".svg", File: data, Format: &excelize.GraphicOptions{Positioning: "oneCell"}}, true
	}
	var ext string
	switch http.DetectContentType(data) {
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		img, err := webp.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, false
		}
		var b bytes.Buffer
		if err := png.Encode(&b, img); err != nil {
			return nil, false
		}
		data, ext = b.Bytes(), ".png"
	default:
		return nil, false
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Height == 0 {
		return nil, false
	}
	scale := payrollLogoPx / float64(cfg.Height)
	return &excelize.Picture{Extension: ext, File: data, Format: &excelize.GraphicOptions{ScaleX: scale, ScaleY: scale, Positioning: "oneCell"}}, true
}
