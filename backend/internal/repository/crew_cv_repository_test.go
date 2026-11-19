package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

type cvWorld struct {
	crewWorld
	repo *repository.CrewCVRepository
	ahts uuid.UUID
}

func setupCVWorld(t *testing.T) cvWorld {
	t.Helper()
	w := cvWorld{crewWorld: setupCrewWorld(t)}
	truncate(t, "master_aliases", "travel_documents", "certificates", "experiences", "crew_next_of_kins")
	w.ahts = mustID(t, `INSERT INTO vessel_types (name) VALUES ('AHTS') RETURNING id`)
	w.repo = repository.NewCrewCVRepository(testDB)
	return w
}

func (w cvWorld) keys(passport, seaman, name string, dob *time.Time) repository.CVDuplicateKeys {
	return repository.CVDuplicateKeys{PassportNumber: passport, SeamanBookNumber: seaman, FullName: name, DateOfBirth: dob,
		PassportType: w.passport, SeamanBookType: w.seamanBook}
}

func (w cvWorld) input(name, passport string) repository.CVCreate {
	dob := d(1990, 5, 17)
	issued, expires := d(2020, 1, 10), d(2025, 1, 10)
	return repository.CVCreate{
		Crew: model.Crew{FullName: name, DateOfBirth: &dob, Nationality: "INDONESIAN", AppliedRankID: &w.ab},
		Keys: w.keys(passport, "", name, &dob),
		NewTypes: []repository.CVNewType{
			{Key: "certificate:SHIP SECURITY OFFICER SSO", Kind: "certificate_type", Name: "Ship Security Officer (SSO)", Code: "ship_security_officer_sso", Category: "other"},
		},
		Aliases: []repository.CVAlias{{Kind: "certificate_type", Alias: "BASIC SAFETY TRAINING", TargetID: w.bst}},
		Kins: []model.NextOfKin{
			{Name: "SITI AMINAH", Relationship: "spouse", Phone: "+6281211112222", IsPrimary: true},
			{Name: "AHMAD SANTOSO", Relationship: "parent"},
		},
		Documents: []repository.CVRecord{{TypeID: w.passport, Doc: model.CrewDocument{Number: passport, IssuedDate: &issued, ExpiredDate: &expires, PlaceOfIssue: "JAKARTA"}}},
		Certificates: []repository.CVRecord{
			{TypeID: w.bst, Doc: model.CrewDocument{Number: "-", IssuedDate: &issued, ExpiredDate: &expires}},
			{NewTypeKey: "certificate:SHIP SECURITY OFFICER SSO", Doc: model.CrewDocument{Number: "-", IssuedDate: &issued}},
		},
		SeaService: []repository.CVSea{{Experience: model.Experience{VesselName: "MV. NUSA 1", RankID: w.ab, VesselTypeID: &w.ahts, GRTHP: "1500",
			CompanyName: "PT. SAMUDERA CONTOH", SignOnDate: d(2019, 1, 1), SignOffDate: ptr(d(2019, 6, 30)), Status: "sign_off", Remarks: "Flag: INDONESIA"}}},
	}
}

func ptr[T any](v T) *T { return &v }

func count(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, testDB.QueryRow(context.Background(), sql, args...).Scan(&n))
	return n
}

