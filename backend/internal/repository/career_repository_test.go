package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type careerWorld struct {
	master, engineer uuid.UUID
	tanker, tug      uuid.UUID
	admin            uuid.UUID
}

func setupCareerWorld(t *testing.T) careerWorld {
	t.Helper()
	truncate(t, "vacancies", "applicants", "ranks", "vessel_types", "files", "admins", "redirects", "contact_messages")
	return careerWorld{
		master:   mustID(t, `INSERT INTO ranks (name, short_code, department, sort_order) VALUES ('Master', 'MST', 'deck', 1) RETURNING id`),
		engineer: mustID(t, `INSERT INTO ranks (name, short_code, department, sort_order) VALUES ('Chief Engineer', 'CE', 'engine', 2) RETURNING id`),
		tanker:   mustID(t, `INSERT INTO vessel_types (name, sort_order) VALUES ('Tanker', 1) RETURNING id`),
		tug:      mustID(t, `INSERT INTO vessel_types (name, sort_order) VALUES ('Tug', 2) RETURNING id`),
		admin:    mustID(t, `INSERT INTO admins (name, email, password_hash) VALUES ('Ambon', 'a@example.com', 'h') RETURNING id`),
	}
}

func date(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return d
}

func newVacancy(w careerWorld, title, slug, status string) model.Vacancy {
	now := time.Now().UTC()
	v := model.Vacancy{
		Title: title, Slug: slug, RankID: w.master, Department: model.DepartmentDeck, Location: "Singapore",
		EmploymentType: model.EmploymentContract, RequiredDocuments: []string{"Passport"}, Status: status,
	}
	if status != model.VacancyDraft {
		v.PublishedAt = &now
	}
	return v
}

func createVacancy(t *testing.T, repo *repository.VacancyRepository, v model.Vacancy, publicID string) uuid.UUID {
	t.Helper()
	id, err := repo.Create(context.Background(), v, publicID)
	require.NoError(t, err)
	return id
}

func TestVacancyUpdateStoresRedirectInTheSameTransaction(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	repo := repository.NewVacancyRepository(testDB)
	id := createVacancy(t, repo, newVacancy(w, "Master Tanker", "master-tanker", model.VacancyOpen), "VAC222222")

	cur, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	cur.Slug = "master-lng"
	require.NoError(t, repo.Update(ctx, cur, &model.Redirect{FromPath: "/career/master-tanker", ToPath: "/career/master-lng"}))

	rd, err := repository.NewRedirectRepository(testDB).Resolve(ctx, "/career/master-tanker")
	require.NoError(t, err)
	assert.Equal(t, "/career/master-lng", rd.ToPath)
	got, err := repo.GetByPublicID(ctx, "VAC222222")
	require.NoError(t, err)
	assert.Equal(t, "master-lng", got.Slug)
	assert.Equal(t, []string{"Passport"}, got.RequiredDocuments)

	conflict := newVacancy(w, "Other", "other", model.VacancyDraft)
	createVacancy(t, repo, conflict, "VAC333333")
	cur.Slug = "other"
	err = repo.Update(ctx, cur, &model.Redirect{FromPath: "/career/master-lng", ToPath: "/career/other"})
	assert.Error(t, err, "slug must stay unique")
	_, err = repository.NewRedirectRepository(testDB).Resolve(ctx, "/career/master-lng")
	assert.ErrorIs(t, err, repository.ErrNotFound, "a failed update must not leave a redirect behind")
}

func TestVacancySlugTakenIgnoresDeletedVacancies(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	repo := repository.NewVacancyRepository(testDB)
	id := createVacancy(t, repo, newVacancy(w, "Master", "master", model.VacancyDraft), "VAC222222")

	taken, err := repo.SlugTaken(ctx, "master", uuid.Nil)
	require.NoError(t, err)
	assert.True(t, taken)
	require.NoError(t, repo.SoftDelete(ctx, id))
	taken, err = repo.SlugTaken(ctx, "master", uuid.Nil)
	require.NoError(t, err)
	assert.False(t, taken)
	assert.ErrorIs(t, repo.SoftDelete(ctx, id), repository.ErrNotFound)
}

