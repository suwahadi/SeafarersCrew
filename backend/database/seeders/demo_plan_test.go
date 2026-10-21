package seeders

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

var demoToday = time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC)

func newTestRNG(stream uint64) *rand.Rand { return rand.New(rand.NewPCG(demoSeed, stream)) }

func TestBuildCrewSpecsProducesSixtyUniqueCrewWithInternationalNames(t *testing.T) {
	specs := buildCrewSpecs(newTestRNG(1), demoToday)

	require.Len(t, specs, demoCrewTotal)
	names, emails := map[string]bool{}, map[string]bool{}
	nationalities := map[string]int{}
	female := 0
	for _, s := range specs {
		assert.False(t, names[s.person.FullName], "duplicate name %s", s.person.FullName)
		assert.False(t, emails[s.email], "duplicate email %s", s.email)
		names[s.person.FullName], emails[s.email] = true, true
		nationalities[s.person.Nationality]++
		if s.person.Gender == "female" {
			female++
		}
		assert.True(t, strings.HasSuffix(s.email, "@"+demoCrewEmailDomain))
		assert.Regexp(t, `^\+[1-9][0-9]{6,14}$`, s.mobile, s.person.FullName)
		assert.Regexp(t, `^[0-9]{5}$`, s.postal)
	}
	assert.Equal(t, 45, nationalities[nationalityIndonesian])
	assert.Equal(t, 8, nationalities[nationalityFilipino])
	assert.Equal(t, 3, nationalities[nationalityIndian])
	assert.Equal(t, 2, nationalities[nationalityUkrainian])
	assert.Equal(t, 2, nationalities[nationalityBurmese])
	assert.Equal(t, 5, female)
}

func TestBuildCrewSpecsIsDeterministic(t *testing.T) {
	assert.Equal(t, buildCrewSpecs(newTestRNG(1), demoToday), buildCrewSpecs(newTestRNG(1), demoToday))
	assert.NotEqual(t, buildCrewSpecs(newTestRNG(1), demoToday), buildCrewSpecs(newTestRNG(2), demoToday))
}

func TestBuildCrewSpecsKeepsAgeAndBodyMeasurementsRealistic(t *testing.T) {
	for _, s := range buildCrewSpecs(newTestRNG(1), demoToday) {
		ages := ageByRank[s.rank]
		years := demoToday.Year() - s.birth.Year()
		assert.GreaterOrEqual(t, years, ages.min-1, s.person.FullName)
		assert.LessOrEqual(t, years, ages.max+1, s.person.FullName)
		assert.GreaterOrEqual(t, s.heightCM, 50)
		assert.LessOrEqual(t, s.heightCM, 250)
		assert.GreaterOrEqual(t, s.weightKG, 20)
		assert.LessOrEqual(t, s.weightKG, 300)
	}
}

func TestAssignRolesGivesEachBusinessRuleExactlyTheExpectedCrew(t *testing.T) {
	specs := buildCrewSpecs(newTestRNG(1), demoToday)
	roles := assignRoles(specs)

	counts := map[crewRole]int{}
	onBoardByContract := map[int]string{}
	for i, r := range roles {
		counts[r.role]++
		if r.role == roleOnBoard {
			_, dup := onBoardByContract[r.contract]
			assert.False(t, dup, "contract %d has two crew", r.contract)
			onBoardByContract[r.contract] = specs[i].rank
			assert.Equal(t, contractPlans[r.contract].rank, specs[i].rank)
		}
		if r.role == roleReliever {
			assert.Equal(t, contractPlans[r.contract].rank, specs[i].rank)
			assert.NotEqual(t, reliefNone, contractPlans[r.contract].reliefStatus)
		}
	}
	assert.Equal(t, 15, counts[roleOnBoard])
	assert.Equal(t, 5, counts[roleReliever])
	assert.Equal(t, 4, counts[roleSignedOff])
	assert.Equal(t, 3, counts[roleMedicalLeave])
	assert.Equal(t, 2, counts[roleBlacklisted])
	assert.Equal(t, demoCrewTotal, counts[roleAvailable]+counts[roleOnBoard]+counts[roleReliever]+counts[roleSignedOff]+counts[roleMedicalLeave]+counts[roleBlacklisted])
}