func TestCrewCVLookups_IncludesAliasesOfLiveTargetsOnly(t *testing.T) {
	w := setupCVWorld(t)
	gone := mustID(t, `INSERT INTO certificate_types (code, name, category, has_expiry, deleted_at) VALUES ('old', 'Old Course', 'other', false, now()) RETURNING id`)
	exec(t, `INSERT INTO master_aliases (kind, alias, target_id) VALUES ('certificate_type', 'BASIC SAFETY', $1), ('certificate_type', 'OLD COURSE X', $2), ('vessel_type', 'AHTS DP 1', $3)`,
		w.bst, gone, w.ahts)

	l, err := w.repo.Lookups(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]uuid.UUID{"BASIC SAFETY": w.bst}, l.Aliases["certificate_type"])
	assert.Equal(t, map[string]uuid.UUID{"AHTS DP 1": w.ahts}, l.Aliases["vessel_type"])
	assert.Equal(t, w.passport, l.PassportType)
	assert.Equal(t, w.seamanBook, l.SeamanBookType)
	assert.Contains(t, l.Ranks, repository.CVMasterItem{ID: w.ab, Code: "AB", Name: "AB"})
	assert.Contains(t, l.CertificateTypes, repository.CVMasterItem{ID: w.bst, Code: "bst", Name: "BST", HasExpiry: true})
	assert.NotContains(t, l.CertificateTypes, repository.CVMasterItem{ID: gone, Code: "old", Name: "Old Course"})
	assert.Contains(t, l.VesselTypes, repository.CVMasterItem{ID: w.ahts, Name: "AHTS"})
	assert.Len(t, l.Companies, 2)
}

func TestCrewCVDuplicates_ByPassportSeamanBookAndNameBirth(t *testing.T) {
	w := setupCVWorld(t)
	crews := repository.NewCrewRepository(testDB)
	dob := d(1990, 5, 17)
	a := createCrew(t, crews, "DUPAAAAA2", "Budi Santoso", nil, nil, "")
	exec(t, `INSERT INTO travel_documents (crew_id, document_type_id, number) VALUES ($1, $2, 'X 1234567'), ($1, $3, 'Y 765432')`, a, w.passport, w.seamanBook)
	createCrew(t, crews, "DUPBBBBB2", "Candra Wijaya", nil, &dob, "")

	ctx := context.Background()
	got, err := w.repo.Duplicates(ctx, w.keys("x1234567", "", "", nil))
	require.NoError(t, err)
	assert.Equal(t, []repository.CVDuplicate{{PublicID: "DUPAAAAA2", FullName: "Budi Santoso", Reason: "passport"}}, got)

	got, err = w.repo.Duplicates(ctx, w.keys("", "Y 765432", "", nil))
	require.NoError(t, err)
	assert.Equal(t, "seaman_book", got[0].Reason)

	got, err = w.repo.Duplicates(ctx, w.keys("", "", "candra WIJAYA", &dob))
	require.NoError(t, err)
	assert.Equal(t, []repository.CVDuplicate{{PublicID: "DUPBBBBB2", FullName: "Candra Wijaya", Reason: "name_and_birth_date"}}, got)

	got, err = w.repo.Duplicates(ctx, w.keys("Z 0000000", "", "Candra Wijaya", ptr(d(1991, 1, 1))))
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestCrewCVCreate_WritesEverythingInOneTransaction(t *testing.T) {
	w := setupCVWorld(t)
	pid, err := w.repo.Create(context.Background(), w.input("BUDI SANTOSO", "X 1234567"), publicid.New, publicid.MaxAttempts)
	require.NoError(t, err)
	assert.Len(t, pid, 9)

	crewID := mustID(t, `SELECT id FROM crews WHERE public_id = $1`, pid)
	assert.Equal(t, 2, count(t, `SELECT count(*) FROM crew_next_of_kins WHERE crew_id = $1`, crewID))
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM crew_next_of_kins WHERE crew_id = $1 AND is_primary`, crewID))
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM travel_documents WHERE crew_id = $1 AND number = 'X 1234567'`, crewID))
	assert.Equal(t, 2, count(t, `SELECT count(*) FROM certificates WHERE crew_id = $1 AND number = '-'`, crewID))
	sso := mustID(t, `SELECT id FROM certificate_types WHERE name = 'Ship Security Officer (SSO)' AND category = 'other' AND code = 'ship_security_officer_sso'`)
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM certificates WHERE crew_id = $1 AND certificate_type_id = $2`, crewID, sso))
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM experiences WHERE crew_id = $1 AND source = 'manual' AND remarks = 'Flag: INDONESIA'`, crewID))
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM master_aliases WHERE kind = 'certificate_type' AND alias = 'BASIC SAFETY TRAINING' AND target_id = $1`, w.bst))
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM crews c JOIN crew_statuses s ON s.id = c.crew_status_id WHERE c.id = $1 AND s.code = 'available'`, crewID))
}

