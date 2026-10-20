package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// ActiveLister lists active, non-deleted rows of one master.
type ActiveLister[T any] interface {
	Active(ctx context.Context) ([]T, error)
}

// MasterListers groups the active listers of every master.
type MasterListers struct {
	Companies        ActiveLister[model.Company]
	Vessels          ActiveLister[model.Vessel]
	Ranks            ActiveLister[model.Rank]
	VesselTypes      ActiveLister[model.VesselType]
	Currencies       ActiveLister[model.Currency]
	CrewStatuses     ActiveLister[model.CrewStatus]
	AllowanceTypes   ActiveLister[model.AllowanceType]
	DocumentTypes    ActiveLister[model.DocumentType]
	CertificateTypes ActiveLister[model.CertificateType]
}

const (
	optionsCacheKey = "masters:options"
	optionsCacheTTL = time.Hour
)

// MasterOptionsService serves every active master list in one cached payload.
type MasterOptionsService struct {
	listers MasterListers
	cache   Cache
}

// NewMasterOptionsService returns a MasterOptionsService.
func NewMasterOptionsService(listers MasterListers, cache Cache) *MasterOptionsService {
	return &MasterOptionsService{listers: listers, cache: cache}
}

// Options returns the cached options, loading them in parallel on a miss.
func (s *MasterOptionsService) Options(ctx context.Context) (model.MasterOptions, error) {
	raw, ok, err := s.cache.Get(ctx, optionsCacheKey)
	if err != nil {
		return model.MasterOptions{}, err
	}
	if ok {
		var cached model.MasterOptions
		if json.Unmarshal(raw, &cached) == nil {
			return cached, nil
		}
	}
	opts, err := s.load(ctx)
	if err != nil {
		return model.MasterOptions{}, err
	}
	payload, err := json.Marshal(opts)
	if err != nil {
		return model.MasterOptions{}, fmt.Errorf("encode master options: %w", err)
	}
	if err := s.cache.Set(ctx, optionsCacheKey, payload, optionsCacheTTL); err != nil {
		return model.MasterOptions{}, err
	}
	return opts, nil
}

// Invalidate drops the cached options.
func (s *MasterOptionsService) Invalidate(ctx context.Context) error {
	return s.cache.Delete(ctx, optionsCacheKey)
}

func (s *MasterOptionsService) load(ctx context.Context) (model.MasterOptions, error) {
	var o model.MasterOptions
	g, gctx := errgroup.WithContext(ctx)
	collectActive(gctx, g, s.listers.Companies, &o.Companies)
	collectActive(gctx, g, s.listers.Vessels, &o.Vessels)
	collectActive(gctx, g, s.listers.Ranks, &o.Ranks)
	collectActive(gctx, g, s.listers.VesselTypes, &o.VesselTypes)
	collectActive(gctx, g, s.listers.Currencies, &o.Currencies)
	collectActive(gctx, g, s.listers.CrewStatuses, &o.CrewStatuses)
	collectActive(gctx, g, s.listers.AllowanceTypes, &o.AllowanceTypes)
	collectActive(gctx, g, s.listers.DocumentTypes, &o.DocumentTypes)
	collectActive(gctx, g, s.listers.CertificateTypes, &o.CertificateTypes)
	if err := g.Wait(); err != nil {
		return model.MasterOptions{}, err
	}
	return o, nil
}

func collectActive[T any](ctx context.Context, g *errgroup.Group, l ActiveLister[T], dst *[]T) {
	g.Go(func() error {
		items, err := l.Active(ctx)
		if err != nil {
			return err
		}
		*dst = items
		return nil
	})
}
