package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

func TestSafeCellNeutralisesFormulas(t *testing.T) {
	cases := map[string]string{
		"=HYPERLINK(\"x\")": "'=HYPERLINK(\"x\")",
		"+62812":            "'+62812",
		"-5":                "'-5",
		"@SUM(A1)":          "'@SUM(A1)",
		"Budi":              "Budi",
		"":                  "",
	}
	for in, want := range cases {
		assert.Equal(t, want, SafeCell(in), in)
	}
}

func TestEncodeExportCSVHasBOMAndEscapes(t *testing.T) {
	data, mime, err := encodeExport("csv", [][]string{{"Name", "Phone"}, {"O'Neil, Jr", "+6281"}})
	require.NoError(t, err)
	assert.Equal(t, "text/csv; charset=utf-8", mime)
	assert.True(t, bytes.HasPrefix(data, []byte(utf8BOM)))
	assert.Contains(t, string(data), "\"O'Neil, Jr\",'+6281")
}

func TestEncodeExportXLSXIsReadable(t *testing.T) {
	data, _, err := encodeExport("xlsx", [][]string{exportHeader, {"ABCDEFGH2", "Budi"}})
	require.NoError(t, err)
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	rows, err := f.GetRows("Crew")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "Crew ID", rows[0][0])
	assert.Equal(t, "Budi", rows[1][1])
}

func TestExportRecordFormatsDatesAndCompliance(t *testing.T) {
	dob := time.Date(1990, 3, 5, 0, 0, 0, 0, time.UTC)
	exp := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	rec := exportRecord(model.CrewRow{PublicID: "ABCDEFGH2", FullName: "Budi", Department: "catering_other", DateOfBirth: &dob,
		Compliant: true, NearestExpiry: &exp, SeaServiceDays: 400}, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), DefaultThresholds, "DD-MMM-YYYY")
	assert.Equal(t, "Catering/Other", rec[3])
	assert.Equal(t, "05-Mar-1990", rec[6])
	assert.Equal(t, "36", rec[7])
	assert.Equal(t, "Compliant", rec[15])
	assert.Equal(t, "01-Nov-2026", rec[16])
	assert.Equal(t, BadgeRed, rec[17])
	assert.Equal(t, "1 y 1 m 5 d", rec[18])
	assert.Len(t, rec, len(exportHeader))
}

type recordingQueue struct {
	kind    string
	payload any
}

func (q *recordingQueue) Enqueue(_ context.Context, kind string, payload any) (model.Job, error) {
	q.kind, q.payload = kind, payload
	return model.Job{ID: uuid.New(), Kind: kind, Status: model.JobQueued}, nil
}

func TestExportExcludesBlacklistedUnlessAskedFor(t *testing.T) {
	q := &recordingQueue{}
	svc := NewCrewExportService(nil, q, nil, nil)
	_, err := svc.Enqueue(context.Background(), "pdf", model.CrewFilter{})
	requireAppErr(t, err, "validation_failed")

	_, err = svc.Enqueue(context.Background(), "csv", model.CrewFilter{GroupBy: "principal"})
	require.NoError(t, err)
	p := q.payload.(CrewExportPayload)
	assert.Equal(t, []string{StatusBlacklisted}, p.Filter.ExcludeStatuses)
	assert.Empty(t, p.Filter.GroupBy)

	_, err = svc.Enqueue(context.Background(), "xlsx", model.CrewFilter{StatusCodes: []string{StatusBlacklisted}})
	require.NoError(t, err)
	assert.Empty(t, q.payload.(CrewExportPayload).Filter.ExcludeStatuses)
}

type fakeImportStore struct {
	lookups   repository.ImportLookups
	existing  map[string]bool
	employees map[string]bool
	imported  []repository.ImportCrew
}

func (f *fakeImportStore) ExistingEmployeeNumbers(_ context.Context, normalized []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, n := range normalized {
		if f.employees[n] {
			out[n] = true
		}
	}
	return out, nil
}

func (f *fakeImportStore) ImportLookups(context.Context) (repository.ImportLookups, error) {
	return f.lookups, nil
}

func (f *fakeImportStore) ExistingDocumentNumbers(_ context.Context, _ uuid.UUID, numbers []string) (map[string]bool, error) {
	out := map[string]bool{}
	for _, n := range numbers {
		if f.existing[n] {
			out[n] = true
		}
	}
	return out, nil
}

func (f *fakeImportStore) ImportCrews(_ context.Context, rows []repository.ImportCrew, _ func() (string, error), _ int) ([]repository.ImportOutcome, error) {
	out := make([]repository.ImportOutcome, len(rows))
	for i, r := range rows {
		f.imported = append(f.imported, r)
		out[i] = repository.ImportOutcome{Line: r.Line, PublicID: "ABCDEFGH2"}
		if r.Crew.FullName == "Fails In DB" {
			out[i] = repository.ImportOutcome{Line: r.Line, Error: "boom"}
		}
	}
	return out, nil
}

func newImportFixture() (*CrewImportService, *fakeImportStore, *memCache, *recordingQueue) {
	rank := uuid.New()
	store := &fakeImportStore{
		lookups:  repository.ImportLookups{Ranks: map[string]uuid.UUID{"CO": rank, "CHIEF OFFICER": rank}, PassportType: uuid.New(), SeamanBook: uuid.New()},
		existing: map[string]bool{"X0000001": true},
	}
	cache, q := newMemCache(), &recordingQueue{}
	return NewCrewImportService(store, cache, q), store, cache, q
}

