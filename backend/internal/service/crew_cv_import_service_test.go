package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
)

type fakeCVStore struct {
	lookups   repository.CVLookups
	dups      []repository.CVDuplicate
	created   *repository.CVCreate
	createErr error
}

func (f *fakeCVStore) Lookups(context.Context) (repository.CVLookups, error) { return f.lookups, nil }

func (f *fakeCVStore) Duplicates(context.Context, repository.CVDuplicateKeys) ([]repository.CVDuplicate, error) {
	return f.dups, nil
}

func (f *fakeCVStore) Create(_ context.Context, in repository.CVCreate, _ func() (string, error), _ int) (string, error) {
	f.created = &in
	if f.createErr != nil {
		return "", f.createErr
	}
	return "CVCREW002", nil
}

type fakeCVFiles struct {
	id       uuid.UUID
	uploaded []UploadInput
	deleted  []uuid.UUID
}

func (f *fakeCVFiles) Upload(_ context.Context, in UploadInput) (model.File, error) {
	f.uploaded = append(f.uploaded, in)
	return model.File{ID: f.id}, nil
}

func (f *fakeCVFiles) Delete(_ context.Context, id uuid.UUID) error {
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeOptions struct{ invalidated int }

func (f *fakeOptions) Invalidate(context.Context) error {
	f.invalidated++
	return nil
}

type cvFixture struct {
	svc     *CrewCVImportService
	store   *fakeCVStore
	cache   *memCache
	files   *fakeCVFiles
	options *fakeOptions
}

func newCVFixture() cvFixture {
	f := cvFixture{store: &fakeCVStore{lookups: cvTestLookups()}, cache: newMemCache(), files: &fakeCVFiles{id: uuid.New()}, options: &fakeOptions{}}
	f.svc = NewCrewCVImportService(f.store, f.cache, f.files, f.options, inlinePool{})
	return f
}

func (f cvFixture) validate(t *testing.T, edits ...func(*excelize.File)) CVPreview {
	t.Helper()
	p, err := f.svc.Validate(context.Background(), "CV RVOS 2024 - AB - BUDI.xlsx", buildCrewCV(t, edits...))
	require.NoError(t, err)
	return p
}

func choiceKeys(p CVPreview) []string {
	keys := make([]string, 0, len(p.Choices))
	for _, c := range p.Choices {
		keys = append(keys, c.Key)
	}
	return keys
}

func field(p CVPreview, label string) string {
	for _, f := range p.Personal {
		if f.Label == label {
			return f.Value
		}
	}
	return "<missing>"
}

const (
	cvCOCKey    = "certificate:CERTIFICATE OF COMPETENCY HOME COUNTRY RATING ABLE DECK"
	cvSSOKey    = "certificate:SHIP SECURITY OFFICER SSO"
	cvAHTSDPKey = "vessel_type:AHTS DP 1"
)

func cvDecisions() map[string]CVDecision {
	return map[string]CVDecision{
		cvCOCKey:    {Action: "create"},
		cvSSOKey:    {Action: "skip"},
		cvAHTSDPKey: {Action: "match", TargetID: cvAHTS.String()},
	}
}

func TestCrewCVValidate_PreviewSections(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t)
	assert.NotEmpty(t, p.Token)
	assert.Equal(t, "BUDI SANTOSO", p.FullName)
	assert.Equal(t, "+6281234567890", field(p, "Mobile phone"))
	assert.Equal(t, "married", field(p, "Marital status"))
	assert.Equal(t, "1990-05-17", field(p, "Date of birth"))
	assert.Equal(t, "JL. MAWAR NO. 5 RT 01 RW 02, SEMARANG 50111", field(p, "Address"))
	assert.Len(t, p.Documents, 2)
	assert.Equal(t, "Seaman Book (CDC)", p.Documents[1].MatchedAs)
	assert.Len(t, p.Certificates, 4)
	assert.Equal(t, "Basic Safety Training", p.Certificates[1].MatchedAs)
	assert.Len(t, p.SeaService, 3)
	require.Len(t, p.NextOfKin, 3)
	assert.Equal(t, "spouse", p.NextOfKin[0].Cells["relationship"])
	assert.Equal(t, "+6281211112222", p.NextOfKin[0].Cells["phone"])
	assert.Equal(t, "parent", p.NextOfKin[1].Cells["relationship"])
	assert.Equal(t, []string{cvCOCKey, cvSSOKey, cvAHTSDPKey}, choiceKeys(p))
	assert.Equal(t, &cvAHTS, p.Choices[2].SuggestedID)
	assert.Contains(t, p.NotImported, "Blood group")
	assert.Empty(t, p.Errors)
	_, staged, err := f.cache.Get(context.Background(), "cv_import:"+p.Token)
	require.NoError(t, err)
	assert.True(t, staged)

	same := f.validate(t, func(x *excelize.File) { require.NoError(t, x.SetCellStr(cvPersonalSheet, "U20", "Siti Aminah")) })
	assert.Len(t, same.NextOfKin, 2, "a parent equal to the primary next of kin is not added twice")
}

