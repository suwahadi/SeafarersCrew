package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type staticList[T any] []T

func (s staticList[T]) Active(context.Context) ([]T, error) { return s, nil }

type memMatrix struct {
	cells []model.ComplianceCell
}

func (m *memMatrix) ActiveCells(context.Context) ([]model.ComplianceCell, error) { return m.cells, nil }

func (m *memMatrix) ReplaceActive(_ context.Context, cells []model.ComplianceCell) error {
	m.cells = cells
	return nil
}

func uuidPtr(id uuid.UUID) *uuid.UUID { return &id }

func TestComplianceMatrixReplace(t *testing.T) {
	rank, doc, cert := uuid.New(), uuid.New(), uuid.New()
	store := &memMatrix{}
	svc := NewComplianceMatrixService(store,
		staticList[model.Rank]{{MasterBase: model.MasterBase{ID: rank}, Name: "Master"}},
		staticList[model.DocumentType]{{MasterBase: model.MasterBase{ID: doc}, Name: "Passport"}},
		staticList[model.CertificateType]{{MasterBase: model.MasterBase{ID: cert}, Name: "BST"}},
	)
	ctx := context.Background()

	m, err := svc.Replace(ctx, []model.ComplianceCell{
		{RankID: rank, DocumentTypeID: uuidPtr(doc)},
		{RankID: rank, CertificateTypeID: uuidPtr(cert)},
		{RankID: rank, DocumentTypeID: uuidPtr(doc)},
	})
	require.NoError(t, err)
	assert.Len(t, m.Cells, 2, "duplicate cells are collapsed")
	assert.Len(t, m.Ranks, 1)

	_, err = svc.Replace(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, store.cells)
}

func TestComplianceMatrixRejectsUnknownOrMalformedCells(t *testing.T) {
	rank, doc := uuid.New(), uuid.New()
	store := &memMatrix{}
	svc := NewComplianceMatrixService(store,
		staticList[model.Rank]{{MasterBase: model.MasterBase{ID: rank}}},
		staticList[model.DocumentType]{{MasterBase: model.MasterBase{ID: doc}}},
		staticList[model.CertificateType]{},
	)
	_, err := svc.Replace(context.Background(), []model.ComplianceCell{
		{RankID: uuid.New(), DocumentTypeID: uuidPtr(doc)},
		{RankID: rank},
		{RankID: rank, DocumentTypeID: uuidPtr(doc), CertificateTypeID: uuidPtr(uuid.New())},
		{RankID: rank, DocumentTypeID: uuidPtr(uuid.New())},
		{RankID: rank, CertificateTypeID: uuidPtr(uuid.New())},
		{RankID: rank, DocumentTypeID: uuidPtr(doc)},
	})
	e := requireAppErr(t, err, "validation_failed")
	assert.Len(t, e.Fields, 5)
	assert.NotContains(t, e.Fields, "cells.5")
	assert.Nil(t, store.cells, "nothing is written when any cell is invalid")
}

func TestMasterOptionsAreCachedUntilInvalidated(t *testing.T) {
	cache := newMemCache()
	calls := 0
	ranks := countingList[model.Rank]{items: []model.Rank{{Name: "Master"}}, calls: &calls}
	svc := NewMasterOptionsService(MasterListers{
		Companies: staticList[model.Company]{}, Vessels: staticList[model.Vessel]{}, Ranks: ranks,
		VesselTypes: staticList[model.VesselType]{}, Currencies: staticList[model.Currency]{}, CrewStatuses: staticList[model.CrewStatus]{},
		AllowanceTypes: staticList[model.AllowanceType]{}, DocumentTypes: staticList[model.DocumentType]{}, CertificateTypes: staticList[model.CertificateType]{},
	}, cache)
	ctx := context.Background()

	first, err := svc.Options(ctx)
	require.NoError(t, err)
	second, err := svc.Options(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Equal(t, first.Ranks[0].Name, second.Ranks[0].Name)

	require.NoError(t, svc.Invalidate(ctx))
	_, err = svc.Options(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

type countingList[T any] struct {
	items []T
	calls *int
}

func (c countingList[T]) Active(context.Context) ([]T, error) {
	*c.calls++
	return c.items, nil
}
