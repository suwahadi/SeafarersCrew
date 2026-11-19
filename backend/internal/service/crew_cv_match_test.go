package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
)

var (
	cvBST     = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	cvAFF     = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	cvMFA     = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	cvPSCRB   = uuid.MustParse("00000000-0000-0000-0000-000000000004")
	cvSAT     = uuid.MustParse("00000000-0000-0000-0000-000000000005")
	cvBOSIET  = uuid.MustParse("00000000-0000-0000-0000-000000000006")
	cvPass    = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	cvSeaman  = uuid.MustParse("00000000-0000-0000-0000-000000000011")
	cvMCU     = uuid.MustParse("00000000-0000-0000-0000-000000000012")
	cvAB      = uuid.MustParse("00000000-0000-0000-0000-000000000020")
	cvOS      = uuid.MustParse("00000000-0000-0000-0000-000000000021")
	cvAHTS    = uuid.MustParse("00000000-0000-0000-0000-000000000030")
	cvBulk    = uuid.MustParse("00000000-0000-0000-0000-000000000031")
	cvCompany = uuid.MustParse("00000000-0000-0000-0000-000000000040")
	cvAliased = uuid.MustParse("00000000-0000-0000-0000-000000000050")
)

func cvTestLookups() repository.CVLookups {
	return repository.CVLookups{
		CertificateTypes: []repository.CVMasterItem{
			{ID: cvBST, Code: "bst", Name: "Basic Safety Training", HasExpiry: true},
			{ID: cvAFF, Code: "aff", Name: "Advanced Fire Fighting", HasExpiry: true},
			{ID: cvMFA, Code: "mfa", Name: "Medical First Aid", HasExpiry: true},
			{ID: cvPSCRB, Code: "pscrb", Name: "Proficiency in Survival Craft and Rescue Boats", HasExpiry: true},
			{ID: cvSAT, Code: "sat", Name: "Security Awareness Training", HasExpiry: true},
			{ID: cvBOSIET, Code: "bosiet", Name: "BOSIET", HasExpiry: true},
			{ID: cvAliased, Code: "aliased", Name: "Aliased Course", HasExpiry: true},
		},
		DocumentTypes: []repository.CVMasterItem{
			{ID: cvPass, Code: "passport", Name: "Passport", HasExpiry: true},
			{ID: cvSeaman, Code: "seaman_book", Name: "Seaman Book (CDC)", HasExpiry: true},
			{ID: cvMCU, Code: "mcu", Name: "Medical Check Up", HasExpiry: true},
		},
		Ranks:       []repository.CVMasterItem{{ID: cvAB, Code: "AB", Name: "AB"}, {ID: cvOS, Code: "OS", Name: "Ordinary Seaman"}},
		VesselTypes: []repository.CVMasterItem{{ID: cvAHTS, Name: "AHTS"}, {ID: cvBulk, Name: "Bulk Carrier"}},
		Companies:   []repository.CVMasterItem{{ID: cvCompany, Code: "SAM", Name: "PT. Samudera Contoh"}},
		Aliases: map[string]map[string]uuid.UUID{
			repository.CVKindCertificateType: {"BASIC SAFETY": cvAliased},
		},
		PassportType: cvPass, SeamanBookType: cvSeaman,
	}
}

func TestCVMatcher_Certificates(t *testing.T) {
	m := newCVMatcher(cvTestLookups())
	cases := []struct {
		text string
		kind string
		id   uuid.UUID
		by   string
	}{
		{"Basic Safety Training / BST", repository.CVKindCertificateType, cvBST, "name"},
		{"Advance Firefighting / AFF", repository.CVKindCertificateType, cvAFF, "code"},
		{"Proficiency in Survival Craft and  Rescue Boats / SCRB - on process revalidation", repository.CVKindCertificateType, cvPSCRB, "name"},
		{"Basic Offshore Safety Induction & Emergency Training / BOSIET", repository.CVKindCertificateType, cvBOSIET, "code"},
		{"Medical Check Up", repository.CVKindDocumentType, cvMCU, "name"},
		{"basic  safety", repository.CVKindCertificateType, cvAliased, "alias"},
	}
	for _, tc := range cases {
		t.Run(tc.text, func(t *testing.T) {
			got, ok := m.certificate(tc.text)
			assert.True(t, ok)
			assert.Equal(t, cvMatch{Kind: tc.kind, ID: tc.id, By: tc.by}, got)
		})
	}
	_, ok := m.certificate("Ship Security Officer (SSO)")
	assert.False(t, ok, "SSO in parentheses is not the code of any type")
	_, ok = m.certificate("Hydrogen Sulfide / H2S")
	assert.False(t, ok)
}

func TestCVMatcher_DocumentsRanksVesselsCompanies(t *testing.T) {
	m := newCVMatcher(cvTestLookups())
	doc, ok := m.document("Seaman Book")
	assert.True(t, ok)
	assert.Equal(t, cvSeaman, doc.ID)
	doc, ok = m.document("Passport")
	assert.True(t, ok)
	assert.Equal(t, cvPass, doc.ID)

	rank, ok := m.rank("ab")
	assert.True(t, ok)
	assert.Equal(t, cvMatch{Kind: repository.CVKindRank, ID: cvAB, By: "code"}, rank)
	rank, ok = m.rank("ORDINARY SEAMAN")
	assert.True(t, ok)
	assert.Equal(t, cvOS, rank.ID)
	_, ok = m.rank("XX")
	assert.False(t, ok)

	vt, ok := m.vesselType("AHTS DP 1")
	assert.True(t, ok)
	assert.Equal(t, cvMatch{Kind: repository.CVKindVesselType, ID: cvAHTS, By: "prefix"}, vt)
	vt, ok = m.vesselType("BULK CARRIER")
	assert.True(t, ok)
	assert.Equal(t, cvMatch{Kind: repository.CVKindVesselType, ID: cvBulk, By: "name"}, vt)
	_, ok = m.vesselType("TANKER")
	assert.False(t, ok)

	assert.Equal(t, &cvCompany, m.company("PT. SAMUDERA CONTOH"))
	assert.Nil(t, m.company("OFFSHORE CONTOH SDN BHD"))
}

func TestNormalizeCVPhone(t *testing.T) {
	cases := []struct {
		raw, nationality, want string
		ok                     bool
	}{
		{"+62.812.3456.7890", "INDONESIAN", "+6281234567890", true},
		{"0812-3456-7890", "INDONESIAN", "+6281234567890", true},
		{"0812 3456 7890", "Indonesia", "+6281234567890", true},
		{"0812-3456-7890", "FILIPINO", "", false},
		{"0063 917 123 4567", "FILIPINO", "+639171234567", true},
		{"12345", "INDONESIAN", "", false},
		{"", "INDONESIAN", "", true},
	}
	for _, tc := range cases {
		got, ok := normalizeCVPhone(tc.raw, tc.nationality)
		assert.Equal(t, tc.want, got, tc.raw)
		assert.Equal(t, tc.ok, ok, tc.raw)
	}
}

func TestMapCVRelationship(t *testing.T) {
	for in, want := range map[string]string{"WIFE": "spouse", "Husband": "spouse", "MOTHER": "parent", "Father": "parent", "SON": "child",
		"daughter": "child", "BROTHER": "sibling", "SISTER": "sibling", "COUSIN": "other", "": "other"} {
		assert.Equal(t, want, mapCVRelationship(in), in)
	}
}