func TestVacancyListCountsApplicationsAndFilters(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	repo := repository.NewVacancyRepository(testDB)
	open := newVacancy(w, "Master Tanker", "master-tanker", model.VacancyOpen)
	open.VesselTypeID = &w.tanker
	openID := createVacancy(t, repo, open, "VAC222222")
	eng := newVacancy(w, "Chief Engineer", "chief-engineer", model.VacancyDraft)
	eng.RankID, eng.Department = w.engineer, model.DepartmentEngine
	createVacancy(t, repo, eng, "VAC333333")
	applicants := repository.NewApplicantRepository(testDB)
	for i, email := range []string{"a@example.com", "b@example.com"} {
		require.NoError(t, applicants.Create(ctx, application(w, openID, "APP22222"+string(rune('2'+i)), email)))
	}
	exec(t, `UPDATE applicants SET stage = 'screening' WHERE email = 'b@example.com'`)

	page, err := repo.List(ctx, model.VacancyFilter{}, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total)
	for _, v := range page.Items {
		if v.PublicID == "VAC222222" {
			assert.Equal(t, 2, v.ApplicationsCount)
			assert.Equal(t, 1, v.NewApplicationsCount)
		}
	}

	tests := []struct {
		name   string
		filter model.VacancyFilter
		want   []string
	}{
		{"by title", model.VacancyFilter{Q: "tanker"}, []string{"VAC222222"}},
		{"by public id", model.VacancyFilter{Q: "vac3333"}, []string{"VAC333333"}},
		{"by status", model.VacancyFilter{Statuses: []string{model.VacancyDraft}}, []string{"VAC333333"}},
		{"by rank", model.VacancyFilter{RankIDs: []uuid.UUID{w.engineer}}, []string{"VAC333333"}},
		{"by vessel type", model.VacancyFilter{VesselTypeIDs: []uuid.UUID{w.tanker}}, []string{"VAC222222"}},
		{"by department", model.VacancyFilter{Departments: []string{model.DepartmentEngine}}, []string{"VAC333333"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := repo.List(ctx, tc.filter, 20, 0)
			require.NoError(t, err)
			var ids []string
			for _, v := range got.Items {
				ids = append(ids, v.PublicID)
			}
			assert.Equal(t, tc.want, ids)
		})
	}

	require.NoError(t, repo.SoftDelete(ctx, openID))
	count, err := applicants.List(ctx, model.ApplicantFilter{}, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, count.Total, "applicants keep their soft-deleted vacancy")
	assert.Equal(t, "Master Tanker", count.Items[0].VacancyTitle)
}

