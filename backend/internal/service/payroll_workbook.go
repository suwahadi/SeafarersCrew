package service

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
)

// WorkbookRow is one transfer row read from a payroll workbook, before it is matched to crew.
type WorkbookRow struct {
	Line           int
	EmployeeNo     string
	BankName       string
	AccountName    string
	AccountNumber  string
	AccountNumeric bool
	USD, IDR       *decimal.Decimal
	Date           *time.Time
	Remark         string
	Errors         map[string]string
}

const (
	payrollHeaderScanRows = 20
	maxPayrollRemark      = 300
)

var payrollHeaderKeys = map[string]string{
	"employeeid": "employee_no", "bank": "bank_name", "nameaccount": "account_name", "accountno": "account_number",
	"usd": "usd", "idr": "idr", "date": "date", "remark": "remark",
}

var textAmount = regexp.MustCompile(`^-?(\d{1,3}(,\d{3})+|\d+)(\.\d{1,2})?$`)

var payrollDateLayouts = []string{"2-Jan-2006", "2-Jan-06", "2006-01-02", "2/1/2006"}

func payrollFileError(msg string) error {
	return apperr.Validation(map[string]string{"file": msg})
}

// ParsePayrollWorkbook reads the transfer rows of the first visible sheet that has the payroll header.
func ParsePayrollWorkbook(data []byte) ([]WorkbookRow, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit: filecheck.MaxXLSXUncompressed, UnzipXMLSizeLimit: filecheck.MaxXLSXUncompressed,
	})
	if err != nil {
		return nil, payrollFileError("The file is not a valid Excel workbook.")
	}
	defer func() { _ = f.Close() }()
	for _, sheet := range f.GetSheetList() {
		if visible, err := f.GetSheetVisible(sheet); err != nil || !visible {
			continue
		}
		rows, err := f.GetRows(sheet, excelize.Options{RawCellValue: true})
		if err != nil {
			return nil, fmt.Errorf("read payroll sheet: %w", err)
		}
		if header, cols := findPayrollHeader(rows); header >= 0 {
			return readPayrollRows(f, sheet, rows, header, cols)
		}
	}
	return nil, payrollFileError("The workbook has no payroll header row (Employee ID, ACCOUNT NO, IDR).")
}

func normalizePayrollHeader(s string) string {
	return strings.NewReplacer(" ", "", ".", "", "_", "", "\n", "").Replace(strings.ToLower(strings.TrimSpace(s)))
}

func findPayrollHeader(rows [][]string) (int, map[string]int) {
	for i := range min(len(rows), payrollHeaderScanRows) {
		cols := map[string]int{}
		for c, v := range rows[i] {
			if key, ok := payrollHeaderKeys[normalizePayrollHeader(v)]; ok {
				cols[key] = c
			}
		}
		_, emp := cols["employee_no"]
		_, acc := cols["account_number"]
		_, idr := cols["idr"]
		if emp && acc && idr {
			return i, cols
		}
	}
	return -1, nil
}

type payrollSheet struct {
	f     *excelize.File
	sheet string
	cols  map[string]int
}

func (p payrollSheet) raw(row []string, key string) string {
	c, ok := p.cols[key]
	if !ok || c >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[c])
}

func (p payrollSheet) axis(key string, rowIndex int) string {
	name, _ := excelize.CoordinatesToCellName(p.cols[key]+1, rowIndex+1)
	return name
}

func (p payrollSheet) numericCell(key string, rowIndex int) bool {
	if _, ok := p.cols[key]; !ok {
		return false
	}
	t, err := p.f.GetCellType(p.sheet, p.axis(key, rowIndex))
	return err == nil && (t == excelize.CellTypeUnset || t == excelize.CellTypeNumber)
}

func (p payrollSheet) cellType(key string, rowIndex int) excelize.CellType {
	if _, ok := p.cols[key]; !ok {
		return excelize.CellTypeUnset
	}
	t, _ := p.f.GetCellType(p.sheet, p.axis(key, rowIndex))
	return t
}

