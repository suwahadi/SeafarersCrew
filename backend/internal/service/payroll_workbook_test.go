package service

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

type wbRow struct {
	emp     any
	bank    string
	holder  string
	account any
	usd     any
	idr     any
	date    any
	remark  string
}

type wbFormula string

type wbCached struct {
	formula string
	value   float64
}

type wbOptions struct {
	hiddenFirst   bool
	headerlessTab bool
	headerOnly    bool
}

var payrollHeader = []string{"No", "Name", "Employee ID", "Rank", "Vessel", "BANK", "NAME ACCOUNT", "ACCOUNT NO", "USD", "IDR", "DATE", "REMARK"}

func setWorkbookCell(t *testing.T, f *excelize.File, sheet, axis string, v any) {
	t.Helper()
	switch x := v.(type) {
	case nil:
	case wbFormula:
		require.NoError(t, f.SetCellFormula(sheet, axis, string(x)))
	case wbCached:
		require.NoError(t, f.SetCellValue(sheet, axis, x.value))
		require.NoError(t, f.SetCellFormula(sheet, axis, x.formula))
	case string:
		require.NoError(t, f.SetCellStr(sheet, axis, x))
	default:
		require.NoError(t, f.SetCellValue(sheet, axis, x))
	}
}

func buildPayrollWorkbook(t *testing.T, opt wbOptions, rows ...wbRow) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	const sheet = "KIRIM GAJI sept 2026"
	require.NoError(t, f.SetSheetName("Sheet1", sheet))
	if opt.hiddenFirst {
		_, err := f.NewSheet("calc")
		require.NoError(t, err)
		require.NoError(t, f.SetCellStr("calc", "B8", "Employee ID"))
		require.NoError(t, f.SetCellStr("calc", "C8", "ACCOUNT NO"))
		require.NoError(t, f.SetCellStr("calc", "D8", "IDR"))
		require.NoError(t, f.SetCellValue("calc", "B9", 999))
		require.NoError(t, f.MoveSheet("calc", sheet))
		require.NoError(t, f.SetSheetVisible("calc", false))
	}
	if opt.headerlessTab {
		_, err := f.NewSheet("notes")
		require.NoError(t, err)
		require.NoError(t, f.SetCellStr("notes", "A1", "Transfer notes"))
		require.NoError(t, f.MoveSheet("notes", sheet))
	}
	for axis, v := range map[string]string{
		"C2": "PT. CONTOH LINTAS", "C3": "Jl. Contoh No. 1", "C4": "Jakarta Utara", "C5": "Tlp. 021 - 000", "C6": "Email : x@example.com",
		"F3": "SALARY SEPTEMBER 2026", "F5": "25 Aug - 24 Sep",
	} {
		require.NoError(t, f.SetCellStr(sheet, axis, v))
	}
	for i, h := range payrollHeader {
		col, _ := excelize.ColumnNumberToName(i + 2)
		require.NoError(t, f.SetCellStr(sheet, col+"8", h))
		require.NoError(t, f.MergeCell(sheet, col+"8", col+"9"))
	}
	if opt.headerOnly {
		buf, err := f.WriteToBuffer()
		require.NoError(t, err)
		return buf.Bytes()
	}
	for i, r := range rows {
		n := strconvRow(10 + i)
		require.NoError(t, f.SetCellValue(sheet, "B"+n, i+1))
		require.NoError(t, f.SetCellStr(sheet, "C"+n, "CREW "+n))
		setWorkbookCell(t, f, sheet, "D"+n, r.emp)
		require.NoError(t, f.SetCellStr(sheet, "E"+n, "2E"))
		require.NoError(t, f.SetCellStr(sheet, "F"+n, "MV. CONTOH"))
		require.NoError(t, f.SetCellStr(sheet, "G"+n, r.bank))
		require.NoError(t, f.SetCellStr(sheet, "H"+n, r.holder))
		setWorkbookCell(t, f, sheet, "I"+n, r.account)
		setWorkbookCell(t, f, sheet, "J"+n, r.usd)
		setWorkbookCell(t, f, sheet, "K"+n, r.idr)
		setWorkbookCell(t, f, sheet, "L"+n, r.date)
		if r.remark != "" {
			require.NoError(t, f.SetCellStr(sheet, "M"+n, r.remark))
		}
	}
	total := strconvRow(10 + len(rows))
	require.NoError(t, f.SetCellFormula(sheet, "J"+total, "SUM(J10:J"+strconvRow(9+len(rows))+")"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	return buf.Bytes()
}

func strconvRow(n int) string { return fmt.Sprint(n) }

func transferDay() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }

