package seeders

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
)

type vacancySeed struct {
	title, rank, vesselType, department, location, duration, salary string
	closingInDays                                                   int
	closed                                                          bool
	documents                                                       []string
	summary                                                         string
}

var demoVacancies = []vacancySeed{
	{
		title: "Master - AHTS Vessel", rank: "MST", vesselType: "AHTS", department: model.DepartmentDeck, location: "Balikpapan, East Kalimantan",
		duration: "6 months", salary: "USD 6,000 - 7,000 per month", closingInDays: 21,
		documents: []string{"Passport", "Seaman Book (CDC)", "COC Master (STCW II/2)", "Medical Check Up (MCU)", "Basic Safety Training (BST)", "BOSIET and HUET"},
		summary:   "command of an anchor handling tug supply vessel on an offshore support contract",
	},
	{
		title: "Chief Engineer - Tanker", rank: "CE", vesselType: "Tanker", department: model.DepartmentEngine, location: "Singapore",
		duration: "4 months", salary: "USD 6,000 - 7,000 per month", closingInDays: 35,
		documents: []string{"Passport", "Seaman Book (CDC)", "COC Chief Engineer (STCW III/2)", "Medical Check Up (MCU)", "Basic Safety Training (BST)", "Advanced Fire Fighting"},
		summary:   "responsibility for the engine department of a product tanker",
	},
	{
		title: "Able Seaman (AB) - Crew Boat", rank: "AB", vesselType: "Crew Boat", department: model.DepartmentDeck, location: "Batam, Riau Islands",
		duration: "3 months", salary: "IDR 8,000,000 - 11,000,000 per month", closingInDays: 14,
		documents: []string{"Passport", "Seaman Book (CDC)", "Rating Forming Part of a Navigational Watch", "Medical Check Up (MCU)", "Basic Safety Training (BST)"},
		summary:   "deck watchkeeping and mooring duties on a crew transfer boat",
	},
	{
		title: "Cook - Offshore Support Vessel", rank: "CK", vesselType: "AHTS", department: model.DepartmentCateringOthers, location: "Jakarta",
		duration: "4 months", salary: "IDR 7,500,000 - 9,500,000 per month", closingInDays: 50,
		documents: []string{"Passport", "Seaman Book (CDC)", "Medical Check Up (MCU)", "Basic Safety Training (BST)", "Food handling certificate"},
		summary:   "galley management and meal preparation for a crew of around twenty",
	},
	{
		title: "Chief Officer - Bulk Carrier", rank: "CO", vesselType: "Bulk Carrier", department: model.DepartmentDeck, location: "Surabaya, East Java",
		duration: "6 months", salary: "USD 4,500 - 5,200 per month", closingInDays: 12, closed: true,
		documents: []string{"Passport", "Seaman Book (CDC)", "COC Chief Officer (STCW II/2)", "Medical Check Up (MCU)", "Basic Safety Training (BST)"},
		summary:   "cargo operations and watch management on a bulk carrier",
	},
	{
		title: "Oiler - ASD Tug", rank: "OLR", vesselType: "ASD Tug", department: model.DepartmentEngine, location: "Surabaya, East Java",
		duration: "3 months", salary: "IDR 8,500,000 - 10,500,000 per month", closingInDays: 30, closed: true,
		documents: []string{"Passport", "Seaman Book (CDC)", "Rating Forming Part of an Engineering Watch", "Medical Check Up (MCU)", "Basic Safety Training (BST)"},
		summary:   "engine room watchkeeping and maintenance support on a harbour tug",
	},
}

func vacancyDescription(v vacancySeed, rankName string) string {
	return fmt.Sprintf("<p>We are looking for a %s for %s. You will join an experienced team and work to the standards of our principal.</p><p>Applications are free of charge and handled by our crewing team.</p>",
		rankName, v.summary)
}

func vacancyRequirements() string {
	return "<ul><li>Valid certificates for the rank, issued under the STCW convention.</li><li>Sea service on similar vessels.</li><li>Valid medical certificate and passport.</li><li>Good English communication for shipboard work.</li></ul>"
}

