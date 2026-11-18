package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

func sheetHead() PayrollSheetHead {
	return PayrollSheetHead{
		Company: "PT Seafarers Crew Indonesia", Address: "Jl. Nusantara No. 123 A Jakarta 12000, Indonesia",
		Phone: "+6282266603388", Email: "demo@seafarerscrew.io", Title: "Salary September 2026",
		Range: FormatPayrollRange(time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)),
	}
}

func sheetLines() []model.PayrollLine {
	usd := decimal.RequireFromString("1800")
	return []model.PayrollLine{
		{LineNo: 1, CrewName: "BUDI SANTOSO", EmployeeNo: "900101", RankCode: "2E", VesselName: "MV. NUSA 52", BankName: "BANK NEGARA INDONESIA",
			AccountName: "SITI RAHAYU", AccountNumber: "0123456789", AmountUSD: &usd, AmountIDR: decimal.RequireFromString("32100000"),
			TransferDate: transferDay()},
		{LineNo: 2, CrewName: "BUDI SANTOSO", EmployeeNo: "900101", RankCode: "2E", VesselName: "MV. NUSA 52", BankName: "BANK CENTRAL ASIA",
			AccountName: "RINA", AccountNumber: "5500000001", AmountIDR: decimal.RequireFromString("8000000"),
			TransferDate: transferDay(), Remark: "=HYPERLINK(\"http://x\")"},
	}
}

func openSheet(t *testing.T, data []byte) (*excelize.File, string) {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f, f.GetSheetList()[0]
}

func cellValue(t *testing.T, f *excelize.File, sheet, axis string) string {
	t.Helper()
	v, err := f.GetCellValue(sheet, axis)
	require.NoError(t, err)
	return v
}

func TestFormatPayrollRange(t *testing.T) {
	assert.Equal(t, "25 Aug - 24 Sep", FormatPayrollRange(time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC), time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)))
}

func TestWritePayrollWorkbook_Layout(t *testing.T) {
	data, err := WritePayrollWorkbook(sheetHead(), sheetLines())
	require.NoError(t, err)
	f, sheet := openSheet(t, data)

	assert.Equal(t, "PT Seafarers Crew Indonesia", cellValue(t, f, sheet, "C2"))
	assert.Equal(t, "Jl. Nusantara No. 123 A Jakarta 12000, Indonesia", cellValue(t, f, sheet, "C3"))
	assert.Equal(t, "Tlp. +6282266603388", cellValue(t, f, sheet, "C4"))
	assert.Equal(t, "Email : demo@seafarerscrew.io", cellValue(t, f, sheet, "C5"))
	assert.Equal(t, "SALARY SEPTEMBER 2026", cellValue(t, f, sheet, "F3"))
	assert.Equal(t, "25 Aug - 24 Sep", cellValue(t, f, sheet, "F5"))
	for i, h := range payrollHeader {
		col, _ := excelize.ColumnNumberToName(i + 2)
		assert.Equal(t, h, cellValue(t, f, sheet, col+"8"))
	}
	merges, err := f.GetMergeCells(sheet)
	require.NoError(t, err)
	var ranges []string
	for _, m := range merges {
		ranges = append(ranges, m.GetStartAxis()+":"+m.GetEndAxis())
	}
	assert.Contains(t, ranges, "B8:B9")
	assert.Contains(t, ranges, "M8:M9")

	raw := func(axis string) string {
		v, err := f.GetCellValue(sheet, axis, excelize.Options{RawCellValue: true})
		require.NoError(t, err)
		return v
	}
	assert.Equal(t, "1", raw("B10"))
	assert.Equal(t, "0123456789", raw("I10"))
	typ, err := f.GetCellType(sheet, "I10")
	require.NoError(t, err)
	assert.NotEqual(t, excelize.CellTypeNumber, typ)
	assert.Equal(t, "1800", raw("J10"))
	assert.Equal(t, "32100000", raw("K10"))
	assert.Empty(t, raw("J11"))

	sumJ, err := f.GetCellFormula(sheet, "J12")
	require.NoError(t, err)
	assert.Equal(t, "SUM(J10:J11)", sumJ)
	sumK, err := f.GetCellFormula(sheet, "K12")
	require.NoError(t, err)
	assert.Equal(t, "SUM(K10:K11)", sumK)

	widthC, err := f.GetColWidth(sheet, "C")
	require.NoError(t, err)
	assert.InDelta(t, 30.7, widthC, 0.01)
	widthF, err := f.GetColWidth(sheet, "F")
	require.NoError(t, err)
	assert.InDelta(t, 63.3, widthF, 0.01)

	styleID, err := f.GetCellStyle(sheet, "J10")
	require.NoError(t, err)
	style, err := f.GetStyle(styleID)
	require.NoError(t, err)
	require.NotNil(t, style.CustomNumFmt)
	assert.Equal(t, `_(* #,##0.00_);_(* \(#,##0.00\);_(* "-"??_);_(@_)`, *style.CustomNumFmt)
}

func pngLogo(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 40, 80))))
	return b.Bytes()
}

func TestWritePayrollWorkbook_LogoOptional(t *testing.T) {
	head := sheetHead()
	head.Logo = pngLogo(t)
	data, err := WritePayrollWorkbook(head, sheetLines())
	require.NoError(t, err)
	f, sheet := openSheet(t, data)
	pics, err := f.GetPictures(sheet, "B2")
	require.NoError(t, err)
	assert.Len(t, pics, 1)

	data, err = WritePayrollWorkbook(sheetHead(), sheetLines())
	require.NoError(t, err)
	f, sheet = openSheet(t, data)
	pics, err = f.GetPictures(sheet, "B2")
	require.NoError(t, err)
	assert.Empty(t, pics)

	head.Logo = []byte("not an image")
	_, err = WritePayrollWorkbook(head, sheetLines())
	require.NoError(t, err)
}