func TestCrewCVValidate_OneChoicePerText(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t, func(x *excelize.File) {
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "U7", "XX"))
		for _, axis := range []string{"F5", "F6", "F7"} {
			require.NoError(t, x.SetCellStr(cvSeaSheet, axis, "XX"))
		}
	})
	var ranks []CVChoice
	for _, c := range p.Choices {
		if c.Kind == "rank" {
			ranks = append(ranks, c)
		}
	}
	require.Len(t, ranks, 1, "one choice no matter how many rows use the same text")
	assert.Equal(t, CVChoice{Key: "rank:XX", Kind: "rank", Text: "XX"}, ranks[0])
	for _, fld := range p.Personal {
		if fld.Label == "Rank" {
			assert.Equal(t, "rank:XX", fld.ChoiceKey)
		}
	}
	for _, row := range p.SeaService {
		assert.Contains(t, row.ChoiceKeys, "rank:XX")
	}
}

func TestCrewCVValidate_DuplicateStops(t *testing.T) {
	f := newCVFixture()
	f.store.dups = []repository.CVDuplicate{{PublicID: "OLDCREW02", FullName: "Budi Santoso", Reason: "passport"}}
	p := f.validate(t)
	require.NotNil(t, p.Duplicate)
	assert.Equal(t, "OLDCREW02", p.Duplicate.PublicID)
	assert.Empty(t, p.Token)
}

func TestCrewCVValidate_ErrorsBlockToken(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t, func(x *excelize.File) { require.NoError(t, x.SetCellStr(cvPersonalSheet, "S36", "31.02.2020")) })
	assert.Equal(t, []CVCellError{{Section: "certificates", Row: 36, Message: "The date could not be read."}}, p.Errors)
	assert.Empty(t, p.Token)

	noName := f.validate(t, func(x *excelize.File) { require.NoError(t, x.SetCellStr(cvPersonalSheet, "H7", "")) })
	assert.Contains(t, noName.Errors, CVCellError{Section: "personal", Row: 7, Message: "The CV has no name."})
	assert.Empty(t, noName.Token)
}

func TestCrewCVCommit_ExpiredToken(t *testing.T) {
	f := newCVFixture()
	_, err := f.svc.Commit(context.Background(), CVCommitInput{Token: "nope"})
	requireAppError(t, err, 410, "import_expired")
	e, _ := apperr.As(err)
	assert.Equal(t, "The preview expired. Upload the CV again.", e.Message)
}