func TestParsePayrollWorkbook_SourceLayout(t *testing.T) {
	data := buildPayrollWorkbook(t, wbOptions{},
		wbRow{emp: 900101, bank: "BANK NEGARA INDONESIA", holder: "BUDI SANTOSO", account: "0123456789", usd: 1800, idr: 32100000, date: transferDay()},
		wbRow{emp: 90105, bank: "BANK RAKYAT INDONESIA", holder: "FAJAR NUGROHO", account: int64(100000000000001), usd: 2500.55, idr: 45000000, date: transferDay()},
		wbRow{emp: "900102", bank: "MANDIRI", holder: "SITI RAHAYU", account: "1300000000001", usd: 3900, idr: 62200000, date: transferDay(), remark: "POTONG 8 juta"},
	)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 3)

	assert.Equal(t, []int{10, 11, 12}, []int{rows[0].Line, rows[1].Line, rows[2].Line})
	assert.Equal(t, "900101", rows[0].EmployeeNo)
	assert.Equal(t, "0123456789", rows[0].AccountNumber)
	assert.False(t, rows[0].AccountNumeric)
	assert.Equal(t, "BANK NEGARA INDONESIA", rows[0].BankName)
	assert.Equal(t, "BUDI SANTOSO", rows[0].AccountName)
	require.NotNil(t, rows[0].USD)
	assert.Equal(t, "1800", rows[0].USD.String())
	require.NotNil(t, rows[0].Date)
	assert.Equal(t, transferDay(), rows[0].Date.UTC())

	assert.Equal(t, "100000000000001", rows[1].AccountNumber)
	assert.True(t, rows[1].AccountNumeric)
	assert.Equal(t, "2500.55", rows[1].USD.String())
	require.NotNil(t, rows[1].IDR)
	assert.Equal(t, "45000000", rows[1].IDR.String())

	assert.Equal(t, "900102", rows[2].EmployeeNo)
	assert.Equal(t, "POTONG 8 juta", rows[2].Remark)
	for _, r := range rows {
		assert.Empty(t, r.Errors, "line %d", r.Line)
	}
}

func TestParsePayrollWorkbook_IDROnlySplitRowAndBlankDate(t *testing.T) {
	data := buildPayrollWorkbook(t, wbOptions{},
		wbRow{emp: 900102, bank: "BANK CENTRAL ASIA", holder: "RINA", account: "5500000001", idr: 8000000, remark: "8 juta"},
	)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Nil(t, rows[0].USD)
	assert.Nil(t, rows[0].Date)
	assert.Equal(t, "8000000", rows[0].IDR.String())
	assert.Empty(t, rows[0].Errors)
}

func TestParsePayrollWorkbook_ReadsSavedFormulaResults(t *testing.T) {
	data := buildPayrollWorkbook(t, wbOptions{},
		wbRow{emp: 900101, bank: "BNI", holder: "A", account: "1", usd: wbCached{formula: "1000+800", value: 1800}, idr: "32,100,000", date: "20-Oct-2026"},
	)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Empty(t, rows[0].Errors)
	require.NotNil(t, rows[0].USD)
	assert.Equal(t, "1800", rows[0].USD.String())
	assert.Equal(t, "32100000", rows[0].IDR.String())
	assert.Equal(t, transferDay(), rows[0].Date.UTC())
}

func TestParsePayrollWorkbook_SkipsHiddenAndHeaderlessSheets(t *testing.T) {
	data := buildPayrollWorkbook(t, wbOptions{hiddenFirst: true, headerlessTab: true},
		wbRow{emp: 900101, bank: "BNI", holder: "A", account: "3200000001", idr: 1000},
	)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "900101", rows[0].EmployeeNo)
	assert.Equal(t, 10, rows[0].Line)
}

func TestParsePayrollWorkbook_FileErrors(t *testing.T) {
	fileError := func(data []byte) string {
		_, err := ParsePayrollWorkbook(data)
		e, ok := apperr.As(err)
		require.True(t, ok, "%v", err)
		return e.Fields["file"]
	}

	f := excelize.NewFile()
	require.NoError(t, f.SetCellStr("Sheet1", "A1", "Name"))
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	assert.Equal(t, "The workbook has no payroll header row (Employee ID, ACCOUNT NO, IDR).", fileError(buf.Bytes()))

	assert.Equal(t, "The file has no data rows.", fileError(buildPayrollWorkbook(t, wbOptions{headerOnly: true})))

	many := make([]wbRow, MaxImportRows+1)
	for i := range many {
		many[i] = wbRow{emp: i + 1, bank: "BNI", holder: "A", account: "1", idr: 1}
	}
	assert.Equal(t, "The file may contain at most 5000 rows.", fileError(buildPayrollWorkbook(t, wbOptions{}, many...)))
}