func TestWritePayrollWorkbook_FormulaInjectionNeutralised(t *testing.T) {
	data, err := WritePayrollWorkbook(sheetHead(), sheetLines())
	require.NoError(t, err)
	f, sheet := openSheet(t, data)
	assert.Equal(t, "'=HYPERLINK(\"http://x\")", cellValue(t, f, sheet, "M11"))
	formula, err := f.GetCellFormula(sheet, "M11")
	require.NoError(t, err)
	assert.Empty(t, formula)
}

func TestPayrollWorkbook_RoundTrip(t *testing.T) {
	lines := sheetLines()
	lines[1].Remark = "POTONG 8 juta"
	data, err := WritePayrollWorkbook(sheetHead(), lines)
	require.NoError(t, err)
	rows, err := ParsePayrollWorkbook(data)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	for i, r := range rows {
		assert.Empty(t, r.Errors)
		assert.Equal(t, lines[i].EmployeeNo, r.EmployeeNo)
		assert.Equal(t, lines[i].AccountNumber, r.AccountNumber)
		assert.False(t, r.AccountNumeric)
		assert.Equal(t, lines[i].BankName, r.BankName)
		assert.Equal(t, lines[i].AccountName, r.AccountName)
		assert.True(t, lines[i].AmountIDR.Equal(*r.IDR))
		assert.Equal(t, lines[i].TransferDate, r.Date.UTC())
		assert.Equal(t, lines[i].Remark, r.Remark)
	}
	assert.True(t, lines[0].AmountUSD.Equal(*rows[0].USD))
	assert.Nil(t, rows[1].USD)
}

func TestPayrollExport_EmptyPeriodReimportsAsNoRows(t *testing.T) {
	data, err := WritePayrollWorkbook(sheetHead(), nil)
	require.NoError(t, err)
	_, err = ParsePayrollWorkbook(data)
	e, ok := apperr.As(err)
	require.True(t, ok, "%v", err)
	assert.Equal(t, "The file has no data rows.", e.Fields["file"])
}

type fakePayrollExportStore struct {
	period model.PayrollPeriod
	lines  []model.PayrollLine
	filter model.PayrollLineFilter
}

func (f *fakePayrollExportStore) GetPeriod(_ context.Context, publicID string) (model.PayrollPeriod, error) {
	if publicID != f.period.PublicID {
		return model.PayrollPeriod{}, fmt.Errorf("get payroll period: %w", repository.ErrNotFound)
	}
	return f.period, nil
}

func (f *fakePayrollExportStore) AllLines(_ context.Context, _ uuid.UUID, filter model.PayrollLineFilter) ([]model.PayrollLine, error) {
	f.filter = filter
	return f.lines, nil
}

type fakeLogoReader struct{ data []byte }

func (f fakeLogoReader) Content(context.Context, uuid.UUID) ([]byte, string, error) {
	return f.data, "image/png", nil
}

func TestPayrollExport_HandleStoresFileWithFilter(t *testing.T) {
	store := &fakePayrollExportStore{
		period: model.PayrollPeriod{ID: uuid.New(), PublicID: "PERAAAAA2", Title: "Salary September 2026",
			PeriodStart: time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC), PeriodEnd: time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)},
		lines: sheetLines(),
	}
	files := &fakeGeneratedFiles{stored: map[uuid.UUID][]byte{}, names: map[uuid.UUID]string{}}
	settings := fakeSettings{"company": {"legal_name": "PT Seafarers Crew Indonesia", "logo_light_file_id": uuid.NewString()}}
	q := &recordingQueue{}
	svc := NewPayrollExportService(store, q, files, fakeLogoReader{data: pngLogo(t)}, settings)

	_, err := svc.Enqueue(context.Background(), "UNKNOWN99", model.PayrollLineFilter{})
	requireAppError(t, err, 404, "not_found")

	filter := model.PayrollLineFilter{Banks: []string{"BANK NEGARA INDONESIA"}, Sort: "amount_idr", Desc: true}
	j, err := svc.Enqueue(context.Background(), "PERAAAAA2", filter)
	require.NoError(t, err)
	assert.Equal(t, PayrollExportJob, q.kind)
	payload, err := json.Marshal(q.payload)
	require.NoError(t, err)
	j.Payload = payload

	res, err := svc.Handle(context.Background(), j)
	require.NoError(t, err)
	result := res.(PayrollExportResult)
	assert.Equal(t, filter, store.filter)
	assert.Equal(t, 2, result.Rows)
	assert.Regexp(t, regexp.MustCompile(`^payroll-PERAAAAA2-\d{8}\.xlsx$`), result.Filename)
	assert.Regexp(t, regexp.MustCompile(`^exports\|payroll-PERAAAAA2-\d{8}\.xlsx\|application/vnd.openxmlformats-officedocument.spreadsheetml.sheet\|xlsx$`), files.names[result.FileID])

	f, sheet := openSheet(t, files.stored[result.FileID])
	assert.Equal(t, "PT Seafarers Crew Indonesia", cellValue(t, f, sheet, "C2"))
	pics, err := f.GetPictures(sheet, "B2")
	require.NoError(t, err)
	assert.Len(t, pics, 1)
}