func TestCrewCVCommit_RequiresEveryDecision(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t)
	d := cvDecisions()
	delete(d, cvSSOKey)
	_, err := f.svc.Commit(context.Background(), CVCommitInput{Token: p.Token, Decisions: d})
	requireAppError(t, err, 422, "validation_failed")
	e, _ := apperr.As(err)
	assert.Equal(t, "Choose what to do with every highlighted item.", e.Fields["decisions"])

	d = cvDecisions()
	d[cvAHTSDPKey] = CVDecision{Action: "match", TargetID: cvBST.String()}
	_, err = f.svc.Commit(context.Background(), CVCommitInput{Token: p.Token, Decisions: d})
	requireAppError(t, err, 422, "validation_failed")
	assert.Nil(t, f.store.created, "nothing is written while a decision is invalid")
}

func TestCrewCVCommit_ConsumesToken(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t)
	res, err := f.svc.Commit(context.Background(), CVCommitInput{Token: p.Token, Decisions: cvDecisions()})
	require.NoError(t, err)
	assert.Equal(t, CVCommitResult{CrewID: "CVCREW002", FullName: "BUDI SANTOSO"}, res)
	_, err = f.svc.Commit(context.Background(), CVCommitInput{Token: p.Token, Decisions: cvDecisions()})
	requireAppError(t, err, 410, "import_expired")
}

func TestCrewCVCommit_BuildsCreateInput(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t, func(x *excelize.File) {
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "C37", "Medical Check Up"))
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "S37", "05.05.2024"))
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "Y37", "05.05.2026"))
	})
	_, err := f.svc.Commit(context.Background(), CVCommitInput{Token: p.Token, Decisions: cvDecisions()})
	require.NoError(t, err)
	c := f.store.created
	require.NotNil(t, c)

	require.Len(t, f.files.uploaded, 1)
	assert.Equal(t, filecheck.ContextCrewCV, f.files.uploaded[0].Context)
	assert.Equal(t, "CV RVOS 2024 - AB - BUDI.xlsx", f.files.uploaded[0].Filename)
	assert.Equal(t, &f.files.id, c.Crew.CVFileID)
	assert.Equal(t, &cvAB, c.Crew.AppliedRankID)
	assert.Equal(t, "+6281234567890", c.Crew.MobilePhone)
	require.NotNil(t, c.Crew.MaritalStatus)
	assert.Equal(t, "married", *c.Crew.MaritalStatus)
	assert.Equal(t, "X 1234567", c.Keys.PassportNumber)
	assert.Equal(t, "Y 765432", c.Keys.SeamanBookNumber)

	require.Len(t, c.NewTypes, 1)
	nt := c.NewTypes[0]
	assert.Equal(t, cvCOCKey, nt.Key)
	assert.Equal(t, repository.CVKindCertificateType, nt.Kind)
	assert.Equal(t, "Certificate of Competency (Home Country) – RATING ABLE DECK", nt.Name)
	assert.Equal(t, "coc", nt.Category)
	assert.True(t, nt.HasExpiry)
	assert.LessOrEqual(t, len(nt.Code), 40)
	assert.True(t, strings.HasPrefix(nt.Code, "certificate_of_competency_home_country"), nt.Code)

	assert.Equal(t, []repository.CVAlias{{Kind: repository.CVKindVesselType, Alias: "AHTS DP 1", TargetID: cvAHTS}}, c.Aliases)

	require.Len(t, c.Documents, 3, "passport, seaman book and the medical row matched to a document type")
	assert.Equal(t, cvMCU, c.Documents[2].TypeID)
	assert.Equal(t, "-", c.Documents[2].Doc.Number)
	require.Len(t, c.Certificates, 3, "competency (new type), BST and MFA; SSO skipped")
	assert.Equal(t, cvCOCKey, c.Certificates[0].NewTypeKey)
	for _, cert := range c.Certificates {
		assert.Equal(t, "-", cert.Doc.Number)
	}

	require.Len(t, c.SeaService, 3)
	dp := c.SeaService[1].Experience
	assert.Equal(t, &cvAHTS, dp.VesselTypeID)
	assert.Equal(t, cvAB, dp.RankID)
	assert.Equal(t, "sign_off", dp.Status)
	assert.Equal(t, time.Date(2019, time.August, 1, 0, 0, 0, 0, time.UTC), dp.SignOnDate)
	assert.Contains(t, dp.Remarks, "Flag: MALAYSIA")
	assert.Contains(t, dp.Remarks, "Vessel type in CV: AHTS DP 1")
	assert.Equal(t, &cvCompany, c.SeaService[0].Experience.CompanyID)
	assert.Equal(t, "OFFSHORE CONTOH SDN BHD", dp.CompanyName)

	require.Len(t, c.Kins, 3)
	assert.Equal(t, model.NextOfKin{Name: "SITI AMINAH", Relationship: "spouse", Phone: "+6281211112222", Address: "JL. MAWAR NO. 5 SEMARANG", IsPrimary: true}, c.Kins[0])
	assert.Equal(t, "parent", c.Kins[2].Relationship)
	assert.Equal(t, 1, f.options.invalidated, "a new type refreshes the cached master options")
}