func TestParsePayrollWorkbook_CellErrors(t *testing.T) {
	data := buildPayrollWorkbook(t, wbOptions{},
		wbRow{emp: 1, bank: "BNI", holder: "A", account: "1", usd: "abc", idr: "1.000x", date: "someday", remark: strings.Repeat("r", 301)},
	)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "USD is not a number.", rows[0].Errors["usd"])
	assert.Equal(t, "IDR is not a number.", rows[0].Errors["idr"])
	assert.Equal(t, "The date could not be read.", rows[0].Errors["date"])
	assert.Equal(t, "The remark may not exceed 300 characters.", rows[0].Errors["remark"])
	for _, msg := range rows[0].Errors {
		assert.NotContains(t, msg, "abc")
	}
}

func TestParsePayrollWorkbook_RejectsLocaleTextNumbers(t *testing.T) {
	data := buildPayrollWorkbook(t, wbOptions{},
		wbRow{emp: 1, bank: "BNI", holder: "A", account: "1", idr: "8.000"},
		wbRow{emp: 2, bank: "BNI", holder: "A", account: "2", usd: "2.570,17", idr: 1},
		wbRow{emp: 3, bank: "BNI", holder: "A", account: "3", idr: "32.100.000"},
		wbRow{emp: 4, bank: "BNI", holder: "A", account: "4", usd: "1800.50", idr: "32,100,000"},
	)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 4)
	assert.Equal(t, "IDR is not a number.", rows[0].Errors["idr"])
	assert.Equal(t, "USD is not a number.", rows[1].Errors["usd"])
	assert.Equal(t, "IDR is not a number.", rows[2].Errors["idr"])
	assert.Empty(t, rows[3].Errors)
	assert.Equal(t, "1800.5", rows[3].USD.String())
	assert.Equal(t, "32100000", rows[3].IDR.String())
}

func TestParsePayrollWorkbook_KeepsKeylessRowsWithAmounts(t *testing.T) {
	data := buildPayrollWorkbook(t, wbOptions{},
		wbRow{emp: 900101, bank: "BNI", holder: "A", account: "3200000001", idr: 1000},
		wbRow{idr: 5000},
	)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 2, "the keyless row with an amount is kept, the SUM total row is still skipped")
	assert.Equal(t, 11, rows[1].Line)
	assert.Empty(t, rows[1].EmployeeNo)
	assert.Equal(t, "5000", rows[1].IDR.String())
}

func TestParsePayrollWorkbook_NeverEvaluatesUploadedFormulas(t *testing.T) {
	data := buildPayrollWorkbook(t, wbOptions{},
		wbRow{emp: 900101, bank: "BNI", holder: "A", account: "1", usd: wbFormula("1000+800"), idr: 1},
	)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Nil(t, rows[0].USD, "a formula without a saved result must not be calculated")
	assert.Equal(t, "USD is a formula with no saved result. Open the file in Excel and save it again.", rows[0].Errors["usd"])
}

func editWorkbook(t *testing.T, data []byte, edit func(f *excelize.File, sheet string)) []byte {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	edit(f, f.GetSheetList()[0])
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	return buf.Bytes()
}

func TestParsePayrollWorkbook_ReadsOnlyTheRowArea(t *testing.T) {
	base := buildPayrollWorkbook(t, wbOptions{}, wbRow{emp: 900101, bank: "BNI", holder: "A", account: "3200000001", idr: 1000})

	far := editWorkbook(t, base, func(f *excelize.File, sheet string) {
		require.NoError(t, f.SetCellValue(sheet, "D20100", 900102))
		require.NoError(t, f.SetCellStr(sheet, "I20100", "3200000002"))
		require.NoError(t, f.SetCellValue(sheet, "K20100", 5000))
	})
	_, err := ParsePayrollWorkbook(far)
	e, ok := apperr.As(err)
	require.True(t, ok, "%v", err)
	assert.Equal(t, "The file may contain at most 5000 rows.", e.Fields["file"], "a row far below the transfer rows is refused instead of loading the whole sheet")

	styled := editWorkbook(t, base, func(f *excelize.File, sheet string) {
		style, err := f.NewStyle(&excelize.Style{Fill: excelize.Fill{Type: "pattern", Color: []string{"FFFF00"}, Pattern: 1}})
		require.NoError(t, err)
		require.NoError(t, f.SetCellStyle(sheet, "A30000", "A30000", style))
	})
	rows, err := ParsePayrollWorkbook(styled)
	require.NoError(t, err, "formatting without values below the rows is fine")
	assert.Len(t, rows, 1)
}
