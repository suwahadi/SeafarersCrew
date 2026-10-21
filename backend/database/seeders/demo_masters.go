package seeders

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type demoCompanySeed struct {
	code, name, kind, address, country, phone, email, website string
}

var demoCompanies = []demoCompanySeed{
	{"SNO", "PT Samudera Nusantara Offshore", model.CompanyTypePrincipal, "Jl. Jenderal Sudirman Kav. 52, Jakarta 12190", "Indonesia", "+622150000101", "info@samudera-offshore.example.com", "https://samudera-offshore.example.com"},
	{"BLE", "PT Bahari Lintas Energi", model.CompanyTypeBoth, "Jl. Jenderal Sudirman No. 14, Balikpapan 76114", "Indonesia", "+6254270000102", "info@bahari-energi.example.com", "https://bahari-energi.example.com"},
	{"MMS", "Meridian Marine Services Pte Ltd", model.CompanyTypePrincipal, "20 Cross Street #08-01, Singapore 048422", "Singapore", "+6562000103", "crewing@meridian-marine.example.com", "https://meridian-marine.example.com"},
	{"KTL", "PT Karya Tirta Logistik", model.CompanyTypeBoth, "Jl. Perak Timur No. 88, Surabaya 60165", "Indonesia", "+62313290104", "operations@karya-tirta.example.com", "https://karya-tirta.example.com"},
	{"FSM", "Fjord Shipping Management AS", model.CompanyTypeCompany, "Strandgata 12, 5013 Bergen", "Norway", "+4755000105", "post@fjord-shipping.example.com", "https://fjord-shipping.example.com"},
}

type demoVesselSeed struct {
	name, vesselType, flag string
	grt                    int64
	hpkw                   string
	year                   int
	owner, principal       string
}

var demoVessels = []demoVesselSeed{
	{"Nusantara Titan", "AHTS", "Indonesia", 2850, "2 x 2,942 kW", 2013, "BLE", "SNO"},
	{"Borneo Pioneer", "AHTS", "Indonesia", 2640, "2 x 2,680 kW", 2014, "BLE", "SNO"},
	{"Selat Sunda Warrior", "AHTS", "Singapore", 2900, "2 x 3,100 kW", 2016, "FSM", "MMS"},
	{"Karimata Express", "Crew Boat", "Indonesia", 285, "3 x 1,200 kW", 2016, "KTL", "KTL"},
	{"Natuna Runner", "Crew Boat", "Indonesia", 310, "3 x 1,350 kW", 2018, "BLE", "BLE"},
	{"Madura Tug 3", "ASD Tug", "Indonesia", 498, "2 x 1,865 kW", 2012, "KTL", "KTL"},
	{"Banda Sentinel", "ASD Tug", "Indonesia", 540, "2 x 2,000 kW", 2019, "KTL", "SNO"},
	{"Orion Star", "Tanker", "Panama", 11850, "6,300 kW", 2009, "FSM", "MMS"},
	{"Celebes Harmony", "Bulk Carrier", "Marshall Islands", 32600, "8,900 kW", 2007, "FSM", "MMS"},
	{"Arafura Spirit", "RORO", "Indonesia", 4900, "2 x 2,400 kW", 2011, "KTL", "KTL"},
}

type demoCompany struct {
	ID   uuid.UUID
	Name string
	Type string
}

type demoVessel struct {
	ID          uuid.UUID
	Name        string
	VesselType  string
	VesselTypeI uuid.UUID
	GRT         decimal.Decimal
	HPKW        string
	OwnerCode   string
	Principal   string
}

func seedDemoMasters(ctx context.Context, d *demoRun, rng *rand.Rand) error {
	byCode := map[string]demoCompany{}
	for i, c := range demoCompanies {
		created, err := d.Companies.Create(ctx, model.Company{
			MasterBase: model.MasterBase{IsActive: true, SortOrder: (i + 1) * 10},
			Code:       c.code, Name: c.name, Type: c.kind, Address: c.address, Country: c.country,
			Phone: c.phone, Email: c.email, Website: c.website,
		})
		if err != nil {
			return fmt.Errorf("create company %s: %w", c.code, err)
		}
		byCode[c.code] = demoCompany{ID: created.ID, Name: created.Name, Type: created.Type}
	}
	imos, err := generateIMOs(rng, len(demoVessels))
	if err != nil {
		return err
	}
	for i, v := range demoVessels {
		grt := decimal.NewFromInt(v.grt)
		year := v.year
		typeID := d.refs.vesselTypes[v.vesselType]
		owner, principal := byCode[v.owner].ID, byCode[v.principal].ID
		if _, err := d.Vessels.Create(ctx, model.Vessel{
			MasterBase: model.MasterBase{IsActive: true, SortOrder: (i + 1) * 10},
			Name:       v.name, IMONumber: imos[i], Flag: v.flag, VesselTypeID: &typeID, GRT: &grt, HPKW: v.hpkw, YearBuilt: &year,
			OwnerCompanyID: &owner, PrincipalID: &principal, Status: "active",
		}); err != nil {
			return fmt.Errorf("create vessel %s: %w", v.name, err)
		}
	}
	return nil
}

func (d *demoRun) loadCompanies(ctx context.Context) (map[string]demoCompany, error) {
	codes := make([]string, len(demoCompanies))
	for i, c := range demoCompanies {
		codes[i] = c.code
	}
	rows, err := d.Pool.Query(ctx, `SELECT code, id, name, type FROM companies WHERE code = ANY($1) AND deleted_at IS NULL`, codes)
	if err != nil {
		return nil, fmt.Errorf("load demo companies: %w", err)
	}
	type row struct {
		code    string
		company demoCompany
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.code, &x.company.ID, &x.company.Name, &x.company.Type)
		return x, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan demo companies: %w", err)
	}
	out := make(map[string]demoCompany, len(items))
	for _, it := range items {
		out[it.code] = it.company
	}
	if len(out) != len(demoCompanies) {
		return nil, fmt.Errorf("found %d of %d demo companies", len(out), len(demoCompanies))
	}
	return out, nil
}

func (d *demoRun) loadVessels(ctx context.Context) ([]demoVessel, error) {
	rows, err := d.Pool.Query(ctx, `SELECT v.name, v.id, t.name, t.id, coalesce(v.grt, 0), v.hp_kw
		FROM vessels v JOIN vessel_types t ON t.id = v.vessel_type_id WHERE v.deleted_at IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("load demo vessels: %w", err)
	}
	all, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (demoVessel, error) {
		var v demoVessel
		err := r.Scan(&v.Name, &v.ID, &v.VesselType, &v.VesselTypeI, &v.GRT, &v.HPKW)
		return v, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan demo vessels: %w", err)
	}
	byName := make(map[string]demoVessel, len(all))
	for _, v := range all {
		byName[v.Name] = v
	}
	out := make([]demoVessel, len(demoVessels))
	for i, seed := range demoVessels {
		v, ok := byName[seed.name]
		if !ok {
			return nil, fmt.Errorf("demo vessel %q not found", seed.name)
		}
		v.OwnerCode, v.Principal = seed.owner, seed.principal
		out[i] = v
	}
	return out, nil
}
