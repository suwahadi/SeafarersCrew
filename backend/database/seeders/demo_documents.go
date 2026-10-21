package seeders

import (
	"fmt"
	"math/rand/v2"
	"time"
)

type documentDraft struct {
	typeCode  string
	number    string
	issued    time.Time
	expiry    *time.Time
	authority string
	place     string
}

type crewDocumentSet struct {
	documents    []documentDraft
	certificates []documentDraft
}

const (
	validityPassport = 1825
	validityMCU      = 730
	validityVisa     = 730
	validityCert     = 1825
	validityOffshore = 1460
)

func buildCrewDocuments(rng *rand.Rand, spec crewSpec, index int, today time.Time) crewDocumentSet {
	indonesian := spec.person.Nationality == nationalityIndonesian
	authority := "National Maritime Authority"
	if indonesian {
		authority = "Directorate General of Sea Transportation"
	}
	place := spec.person.City
	dated := func(code, number string, validityDays int, place, authority string) documentDraft {
		item := expiryFor(rng, pickExpiryBucket(rng), today, validityDays)
		expiry := item.expiry
		return documentDraft{typeCode: code, number: number, issued: item.issued, expiry: &expiry, authority: authority, place: place}
	}
	undated := func(code, number string, place, authority string) documentDraft {
		issued := today.AddDate(0, 0, -(200 + rng.IntN(1500)))
		return documentDraft{typeCode: code, number: number, issued: issued, authority: authority, place: place}
	}

	var set crewDocumentSet
	if indonesian {
		set.documents = append(set.documents, undated("ktp", digits(rng, 16), place, ""))
	}
	set.documents = append(set.documents, dated("passport", letterAndDigits(rng, 7), validityPassport, immigrationOffice(spec.person), ""))
	set.documents = append(set.documents, dated("seaman_book", letterAndDigits(rng, 6), validityPassport, place, ""))
	if indonesian {
		set.documents = append(set.documents, dated("buku_pelaut", letterAndDigits(rng, 6), validityPassport, place, ""))
	}
	if index%13 != 5 {
		set.documents = append(set.documents, dated("mcu", fmt.Sprintf("MCU-%d-%04d", today.Year(), rng.IntN(10000)), validityMCU, place+" Seafarers Clinic", ""))
	}
	set.documents = append(set.documents, undated("yellow_fever", fmt.Sprintf("YF-%06d", rng.IntN(1_000_000)), place, ""))
	if rng.IntN(5) == 0 {
		set.documents = append(set.documents, dated("visa", letterAndDigits(rng, 8), validityVisa, "Singapore", ""))
	}

	cert := func(code string, validity int) {
		set.certificates = append(set.certificates, dated(code, fmt.Sprintf("%s-%s", certPrefix(code), digits(rng, 8)), validity, place, authority))
	}
	undatedCert := func(code string) {
		set.certificates = append(set.certificates, undated(code, fmt.Sprintf("%s-%s", certPrefix(code), digits(rng, 8)), place, authority))
	}
	if index%17 != 9 {
		cert("bst", validityCert)
	}
	undatedCert("sat")
	if coc, ok := cocByRank[spec.rank]; ok {
		cert(coc, validityCert)
	}
	switch spec.rank {
	case "AB", "OS", "BSN":
		undatedCert("rating_nav_watch")
	case "OLR", "WPR":
		undatedCert("rating_eng_watch")
	}
	officer := cocByRank[spec.rank] != ""
	chance := func(officerOdds, otherOdds int) bool {
		odds := otherOdds
		if officer {
			odds = officerOdds
		}
		return rng.IntN(100) < odds
	}
	if chance(90, 55) {
		cert("aff", validityCert)
	}
	if chance(80, 45) {
		cert("mfa", validityCert)
	}
	if chance(60, 35) {
		cert("pscrb", validityCert)
	}
	if (spec.rank == "MST" || spec.rank == "CO" || spec.rank == "2O" || spec.rank == "3O") && rng.IntN(100) < 60 {
		cert("gmdss", validityCert)
	}
	if rng.IntN(100) < 45 {
		cert("bosiet", validityOffshore)
	}
	if rng.IntN(100) < 45 {
		cert("huet", validityOffshore)
	}
	return set
}

func certPrefix(code string) string {
	switch code {
	case "bst":
		return "BST"
	case "sat":
		return "SAT"
	case "aff":
		return "AFF"
	case "mfa":
		return "MFA"
	case "pscrb":
		return "PSCRB"
	case "gmdss":
		return "GMDSS"
	case "bosiet":
		return "BOSIET"
	case "huet":
		return "HUET"
	case "rating_nav_watch":
		return "NAV"
	case "rating_eng_watch":
		return "ENG"
	}
	return "COC"
}

func immigrationOffice(p demoPerson) string {
	if p.Country == "Indonesia" {
		return "Imigrasi " + p.City
	}
	return p.City
}

func digits(rng *rand.Rand, n int) string {
	const alphabet = "0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rng.IntN(len(alphabet))]
	}
	if b[0] == '0' {
		b[0] = '1'
	}
	return string(b)
}

func letterAndDigits(rng *rand.Rand, n int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return string(letters[rng.IntN(len(letters))]) + digits(rng, n)
}