func TestListOpenOnlyReturnsVacanciesAcceptingApplications(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	repo := repository.NewVacancyRepository(testDB)
	today := date(t, "2026-10-21")
	yesterday, tomorrow := today.AddDate(0, 0, -1), today.AddDate(0, 0, 1)

	forever := newVacancy(w, "Master Tanker", "master-tanker", model.VacancyOpen)
	forever.VesselTypeID = &w.tanker
	forever.DescriptionHTML = "<p>Command a product tanker</p>"
	createVacancy(t, repo, forever, "VAC222222")
	lastDay := newVacancy(w, "Chief Engineer", "chief-engineer", model.VacancyOpen)
	lastDay.RankID, lastDay.Department, lastDay.Location, lastDay.ClosingDate = w.engineer, model.DepartmentEngine, "Rotterdam", &today
	createVacancy(t, repo, lastDay, "VAC333333")
	expired := newVacancy(w, "Expired", "expired", model.VacancyOpen)
	expired.ClosingDate = &yesterday
	createVacancy(t, repo, expired, "VAC444444")
	createVacancy(t, repo, newVacancy(w, "Draft", "draft", model.VacancyDraft), "VAC555555")
	createVacancy(t, repo, newVacancy(w, "Closed", "closed", model.VacancyClosed), "VAC666666")
	deleted := createVacancy(t, repo, newVacancy(w, "Deleted", "deleted", model.VacancyOpen), "VAC777777")
	require.NoError(t, repo.SoftDelete(ctx, deleted))
	later := newVacancy(w, "Later", "later", model.VacancyOpen)
	later.ClosingDate = &tomorrow
	createVacancy(t, repo, later, "VAC888888")

	list := func(f model.PublicVacancyFilter) []string {
		f.Today = today
		page, err := repo.ListOpen(ctx, f, 20, 0)
		require.NoError(t, err)
		assert.Equal(t, len(page.Items), page.Total)
		var slugs []string
		for _, v := range page.Items {
			slugs = append(slugs, v.Slug)
		}
		return slugs
	}

	assert.ElementsMatch(t, []string{"master-tanker", "chief-engineer", "later"}, list(model.PublicVacancyFilter{}))
	assert.Equal(t, []string{"chief-engineer"}, list(model.PublicVacancyFilter{Q: "chief"}))
	assert.Equal(t, []string{"chief-engineer"}, list(model.PublicVacancyFilter{Q: "ENGINEER"}), "rank names are searched case-insensitively")
	assert.Equal(t, []string{"master-tanker"}, list(model.PublicVacancyFilter{Q: "product tanker"}), "descriptions are searched")
	assert.Equal(t, []string{"chief-engineer"}, list(model.PublicVacancyFilter{RankID: &w.engineer}))
	assert.Equal(t, []string{"master-tanker"}, list(model.PublicVacancyFilter{VesselTypeID: &w.tanker}))
	assert.Equal(t, []string{"chief-engineer"}, list(model.PublicVacancyFilter{Department: model.DepartmentEngine}))
	assert.Equal(t, []string{"chief-engineer"}, list(model.PublicVacancyFilter{Location: "rotterdam"}))
	assert.Empty(t, list(model.PublicVacancyFilter{Q: "%"}), "LIKE wildcards are escaped")

	card, err := repo.ListOpen(ctx, model.PublicVacancyFilter{Today: today}, 2, 0)
	require.NoError(t, err)
	assert.Len(t, card.Items, 2)
	assert.Equal(t, 3, card.Total)
	assert.Empty(t, card.Items[0].DescriptionHTML, "cards do not carry the description")

	detail, err := repo.GetBySlug(ctx, "closed")
	require.NoError(t, err)
	assert.Equal(t, model.VacancyClosed, detail.Status)
	_, err = repo.GetBySlug(ctx, "deleted")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestFilterOptionsListOnlyValuesUsedByOpenVacancies(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	repo := repository.NewVacancyRepository(testDB)
	today := date(t, "2026-10-21")
	open := newVacancy(w, "Master Tanker", "master-tanker", model.VacancyOpen)
	open.VesselTypeID = &w.tanker
	createVacancy(t, repo, open, "VAC222222")
	hidden := newVacancy(w, "Chief Engineer", "chief-engineer", model.VacancyDraft)
	hidden.RankID, hidden.Department, hidden.Location, hidden.VesselTypeID = w.engineer, model.DepartmentEngine, "Rotterdam", &w.tug
	createVacancy(t, repo, hidden, "VAC333333")

	got, err := repo.FilterOptions(ctx, today)
	require.NoError(t, err)
	require.Len(t, got.Ranks, 1)
	assert.Equal(t, "Master", got.Ranks[0].Name)
	assert.Equal(t, "MST", got.Ranks[0].ShortCode)
	require.Len(t, got.VesselTypes, 1)
	assert.Equal(t, "Tanker", got.VesselTypes[0].Name)
	assert.Equal(t, []string{model.DepartmentDeck}, got.Departments)
	assert.Equal(t, []string{"Singapore"}, got.Locations)
}

func TestRankAndVesselTypeLookupsRequireActiveRecords(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	repo := repository.NewVacancyRepository(testDB)
	exec(t, `UPDATE ranks SET is_active = false WHERE id = $1`, w.engineer)
	exec(t, `UPDATE vessel_types SET deleted_at = now() WHERE id = $1`, w.tug)

	for id, want := range map[uuid.UUID]bool{w.master: true, w.engineer: false, uuid.New(): false} {
		ok, err := repo.RankActive(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, want, ok)
	}
	for id, want := range map[uuid.UUID]bool{w.tanker: true, w.tug: false} {
		ok, err := repo.VesselTypeActive(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, want, ok)
	}
}

func application(w careerWorld, vacancyID uuid.UUID, publicID, email string) model.NewApplication {
	now := time.Now().UTC()
	return model.NewApplication{
		ID: uuid.New(), PublicID: publicID, VacancyID: vacancyID, FullName: "Siti Aminah", Email: email, Phone: "+6281234567890",
		DateOfBirth: time.Date(1990, 5, 1, 0, 0, 0, 0, time.UTC), Nationality: "Indonesian", RankAppliedID: w.master,
		YearsExperience: 8, Message: "Hello", ConsentAt: now, IPAddress: "203.0.113.9", DuplicateSince: now.AddDate(0, 0, -30),
	}
}

func storedFile(t *testing.T, key string) uuid.UUID {
	t.Helper()
	return mustID(t, `INSERT INTO files (storage_key, original_name, mime_type, size_original, size_stored, sha256)
		VALUES ($1, 'cv.docx', 'application/pdf', 100, 80, 'abc') RETURNING id`, key)
}

func TestApplicantCreateStoresFilesAndFirstStageLog(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	vacancies := repository.NewVacancyRepository(testDB)
	repo := repository.NewApplicantRepository(testDB)
	vid := createVacancy(t, vacancies, newVacancy(w, "Master", "master", model.VacancyOpen), "VAC222222")
	n := application(w, vid, "APP222222", "Siti@Example.com")
	n.Files = []model.NewApplicantFile{{FileID: storedFile(t, "applicant/a.pdf"), Kind: model.ApplicantFileCV}, {FileID: storedFile(t, "applicant/b.pdf"), Kind: model.ApplicantFileDocument}}
	require.NoError(t, repo.Create(ctx, n))

	got, err := repo.GetByPublicID(ctx, "APP222222")
	require.NoError(t, err)
	assert.Equal(t, "Siti Aminah", got.FullName)
	assert.Equal(t, "Master", got.RankAppliedName)
	assert.Equal(t, "VAC222222", got.VacancyPublicID)
	assert.Equal(t, model.StageNew, got.Stage)
	assert.Equal(t, "203.0.113.9", got.IPAddress)
	require.NotNil(t, got.YearsExperience)
	assert.Equal(t, 8, *got.YearsExperience)
	require.Len(t, got.Files, 2)
	assert.Equal(t, model.ApplicantFileCV, got.Files[0].Kind, "the CV is listed first")
	require.Len(t, got.StageLogs, 1)
	assert.Equal(t, "", got.StageLogs[0].FromStage)
	assert.Equal(t, model.StageNew, got.StageLogs[0].ToStage)
	assert.Nil(t, got.StageLogs[0].AdminID)

	exec(t, `UPDATE files SET deleted_at = now() WHERE storage_key = 'applicant/b.pdf'`)
	got, err = repo.GetByPublicID(ctx, "APP222222")
	require.NoError(t, err)
	assert.Len(t, got.Files, 1, "deleted files are not listed")
}

func TestApplicantCreateRefusesDuplicateInsideTheWindow(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	vacancies := repository.NewVacancyRepository(testDB)
	repo := repository.NewApplicantRepository(testDB)
	vid := createVacancy(t, vacancies, newVacancy(w, "Master", "master", model.VacancyOpen), "VAC222222")
	other := createVacancy(t, vacancies, newVacancy(w, "Other", "other", model.VacancyOpen), "VAC333333")
	require.NoError(t, repo.Create(ctx, application(w, vid, "APP222222", "siti@example.com")))

	dup := application(w, vid, "APP333333", "SITI@example.com")
	assert.ErrorIs(t, repo.Create(ctx, dup), repository.ErrDuplicateApplication, "email comparison ignores case")
	recent, err := repo.HasRecent(ctx, dup)
	require.NoError(t, err)
	assert.True(t, recent)

	assert.NoError(t, repo.Create(ctx, application(w, other, "APP444444", "siti@example.com")), "another vacancy is allowed")

	exec(t, `UPDATE applicants SET created_at = now() - interval '31 days' WHERE public_id = 'APP222222'`)
	assert.NoError(t, repo.Create(ctx, dup), "the guard only looks 30 days back")

	exec(t, `UPDATE applicants SET deleted_at = now() WHERE public_id = 'APP333333'`)
	again := application(w, vid, "APP555555", "siti@example.com")
	assert.NoError(t, repo.Create(ctx, again), "deleted applications do not block a new one")
}

func TestApplicantCreateLeavesNothingBehindOnFailure(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	vacancies := repository.NewVacancyRepository(testDB)
	repo := repository.NewApplicantRepository(testDB)
	vid := createVacancy(t, vacancies, newVacancy(w, "Master", "master", model.VacancyOpen), "VAC222222")
	n := application(w, vid, "APP222222", "siti@example.com")
	n.Files = []model.NewApplicantFile{{FileID: uuid.New(), Kind: model.ApplicantFileCV}}

	assert.Error(t, repo.Create(ctx, n), "a file that does not exist breaks the transaction")
	var count int
	require.NoError(t, testDB.QueryRow(ctx, `SELECT count(*) FROM applicants`).Scan(&count))
	assert.Zero(t, count)
}

func TestApplicantChangeStageLogsTheMoveAndRejectsStaleStage(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	vacancies := repository.NewVacancyRepository(testDB)
	repo := repository.NewApplicantRepository(testDB)
	vid := createVacancy(t, vacancies, newVacancy(w, "Master", "master", model.VacancyOpen), "VAC222222")
	n := application(w, vid, "APP222222", "siti@example.com")
	require.NoError(t, repo.Create(ctx, n))

	require.NoError(t, repo.ChangeStage(ctx, n.ID, model.StageNew, model.StageInterview, "Strong profile", w.admin))
	assert.ErrorIs(t, repo.ChangeStage(ctx, n.ID, model.StageNew, model.StageRejected, "late", w.admin), repository.ErrStateChanged)

	got, err := repo.GetByPublicID(ctx, "APP222222")
	require.NoError(t, err)
	assert.Equal(t, model.StageInterview, got.Stage)
	require.Len(t, got.StageLogs, 2)
	last := got.StageLogs[1]
	assert.Equal(t, model.StageNew, last.FromStage)
	assert.Equal(t, model.StageInterview, last.ToStage)
	assert.Equal(t, "Strong profile", last.Note)
	require.NotNil(t, last.AdminID)
	assert.Equal(t, w.admin, *last.AdminID)
	assert.Equal(t, "Ambon", last.AdminName)
}

func TestApplicantListFiltersAndStageCounts(t *testing.T) {
	w := setupCareerWorld(t)
	ctx := context.Background()
	vacancies := repository.NewVacancyRepository(testDB)
	repo := repository.NewApplicantRepository(testDB)
	v1 := createVacancy(t, vacancies, newVacancy(w, "Master", "master", model.VacancyOpen), "VAC222222")
	v2 := createVacancy(t, vacancies, newVacancy(w, "Engineer", "engineer", model.VacancyOpen), "VAC333333")
	a := application(w, v1, "APP222222", "siti@example.com")
	b := application(w, v1, "APP333333", "budi@example.com")
	b.FullName, b.RankAppliedID = "Budi Santoso", w.engineer
	c := application(w, v2, "APP444444", "dewi@example.com")
	c.FullName = "Dewi Lestari"
	for _, n := range []model.NewApplication{a, b, c} {
		require.NoError(t, repo.Create(ctx, n))
	}
	exec(t, `UPDATE applicants SET stage = 'screening' WHERE public_id = 'APP333333'`)
	exec(t, `UPDATE applicants SET created_at = '2026-09-01T10:00:00Z' WHERE public_id = 'APP444444'`)
	from, before := date(t, "2026-10-01").UTC(), date(t, "2026-12-01").UTC()

	tests := []struct {
		name   string
		filter model.ApplicantFilter
		want   []string
	}{
		{"all newest first", model.ApplicantFilter{}, []string{"APP333333", "APP222222", "APP444444"}},
		{"by name", model.ApplicantFilter{Q: "budi"}, []string{"APP333333"}},
		{"by email", model.ApplicantFilter{Q: "dewi@"}, []string{"APP444444"}},
		{"by public id", model.ApplicantFilter{Q: "app2222"}, []string{"APP222222"}},
		{"by vacancy", model.ApplicantFilter{VacancyPublic: "VAC333333"}, []string{"APP444444"}},
		{"by stage", model.ApplicantFilter{Stages: []string{model.StageScreening}}, []string{"APP333333"}},
		{"by rank", model.ApplicantFilter{RankIDs: []uuid.UUID{w.engineer}}, []string{"APP333333"}},
		{"by date range", model.ApplicantFilter{CreatedFrom: &from, CreatedBefore: &before}, []string{"APP333333", "APP222222"}},
		{"sorted by name", model.ApplicantFilter{Sort: "full_name"}, []string{"APP333333", "APP444444", "APP222222"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			page, err := repo.List(ctx, tc.filter, 20, 0)
			require.NoError(t, err)
			var ids []string
			for _, a := range page.Items {
				ids = append(ids, a.PublicID)
			}
			assert.Equal(t, tc.want, ids)
			assert.Equal(t, len(tc.want), page.Total)
		})
	}

	counts, err := repo.StageCounts(ctx, model.ApplicantFilter{VacancyPublic: "VAC222222", Stages: []string{model.StageNew}})
	require.NoError(t, err)
	assert.Equal(t, map[string]int{model.StageNew: 1, model.StageScreening: 1}, counts, "the stage filter does not narrow the counts")

	require.NoError(t, repo.SoftDelete(ctx, a.ID))
	assert.ErrorIs(t, repo.SoftDelete(ctx, a.ID), repository.ErrNotFound)
	_, err = repo.GetByPublicID(ctx, "APP222222")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestContactInboxFlagsAndFilters(t *testing.T) {
	setupCareerWorld(t)
	ctx := context.Background()
	repo := repository.NewContactRepository(testDB)
	create := func(name, subject string) model.ContactMessage {
		m, err := repo.Create(ctx, model.ContactMessage{Name: name, Email: name + "@example.com", Subject: subject, Message: "Hi", IPAddress: "203.0.113.9"})
		require.NoError(t, err)
		return m
	}
	first, second, third := create("ann", "Crew request"), create("bob", "Quotation"), create("cy", "Other")
	assert.Equal(t, "203.0.113.9", first.IPAddress)
	assert.False(t, first.IsRead)

	read := true
	archived := true
	m, err := repo.SetFlags(ctx, second.ID, &read, nil)
	require.NoError(t, err)
	assert.True(t, m.IsRead)
	assert.False(t, m.IsArchived)
	_, err = repo.SetFlags(ctx, third.ID, nil, &archived)
	require.NoError(t, err)
	_, err = repo.SetFlags(ctx, uuid.New(), &read, nil)
	assert.ErrorIs(t, err, repository.ErrNotFound)

	n, err := repo.UnreadCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, n, "archived and read messages are not unread")

	tests := []struct {
		status, q string
		want      []string
	}{
		{model.ContactAll, "", []string{"cy", "bob", "ann"}},
		{model.ContactUnread, "", []string{"ann"}},
		{model.ContactRead, "", []string{"bob"}},
		{model.ContactArchived, "", []string{"cy"}},
		{model.ContactAll, "quot", []string{"bob"}},
		{model.ContactAll, "ann@", []string{"ann"}},
	}
	for _, tc := range tests {
		t.Run(tc.status+tc.q, func(t *testing.T) {
			page, err := repo.List(ctx, model.ContactFilter{Status: tc.status, Q: tc.q}, 20, 0)
			require.NoError(t, err)
			var names []string
			for _, m := range page.Items {
				names = append(names, m.Name)
			}
			assert.Equal(t, tc.want, names)
		})
	}

	got, err := repo.Get(ctx, first.ID)
	require.NoError(t, err)
	assert.False(t, got.IsRead, "viewing a message does not mark it read")
}
