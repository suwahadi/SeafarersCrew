package seeders

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

type demoCrew struct {
	spec     crewSpec
	role     crewAssignment
	id       uuid.UUID
	publicID string
}

func seedDemoCrew(ctx context.Context, d *demoRun, rng *rand.Rand) error {
	specs := buildCrewSpecs(rng, d.today)
	for i, spec := range specs {
		crew, err := d.Crews.Create(ctx, crewModel(spec, d.refs.ranks[spec.rank].ID))
		if err != nil {
			return fmt.Errorf("create crew %d: %w", i+1, err)
		}
		if err := d.saveCrewRecords(ctx, rng, crew.PublicID, spec, i); err != nil {
			return fmt.Errorf("save records of crew %d: %w", i+1, err)
		}
	}
	return nil
}

func crewModel(spec crewSpec, rankID uuid.UUID) model.Crew {
	birth, gender, marital := spec.birth, spec.person.Gender, spec.marital
	height, weight := smallInt(spec.heightCM), smallInt(spec.weightKG)
	return model.Crew{
		FullName: spec.person.FullName, PlaceOfBirth: spec.person.City, DateOfBirth: &birth, Gender: &gender,
		Nationality: spec.person.Nationality, Religion: spec.person.Religion, MaritalStatus: &marital,
		HeightCM: &height, WeightKG: &weight, ShoeSize: spec.shoe, CoverallSize: spec.coverall, AppliedRankID: &rankID,
		Address: spec.address, City: spec.person.City, Province: spec.person.Province, Country: spec.person.Country,
		PostalCode: spec.postal, MobilePhone: spec.mobile, Email: spec.email, NearestAirport: spec.person.Airport,
	}
}

func (d *demoRun) saveCrewRecords(ctx context.Context, rng *rand.Rand, publicID string, spec crewSpec, index int) error {
	set := buildCrewDocuments(rng, spec, index, d.today)
	for _, doc := range set.documents {
		t := d.refs.documentTypes[doc.typeCode]
		if _, err := d.Crews.SaveDocument(ctx, repository.TravelDocuments, publicID, documentModel(doc, t.ID)); err != nil {
			return fmt.Errorf("save document %s: %w", doc.typeCode, err)
		}
	}
	for _, cert := range set.certificates {
		t := d.refs.certificates[cert.typeCode]
		if _, err := d.Crews.SaveDocument(ctx, repository.Certificates, publicID, documentModel(cert, t.ID)); err != nil {
			return fmt.Errorf("save certificate %s: %w", cert.typeCode, err)
		}
	}
	if _, err := d.Crews.SaveBankAccount(ctx, publicID, bankAccountFor(rng, spec)); err != nil {
		return fmt.Errorf("save bank account: %w", err)
	}
	if _, err := d.Crews.SaveNextOfKin(ctx, publicID, nextOfKinFor(rng, spec)); err != nil {
		return fmt.Errorf("save next of kin: %w", err)
	}
	return nil
}

func documentModel(doc documentDraft, typeID uuid.UUID) model.CrewDocument {
	issued := doc.issued
	return model.CrewDocument{
		TypeID: typeID, Number: doc.number, IssuedDate: &issued, ExpiredDate: doc.expiry,
		IssuingAuthority: doc.authority, PlaceOfIssue: doc.place,
	}
}

func bankAccountFor(rng *rand.Rand, spec crewSpec) model.BankAccount {
	if spec.person.Nationality == nationalityIndonesian {
		return model.BankAccount{
			BankName:    pick(rng, []string{"Bank Mandiri", "Bank Central Asia", "Bank Negara Indonesia", "Bank Rakyat Indonesia"}),
			AccountName: spec.person.FullName, AccountNumber: digits(rng, 10+rng.IntN(4)), CurrencyCode: "IDR", Remarks: "DIRECT", IsPrimary: true,
		}
	}
	return model.BankAccount{
		BankName: pick(rng, []string{"Standard Chartered", "DBS Bank", "HSBC"}), AccountName: spec.person.FullName,
		AccountNumber: digits(rng, 10+rng.IntN(4)), CurrencyCode: "USD", CoverBank: "Citibank New York", Remarks: "DIRECT", IsPrimary: true,
	}
}

func nextOfKinFor(rng *rand.Rand, spec crewSpec) model.NextOfKin {
	relationship, gender := "parent", pick(rng, []string{"male", "female"})
	if spec.marital == "married" {
		relationship, gender = "spouse", "female"
		if spec.person.Gender == "female" {
			gender = "male"
		}
	}
	pool := namePools[spec.person.Nationality]
	firsts := pool.male
	if gender == "female" {
		firsts = pool.female
	}
	return model.NextOfKin{
		Name: pick(rng, firsts) + " " + pick(rng, pool.last), Relationship: relationship,
		Phone: mobileFor(rng, spec.person.Nationality), Address: spec.address + ", " + spec.person.City, IsPrimary: true,
	}
}