func TestCrewCVCommit_SkippedRankDropsSeaRows(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t, func(x *excelize.File) {
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "U7", "XX"))
		for _, axis := range []string{"F5", "F6", "F7"} {
			require.NoError(t, x.SetCellStr(cvSeaSheet, axis, "XX"))
		}
	})
	d := cvDecisions()
	d["rank:XX"] = CVDecision{Action: "skip"}
	d[cvCOCKey] = CVDecision{Action: "skip"}
	_, err := f.svc.Commit(context.Background(), CVCommitInput{Token: p.Token, Decisions: d})
	require.NoError(t, err)
	assert.Nil(t, f.store.created.Crew.AppliedRankID)
	assert.Empty(t, f.store.created.SeaService)
	assert.Zero(t, f.options.invalidated, "no new type, no cache refresh")
}

func TestCrewCVCommit_DeletesFileOnFailure(t *testing.T) {
	f := newCVFixture()
	f.store.createErr = repository.ErrCVDuplicate
	p := f.validate(t)
	_, err := f.svc.Commit(context.Background(), CVCommitInput{Token: p.Token, Decisions: cvDecisions()})
	requireAppError(t, err, 409, "crew_exists")
	assert.Equal(t, []uuid.UUID{f.files.id}, f.files.deleted)
	assert.Zero(t, f.options.invalidated)
}

