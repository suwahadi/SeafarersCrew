package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

func listAll(q repository.ListQuery) repository.ListQuery {
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 100
	}
	return q
}

func TestCompanyRepositoryCRUDAndFilters(t *testing.T) {
	truncate(t, "companies")
	ctx := context.Background()
	repo := repository.NewCompanyRepository(testDB)

	acme, err := repo.Insert(ctx, model.Company{MasterBase: model.MasterBase{IsActive: true}, Code: "ACME", Name: "Acme Shipping", Type: "both", Country: "Indonesia"})
	require.NoError(t, err)
	assert.Equal(t, 10, acme.SortOrder)
	ocean, err := repo.Insert(ctx, model.Company{MasterBase: model.MasterBase{IsActive: true}, Code: "OCN", Name: "Ocean Principal", Type: "principal"})
	require.NoError(t, err)
	assert.Equal(t, 20, ocean.SortOrder)
	_, err = repo.Insert(ctx, model.Company{MasterBase: model.MasterBase{IsActive: false}, Code: "OLD", Name: "Old Co", Type: "company"})
	require.NoError(t, err)

	page, err := repo.List(ctx, listAll(repository.ListQuery{Filters: map[string]any{"type": "principal"}}))
	require.NoError(t, err)
	assert.Equal(t, 1, page.Total)
	assert.Equal(t, "OCN", page.Items[0].Code)

	page, err = repo.List(ctx, listAll(repository.ListQuery{Filters: map[string]any{"is_active": false}}))
	require.NoError(t, err)
	assert.Equal(t, "OLD", page.Items[0].Code)

	page, err = repo.List(ctx, listAll(repository.ListQuery{Q: "indo"}))
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "ACME", page.Items[0].Code)

	page, err = repo.List(ctx, listAll(repository.ListQuery{Q: "%"}))
	require.NoError(t, err)
	assert.Zero(t, page.Total, "LIKE wildcards in the search term are literal")

	page, err = repo.List(ctx, repository.ListQuery{Page: 2, PerPage: 1, Sort: "name", Desc: true})
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "Ocean Principal", page.Items[0].Name, "second row of name descending")

	page, err = repo.List(ctx, listAll(repository.ListQuery{Sort: "name; DROP TABLE companies"}))
	require.NoError(t, err)
	assert.Equal(t, 3, page.Total, "unknown sort keys fall back to the default order")

	acme.Name = "Acme Maritime"
	updated, err := repo.Update(ctx, acme.ID, acme)
	require.NoError(t, err)
	assert.Equal(t, "Acme Maritime", updated.Name)
	assert.Equal(t, 10, updated.SortOrder, "updates keep the sort order")

	require.NoError(t, repo.SoftDelete(ctx, acme.ID))
	_, err = repo.Get(ctx, acme.ID)
	assert.ErrorIs(t, err, repository.ErrNotFound)
	assert.ErrorIs(t, repo.SoftDelete(ctx, acme.ID), repository.ErrNotFound)
	page, err = repo.List(ctx, listAll(repository.ListQuery{}))
	require.NoError(t, err)
	assert.Equal(t, 2, page.Total, "soft-deleted rows are excluded")

	taken, err := repo.Taken(ctx, "code", "acme", uuid.Nil)
	require.NoError(t, err)
	assert.False(t, taken, "a soft-deleted code may be reused")
	_, err = repo.Insert(ctx, model.Company{MasterBase: model.MasterBase{IsActive: true}, Code: "ACME", Name: "Acme Reborn", Type: "company"})
	require.NoError(t, err, "partial unique index ignores soft-deleted rows")

	_, err = repo.Insert(ctx, model.Company{MasterBase: model.MasterBase{IsActive: true}, Code: "acme", Name: "Clash", Type: "company"})
	assert.ErrorIs(t, err, repository.ErrDuplicate)
}