func TestContractPlansAreConsistent(t *testing.T) {
	vessels := map[int]bool{}
	reliefs := 0
	ranksPerVessel := map[[2]interface{}]bool{}
	for i, c := range contractPlans {
		vessels[c.vessel] = true
		assert.LessOrEqual(t, c.remaining, c.months*30-5, "contract %d would start in the future", i)
		key := [2]interface{}{c.vessel, c.rank}
		assert.False(t, ranksPerVessel[key], "vessel %d has two %s", c.vessel, c.rank)
		ranksPerVessel[key] = true
		if c.reliefStatus != reliefNone {
			reliefs++
			assert.Contains(t, []string{reliefPlanned, reliefConfirmed}, c.reliefStatus)
		}
	}
	assert.Len(t, contractPlans, 15)
	assert.Len(t, vessels, demoVesselTotal)
	assert.Equal(t, 5, reliefs)
	assert.Contains(t, contractRemaining(), -12, "one contract is overdue")
}

func contractRemaining() []int {
	out := make([]int, len(contractPlans))
	for i, c := range contractPlans {
		out[i] = c.remaining
	}
	return out
}

func TestExperienceCountsSumToTheRequestedTotal(t *testing.T) {
	counts := experienceCounts(newTestRNG(3), demoCrewTotal, 116)
	sum := 0
	for _, c := range counts {
		assert.GreaterOrEqual(t, c, 1)
		sum += c
	}
	assert.Equal(t, 116, sum)
}

func TestIMOWithCheckDigit(t *testing.T) {
	got, err := imoWithCheckDigit(907472)
	require.NoError(t, err)
	assert.Equal(t, "9074729", got, "check digit of the IMO weighting 7 to 2")
	assert.True(t, service.ValidIMO(got))

	_, err = imoWithCheckDigit(99)
	assert.Error(t, err)
}

func TestGenerateIMOsReturnsUniqueValidNumbers(t *testing.T) {
	imos, err := generateIMOs(newTestRNG(4), 10)
	require.NoError(t, err)
	seen := map[string]bool{}
	for _, imo := range imos {
		assert.True(t, service.ValidIMO(imo), imo)
		assert.False(t, seen[imo])
		seen[imo] = true
	}
	assert.Len(t, imos, 10)
}

func TestExpiryForPlacesTheDateInsideTheRequestedBucket(t *testing.T) {
	tests := []struct {
		name   string
		bucket expiryBucket
		min    int
		max    int
	}{
		{"expired", bucketExpired, -180, -1},
		{"within 30 days", bucketWithin30, 1, 30},
		{"within 60 days", bucketWithin60, 31, 60},
		{"within 90 days", bucketWithin90, 61, 90},
		{"valid", bucketValid, 91, 1824},
	}
	rng := newTestRNG(5)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for range 200 {
				item := expiryFor(rng, tc.bucket, demoToday, 1825)
				days := int(item.expiry.Sub(demoToday).Hours() / 24)
				assert.GreaterOrEqual(t, days, tc.min)
				assert.LessOrEqual(t, days, tc.max)
				assert.Equal(t, tc.bucket, bucketOf(item.expiry, demoToday))
				assert.True(t, item.issued.Before(demoToday), "issued date must be in the past")
				assert.True(t, item.issued.Before(item.expiry))
			}
		})
	}
}

func TestPickExpiryBucketCoversEveryBucketAndMostAreValid(t *testing.T) {
	rng := newTestRNG(6)
	counts := map[expiryBucket]int{}
	for range 1000 {
		counts[pickExpiryBucket(rng)]++
	}
	for _, b := range []expiryBucket{bucketExpired, bucketWithin30, bucketWithin60, bucketWithin90, bucketValid} {
		assert.Positive(t, counts[b], "bucket %d", b)
	}
	assert.Greater(t, counts[bucketValid], 600)
}

func TestBucketOfUsesTheBadgeBoundaries(t *testing.T) {
	tests := []struct {
		days int
		want expiryBucket
	}{
		{-1, bucketExpired}, {0, bucketWithin30}, {30, bucketWithin30}, {31, bucketWithin60}, {60, bucketWithin60},
		{61, bucketWithin90}, {90, bucketWithin90}, {91, bucketValid},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, bucketOf(demoToday.AddDate(0, 0, tc.days), demoToday), "%d days", tc.days)
	}
}