func TestImportTemplateRoundTripsThroughValidation(t *testing.T) {
	svc, _, _, _ := newImportFixture()
	preview, err := svc.Validate(context.Background(), svc.Template())
	require.NoError(t, err)
	assert.Equal(t, 1, preview.Valid, "%+v", preview.Rows)
}

func TestImportValidatesEveryRowAndFlagsDuplicates(t *testing.T) {
	svc, _, _, _ := newImportFixture()
	csvData := "\xEF\xBB\xBFFull_Name,date_of_birth,applied_rank,passport_number,passport_expiry,email\n" +
		"Valid One,1990-01-01,co,X1111111,2030-01-01,one@example.com\n" +
		",1990-01-01,CO,,,\n" +
		"Bad Rank,1990-13-01,Bosun,,,\n" +
		"Existing,1990-01-01,CO,x0000001,2030-01-01,\n" +
		"Copy,1990-01-01,CO,X1111111,2030-01-01,\n" +
		"No Expiry,1990-01-01,CO,X2222222,,\n" +
		"\n"
	preview, err := svc.Validate(context.Background(), []byte(csvData))
	require.NoError(t, err)
	assert.Equal(t, 6, preview.Total)
	assert.Equal(t, 1, preview.Valid)
	assert.Equal(t, 3, preview.Invalid)
	assert.Equal(t, 2, preview.Duplicate)
	assert.NotEmpty(t, preview.Token)

	byLine := map[int]ImportPreviewRow{}
	for _, r := range preview.Rows {
		byLine[r.Line] = r
	}
	assert.Equal(t, RowValid, byLine[2].Status)
	assert.Contains(t, byLine[3].Errors, "full_name")
	assert.Contains(t, byLine[4].Errors, "applied_rank")
	assert.Contains(t, byLine[4].Errors, "date_of_birth")
	assert.Equal(t, RowDuplicate, byLine[5].Status)
	assert.Contains(t, byLine[6].Errors["passport_number"], "line 2")
	assert.Contains(t, byLine[7].Errors, "passport_expiry")
}

func TestImportRejectsBadFiles(t *testing.T) {
	svc, _, _, _ := newImportFixture()
	cases := map[string]string{
		"unknown column": "full_name,shoe\nA,42\n",
		"missing name":   "email\na@example.com\n",
		"header only":    "full_name\n",
		"binary":         "full_name\n\x00\x01",
		"broken quotes":  "full_name\n\"unterminated\n",
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Validate(context.Background(), []byte(data))
			assert.Error(t, err)
		})
	}
	big := "full_name\n" + strings.Repeat("A\n", MaxImportRows+1)
	_, err := svc.Validate(context.Background(), []byte(big))
	requireAppErr(t, err, "validation_failed")
}

func TestImportCommitRunsStagedRowsAndReports(t *testing.T) {
	svc, store, cache, q := newImportFixture()
	ctx := context.Background()
	preview, err := svc.Validate(ctx, []byte("full_name\nAlpha\nFails In DB\nBeta\n,\n"))
	require.NoError(t, err)

	j, err := svc.Commit(ctx, preview.Token)
	require.NoError(t, err)
	assert.Equal(t, CrewImportJob, q.kind)

	payload, _ := json.Marshal(q.payload)
	j.Payload = payload
	res, err := svc.Handle(ctx, j)
	require.NoError(t, err)
	report := res.(ImportReport)
	assert.Equal(t, 2, report.Created)
	assert.Equal(t, 1, report.Failed)
	assert.Len(t, store.imported, 3, "invalid rows are never sent to the database")
	assert.Empty(t, cache.data, "staged rows are removed after the import")

	_, err = svc.Commit(ctx, preview.Token)
	requireAppErr(t, err, "import_expired")
}

func TestCrewImport_EmployeeIDColumn(t *testing.T) {
	svc, store, _, _ := newImportFixture()
	store.employees = map[string]bool{"777": true}
	assert.True(t, strings.HasSuffix(strings.SplitN(string(svc.Template()), "\n", 2)[0], ",employee_id"))

	csvData := "full_name,applied_rank,employee_id\n" +
		"New Hand,CO,900101\n" +
		"Taken,CO,00777\n" +
		"Twice,CO,0900101\n"
	preview, err := svc.Validate(context.Background(), []byte(csvData))
	require.NoError(t, err)
	byLine := map[int]ImportPreviewRow{}
	for _, r := range preview.Rows {
		byLine[r.Line] = r
	}
	assert.Equal(t, RowValid, byLine[2].Status)
	assert.Equal(t, RowInvalid, byLine[3].Status)
	assert.Equal(t, "Another crew member already has this Employee ID.", byLine[3].Errors["employee_id"])
	assert.Equal(t, RowInvalid, byLine[4].Status)
	assert.Contains(t, byLine[4].Errors["employee_id"], "line 2")

	_, err = svc.Commit(context.Background(), preview.Token)
	require.NoError(t, err)
	var staged stagedImport
	raw, ok, err := svc.cache.Get(context.Background(), importKey(preview.Token))
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, json.Unmarshal(raw, &staged))
	require.Len(t, staged.Rows, 1)
	assert.Equal(t, "900101", staged.Rows[0].Crew.EmployeeNo)
}

func TestCrewWriteError_EmployeeNoDuplicate(t *testing.T) {
	err := crewWriteError(fmt.Errorf("insert crew: %w", &pgconn.PgError{Code: "23505", ConstraintName: "crews_employee_no_key"}))
	var appErr *apperr.Error
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "Another crew member already has this Employee ID.", appErr.Fields["employee_no"])
}