func TestCrewCVValidate_EmptySectionsAreLists(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t, func(x *excelize.File) { require.NoError(t, x.DeleteSheet(cvSeaSheet)) })
	out, err := json.Marshal(p)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"sea_service":[]`, "the admin reads every section as a list")
}

func TestCrewCVValidate_RowRulesMatchManualEntry(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t, func(x *excelize.File) {
		require.NoError(t, x.SetCellValue(cvPersonalSheet, "S23", cvDay(2019, time.January, 1)))
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "H24", strings.Repeat("9", 61)))
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "Y23", strings.Repeat("P", 101)))
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "Y36", "01.01.2018"))
		require.NoError(t, x.SetCellStr(cvPersonalSheet, "H20", strings.Repeat("F", 151)))
		require.NoError(t, x.SetCellStr(cvSeaSheet, "Q5", "01.01.2018"))
		require.NoError(t, x.SetCellStr(cvSeaSheet, "K6", strings.Repeat("C", 151)))
	})
	assert.ElementsMatch(t, []CVCellError{
		{Section: "documents", Row: 23, Message: "The expiry date must be on or after the issued date."},
		{Section: "documents", Row: 24, Message: "The number may not be longer than 60 characters."},
		{Section: "certificates", Row: 36, Message: "The expiry date must be on or after the issued date."},
		{Section: "sea_service", Row: 5, Message: "The sign-off date must be on or after the sign-on date."},
	}, p.Errors)
	assert.Empty(t, p.Token, "Confirm would fail on these rows, so the preview blocks it")
	assert.Contains(t, p.Warnings, "Documents row 23: the place of issue is longer than 100 characters and was left empty.")
	assert.Contains(t, p.Warnings, "Sea service row 6: the company is longer than 150 characters and was left empty.")
	assert.Contains(t, p.Warnings, "A next of kin name is longer than 150 characters and was not imported.")
	assert.Len(t, p.NextOfKin, 2)
}

func TestCrewCVValidate_DocumentWithoutTypeIsAnError(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t, func(x *excelize.File) { require.NoError(t, x.SetCellStr(cvPersonalSheet, "B24", "")) })
	assert.Contains(t, p.Errors, CVCellError{Section: "documents", Row: 24, Message: "The document type is missing."})
	assert.Empty(t, p.Token)
}

func withMacro(t *testing.T, data []byte) []byte {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, file := range r.File {
		w, err := zw.Create(file.Name)
		require.NoError(t, err)
		rc, err := file.Open()
		require.NoError(t, err)
		_, err = io.Copy(w, rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
	}
	w, err := zw.Create("xl/vbaProject.bin")
	require.NoError(t, err)
	_, err = w.Write([]byte("macro"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return out.Bytes()
}

func TestCrewCVValidate_RunsTheUploadChecks(t *testing.T) {
	f := newCVFixture()
	_, err := f.svc.Validate(context.Background(), "CV RVOS 2024.xls.xlsx", buildCrewCV(t))
	require.Error(t, err, "a double extension is refused before the preview, not at Confirm")
	_, err = f.svc.Validate(context.Background(), "CV RVOS 2024.xlsx", withMacro(t, buildCrewCV(t)))
	require.Error(t, err, "a workbook with macros is refused before the preview")
	_, err = f.svc.Validate(context.Background(), "CV RVOS 2024 - AB - BUDI.xlsx", buildCrewCV(t))
	require.NoError(t, err)
}

func TestCrewCVValidate_WarnsAboutMissingSections(t *testing.T) {
	f := newCVFixture()
	p := f.validate(t, func(x *excelize.File) { require.NoError(t, x.DeleteSheet(cvSeaSheet)) })
	assert.Contains(t, p.Warnings, "No shipboard experience section was found in the CV, so nothing was imported from it.")
}

func TestCrewCVCommit_LongTextAliasFitsTheColumn(t *testing.T) {
	f := newCVFixture()
	long := "Ship Security Officer " + strings.Repeat("Refresher ", 30)
	p := f.validate(t, func(x *excelize.File) { require.NoError(t, x.SetCellStr(cvPersonalSheet, "C39", long)) })
	d := cvDecisions()
	delete(d, cvSSOKey)
	for _, c := range p.Choices {
		if c.Kind == "certificate" && strings.HasPrefix(c.Text, "Ship Security Officer") {
			d[c.Key] = CVDecision{Action: "match", TargetID: cvSAT.String()}
		}
	}
	_, err := f.svc.Commit(context.Background(), CVCommitInput{Token: p.Token, Decisions: d})
	require.NoError(t, err)
	var found bool
	for _, a := range f.store.created.Aliases {
		if a.TargetID == cvSAT {
			found = true
			assert.LessOrEqual(t, utf8.RuneCountInString(a.Alias), 200)
		}
	}
	assert.True(t, found)
}

func withPart(t *testing.T, data []byte, name, body string) []byte {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, file := range r.File {
		w, err := zw.Create(file.Name)
		require.NoError(t, err)
		rc, err := file.Open()
		require.NoError(t, err)
		_, err = io.Copy(w, rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
	}
	w, err := zw.Create(name)
	require.NoError(t, err)
	_, err = w.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return out.Bytes()
}

func TestCrewCVValidate_RefusesActiveContent(t *testing.T) {
	f := newCVFixture()
	_, err := f.svc.Validate(context.Background(), "CV RVOS 2024.xlsx", withPart(t, buildCrewCV(t), "xl/externalLinks/externalLink1.xml", "<externalLink/>"))
	require.Error(t, err, "a CV that will be stored and opened later may not carry external links")
}