func seedDemoVacancies(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	for i, v := range demoVacancies {
		rank := d.refs.ranks[v.rank]
		typeID := d.refs.vesselTypes[v.vesselType]
		closing := d.today.AddDate(0, 0, v.closingInDays)
		if v.closed {
			closing = d.today.AddDate(0, 0, -v.closingInDays)
		}
		in := model.Vacancy{
			Title: v.title, RankID: rank.ID, VesselTypeID: &typeID, Department: v.department, Location: v.location,
			EmploymentType: model.EmploymentContract, Duration: v.duration, SalaryText: v.salary,
			DescriptionHTML: vacancyDescription(v, rank.Name), RequirementsHTML: vacancyRequirements(), RequiredDocuments: v.documents,
			ClosingDate: &closing, Status: model.VacancyOpen,
		}
		if v.closed {
			in.ClosingDate = ptrTime(d.today.AddDate(0, 0, 7))
		}
		created, err := d.Vacancies.Create(ctx, in)
		if err != nil {
			return fmt.Errorf("create vacancy %d: %w", i+1, err)
		}
		if !v.closed {
			continue
		}
		created.Status, created.ClosingDate = model.VacancyClosed, &closing
		if _, err := d.Vacancies.Update(ctx, created.PublicID, created); err != nil {
			return fmt.Errorf("close vacancy %d: %w", i+1, err)
		}
	}
	return nil
}

func ptrTime(t time.Time) *time.Time { return &t }

var applicantPaths = [][]string{
	{model.StageNew}, {model.StageNew}, {model.StageNew}, {model.StageNew},
	{model.StageNew, model.StageScreening}, {model.StageNew, model.StageScreening}, {model.StageNew, model.StageScreening},
	{model.StageNew, model.StageScreening, model.StageInterview}, {model.StageNew, model.StageScreening, model.StageInterview},
	{model.StageNew, model.StageScreening, model.StageInterview},
	{model.StageNew, model.StageScreening, model.StageInterview, model.StageAccepted},
	{model.StageNew, model.StageScreening, model.StageInterview, model.StageAccepted},
	{model.StageNew, model.StageRejected},
	{model.StageNew, model.StageScreening, model.StageRejected},
	{model.StageNew, model.StageScreening, model.StageInterview, model.StageRejected},
}

var stageNotes = map[string][]string{
	model.StageScreening: {"CV and documents reviewed; profile matches the vacancy.", "Documents complete; moving to screening."},
	model.StageInterview: {"Interview scheduled with the crewing team.", "Shortlisted after screening call."},
	model.StageAccepted:  {"Offer accepted; documents to be verified before sign-on.", "Selected by the principal; joining date to be confirmed."},
	model.StageRejected:  {"Does not meet the minimum sea service requirement.", "Certificates expired and not renewed in time.", "Position filled by another candidate."},
}

const applicantMessage = "I would like to apply for this position. My CV and certificates are available on request."

func seedDemoApplicants(ctx context.Context, d *demoRun, rng *rand.Rand) error {
	vacancies, err := d.vacancyIDs(ctx)
	if err != nil {
		return err
	}
	taken := map[string]bool{}
	for i, path := range applicantPaths {
		vacancy := vacancies[i%len(vacancies)]
		nationality := nationalityIndonesian
		if i%5 == 4 {
			nationality = pick(rng, []string{nationalityFilipino, nationalityIndian})
		}
		person := newPerson(rng, nationality, "male", taken)
		created := d.today.AddDate(0, 0, -(4 + i*3)).Add(time.Duration(8+rng.IntN(9)) * time.Hour)
		years := 2 + rng.IntN(20)
		app, err := d.newApplication(person, vacancy, created, years, i)
		if err != nil {
			return err
		}
		if err := d.Applicants.Create(ctx, app); err != nil {
			return fmt.Errorf("create applicant %d: %w", i+1, err)
		}
		if err := d.moveApplicant(ctx, rng, app.ID, path, created); err != nil {
			return fmt.Errorf("move applicant %d: %w", i+1, err)
		}
	}
	return nil
}

type demoVacancy struct {
	id     uuid.UUID
	rankID uuid.UUID
}

func (d *demoRun) vacancyIDs(ctx context.Context) ([]demoVacancy, error) {
	titles := make([]string, len(demoVacancies))
	for i, v := range demoVacancies {
		titles[i] = v.title
	}
	rows, err := d.Pool.Query(ctx, `SELECT id, rank_id, title FROM vacancies WHERE title = ANY($1) AND deleted_at IS NULL`, titles)
	if err != nil {
		return nil, fmt.Errorf("load demo vacancies: %w", err)
	}
	type row struct {
		demoVacancy
		title string
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (row, error) {
		var x row
		err := r.Scan(&x.id, &x.rankID, &x.title)
		return x, err
	})
	if err != nil {
		return nil, fmt.Errorf("scan demo vacancies: %w", err)
	}
	byTitle := make(map[string]demoVacancy, len(items))
	for _, it := range items {
		byTitle[it.title] = it.demoVacancy
	}
	out := make([]demoVacancy, len(titles))
	for i, title := range titles {
		v, ok := byTitle[title]
		if !ok {
			return nil, fmt.Errorf("demo vacancy %q not found", title)
		}
		out[i] = v
	}
	return out, nil
}