func (d *demoRun) loadCrew(ctx context.Context) ([]demoCrew, error) {
	specs := buildCrewSpecs(stepRNG("crew"), d.today)
	roles := assignRoles(specs)
	rows, err := d.Pool.Query(ctx, `SELECT email, id, public_id FROM crews WHERE email LIKE $1 AND deleted_at IS NULL`, "%@"+demoCrewEmailDomain)
	if err != nil {
		return nil, fmt.Errorf("load demo crew: %w", err)
	}
	type row struct {
		email    string
		id       uuid.UUID
		publicID string
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.email, &x.id, &x.publicID)
		return x, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan demo crew: %w", err)
	}
	byEmail := make(map[string]row, len(items))
	for _, it := range items {
		byEmail[it.email] = it
	}
	out := make([]demoCrew, len(specs))
	for i, spec := range specs {
		r, ok := byEmail[spec.email]
		if !ok {
			return nil, fmt.Errorf("demo crew member %d not found", i+1)
		}
		out[i] = demoCrew{spec: spec, role: roles[i], id: r.id, publicID: r.publicID}
	}
	return out, nil
}

func seedDemoExperiences(ctx context.Context, d *demoRun, rng *rand.Rand) error {
	crew, err := d.loadCrew(ctx)
	if err != nil {
		return err
	}
	companies, err := d.loadCompanies(ctx)
	if err != nil {
		return err
	}
	vessels, err := d.loadVessels(ctx)
	if err != nil {
		return err
	}
	manualRows := demoExperienceTotal - len(signedOffRanks)
	counts := experienceCounts(rng, len(crew), manualRows)
	for i, c := range crew {
		end := d.experienceEnd(rng, c)
		for _, x := range d.experienceHistory(rng, c, counts[i], end, companies, vessels) {
			if _, err := d.Crews.SaveExperience(ctx, c.publicID, x); err != nil {
				return fmt.Errorf("save experience of crew %d: %w", i+1, err)
			}
		}
	}
	return nil
}

func (d *demoRun) experienceEnd(rng *rand.Rand, c demoCrew) time.Time {
	if c.role.role == roleOnBoard {
		plan := contractPlans[c.role.contract]
		return contractSignOn(d.today, plan).AddDate(0, 0, -(20 + rng.IntN(40)))
	}
	if c.role.role == roleSignedOff {
		return d.today.AddDate(0, 0, -(300 + rng.IntN(60)))
	}
	return d.today.AddDate(0, 0, -(25 + rng.IntN(220)))
}

var pastVesselNames = []string{
	"Pacific Provider", "Java Ranger", "Sumba Trader", "Malacca Star", "Kalimantan Express", "Sulawesi Pride", "Lombok Strait",
	"Bintan Voyager", "Halmahera Wind", "Timor Dawn", "Riau Navigator", "Maluku Pearl", "Karang Jaya", "Laut Biru", "Samudra Indah",
}

func (d *demoRun) experienceHistory(rng *rand.Rand, c demoCrew, count int, end time.Time, companies map[string]demoCompany, vessels []demoVessel) []model.Experience {
	rank := d.refs.ranks[c.spec.rank]
	rankID := rank.ID
	out := make([]model.Experience, 0, count)
	for range count {
		months := 4 + rng.IntN(6)
		signOff := end
		signOn := signOff.AddDate(0, -months, -rng.IntN(20))
		x := model.Experience{
			RankID: rankID, SignOnDate: signOn, SignOffDate: &signOff, Status: "sign_off", Source: "manual",
			Remarks: pick(rng, []string{"Completed contract", "Completed contract", "Contract finished, good performance", ""}),
		}
		if rng.IntN(4) == 0 {
			v := pick(rng, vessels)
			vesselID, typeID := v.ID, v.VesselTypeI
			x.VesselName, x.VesselID, x.VesselTypeID, x.VesselType = v.Name, &vesselID, &typeID, v.VesselType
			x.GRTHP = fmt.Sprintf("GRT %s / %s", v.GRT.StringFixed(0), v.HPKW)
		} else {
			name := pick(rng, demoVessels)
			x.VesselName = pick(rng, pastVesselNames)
			typeID := d.refs.vesselTypes[name.vesselType]
			x.VesselTypeID, x.VesselType = &typeID, name.vesselType
			x.GRTHP = fmt.Sprintf("GRT %d / %s", name.grt, name.hpkw)
		}
		employer, principal := companies[pick(rng, []string{"BLE", "KTL", "FSM"})], companies[pick(rng, []string{"SNO", "BLE", "MMS", "KTL"})]
		employerID, principalID := employer.ID, principal.ID
		x.CompanyID, x.CompanyName, x.PrincipalID, x.PrincipalName = &employerID, employer.Name, &principalID, principal.Name
		out = append(out, x)
		end = signOn.AddDate(0, 0, -(20 + rng.IntN(70)))
	}
	return out
}

func smallInt(v int) int16 {
	if v < 0 || v > math.MaxInt16 {
		return 0
	}
	return int16(v)
}
