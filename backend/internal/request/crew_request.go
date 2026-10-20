package request

import (
	"strings"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// Crew is the body of POST /crews and PUT /crews/{id} (Personal Data and Contact tabs).
type Crew struct {
	FullName       string `json:"full_name"`
	PlaceOfBirth   string `json:"place_of_birth"`
	DateOfBirth    string `json:"date_of_birth"`
	Gender         string `json:"gender"`
	Nationality    string `json:"nationality"`
	Religion       string `json:"religion"`
	MaritalStatus  string `json:"marital_status"`
	HeightCM       *int   `json:"height_cm"`
	WeightKG       *int   `json:"weight_kg"`
	ShoeSize       string `json:"shoe_size"`
	CoverallSize   string `json:"coverall_size"`
	AppliedRankID  string `json:"applied_rank_id"`
	PhotoFileID    string `json:"photo_file_id"`
	CVFileID       string `json:"cv_file_id"`
	Address        string `json:"address"`
	City           string `json:"city"`
	Province       string `json:"province"`
	Country        string `json:"country"`
	PostalCode     string `json:"postal_code"`
	HomePhone      string `json:"home_phone"`
	MobilePhone    string `json:"mobile_phone"`
	Email          string `json:"email"`
	NearestAirport string `json:"nearest_airport"`

	parsed model.Crew
}

// Validate implements Validator.
func (r *Crew) Validate() map[string]string {
	Trim(&r.FullName, &r.PlaceOfBirth, &r.DateOfBirth, &r.Gender, &r.Nationality, &r.Religion, &r.MaritalStatus,
		&r.ShoeSize, &r.CoverallSize, &r.AppliedRankID, &r.PhotoFileID, &r.CVFileID, &r.Address, &r.City, &r.Province,
		&r.Country, &r.PostalCode, &r.HomePhone, &r.MobilePhone, &r.Email, &r.NearestAirport)
	r.Email = strings.ToLower(r.Email)
	e := Errors{}
	e.Required("full_name", r.FullName, 150)
	for field, v := range map[string]string{
		"place_of_birth": r.PlaceOfBirth, "nationality": r.Nationality, "religion": r.Religion, "city": r.City,
		"province": r.Province, "country": r.Country, "nearest_airport": r.NearestAirport,
	} {
		e.MaxLen(field, v, 100)
	}
	e.MaxLen("address", r.Address, 500)
	e.MaxLen("postal_code", r.PostalCode, 20)
	e.MaxLen("shoe_size", r.ShoeSize, 10)
	e.MaxLen("coverall_size", r.CoverallSize, 10)
	e.OneOf("gender", r.Gender, "male", "female")
	e.OneOf("marital_status", r.MaritalStatus, "single", "married", "divorced", "widowed")
	e.Range("height_cm", r.HeightCM, 50, 250)
	e.Range("weight_kg", r.WeightKG, 20, 300)
	e.Phone("mobile_phone", r.MobilePhone)
	e.Phone("home_phone", r.HomePhone)
	e.Email("email", r.Email)

	c := model.Crew{
		FullName: r.FullName, PlaceOfBirth: r.PlaceOfBirth, Nationality: r.Nationality, Religion: r.Religion,
		ShoeSize: r.ShoeSize, CoverallSize: r.CoverallSize, Address: r.Address, City: r.City, Province: r.Province,
		Country: r.Country, PostalCode: r.PostalCode, HomePhone: r.HomePhone, MobilePhone: r.MobilePhone,
		Email: r.Email, NearestAirport: r.NearestAirport,
	}
	c.DateOfBirth = e.Date("date_of_birth", r.DateOfBirth)
	c.AppliedRankID = e.UUID("applied_rank_id", r.AppliedRankID)
	c.PhotoFileID = e.UUID("photo_file_id", r.PhotoFileID)
	c.CVFileID = e.UUID("cv_file_id", r.CVFileID)
	c.Gender = optString(r.Gender)
	c.MaritalStatus = optString(r.MaritalStatus)
	c.HeightCM = optInt16(r.HeightCM)
	c.WeightKG = optInt16(r.WeightKG)
	r.parsed = c
	return e
}

// Model returns the validated crew.
func (r *Crew) Model() model.Crew { return r.parsed }

// CrewStatus is the body of PATCH /crews/{id}/status.
type CrewStatus struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

// BankAccount is the body for crew bank accounts.
type BankAccount struct {
	BankName      string `json:"bank_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	CurrencyCode  string `json:"currency_code"`
	CoverBank     string `json:"cover_bank"`
	Remarks       string `json:"remarks"`
	IsPrimary     bool   `json:"is_primary"`
}

// Validate implements Validator.
func (r *BankAccount) Validate() map[string]string {
	Trim(&r.BankName, &r.AccountName, &r.AccountNumber, &r.CurrencyCode, &r.CoverBank, &r.Remarks)
	r.CurrencyCode = strings.ToUpper(r.CurrencyCode)
	e := Errors{}
	e.Required("bank_name", r.BankName, 100)
	e.Required("account_name", r.AccountName, 150)
	e.Required("account_number", r.AccountNumber, 50)
	e.Required("currency_code", r.CurrencyCode, 3)
	e.MaxLen("cover_bank", r.CoverBank, 150)
	e.MaxLen("remarks", r.Remarks, 300)
	return e
}

// Model maps the request onto a bank account.
func (r *BankAccount) Model(crewID uuid.UUID) model.BankAccount {
	return model.BankAccount{CrewID: crewID, BankName: r.BankName, AccountName: r.AccountName, AccountNumber: r.AccountNumber,
		CurrencyCode: r.CurrencyCode, CoverBank: r.CoverBank, Remarks: r.Remarks, IsPrimary: r.IsPrimary}
}

// NextOfKin is the body for next of kin rows.
type NextOfKin struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	IsPrimary    bool   `json:"is_primary"`
}

// Validate implements Validator.
func (r *NextOfKin) Validate() map[string]string {
	Trim(&r.Name, &r.Relationship, &r.Phone, &r.Address)
	e := Errors{}
	e.Required("name", r.Name, 150)
	e.Required("relationship", r.Relationship, 20)
	e.OneOf("relationship", r.Relationship, "spouse", "parent", "child", "sibling", "other")
	e.Phone("phone", r.Phone)
	e.MaxLen("address", r.Address, 500)
	return e
}

// Model maps the request onto a next of kin.
func (r *NextOfKin) Model(crewID uuid.UUID) model.NextOfKin {
	return model.NextOfKin{CrewID: crewID, Name: r.Name, Relationship: r.Relationship, Phone: r.Phone, Address: r.Address, IsPrimary: r.IsPrimary}
}

// CrewDocument is the body for documents and certificates.
type CrewDocument struct {
	TypeID           string `json:"type_id"`
	Number           string `json:"number"`
	IssuedDate       string `json:"issued_date"`
	ExpiredDate      string `json:"expired_date"`
	IssuingAuthority string `json:"issuing_authority"`
	PlaceOfIssue     string `json:"place_of_issue"`
	FileID           string `json:"file_id"`

	parsed model.CrewDocument
}

// Validate implements Validator.
func (r *CrewDocument) Validate() map[string]string {
	Trim(&r.TypeID, &r.Number, &r.IssuedDate, &r.ExpiredDate, &r.IssuingAuthority, &r.PlaceOfIssue, &r.FileID)
	e := Errors{}
	d := model.CrewDocument{Number: r.Number, IssuingAuthority: r.IssuingAuthority, PlaceOfIssue: r.PlaceOfIssue}
	d.TypeID = e.RequiredUUID("type_id", r.TypeID)
	e.Required("number", r.Number, 60)
	e.MaxLen("issuing_authority", r.IssuingAuthority, 150)
	e.MaxLen("place_of_issue", r.PlaceOfIssue, 100)
	d.IssuedDate = e.Date("issued_date", r.IssuedDate)
	d.ExpiredDate = e.Date("expired_date", r.ExpiredDate)
	d.FileID = e.UUID("file_id", r.FileID)
	e.After("expired_date", d.IssuedDate, d.ExpiredDate, "The expiry date must be on or after the issued date.")
	r.parsed = d
	return e
}

// Model returns the validated row for crewID.
func (r *CrewDocument) Model(crewID uuid.UUID) model.CrewDocument {
	d := r.parsed
	d.CrewID = crewID
	return d
}

// Experience is the body for manual sea service rows.
type Experience struct {
	VesselName    string `json:"vessel_name"`
	VesselID      string `json:"vessel_id"`
	RankID        string `json:"rank_id"`
	VesselTypeID  string `json:"vessel_type_id"`
	GRTHP         string `json:"grt_hp"`
	CompanyID     string `json:"company_id"`
	CompanyName   string `json:"company_name"`
	PrincipalID   string `json:"principal_id"`
	PrincipalName string `json:"principal_name"`
	SignOnDate    string `json:"sign_on_date"`
	SignOffDate   string `json:"sign_off_date"`
	Remarks       string `json:"remarks"`

	parsed model.Experience
}

// Validate implements Validator.
func (r *Experience) Validate() map[string]string {
	Trim(&r.VesselName, &r.VesselID, &r.RankID, &r.VesselTypeID, &r.GRTHP, &r.CompanyID, &r.CompanyName,
		&r.PrincipalID, &r.PrincipalName, &r.SignOnDate, &r.SignOffDate, &r.Remarks)
	e := Errors{}
	x := model.Experience{VesselName: r.VesselName, GRTHP: r.GRTHP, CompanyName: r.CompanyName, PrincipalName: r.PrincipalName, Remarks: r.Remarks, Source: "manual"}
	e.Required("vessel_name", r.VesselName, 150)
	e.MaxLen("grt_hp", r.GRTHP, 60)
	e.MaxLen("company_name", r.CompanyName, 150)
	e.MaxLen("principal_name", r.PrincipalName, 150)
	e.MaxLen("remarks", r.Remarks, 500)
	x.VesselID = e.UUID("vessel_id", r.VesselID)
	x.RankID = e.RequiredUUID("rank_id", r.RankID)
	x.VesselTypeID = e.UUID("vessel_type_id", r.VesselTypeID)
	x.CompanyID = e.UUID("company_id", r.CompanyID)
	x.PrincipalID = e.UUID("principal_id", r.PrincipalID)
	x.SignOnDate = e.RequiredDate("sign_on_date", r.SignOnDate)
	x.SignOffDate = e.Date("sign_off_date", r.SignOffDate)
	e.After("sign_off_date", &x.SignOnDate, x.SignOffDate, "The sign-off date must be on or after the sign-on date.")
	x.Status = "sign_off"
	if x.SignOffDate == nil {
		x.Status = "on_board"
	}
	r.parsed = x
	return e
}

// Model returns the validated experience for crewID.
func (r *Experience) Model(crewID uuid.UUID) model.Experience {
	x := r.parsed
	x.CrewID = crewID
	return x
}

func optString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optInt16(n *int) *int16 {
	if n == nil {
		return nil
	}
	v := int16(min(max(*n, -32768), 32767))
	return &v
}