func (d *demoRun) newApplication(p demoPerson, v demoVacancy, created time.Time, years, index int) (model.NewApplication, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return model.NewApplication{}, fmt.Errorf("applicant id: %w", err)
	}
	public, err := publicid.New()
	if err != nil {
		return model.NewApplication{}, err
	}
	local := strings.NewReplacer(" ", ".", "'", "").Replace(strings.ToLower(p.FullName))
	rng := stepRNG(fmt.Sprintf("applicant-%d", index))
	birth := d.today.AddDate(-(24 + rng.IntN(25)), -rng.IntN(12), -rng.IntN(28))
	return model.NewApplication{
		ID: id, PublicID: public, VacancyID: v.id, FullName: p.FullName,
		Email: fmt.Sprintf("%s.%02d@applicants.example.com", local, index+1), Phone: mobileFor(rng, p.Nationality),
		DateOfBirth: birth, Nationality: p.Nationality, RankAppliedID: v.rankID, YearsExperience: years, Message: applicantMessage,
		ConsentAt: created, IPAddress: fmt.Sprintf("203.0.113.%d", 1+index), DuplicateSince: created.AddDate(0, 0, -30),
	}, nil
}

func (d *demoRun) moveApplicant(ctx context.Context, rng *rand.Rand, id uuid.UUID, path []string, created time.Time) error {
	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin applicant transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	final := path[len(path)-1]
	if _, err := tx.Exec(ctx, `UPDATE applicants SET created_at = $2, stage = $3 WHERE id = $1`, id, created, final); err != nil {
		return fmt.Errorf("update applicant: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE applicant_stage_logs SET created_at = $2 WHERE applicant_id = $1`, id, created); err != nil {
		return fmt.Errorf("update first stage log: %w", err)
	}
	at := created
	for i := 1; i < len(path); i++ {
		at = at.AddDate(0, 0, 1+rng.IntN(3))
		note := pick(rng, stageNotes[path[i]])
		if _, err := tx.Exec(ctx, `INSERT INTO applicant_stage_logs (applicant_id, from_stage, to_stage, note, created_at) VALUES ($1, $2, $3, $4, $5)`,
			id, path[i-1], path[i], note, at); err != nil {
			return fmt.Errorf("insert stage log: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit applicant transaction: %w", err)
	}
	return nil
}

type contactSeed struct {
	name, email, phone, subject, message string
	daysAgo                              int
	read, archived                       bool
}

var demoContacts = []contactSeed{
	{"Hendro Gunawan", "hendro.gunawan@samudera-offshore.example.com", "+622150000199", "Crew for an AHTS charter in Q1", "We are planning a three-month charter and need a master, a chief engineer and four ABs. Could you share availability and indicative rates?", 1, false, false},
	{"Rina Kurniawati", "rina.kurniawati@mail.example.com", "", "Status of my application", "I applied for the Able Seaman vacancy last week and would like to know whether you received my documents.", 2, false, false},
	{"Marcus Lee", "marcus.lee@meridian-marine.example.com", "+6562000188", "Partnership proposal", "Meridian Marine Services would like to discuss a framework agreement for tanker crew. Please let us know a convenient time for a call.", 6, true, false},
	{"Budi Hartono", "budi.hartono@mail.example.com", "+628123450011", "Medical check-up clinics", "Could you share a list of clinics approved for seafarer medical check-ups in Surabaya?", 11, true, false},
	{"Website Promotions", "offers@promo.example.com", "", "Boost your website ranking", "We can improve your search ranking within days. Reply for a free audit.", 19, true, true},
}

func seedDemoContacts(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	for i, c := range demoContacts {
		created, err := d.Contacts.Create(ctx, model.ContactMessage{
			Name: c.name, Email: c.email, Phone: c.phone, Subject: c.subject, Message: c.message, IPAddress: fmt.Sprintf("198.51.100.%d", 10+i),
		})
		if err != nil {
			return fmt.Errorf("create contact message %d: %w", i+1, err)
		}
		if c.read || c.archived {
			read, archived := c.read, c.archived
			if _, err := d.Contacts.SetFlags(ctx, created.ID, &read, &archived); err != nil {
				return fmt.Errorf("flag contact message %d: %w", i+1, err)
			}
		}
		at := d.today.AddDate(0, 0, -c.daysAgo).Add(10 * time.Hour)
		if _, err := d.Pool.Exec(ctx, `UPDATE contact_messages SET created_at = $2 WHERE id = $1`, created.ID, at); err != nil {
			return fmt.Errorf("date contact message %d: %w", i+1, err)
		}
	}
	return nil
}
