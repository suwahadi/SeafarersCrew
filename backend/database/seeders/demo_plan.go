package seeders

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

const (
	demoCrewTotal       = 60
	demoExperienceTotal = 120
	demoVesselTotal     = 10
)

type crewSlot struct {
	rank        string
	nationality string
	count       int
}

var crewSlots = []crewSlot{
	{"MST", nationalityIndonesian, 3}, {"MST", nationalityUkrainian, 1},
	{"CO", nationalityIndonesian, 3}, {"CO", nationalityIndian, 1},
	{"2O", nationalityIndonesian, 3},
	{"3O", nationalityIndonesian, 3},
	{"BSN", nationalityIndonesian, 4},
	{"AB", nationalityIndonesian, 4}, {"AB", nationalityFilipino, 3}, {"AB", nationalityBurmese, 1},
	{"OS", nationalityIndonesian, 5},
	{"CE", nationalityIndonesian, 2}, {"CE", nationalityIndian, 1}, {"CE", nationalityUkrainian, 1},
	{"2E", nationalityIndonesian, 2}, {"2E", nationalityIndian, 1},
	{"3E", nationalityIndonesian, 3},
	{"4E", nationalityIndonesian, 2},
	{"EC", nationalityIndonesian, 2},
	{"OLR", nationalityIndonesian, 3}, {"OLR", nationalityFilipino, 2}, {"OLR", nationalityBurmese, 1},
	{"WPR", nationalityIndonesian, 3}, {"WPR", nationalityFilipino, 1},
	{"CK", nationalityIndonesian, 2}, {"CK", nationalityFilipino, 1},
	{"STW", nationalityIndonesian, 1}, {"STW", nationalityFilipino, 1},
}

var femaleByRank = map[string]int{"STW": 2, "CK": 1, "3O": 1, "EC": 1}

type ageRange struct{ min, max int }

var ageByRank = map[string]ageRange{
	"MST": {40, 58}, "CE": {40, 58}, "CO": {34, 50}, "2E": {32, 48}, "2O": {30, 44}, "3O": {25, 38}, "3E": {26, 38},
	"4E": {24, 34}, "EC": {20, 24}, "BSN": {32, 50}, "AB": {24, 45}, "OS": {20, 32}, "OLR": {23, 42}, "WPR": {20, 32},
	"CK": {28, 50}, "STW": {22, 40},
}

type crewSpec struct {
	person   demoPerson
	rank     string
	birth    time.Time
	heightCM int
	weightKG int
	marital  string
	shoe     string
	coverall string
	email    string
	mobile   string
	address  string
	postal   string
}

func buildCrewSpecs(rng *rand.Rand, today time.Time) []crewSpec {
	taken := map[string]bool{}
	var specs []crewSpec
	seenInRank := map[string]int{}
	totalInRank := map[string]int{}
	for _, s := range crewSlots {
		totalInRank[s.rank] += s.count
	}
	for _, slot := range crewSlots {
		for range slot.count {
			position := seenInRank[slot.rank]
			seenInRank[slot.rank]++
			gender := "male"
			if position >= totalInRank[slot.rank]-femaleByRank[slot.rank] {
				gender = "female"
			}
			specs = append(specs, newCrewSpec(rng, today, slot, gender, taken, len(specs)))
		}
	}
	return specs
}

func newCrewSpec(rng *rand.Rand, today time.Time, slot crewSlot, gender string, taken map[string]bool, index int) crewSpec {
	person := newPerson(rng, slot.nationality, gender, taken)
	ages := ageByRank[slot.rank]
	birth := today.AddDate(-(ages.min + rng.IntN(ages.max-ages.min+1)), -rng.IntN(12), -rng.IntN(28))
	height, weight := 162+rng.IntN(24), 58+rng.IntN(32)
	if gender == "female" {
		height, weight = 152+rng.IntN(18), 46+rng.IntN(20)
	}
	marital := pick(rng, []string{"single", "married", "married", "married", "divorced"})
	if birth.After(today.AddDate(-26, 0, 0)) {
		marital = pick(rng, []string{"single", "single", "married"})
	}
	return crewSpec{
		person:   person,
		rank:     slot.rank,
		birth:    birth,
		heightCM: height,
		weightKG: weight,
		marital:  marital,
		shoe:     fmt.Sprintf("%d", 39+rng.IntN(7)),
		coverall: pick(rng, []string{"M", "L", "L", "XL", "XXL"}),
		email:    emailFor(person.FullName, index),
		mobile:   mobileFor(rng, slot.nationality),
		address:  addressFor(rng, person),
		postal:   fmt.Sprintf("%05d", 10000+rng.IntN(89999)),
	}
}