func TestCrewCVCreate_ReusesExistingTypeWithTheSameName(t *testing.T) {
	w := setupCVWorld(t)
	existing := mustID(t, `INSERT INTO certificate_types (code, name, category, has_expiry) VALUES ('sso', 'Ship Security Officer (SSO)', 'cop', true) RETURNING id`)
	clash := mustID(t, `INSERT INTO certificate_types (code, name, category, has_expiry) VALUES ('h2s', 'Hydrogen Sulphide', 'other', true) RETURNING id`)
	in := w.input("BUDI SANTOSO", "X 1234567")
	in.NewTypes = append(in.NewTypes, repository.CVNewType{Key: "certificate:H2S", Kind: "certificate_type", Name: "Hydrogen Sulfide / H2S", Code: "h2s", Category: "other"})
	in.Certificates = append(in.Certificates, repository.CVRecord{NewTypeKey: "certificate:H2S", Doc: model.CrewDocument{Number: "-"}})

	_, err := w.repo.Create(context.Background(), in, publicid.New, publicid.MaxAttempts)
	require.NoError(t, err)
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM certificates WHERE certificate_type_id = $1`, existing), "same name reuses the existing type")
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM certificate_types WHERE code = 'h2s_2' AND name = 'Hydrogen Sulfide / H2S'`), "a taken code gets a suffix")
	assert.Equal(t, 0, count(t, `SELECT count(*) FROM certificates WHERE certificate_type_id = $1`, clash))
}

func TestCrewCVCreate_RollsBackOnFailure(t *testing.T) {
	w := setupCVWorld(t)
	in := w.input("BUDI SANTOSO", "X 1234567")
	in.SeaService[0].Experience.RankID = uuid.New()

	_, err := w.repo.Create(context.Background(), in, publicid.New, publicid.MaxAttempts)
	require.Error(t, err)
	assert.Zero(t, count(t, `SELECT count(*) FROM crews`))
	assert.Zero(t, count(t, `SELECT count(*) FROM certificate_types WHERE code = 'ship_security_officer_sso'`))
	assert.Zero(t, count(t, `SELECT count(*) FROM master_aliases`))
}

func TestCrewCVCreate_SecondCreateIsDuplicate(t *testing.T) {
	w := setupCVWorld(t)
	_, err := w.repo.Create(context.Background(), w.input("BUDI SANTOSO", "X 1234567"), publicid.New, publicid.MaxAttempts)
	require.NoError(t, err)
	_, err = w.repo.Create(context.Background(), w.input("BUDI SANTOSO", "X 1234567"), publicid.New, publicid.MaxAttempts)
	assert.True(t, errors.Is(err, repository.ErrCVDuplicate), "%v", err)
	assert.Equal(t, 1, count(t, `SELECT count(*) FROM crews`))
}

func TestCrewCVLookups_AcceptsTypesWithoutCode(t *testing.T) {
	w := setupCVWorld(t)
	cert := mustID(t, `INSERT INTO certificate_types (name, category, has_expiry) VALUES ('Course From The Master Screen', 'other', true) RETURNING id`)
	doc := mustID(t, `INSERT INTO document_types (name, has_expiry) VALUES ('Document From The Master Screen', false) RETURNING id`)

	l, err := w.repo.Lookups(context.Background())
	require.NoError(t, err, "types created in Master Data have no code")
	assert.Contains(t, l.CertificateTypes, repository.CVMasterItem{ID: cert, Name: "Course From The Master Screen", HasExpiry: true})
	assert.Contains(t, l.DocumentTypes, repository.CVMasterItem{ID: doc, Name: "Document From The Master Screen"})
}