func TestReorderSetsSortOrderFromPosition(t *testing.T) {
	truncate(t, "vessel_types")
	ctx := context.Background()
	repo := repository.NewVesselTypeRepository(testDB)
	var ids []uuid.UUID
	for _, name := range []string{"AHTS", "Tanker", "Tug"} {
		v, err := repo.Insert(ctx, model.VesselType{MasterBase: model.MasterBase{IsActive: true}, Name: name})
		require.NoError(t, err)
		ids = append(ids, v.ID)
	}
	require.NoError(t, repo.Reorder(ctx, []uuid.UUID{ids[2], ids[0], ids[1]}))
	active, err := repo.Active(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"Tug", "AHTS", "Tanker"}, []string{active[0].Name, active[1].Name, active[2].Name})
}

func TestVesselRepositoryHandlesNullableNumbers(t *testing.T) {
	truncate(t, "vessels", "vessel_types")
	ctx := context.Background()
	repo := repository.NewVesselRepository(testDB)
	types := repository.NewVesselTypeRepository(testDB)
	ahts, err := types.Insert(ctx, model.VesselType{MasterBase: model.MasterBase{IsActive: true}, Name: "AHTS"})
	require.NoError(t, err)

	bare, err := repo.Insert(ctx, model.Vessel{MasterBase: model.MasterBase{IsActive: true}, Name: "MV Bare", IMONumber: "9074729", Status: "active"})
	require.NoError(t, err)
	assert.Nil(t, bare.GRT)
	assert.Nil(t, bare.YearBuilt)

	grt := decimal.RequireFromString("1234.50")
	year := 2015
	full, err := repo.Insert(ctx, model.Vessel{MasterBase: model.MasterBase{IsActive: true}, Name: "MV Full", IMONumber: "9176187",
		VesselTypeID: &ahts.ID, GRT: &grt, YearBuilt: &year, Status: "laid_up"})
	require.NoError(t, err)
	assert.True(t, grt.Equal(*full.GRT))
	assert.Equal(t, 2015, *full.YearBuilt)

	page, err := repo.List(ctx, listAll(repository.ListQuery{Filters: map[string]any{"vessel_type_id": ahts.ID, "status": "laid_up"}}))
	require.NoError(t, err)
	require.Equal(t, 1, page.Total)
	assert.Equal(t, "MV Full", page.Items[0].Name)

	usages, err := types.Usages(ctx, ahts.ID)
	require.NoError(t, err)
	assert.Equal(t, []model.Usage{{Label: "vessels", Count: 1}}, usages)
}

func TestRankUsagesCountOnlyActiveReferences(t *testing.T) {
	truncate(t, "crews", "ranks", "crew_statuses")
	ctx := context.Background()
	ranks := repository.NewRankRepository(testDB)
	rank, err := ranks.Insert(ctx, model.Rank{MasterBase: model.MasterBase{IsActive: true}, Name: "Master", ShortCode: "MST", Department: "deck"})
	require.NoError(t, err)

	usages, err := ranks.Usages(ctx, rank.ID)
	require.NoError(t, err)
	assert.Empty(t, usages)

	var statusID uuid.UUID
	require.NoError(t, testDB.QueryRow(ctx, `INSERT INTO crew_statuses (code, name, is_system) VALUES ('available', 'Available', true) RETURNING id`).Scan(&statusID))
	_, err = testDB.Exec(ctx, `INSERT INTO crews (public_id, full_name, crew_status_id, applied_rank_id) VALUES ('ABCDEFGH2', 'A', $1, $2), ('ABCDEFGH3', 'B', $1, $2)`, statusID, rank.ID)
	require.NoError(t, err)
	_, err = testDB.Exec(ctx, `UPDATE crews SET deleted_at = now() WHERE public_id = 'ABCDEFGH3'`)
	require.NoError(t, err)

	usages, err = ranks.Usages(ctx, rank.ID)
	require.NoError(t, err)
	assert.Equal(t, []model.Usage{{Label: "crews", Count: 1}}, usages)

	taken, err := ranks.Taken(ctx, "short_code", "mst", uuid.Nil)
	require.NoError(t, err)
	assert.True(t, taken)
	taken, err = ranks.Taken(ctx, "short_code", "mst", rank.ID)
	require.NoError(t, err)
	assert.False(t, taken)
}