func emailFor(fullName string, index int) string {
	local := strings.NewReplacer(" ", ".", "'", "").Replace(strings.ToLower(fullName))
	return fmt.Sprintf("%s.%02d@%s", local, index+1, demoCrewEmailDomain)
}

const demoCrewEmailDomain = "crew.example.com"

func mobileFor(rng *rand.Rand, nationality string) string {
	switch nationality {
	case nationalityFilipino:
		return fmt.Sprintf("+639%09d", rng.IntN(1_000_000_000))
	case nationalityIndian:
		return fmt.Sprintf("+9198%08d", rng.IntN(100_000_000))
	case nationalityUkrainian:
		return fmt.Sprintf("+38050%07d", rng.IntN(10_000_000))
	case nationalityBurmese:
		return fmt.Sprintf("+959%08d", rng.IntN(100_000_000))
	}
	return fmt.Sprintf("+62812%08d", rng.IntN(100_000_000))
}

func addressFor(rng *rand.Rand, p demoPerson) string {
	street := pick(rng, []string{"Jl. Mawar", "Jl. Melati", "Jl. Pelabuhan", "Jl. Bahari", "Jl. Kenanga", "Jl. Nusantara"})
	if p.Country != "Indonesia" {
		street = pick(rng, []string{"Harbour Road", "Marina Street", "Seaview Avenue", "Port Lane"})
	}
	return fmt.Sprintf("%s No. %d", street, 1+rng.IntN(120))
}

type crewRole int

const (
	roleAvailable crewRole = iota
	roleOnBoard
	roleReliever
	roleSignedOff
	roleMedicalLeave
	roleBlacklisted
)

type crewAssignment struct {
	role     crewRole
	contract int
}

type contractPlan struct {
	rank         string
	vessel       int
	remaining    int
	months       int
	reliefDays   int
	reliefStatus string
}

const (
	reliefNone      = ""
	reliefPlanned   = "planned"
	reliefConfirmed = "confirmed"
)

var contractPlans = []contractPlan{
	{"MST", 0, -12, 6, 4, reliefPlanned},
	{"CE", 0, 8, 6, 8, reliefConfirmed},
	{"CO", 1, 20, 6, 20, reliefPlanned},
	{"AB", 1, 45, 6, 45, reliefConfirmed},
	{"OLR", 2, 75, 8, 75, reliefPlanned},
	{"MST", 2, 110, 6, 0, reliefNone},
	{"MST", 3, 140, 8, 0, reliefNone},
	{"CE", 3, 25, 4, 0, reliefNone},
	{"CE", 4, 95, 6, 0, reliefNone},
	{"CO", 4, 55, 6, 0, reliefNone},
	{"2E", 5, 160, 8, 0, reliefNone},
	{"3O", 6, 70, 6, 0, reliefNone},
	{"AB", 7, 35, 4, 0, reliefNone},
	{"OLR", 8, 120, 8, 0, reliefNone},
	{"CK", 9, 15, 4, 0, reliefNone},
}

var (
	signedOffRanks     = []string{"AB", "OS", "WPR", "STW"}
	medicalLeaveRanks  = []string{"OS", "BSN", "EC"}
	blacklistedRanks   = []string{"OS", "WPR"}
	completedMonthsFor = map[string]int{"AB": 6, "OS": 4, "WPR": 6, "STW": 4}
)

func assignRoles(specs []crewSpec) []crewAssignment {
	out := make([]crewAssignment, len(specs))
	taken := make([]bool, len(specs))
	claim := func(rank string, role crewRole, contract int) {
		for i, s := range specs {
			if s.rank == rank && !taken[i] {
				taken[i] = true
				out[i] = crewAssignment{role: role, contract: contract}
				return
			}
		}
	}
	for i, c := range contractPlans {
		claim(c.rank, roleOnBoard, i)
	}
	for i, c := range contractPlans {
		if c.reliefStatus != reliefNone {
			claim(c.rank, roleReliever, i)
		}
	}
	for _, rank := range signedOffRanks {
		claim(rank, roleSignedOff, -1)
	}
	for _, rank := range medicalLeaveRanks {
		claim(rank, roleMedicalLeave, -1)
	}
	for _, rank := range blacklistedRanks {
		claim(rank, roleBlacklisted, -1)
	}
	return out
}

func experienceCounts(rng *rand.Rand, crew, total int) []int {
	counts := make([]int, crew)
	for i := range counts {
		counts[i] = 1
	}
	for range total - crew {
		counts[rng.IntN(crew)]++
	}
	return counts
}
