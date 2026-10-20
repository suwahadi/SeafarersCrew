package repository_test

import (
	"context"
	"html"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/database/seeders"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

func money(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestQuotationRepositoryLifecycle(t *testing.T) {
	w := setupCrewWorld(t)
	ctx := context.Background()
	exec(t, `INSERT INTO currencies (code, decimals) VALUES ('IDR', 0)`)
	repo := repository.NewQuotationRepository(testDB)

	refs, err := repo.CheckRefs(ctx, w.principalA, &w.vessel, "IDR", []uuid.UUID{w.master, uuid.New()})
	require.NoError(t, err)
	assert.Equal(t, "principal", refs.ClientType)
	assert.True(t, refs.ClientActive)
	assert.True(t, refs.VesselExists)
	assert.True(t, refs.CurrencyActive)
	assert.Equal(t, int32(0), refs.CurrencyDecimal)
	assert.Equal(t, 1, refs.MissingRanks)

	refs, err = repo.CheckRefs(ctx, uuid.New(), nil, "XXX", nil)
	require.NoError(t, err)
	assert.Empty(t, refs.ClientType)
	assert.False(t, refs.CurrencyActive)

	crew, days, rate := 2, 30, money("250.50")
	q := model.Quotation{PublicID: "QTAAAAAA2", ClientCompanyID: w.principalA, VesselID: &w.vessel, CurrencyCode: "USD",
		QuotationDate: d(2026, 10, 1), ValidUntil: d(2026, 10, 31), TaxPercent: money("10"), Subtotal: money("16030"),
		TaxAmount: money("1603"), GrandTotal: money("17633"), Lines: []model.QuotationLine{
			{LineType: model.LineDailyRate, RankID: &w.master, CrewCount: &crew, Days: &days, DailyRate: &rate, Amount: money("15030")},
			{LineType: model.LineLumpSum, Description: "Mobilisation", Amount: money("1000")},
		}}
	id, err := repo.Create(ctx, q)
	require.NoError(t, err)

	got, err := repo.GetByPublicID(ctx, "QTAAAAAA2")
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "Alpha Offshore", got.ClientName)
	assert.Equal(t, "MV Ocean", got.VesselName)
	assert.Equal(t, int32(2), got.CurrencyDecimal)
	assert.Equal(t, model.QuotationDraft, got.Status)
	require.Len(t, got.Lines, 2)
	assert.Equal(t, "Master", got.Lines[0].RankName)
	assert.True(t, rate.Equal(*got.Lines[0].DailyRate))
	assert.True(t, money("17633").Equal(got.GrandTotal))

	got.Lines = got.Lines[1:]
	got.Subtotal, got.GrandTotal = money("1000"), money("1100")
	require.NoError(t, repo.UpdateDraft(ctx, got))
	got, err = repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Len(t, got.Lines, 1)

	ok, err := repo.SetStatus(ctx, id, model.QuotationSent, model.QuotationAccepted)
	require.NoError(t, err)
	assert.False(t, ok, "the from status must match")
	ok, err = repo.SetStatus(ctx, id, model.QuotationDraft, model.QuotationSent)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.ErrorIs(t, repo.UpdateDraft(ctx, got), repository.ErrNotFound, "sent quotations are locked")

	q2 := q
	q2.PublicID, q2.ClientCompanyID, q2.CurrencyCode, q2.GrandTotal = "QTBBBBBB2", w.principalB, "IDR", money("5")
	q2.ValidUntil = d(2026, 12, 31)
	q2.Lines = []model.QuotationLine{{LineType: model.LineLumpSum, Description: "Fee", Amount: money("5")}}
	_, err = repo.Create(ctx, q2)
	require.NoError(t, err)

	list := func(f model.QuotationFilter) []string {
		f.Limit = 50
		page, err := repo.List(ctx, f)
		require.NoError(t, err)
		ids := make([]string, len(page.Items))
		for i, it := range page.Items {
			ids[i] = it.PublicID
		}
		assert.Equal(t, len(ids), page.Total)
		return ids
	}
	assert.Equal(t, []string{"QTBBBBBB2"}, list(model.QuotationFilter{Q: "beta"}))
	assert.Equal(t, []string{"QTAAAAAA2"}, list(model.QuotationFilter{Statuses: []string{"sent"}}))
	assert.Equal(t, []string{"QTBBBBBB2"}, list(model.QuotationFilter{Currencies: []string{"IDR"}}))
	assert.Equal(t, []string{"QTAAAAAA2", "QTBBBBBB2"}, list(model.QuotationFilter{Sort: "grand_total", Desc: true}))
	from := d(2026, 10, 2)
	assert.Empty(t, list(model.QuotationFilter{DateFrom: &from}))

	n, err := repo.ExpireOverdue(ctx, d(2026, 11, 1))
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	got, err = repo.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, model.QuotationExpired, got.Status)

	require.NoError(t, repo.SoftDelete(ctx, id))
	_, err = repo.GetByPublicID(ctx, "QTAAAAAA2")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestDocumentTemplateRepositoryKeepsOneDefaultPerKind(t *testing.T) {
	truncate(t, "document_templates")
	ctx := context.Background()
	repo := repository.NewDocumentTemplateRepository(testDB)
	a, err := repo.Save(ctx, model.DocumentTemplate{Name: "A", Kind: "sea", PaperSize: "a4", Orientation: "portrait", BodyHTML: "a", IsDefault: true})
	require.NoError(t, err)
	b, err := repo.Save(ctx, model.DocumentTemplate{Name: "B", Kind: "sea", PaperSize: "letter", Orientation: "landscape", BodyHTML: "b", IsDefault: true})
	require.NoError(t, err)
	_, err = repo.Save(ctx, model.DocumentTemplate{Name: "C", Kind: "cv", PaperSize: "a4", Orientation: "portrait", BodyHTML: "c", IsDefault: true})
	require.NoError(t, err)

	def, err := repo.Default(ctx, "sea")
	require.NoError(t, err)
	assert.Equal(t, b.ID, def.ID)
	a, err = repo.Get(ctx, a.ID)
	require.NoError(t, err)
	assert.False(t, a.IsDefault)

	a.IsDefault = true
	a.Name = "A2"
	_, err = repo.Save(ctx, a)
	require.NoError(t, err)
	def, err = repo.Default(ctx, "sea")
	require.NoError(t, err)
	assert.Equal(t, "A2", def.Name)

	sea, err := repo.List(ctx, "sea")
	require.NoError(t, err)
	assert.Len(t, sea, 2)
	n, err := repo.CountByKind(ctx, "cv")
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	require.NoError(t, repo.SoftDelete(ctx, b.ID))
	_, err = repo.Get(ctx, b.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	all, err := repo.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestDocumentSourcesAndGeneratedHistory(t *testing.T) {
	w := setupCrewWorld(t)
	truncate(t, "allowance_types", "document_templates")
	ctx := context.Background()
	crewRepo := repository.NewCrewRepository(testDB)
	records := repository.NewCrewRecordRepository(testDB)
	dob := d(1985, 5, 20)
	crewID := createCrew(t, crewRepo, "SRCAAAAA2", "Budi Santoso", &w.master, &dob, "Indonesian")
	exp := d(2030, 1, 1)
	_, err := records.SaveDocument(ctx, repository.TravelDocuments, model.CrewDocument{CrewID: crewID, TypeID: w.passport, Number: "X1234567", ExpiredDate: &exp})
	require.NoError(t, err)
	_, err = records.SaveDocument(ctx, repository.TravelDocuments, model.CrewDocument{CrewID: crewID, TypeID: w.seamanBook, Number: "SB-1", ExpiredDate: &exp})
	require.NoError(t, err)

	contractID := mustID(t, `INSERT INTO contracts (public_id, crew_id, vessel_id, rank_id, principal_id, employer_company_id, sign_on_date,
		duration_months, planned_sign_off_date, currency_code, basic_salary, working_hours_week, leave_days_month, point_of_hire)
		VALUES ('SEAAAAAA2', $1, $2, $3, $4, $5, '2026-11-01', 6, '2027-05-01', 'USD', 8000, 48, 2.5, 'Jakarta') RETURNING id`,
		crewID, w.vessel, w.master, w.principalA, w.principalB)
	allowance := mustID(t, `INSERT INTO allowance_types (name, basis) VALUES ('Overtime', 'fixed_month') RETURNING id`)
	exec(t, `INSERT INTO contract_allowances (contract_id, allowance_type_id, basis, value, monthly_amount) VALUES ($1, $2, 'fixed_month', 1200, 1200)`, contractID, allowance)

	src := repository.NewDocumentSourceRepository(testDB)
	sea, err := src.SEA(ctx, "SEAAAAAA2")
	require.NoError(t, err)
	assert.Equal(t, contractID, sea.ContractID)
	assert.Equal(t, "Budi Santoso", sea.Crew.FullName)
	assert.Equal(t, "X1234567", sea.Crew.PassportNo)
	assert.Equal(t, "SB-1", sea.Crew.SeamanBookNo)
	assert.Equal(t, "MV Ocean", sea.VesselName)
	assert.Equal(t, "9074729", sea.IMONumber)
	assert.Equal(t, "Alpha Offshore", sea.PrincipalName)
	assert.Equal(t, "Beta Marine", sea.EmployerName)
	assert.Equal(t, int32(2), sea.CurrencyDecimal)
	assert.True(t, money("2.5").Equal(*sea.LeaveDays))
	require.Len(t, sea.Allowances, 1)
	assert.True(t, money("1200").Equal(sea.Allowances[0].MonthlyAmount))

	_, err = src.SEA(ctx, "ZZZZZZZZ2")
	assert.ErrorIs(t, err, repository.ErrNotFound)
	profile, err := src.CrewProfile(ctx, "SRCAAAAA2")
	require.NoError(t, err)
	assert.Equal(t, "Master", profile.AppliedRank)
	id, err := src.ContractID(ctx, "SEAAAAAA2")
	require.NoError(t, err)
	assert.Equal(t, contractID, id)

	fileID := mustID(t, `INSERT INTO files (storage_key, original_name, mime_type, size_original, size_stored, sha256)
		VALUES ($1, 'SEA.pdf', 'application/pdf', 100, 80, 'x') RETURNING id`, "generated_documents/"+uuid.NewString()+".pdf")
	history := repository.NewGeneratedDocumentRepository(testDB)
	g, err := history.Create(ctx, model.GeneratedDocument{PublicID: "GENAAAAA2", Kind: "sea", RelatedType: "contract", RelatedID: contractID, FileID: fileID})
	require.NoError(t, err)
	assert.Equal(t, int64(80), g.FileSize)

	_, err = history.Create(ctx, model.GeneratedDocument{PublicID: "GENAAAAA2", Kind: "sea", RelatedType: "contract", RelatedID: contractID, FileID: fileID})
	assert.Error(t, err, "public IDs are unique")

	list, err := history.List(ctx, "contract", &contractID, 10)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	other := uuid.New()
	list, err = history.List(ctx, "contract", &other, 10)
	require.NoError(t, err)
	assert.Empty(t, list)
}

type noSettings struct{}

func (noSettings) SeedDefaults(context.Context, map[string]map[string]any) error { return nil }

func TestReferenceSeedAddsOneDefaultTemplatePerKindOnce(t *testing.T) {
	truncate(t, "document_templates")
	ctx := context.Background()
	require.NoError(t, seeders.Reference(ctx, testDB, noSettings{}, nil))
	require.NoError(t, seeders.Reference(ctx, testDB, noSettings{}, nil))
	repo := repository.NewDocumentTemplateRepository(testDB)
	all, err := repo.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 3)
	for _, tp := range all {
		assert.True(t, tp.IsDefault, tp.Kind)
		assert.Empty(t, service.UnknownTags(tp.Kind, tp.HeaderHTML+tp.BodyHTML+tp.FooterHTML), "seeded %s template uses only known tags", tp.Kind)
		// Only entity encoding of quotes may differ; no markup may be stripped.
		for _, part := range []string{tp.HeaderHTML, tp.BodyHTML, tp.FooterHTML} {
			assert.Equal(t, part, html.UnescapeString(service.NewTemplatePolicy().Sanitize(part)), "seeded %s template survives sanitizing", tp.Kind)
		}
	}
}