func TestCurrencyUsagesFollowTheCode(t *testing.T) {
	truncate(t, "currencies", "crews", "crew_statuses")
	ctx := context.Background()
	repo := repository.NewCurrencyRepository(testDB)
	usd, err := repo.Insert(ctx, model.Currency{MasterBase: model.MasterBase{IsActive: true}, Code: "USD", Decimals: 2})
	require.NoError(t, err)

	var statusID, crewID uuid.UUID
	require.NoError(t, testDB.QueryRow(ctx, `INSERT INTO crew_statuses (name) VALUES ('Available') RETURNING id`).Scan(&statusID))
	require.NoError(t, testDB.QueryRow(ctx, `INSERT INTO crews (public_id, full_name, crew_status_id) VALUES ('ABCDEFGH4', 'C', $1) RETURNING id`, statusID).Scan(&crewID))
	_, err = testDB.Exec(ctx, `INSERT INTO crew_bank_accounts (crew_id, bank_name, account_name, account_number, currency_code) VALUES ($1, 'BRI', 'C', '1', 'USD')`, crewID)
	require.NoError(t, err)

	usages, err := repo.Usages(ctx, usd.ID)
	require.NoError(t, err)
	assert.Equal(t, []model.Usage{{Label: "bank accounts", Count: 1}}, usages)
}

func TestComplianceMatrixReplaceKeepsInactiveRows(t *testing.T) {
	truncate(t, "compliance_requirements", "ranks", "document_types", "certificate_types")
	ctx := context.Background()
	ranks := repository.NewRankRepository(testDB)
	docs := repository.NewDocumentTypeRepository(testDB)
	certs := repository.NewCertificateTypeRepository(testDB)
	matrix := repository.NewComplianceMatrixRepository(testDB)

	master, err := ranks.Insert(ctx, model.Rank{MasterBase: model.MasterBase{IsActive: true}, Name: "Master", ShortCode: "MST", Department: "deck"})
	require.NoError(t, err)
	retired, err := ranks.Insert(ctx, model.Rank{MasterBase: model.MasterBase{IsActive: true}, Name: "Radio Officer", ShortCode: "RO", Department: "deck"})
	require.NoError(t, err)
	passport, err := docs.Insert(ctx, model.DocumentType{MasterBase: model.MasterBase{IsActive: true}, Name: "Passport", HasExpiry: true})
	require.NoError(t, err)
	bst, err := certs.Insert(ctx, model.CertificateType{MasterBase: model.MasterBase{IsActive: true}, Name: "BST", Category: "cop", HasExpiry: true})
	require.NoError(t, err)

	require.NoError(t, matrix.ReplaceActive(ctx, []model.ComplianceCell{
		{RankID: master.ID, DocumentTypeID: &passport.ID},
		{RankID: master.ID, CertificateTypeID: &bst.ID},
		{RankID: retired.ID, DocumentTypeID: &passport.ID},
	}))
	cells, err := matrix.ActiveCells(ctx)
	require.NoError(t, err)
	assert.Len(t, cells, 3)

	retired.IsActive = false
	_, err = ranks.Update(ctx, retired.ID, retired)
	require.NoError(t, err)

	require.NoError(t, matrix.ReplaceActive(ctx, []model.ComplianceCell{{RankID: master.ID, CertificateTypeID: &bst.ID}}))
	cells, err = matrix.ActiveCells(ctx)
	require.NoError(t, err)
	require.Len(t, cells, 1)
	assert.Equal(t, bst.ID, *cells[0].CertificateTypeID)

	var total int
	require.NoError(t, testDB.QueryRow(ctx, `SELECT count(*) FROM compliance_requirements`).Scan(&total))
	assert.Equal(t, 2, total, "rows of inactive ranks are kept for when they are reactivated")

	lookup := repository.NewLookupRepository(testDB)
	_, err = lookup.CompanyType(ctx, uuid.New())
	assert.ErrorIs(t, err, repository.ErrNotFound)
	ok, err := lookup.FileExists(ctx, uuid.New())
	require.NoError(t, err)
	assert.False(t, ok)
}
