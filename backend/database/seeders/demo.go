package seeders

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suwahadi/seafarerscrew/backend/internal/config"
	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

const demoSeed uint64 = 20261021

const demoMarkerGroup = "seed"

// ErrDemoInProduction is returned when demo data is requested in production without force.
var ErrDemoInProduction = errors.New("refusing to load demo data when APP_ENV is production; pass --force to override")

// CheckDemoAllowed refuses demo data in production unless force is set.
func CheckDemoAllowed(env string, force bool) error {
	if env == config.EnvProduction && !force {
		return ErrDemoInProduction
	}
	return nil
}

// DemoDeps are the services the demo seed loads data through.
type DemoDeps struct {
	Pool       *pgxpool.Pool
	Logger     *slog.Logger
	Site       service.SiteChanger
	Files      *service.FileService
	Companies  *service.MasterService[model.Company]
	Vessels    *service.MasterService[model.Vessel]
	Crews      *service.CrewService
	Contracts  *service.ContractService
	Rotation   *service.RotationService
	Quotations *service.QuotationService
	CMS        service.CMS
	News       *service.NewsService
	Pages      *service.PageService
	Vacancies  *service.VacancyService
	Applicants *repository.ApplicantRepository
	Contacts   *repository.ContactRepository
}

// DemoStep reports what happened to one demo seed step.
type DemoStep struct {
	Name    string
	Skipped bool
}

type demoRun struct {
	DemoDeps
	today time.Time
	refs  demoRefs
}

type demoStepFunc func(ctx context.Context, d *demoRun, rng *rand.Rand) error

var demoSteps = []struct {
	name string
	run  demoStepFunc
}{
	{"masters", seedDemoMasters},
	{"images", seedDemoImages},
	{"crew", seedDemoCrew},
	{"experiences", seedDemoExperiences},
	{"contracts", seedDemoContracts},
	{"reliefs", seedDemoReliefs},
	{"statuses", seedDemoStatuses},
	{"quotations", seedDemoQuotations},
	{"cms", seedDemoCMS},
	{"news", seedDemoNews},
	{"about", seedDemoAboutPage},
	{"vacancies", seedDemoVacancies},
	{"applicants", seedDemoApplicants},
	{"contacts", seedDemoContacts},
}

// Demo loads the demo data set; every step records a marker, so running it again adds nothing.
func Demo(ctx context.Context, deps DemoDeps) ([]DemoStep, error) {
	today, err := deps.Crews.Today(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve today: %w", err)
	}
	refs, err := loadDemoRefs(ctx, deps.Pool)
	if err != nil {
		return nil, err
	}
	d := &demoRun{DemoDeps: deps, today: today, refs: refs}
	report := make([]DemoStep, 0, len(demoSteps))
	for _, s := range demoSteps {
		done, err := d.stepDone(ctx, s.name)
		if err != nil {
			return report, err
		}
		if done {
			report = append(report, DemoStep{Name: s.name, Skipped: true})
			continue
		}
		if err := s.run(ctx, d, stepRNG(s.name)); err != nil {
			return report, fmt.Errorf("demo step %s: %w", s.name, err)
		}
		if err := d.markStep(ctx, s.name); err != nil {
			return report, err
		}
		d.Logger.InfoContext(ctx, "demo seed step finished", "step", s.name)
		report = append(report, DemoStep{Name: s.name})
	}
	d.announceChange(ctx)
	return report, nil
}

func stepRNG(name string) *rand.Rand {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return rand.New(rand.NewPCG(demoSeed, h.Sum64()))
}

func (d *demoRun) stepDone(ctx context.Context, name string) (bool, error) {
	var done bool
	err := d.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM settings WHERE group_name = $1 AND key = $2)`, demoMarkerGroup, "demo_"+name).Scan(&done)
	if err != nil {
		return false, fmt.Errorf("read demo marker %s: %w", name, err)
	}
	return done, nil
}

func (d *demoRun) markStep(ctx context.Context, name string) error {
	_, err := d.Pool.Exec(ctx, `INSERT INTO settings (group_name, key, value) VALUES ($1, $2, jsonb_build_object('completed_at', to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')))
		ON CONFLICT (group_name, key) DO NOTHING`, demoMarkerGroup, "demo_"+name)
	if err != nil {
		return fmt.Errorf("write demo marker %s: %w", name, err)
	}
	return nil
}

func (d *demoRun) announceChange(ctx context.Context) {
	tags := []string{model.TagSite, model.TagHome, model.TagPages, model.TagNews, model.TagGallery, model.TagVacancies, model.TagRedirects, model.TagSitemap}
	if err := d.Site.Changed(ctx, tags...); err != nil {
		d.Logger.WarnContext(ctx, "public cache not invalidated after demo seed", "error", err)
	}
}