func (p payrollSheet) literalAmount(row []string, rowIndex int) bool {
	for _, key := range []string{"usd", "idr"} {
		if p.raw(row, key) == "" {
			continue
		}
		if formula, err := p.f.GetCellFormula(p.sheet, p.axis(key, rowIndex)); err == nil && formula == "" {
			return true
		}
	}
	return false
}

func (p payrollSheet) amount(row []string, key string, rowIndex int) (*decimal.Decimal, bool) {
	v := p.value(row, key, rowIndex)
	if t := p.cellType(key, rowIndex); (t == excelize.CellTypeSharedString || t == excelize.CellTypeInlineString) && v != "" && !textAmount.MatchString(v) {
		return nil, false
	}
	return parseAmount(v)
}

func (p payrollSheet) value(row []string, key string, rowIndex int) string {
	v := p.raw(row, key)
	if v != "" {
		return v
	}
	if _, ok := p.cols[key]; !ok {
		return ""
	}
	axis := p.axis(key, rowIndex)
	if formula, err := p.f.GetCellFormula(p.sheet, axis); err != nil || formula == "" {
		return ""
	}
	calc, err := p.f.CalcCellValue(p.sheet, axis, excelize.Options{RawCellValue: true})
	if err != nil {
		return ""
	}
	return strings.TrimSpace(calc)
}

func readPayrollRows(f *excelize.File, sheet string, rows [][]string, header int, cols map[string]int) ([]WorkbookRow, error) {
	p := payrollSheet{f: f, sheet: sheet, cols: cols}
	var data []int
	for i := header + 1; i < len(rows); i++ {
		if p.raw(rows[i], "employee_no") != "" || p.raw(rows[i], "account_number") != "" || p.literalAmount(rows[i], i) {
			data = append(data, i)
		}
	}
	if len(data) == 0 {
		return nil, payrollFileError("The file has no data rows.")
	}
	if len(data) > MaxImportRows {
		return nil, payrollFileError(fmt.Sprintf("The file may contain at most %d rows.", MaxImportRows))
	}
	out := make([]WorkbookRow, 0, len(data))
	for _, i := range data {
		out = append(out, p.row(rows[i], i))
	}
	return out, nil
}

func (p payrollSheet) row(row []string, i int) WorkbookRow {
	r := WorkbookRow{
		Line:          i + 1,
		EmployeeNo:    p.raw(row, "employee_no"),
		BankName:      p.raw(row, "bank_name"),
		AccountName:   p.raw(row, "account_name"),
		AccountNumber: p.raw(row, "account_number"),
		Remark:        p.raw(row, "remark"),
	}
	if p.numericCell("employee_no", i) {
		r.EmployeeNo = plainNumber(r.EmployeeNo)
	}
	if r.AccountNumber != "" && p.numericCell("account_number", i) {
		r.AccountNumber, r.AccountNumeric = plainNumber(r.AccountNumber), true
	}
	errs := map[string]string{}
	var ok bool
	if r.USD, ok = p.amount(row, "usd", i); !ok {
		errs["usd"] = "USD is not a number."
	}
	if r.IDR, ok = p.amount(row, "idr", i); !ok {
		errs["idr"] = "IDR is not a number."
	}
	if r.Date, ok = parsePayrollDate(p.raw(row, "date")); !ok {
		errs["date"] = "The date could not be read."
	}
	if len([]rune(r.Remark)) > maxPayrollRemark {
		errs["remark"] = fmt.Sprintf("The remark may not exceed %d characters.", maxPayrollRemark)
	}
	if len(errs) > 0 {
		r.Errors = errs
	}
	return r
}

func plainNumber(s string) string {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f != math.Trunc(f) {
		return s
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func parseAmount(s string) (*decimal.Decimal, bool) {
	s = strings.ReplaceAll(strings.ReplaceAll(s, ",", ""), " ", "")
	if s == "" {
		return nil, true
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return nil, false
	}
	return &d, true
}

func parsePayrollDate(s string) (*time.Time, bool) {
	if s == "" {
		return nil, true
	}
	if serial, err := strconv.ParseFloat(s, 64); err == nil {
		t, err := excelize.ExcelDateToTime(serial, false)
		if err != nil {
			return nil, false
		}
		day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return &day, true
	}
	for _, layout := range payrollDateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return &t, true
		}
	}
	return nil, false
}
