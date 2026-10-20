package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// ComplianceMatrixStore persists the compliance matrix.
type ComplianceMatrixStore interface {
	ActiveCells(ctx context.Context) ([]model.ComplianceCell, error)
	ReplaceActive(ctx context.Context, cells []model.ComplianceCell) error
}

// ComplianceMatrixService reads and replaces the compliance matrix.
type ComplianceMatrixService struct {
	store            ComplianceMatrixStore
	ranks            ActiveLister[model.Rank]
	documentTypes    ActiveLister[model.DocumentType]
	certificateTypes ActiveLister[model.CertificateType]
}

// NewComplianceMatrixService returns a ComplianceMatrixService.
func NewComplianceMatrixService(store ComplianceMatrixStore, ranks ActiveLister[model.Rank], docs ActiveLister[model.DocumentType], certs ActiveLister[model.CertificateType]) *ComplianceMatrixService {
	return &ComplianceMatrixService{store: store, ranks: ranks, documentTypes: docs, certificateTypes: certs}
}

// Matrix returns active ranks, document types, certificate types and checked cells.
func (s *ComplianceMatrixService) Matrix(ctx context.Context) (model.ComplianceMatrix, error) {
	var m model.ComplianceMatrix
	g, gctx := errgroup.WithContext(ctx)
	collectActive(gctx, g, s.ranks, &m.Ranks)
	collectActive(gctx, g, s.documentTypes, &m.DocumentTypes)
	collectActive(gctx, g, s.certificateTypes, &m.CertificateTypes)
	g.Go(func() error {
		cells, err := s.store.ActiveCells(gctx)
		m.Cells = cells
		return err
	})
	if err := g.Wait(); err != nil {
		return model.ComplianceMatrix{}, err
	}
	return m, nil
}

// Replace validates cells against active ranks and types and replaces the active part of the matrix.
func (s *ComplianceMatrixService) Replace(ctx context.Context, cells []model.ComplianceCell) (model.ComplianceMatrix, error) {
	current, err := s.Matrix(ctx)
	if err != nil {
		return model.ComplianceMatrix{}, err
	}
	ranks := activeIDSet(current.Ranks, func(r model.Rank) uuid.UUID { return r.ID })
	docs := activeIDSet(current.DocumentTypes, func(d model.DocumentType) uuid.UUID { return d.ID })
	certs := activeIDSet(current.CertificateTypes, func(c model.CertificateType) uuid.UUID { return c.ID })

	fields := map[string]string{}
	seen := map[[2]uuid.UUID]struct{}{}
	clean := make([]model.ComplianceCell, 0, len(cells))
	for i, c := range cells {
		key := fmt.Sprintf("cells.%d", i)
		switch {
		case !ranks[c.RankID]:
			fields[key] = "The rank does not exist or is inactive."
		case (c.DocumentTypeID == nil) == (c.CertificateTypeID == nil):
			fields[key] = "Exactly one of document_type_id or certificate_type_id is required."
		case c.DocumentTypeID != nil && !docs[*c.DocumentTypeID]:
			fields[key] = "The document type does not exist or is inactive."
		case c.CertificateTypeID != nil && !certs[*c.CertificateTypeID]:
			fields[key] = "The certificate type does not exist or is inactive."
		default:
			typeID := c.CertificateTypeID
			if c.DocumentTypeID != nil {
				typeID = c.DocumentTypeID
			}
			k := [2]uuid.UUID{c.RankID, *typeID}
			if _, dup := seen[k]; !dup {
				seen[k] = struct{}{}
				clean = append(clean, c)
			}
		}
	}
	if len(fields) > 0 {
		return model.ComplianceMatrix{}, apperr.Validation(fields)
	}
	if err := s.store.ReplaceActive(ctx, clean); err != nil {
		return model.ComplianceMatrix{}, err
	}
	return s.Matrix(ctx)
}

func activeIDSet[T any](items []T, id func(T) uuid.UUID) map[uuid.UUID]bool {
	out := make(map[uuid.UUID]bool, len(items))
	for _, it := range items {
		out[id(it)] = true
	}
	return out
}
