package service

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

const (
	cvPersonalSheet = "PERSONAL INFORMATION"
	cvSeaSheet      = "SHIPBOARD EXPERIENCE"
)

func cvDay(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// buildCrewCV writes a synthetic RVOS-shaped CV that follows the cell map in the design;
// every value is invented. Edits run after the defaults so a test can change single cells.
func buildCrewCV(t *testing.T, edits ...func(f *excelize.File)) []byte {
	t.Helper()
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	require.NoError(t, f.SetSheetName("Sheet1", cvPersonalSheet))
	_, err := f.NewSheet(cvSeaSheet)
	require.NoError(t, err)

	set := func(sheet, axis string, v any) {
		t.Helper()
		if s, ok := v.(string); ok {
			require.NoError(t, f.SetCellStr(sheet, axis, s))
			return
		}
		require.NoError(t, f.SetCellValue(sheet, axis, v))
	}
	merge := func(sheet string, ranges ...string) {
		t.Helper()
		for i := 0; i < len(ranges); i += 2 {
			require.NoError(t, f.MergeCell(sheet, ranges[i], ranges[i+1]))
		}
	}
	p := cvPersonalSheet
	for row := 7; row <= 12; row++ {
		r := strconv.Itoa(row)
		merge(p, "B"+r, "G"+r, "H"+r, "Q"+r, "R"+r, "T"+r, "U"+r, "Z"+r)
	}
	merge(p, "B6", "T6", "B13", "G14", "H13", "Z13", "H14", "Z14", "B16", "Z16", "B17", "G17", "H17", "Q17", "R17", "T17", "U17", "Z17",
		"B18", "G19", "H18", "Z18", "R19", "T19", "U19", "Z19", "B20", "G20", "H20", "Q20", "R20", "T20", "U20", "Z20")
	for _, r := range []string{"22", "23", "24"} {
		merge(p, "B"+r, "G"+r, "H"+r, "L"+r, "M"+r, "R"+r, "S"+r, "X"+r, "Y"+r, "Z"+r)
	}
	for _, r := range []string{"26", "27", "28", "29"} {
		merge(p, "B"+r, "L"+r, "M"+r, "R"+r, "S"+r, "X"+r, "Y"+r, "Z"+r)
	}
	for row := 35; row <= 40; row++ {
		r := strconv.Itoa(row)
		merge(p, "C"+r, "R"+r, "S"+r, "X"+r, "Y"+r, "Z"+r)
	}

	for axis, v := range map[string]any{
		"B6": "CURRICULUM  VITAE", "U6": "Date of Join", "Y6": "/       /",
		"B7": "Name", "H7": "BUDI SANTOSO", "R7": "Rank", "U7": "AB",
		"B8": "Date Of Birth", "H8": cvDay(1990, time.May, 17), "R8": "Place of Birth", "U8": "SEMARANG",
		"B9": "Marital Status", "H9": "MARRIED", "R9": "Nationality", "U9": "INDONESIAN",
		"B10": "Religion", "H10": "ISLAM", "R10": "Blood Group", "U10": "O",
		"B11": "E-mail", "H11": "budi.santoso@example.com", "R11": "PPE Size", "U11": "XL / 42",
		"B12": "Home Tel. No", "R12": "Mobile No.", "U12": "+62.812.3456.7890",
		"B13": "Home Address", "H13": "JL. MAWAR NO. 5 RT 01 RW 02", "H14": "SEMARANG 50111",
		"B16": "NEXT OF KIN (NOK)  e.g Wife",
		"B17": "Name", "H17": "SITI AMINAH", "R17": "Relationship", "U17": "WIFE",
		"B18": "Address", "H18": "JL. MAWAR NO. 5 SEMARANG",
		"R19": "Tel. No", "U19": "0812-1111-2222",
		"B20": "Father's Name", "H20": "AHMAD SANTOSO", "R20": "Mother's Name", "U20": "SRI WAHYUNI",
		"B22": "TRAVELLING DOCUMENT", "H22": "NUMBER", "M22": "DATE OF ISSUE", "S22": "DATE OF EXPIRY", "Y22": "PLACE OF ISSUE",
		"B23": "Passport", "H23": "X 1234567", "M23": cvDay(2020, time.January, 10), "S23": cvDay(2025, time.January, 10), "Y23": "JAKARTA",
		"B24": "Seaman Book", "H24": "Y 765432", "M24": cvDay(2021, time.March, 1), "S24": cvDay(2026, time.March, 1), "Y24": "SEMARANG",
		"B26": "CERTIFICATE OF COMPETENCY", "M26": "CAPACITY", "S26": "DATE OF ISSUE", "Y26": "DATE OF EXPIRY",
		"B27": "Certificate of Competency (Home Country)", "M27": "RATING ABLE DECK", "S27": "03.04.2020", "Y27": "03.04.2025",
		"B28": "Certificate of Endorsement (Home Country)",
		"B29": "General Operator Certificate (GOC) (Home Country)",
		"B35": "NO", "C35": "SAFETY AND SKILL TRAINING CERTIFICATES", "S35": "DATE OF ISSUE", "Y35": "DATE OF EXPIRY",
		"B36": 1, "C36": "Basic Safety Training / BST", "S36": "01.02.2019", "Y36": "01.02.2024",
		"B37": 2, "C37": "Personal Survival Techniques",
		"B38": 3, "C38": "Medical First Aid / MFA", "S38": "15.06.2018", "Y38": "UNLIMITED",
		"B39": 4, "C39": "Ship Security Officer (SSO)", "S39": "10.10.2020",
		"B40": 5, "C40": "Elementary First Aid",
	} {
		set(p, axis, v)
	}

	e := cvSeaSheet
	merge(e, "A3", "A4", "B3", "E4", "F3", "F4", "G3", "G4", "H3", "I4", "J3", "J4", "K3", "O4", "P3", "Q3")
	for _, r := range []string{"5", "6", "7", "8", "9"} {
		merge(e, "B"+r, "E"+r, "H"+r, "I"+r, "K"+r, "O"+r)
	}
	for axis, v := range map[string]any{
		"Q1": "PT. CONTOH MANNING", "A2": "SHIPBOARD EXPERIENCE OF THE APPLICANT",
		"A3": "No.", "B3": "NAME OF VESSEL", "F3": "RANK", "G3": "Flag / Registry", "H3": "VESSEL TYPE", "J3": "GRT / HP", "K3": "COMPANY",
		"P3": "Period Onboard", "P4": "From", "Q4": "To",
		"A5": 1, "B5": "MV. NUSA 1", "F5": "AB", "G5": "INDONESIA", "H5": "AHTS", "J5": 1500, "K5": "PT. SAMUDERA CONTOH", "P5": "01.01.2019", "Q5": "30.06.2019",
		"A6": 2, "B6": "MV. NUSA 2", "F6": "AB", "G6": "MALAYSIA", "H6": "AHTS DP 1", "J6": 2400, "K6": "OFFSHORE CONTOH SDN BHD", "P6": "01.08.2019", "Q6": "31.01.2020",
		"A7": 3, "B7": "MV. NUSA 3", "F7": "AB", "G7": "PANAMA", "H7": "BULK CARRIER", "J7": 32000, "K7": "PT. SAMUDERA CONTOH", "P7": "01.03.2020", "Q7": "30.09.2020",
		"A8": 4, "A9": 5,
		"A15": "I hereby declare that the information above is true.", "M16": "Signature:",
	} {
		set(e, axis, v)
	}
	for _, edit := range edits {
		edit(f)
	}
	buf, err := f.WriteToBuffer()
	require.NoError(t, err)
	return buf.Bytes()
}

func parseCV(t *testing.T, edits ...func(f *excelize.File)) CVWorkbook {
	t.Helper()
	wb, err := ParseCrewCV(buildCrewCV(t, edits...))
	require.NoError(t, err)
	return wb
}

func TestParseCrewCV_PersonalAndKin(t *testing.T) {
	wb := parseCV(t)
	p := wb.Personal
	assert.Equal(t, "BUDI SANTOSO", p.FullName)
	assert.Equal(t, "AB", p.Rank)
	require.NotNil(t, p.DateOfBirth)
	assert.Equal(t, cvDay(1990, time.May, 17), *p.DateOfBirth)
	assert.Equal(t, "SEMARANG", p.PlaceOfBirth)
	assert.Equal(t, "MARRIED", p.MaritalStatus)
	assert.Equal(t, "INDONESIAN", p.Nationality)
	assert.Equal(t, "ISLAM", p.Religion)
	assert.Equal(t, "budi.santoso@example.com", p.Email)
	assert.Equal(t, "XL / 42", p.PPESize)
	assert.Empty(t, p.HomePhone)
	assert.Equal(t, "+62.812.3456.7890", p.MobilePhone)
	assert.Equal(t, "JL. MAWAR NO. 5 RT 01 RW 02, SEMARANG 50111", p.Address)
	assert.Equal(t, CVKin{Name: "SITI AMINAH", Relationship: "WIFE", Address: "JL. MAWAR NO. 5 SEMARANG", Phone: "0812-1111-2222"}, wb.NextOfKin)
	assert.Equal(t, "AHMAD SANTOSO", wb.Father)
	assert.Equal(t, "SRI WAHYUNI", wb.Mother)
	assert.Empty(t, wb.Errors)
}

func TestParseCrewCV_DocumentsAndTables(t *testing.T) {
	wb := parseCV(t)
	require.Len(t, wb.Documents, 2)
	pass := wb.Documents[0]
	assert.Equal(t, "Passport", pass.Label)
	assert.Equal(t, "X 1234567", pass.Number)
	assert.Equal(t, "JAKARTA", pass.PlaceOfIssue)
	assert.Equal(t, cvDay(2020, time.January, 10), *pass.Issued)
	assert.Equal(t, cvDay(2025, time.January, 10), *pass.Expires)
	assert.Equal(t, 23, pass.Row)
	assert.Equal(t, "Seaman Book", wb.Documents[1].Label)

	var competency []CVCertificate
	for _, c := range wb.Certificates {
		if c.Competency {
			competency = append(competency, c)
		}
	}
	require.Len(t, competency, 1)
	assert.Equal(t, "Certificate of Competency (Home Country) – RATING ABLE DECK", competency[0].Text)
	assert.Equal(t, 27, competency[0].Row)

	require.Len(t, wb.SeaService, 3, "numbered empty template lines are ignored")
	s := wb.SeaService[1]
	assert.Equal(t, CVSeaService{Row: 6, Vessel: "MV. NUSA 2", Rank: "AB", Flag: "MALAYSIA", VesselType: "AHTS DP 1", GRTHP: "2400",
		Company: "OFFSHORE CONTOH SDN BHD", From: s.From, To: s.To}, s)
	assert.Equal(t, cvDay(2019, time.August, 1), *s.From)
	assert.Equal(t, cvDay(2020, time.January, 31), *s.To)
}

func TestParseCrewCV_CertificateRowsWithDatesOnly(t *testing.T) {
	wb := parseCV(t)
	var training []CVCertificate
	for _, c := range wb.Certificates {
		if !c.Competency {
			training = append(training, c)
		}
	}
	require.Len(t, training, 3)
	assert.Equal(t, "Basic Safety Training / BST", training[0].Text)
	assert.Equal(t, cvDay(2024, time.February, 1), *training[0].Expires)
	assert.Equal(t, "Medical First Aid / MFA", training[1].Text)
	assert.Nil(t, training[1].Expires, "UNLIMITED means no expiry")
	assert.Equal(t, cvDay(2018, time.June, 15), *training[1].Issued)
	assert.Equal(t, "Ship Security Officer (SSO)", training[2].Text)
	assert.Nil(t, training[2].Expires)
	assert.Equal(t, 39, training[2].Row)
}

func TestParseCrewCV_DatesFromCellsAndText(t *testing.T) {
	wb := parseCV(t, func(f *excelize.File) {
		require.NoError(t, f.SetCellStr(cvPersonalSheet, "H8", "17.05.1990"))
		require.NoError(t, f.SetCellValue(cvPersonalSheet, "S36", cvDay(2020, time.April, 3)))
		require.NoError(t, f.SetCellStr(cvPersonalSheet, "Y36", "3.4.2025"))
	})
	assert.Equal(t, cvDay(1990, time.May, 17), *wb.Personal.DateOfBirth)
	assert.Equal(t, cvDay(2020, time.April, 3), *wb.Certificates[1].Issued)
	assert.Equal(t, cvDay(2025, time.April, 3), *wb.Certificates[1].Expires, "text dates are day first")

	bad := parseCV(t, func(f *excelize.File) {
		require.NoError(t, f.SetCellStr(cvPersonalSheet, "S36", "31.02.2020"))
	})
	assert.Contains(t, bad.Errors, CVCellError{Section: "certificates", Row: 36, Message: "The date could not be read."})
}

func TestParseCrewCV_LabelVariants(t *testing.T) {
	wb := parseCV(t, func(f *excelize.File) {
		require.NoError(t, f.SetCellStr(cvPersonalSheet, "B7", "NAME"))
		require.NoError(t, f.SetCellStr(cvPersonalSheet, "B8", "Date of  Birth :"))
		require.NoError(t, f.SetCellStr(cvPersonalSheet, "B11", "E-mail:"))
	})
	assert.Equal(t, "BUDI SANTOSO", wb.Personal.FullName)
	assert.Equal(t, cvDay(1990, time.May, 17), *wb.Personal.DateOfBirth)
	assert.Equal(t, "budi.santoso@example.com", wb.Personal.Email)
}

func TestParseCrewCV_RejectsOtherLayouts(t *testing.T) {
	payroll := buildPayrollWorkbook(t, wbOptions{}, wbRow{emp: 900101, bank: "BNI", holder: "A", account: "1", idr: 1})
	for name, data := range map[string][]byte{"payroll sheet": payroll, "not a workbook": []byte("PK\x03\x04 nope")} {
		t.Run(name, func(t *testing.T) {
			_, err := ParseCrewCV(data)
			e, ok := apperr.As(err)
			require.True(t, ok, "%v", err)
			assert.Equal(t, "This file does not look like an RVOS CV.", e.Fields["file"])
		})
	}
}

func TestParseCrewCV_CompactLabels(t *testing.T) {
	wb := parseCV(t, func(f *excelize.File) {
		require.NoError(t, f.SetCellStr(cvPersonalSheet, "R19", "Tel.No"))
		require.NoError(t, f.SetCellStr(cvSeaSheet, "G3", "Flag/Registry"))
		require.NoError(t, f.SetCellStr(cvSeaSheet, "J3", "GRT/HP"))
	})
	assert.Equal(t, "0812-1111-2222", wb.NextOfKin.Phone)
	require.Len(t, wb.SeaService, 3)
	assert.Equal(t, "MALAYSIA", wb.SeaService[1].Flag)
	assert.Equal(t, "2400", wb.SeaService[1].GRTHP)
}

func TestParseCrewCV_ReportsMissingSections(t *testing.T) {
	wb := parseCV(t, func(f *excelize.File) {
		require.NoError(t, f.SetCellStr(cvPersonalSheet, "C35", "TRAINING COURSES"))
		require.NoError(t, f.DeleteSheet(cvSeaSheet))
	})
	assert.Equal(t, []string{"safety and skill training certificates", "shipboard experience"}, wb.Missing)
	assert.Empty(t, wb.Errors, "the renamed header row is not read as a certificate")
	require.Len(t, wb.Certificates, 1, "only the competency row; numbered training rows are not swallowed by the competency block")
	assert.True(t, wb.Certificates[0].Competency)
	assert.Empty(t, parseCV(t).Missing)
}
