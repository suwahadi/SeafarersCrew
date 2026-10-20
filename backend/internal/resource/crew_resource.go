package resource

import (
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// Date marshals a date-only value as YYYY-MM-DD.
type Date time.Time

// MarshalJSON implements json.Marshaler.
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format(datetime.DateLayout) + `"`), nil
}

// NewDate converts an optional time to an optional Date.
func NewDate(t *time.Time) *Date {
	if t == nil {
		return nil
	}
	d := Date(*t)
	return &d
}

// Status is a crew status badge.
type Status struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Crew is the crew record returned by the API.
type Crew struct {
	ID              string     `json:"id"`
	FullName        string     `json:"full_name"`
	PlaceOfBirth    string     `json:"place_of_birth"`
	DateOfBirth     *Date      `json:"date_of_birth"`
	Age             *int       `json:"age"`
	Gender          *string    `json:"gender"`
	Nationality     string     `json:"nationality"`
	Religion        string     `json:"religion"`
	MaritalStatus   *string    `json:"marital_status"`
	HeightCM        *int16     `json:"height_cm"`
	WeightKG        *int16     `json:"weight_kg"`
	ShoeSize        string     `json:"shoe_size"`
	CoverallSize    string     `json:"coverall_size"`
	AppliedRankID   *uuid.UUID `json:"applied_rank_id"`
	AppliedRankName string     `json:"applied_rank_name"`
	Status          Status     `json:"status"`
	StatusReason    string     `json:"status_reason"`
	PhotoFileID     *uuid.UUID `json:"photo_file_id"`
	CVFileID        *uuid.UUID `json:"cv_file_id"`
	Address         string     `json:"address"`
	City            string     `json:"city"`
	Province        string     `json:"province"`
	Country         string     `json:"country"`
	PostalCode      string     `json:"postal_code"`
	HomePhone       string     `json:"home_phone"`
	MobilePhone     string     `json:"mobile_phone"`
	Email           string     `json:"email"`
	NearestAirport  string     `json:"nearest_airport"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// NewCrew maps a crew record; today is used to compute the age.
func NewCrew(c model.Crew, today time.Time) Crew {
	return Crew{
		ID: c.PublicID, FullName: c.FullName, PlaceOfBirth: c.PlaceOfBirth, DateOfBirth: NewDate(c.DateOfBirth),
		Age: Age(c.DateOfBirth, today), Gender: c.Gender, Nationality: c.Nationality, Religion: c.Religion,
		MaritalStatus: c.MaritalStatus, HeightCM: c.HeightCM, WeightKG: c.WeightKG, ShoeSize: c.ShoeSize,
		CoverallSize: c.CoverallSize, AppliedRankID: c.AppliedRankID, AppliedRankName: c.AppliedRankName,
		Status: Status{Code: c.StatusCode, Name: c.StatusName, Color: c.StatusColor}, StatusReason: c.StatusReason,
		PhotoFileID: c.PhotoFileID, CVFileID: c.CVFileID, Address: c.Address, City: c.City, Province: c.Province,
		Country: c.Country, PostalCode: c.PostalCode, HomePhone: c.HomePhone, MobilePhone: c.MobilePhone,
		Email: c.Email, NearestAirport: c.NearestAirport, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

// Age returns completed years between dob and today.
func Age(dob *time.Time, today time.Time) *int {
	if dob == nil {
		return nil
	}
	years := today.Year() - dob.Year()
	if today.Month() < dob.Month() || (today.Month() == dob.Month() && today.Day() < dob.Day()) {
		years--
	}
	return &years
}

// BankAccount is a crew bank account.
type BankAccount struct {
	ID            uuid.UUID `json:"id"`
	BankName      string    `json:"bank_name"`
	AccountName   string    `json:"account_name"`
	AccountNumber string    `json:"account_number"`
	CurrencyCode  string    `json:"currency_code"`
	CoverBank     string    `json:"cover_bank"`
	Remarks       string    `json:"remarks"`
	IsPrimary     bool      `json:"is_primary"`
}

// NewBankAccounts maps bank accounts.
func NewBankAccounts(in []model.BankAccount) []BankAccount {
	out := make([]BankAccount, len(in))
	for i, b := range in {
		out[i] = NewBankAccount(b)
	}
	return out
}

// NewBankAccount maps a bank account.
func NewBankAccount(b model.BankAccount) BankAccount {
	return BankAccount{ID: b.ID, BankName: b.BankName, AccountName: b.AccountName, AccountNumber: b.AccountNumber,
		CurrencyCode: b.CurrencyCode, CoverBank: b.CoverBank, Remarks: b.Remarks, IsPrimary: b.IsPrimary}
}

// NextOfKin is an emergency contact.
type NextOfKin struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Relationship string    `json:"relationship"`
	Phone        string    `json:"phone"`
	Address      string    `json:"address"`
	IsPrimary    bool      `json:"is_primary"`
}

// NewNextOfKins maps next of kin rows.
func NewNextOfKins(in []model.NextOfKin) []NextOfKin {
	out := make([]NextOfKin, len(in))
	for i, k := range in {
		out[i] = NewNextOfKin(k)
	}
	return out
}

// NewNextOfKin maps a next of kin.
func NewNextOfKin(k model.NextOfKin) NextOfKin {
	return NextOfKin{ID: k.ID, Name: k.Name, Relationship: k.Relationship, Phone: k.Phone, Address: k.Address, IsPrimary: k.IsPrimary}
}

// Document is a document or certificate row with its expiry badge.
type Document struct {
	ID               uuid.UUID  `json:"id"`
	TypeID           uuid.UUID  `json:"type_id"`
	TypeName         string     `json:"type_name"`
	HasExpiry        bool       `json:"has_expiry"`
	Number           string     `json:"number"`
	IssuedDate       *Date      `json:"issued_date"`
	ExpiredDate      *Date      `json:"expired_date"`
	IssuingAuthority string     `json:"issuing_authority,omitempty"`
	PlaceOfIssue     string     `json:"place_of_issue"`
	FileID           *uuid.UUID `json:"file_id"`
	Badge            string     `json:"badge"`
	DaysLeft         *int       `json:"days_left"`
}

// NewDocuments maps document views.
func NewDocuments(in []service.DocumentView) []Document {
	out := make([]Document, len(in))
	for i, d := range in {
		out[i] = NewDocument(d)
	}
	return out
}

// NewDocument maps a document view.
func NewDocument(d service.DocumentView) Document {
	return Document{ID: d.ID, TypeID: d.TypeID, TypeName: d.TypeName, HasExpiry: d.HasExpiry, Number: d.Number,
		IssuedDate: NewDate(d.IssuedDate), ExpiredDate: NewDate(d.ExpiredDate), IssuingAuthority: d.IssuingAuthority,
		PlaceOfIssue: d.PlaceOfIssue, FileID: d.FileID, Badge: d.Badge, DaysLeft: d.DaysLeft}
}

// Experience is a sea service row.
type Experience struct {
	ID            uuid.UUID     `json:"id"`
	VesselName    string        `json:"vessel_name"`
	VesselID      *uuid.UUID    `json:"vessel_id"`
	RankID        uuid.UUID     `json:"rank_id"`
	RankName      string        `json:"rank_name"`
	VesselTypeID  *uuid.UUID    `json:"vessel_type_id"`
	VesselType    string        `json:"vessel_type"`
	GRTHP         string        `json:"grt_hp"`
	CompanyID     *uuid.UUID    `json:"company_id"`
	CompanyName   string        `json:"company_name"`
	PrincipalID   *uuid.UUID    `json:"principal_id"`
	PrincipalName string        `json:"principal_name"`
	SignOnDate    Date          `json:"sign_on_date"`
	SignOffDate   *Date         `json:"sign_off_date"`
	Status        string        `json:"status"`
	Remarks       string        `json:"remarks"`
	Source        string        `json:"source"`
	ContractID    *uuid.UUID    `json:"contract_id"`
	Days          int           `json:"days,omitempty"`
	Duration      *service.Span `json:"duration,omitempty"`
}

// NewExperience maps a sea service row.
func NewExperience(x model.Experience) Experience {
	return Experience{ID: x.ID, VesselName: x.VesselName, VesselID: x.VesselID, RankID: x.RankID, RankName: x.RankName,
		VesselTypeID: x.VesselTypeID, VesselType: x.VesselType, GRTHP: x.GRTHP, CompanyID: x.CompanyID, CompanyName: x.CompanyName,
		PrincipalID: x.PrincipalID, PrincipalName: x.PrincipalName, SignOnDate: Date(x.SignOnDate), SignOffDate: NewDate(x.SignOffDate),
		Status: x.Status, Remarks: x.Remarks, Source: x.Source, ContractID: x.ContractID}
}

// SeaService is the Experiences tab with totals.
type SeaService struct {
	Rows   []Experience             `json:"rows"`
	ByRank []service.RankSeaService `json:"by_rank"`
	Total  service.Span             `json:"total"`
}

// NewSeaService maps the sea service summary.
func NewSeaService(s service.CrewSeaService) SeaService {
	rows := make([]Experience, len(s.Rows))
	for i, r := range s.Rows {
		e := NewExperience(r.Experience)
		e.Days = r.Days
		d := r.Duration
		e.Duration = &d
		rows[i] = e
	}
	return SeaService{Rows: rows, ByRank: s.ByRank, Total: s.Total}
}

// CrewRow is one datagrid row.
type CrewRow struct {
	ID             string       `json:"id"`
	FullName       string       `json:"full_name"`
	DateOfBirth    *Date        `json:"date_of_birth"`
	Age            *int         `json:"age"`
	Nationality    string       `json:"nationality"`
	MobilePhone    string       `json:"mobile_phone"`
	Email          string       `json:"email"`
	Status         Status       `json:"status"`
	RankID         *uuid.UUID   `json:"rank_id"`
	RankName       string       `json:"rank_name"`
	Department     string       `json:"department"`
	ContractID     string       `json:"contract_id"`
	VesselID       *uuid.UUID   `json:"vessel_id"`
	VesselName     string       `json:"vessel_name"`
	VesselType     string       `json:"vessel_type"`
	PrincipalID    *uuid.UUID   `json:"principal_id"`
	PrincipalName  string       `json:"principal_name"`
	EmployerName   string       `json:"employer_name"`
	PassportNumber string       `json:"passport_number"`
	SeamanBook     string       `json:"seaman_book_number"`
	Compliant      bool         `json:"compliant"`
	NearestExpiry  *Date        `json:"nearest_expiry"`
	ExpiryBadge    string       `json:"expiry_badge"`
	SeaService     service.Span `json:"sea_service"`
	GroupKey       string       `json:"group_key,omitempty"`
}

// CrewGroup is one datagrid group.
type CrewGroup struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// CrewList is the datagrid response body.
type CrewList struct {
	Items  []CrewRow   `json:"items"`
	Groups []CrewGroup `json:"groups,omitempty"`
}

// NewCrewRow maps a datagrid row.
func NewCrewRow(c model.CrewRow, today time.Time, t service.Thresholds) CrewRow {
	return CrewRow{
		ID: c.PublicID, FullName: c.FullName, DateOfBirth: NewDate(c.DateOfBirth), Age: Age(c.DateOfBirth, today),
		Nationality: c.Nationality, MobilePhone: c.MobilePhone, Email: c.Email,
		Status: Status{Code: c.StatusCode, Name: c.StatusName, Color: c.StatusColor}, RankID: c.RankID, RankName: c.RankName,
		Department: c.Department, ContractID: c.ContractID, VesselID: c.VesselID, VesselName: c.VesselName, VesselType: c.VesselType,
		PrincipalID: c.PrincipalID, PrincipalName: c.PrincipalName, EmployerName: c.EmployerName, PassportNumber: c.PassportNumber,
		SeamanBook: c.SeamanBook, Compliant: c.Compliant, NearestExpiry: NewDate(c.NearestExpiry),
		ExpiryBadge: service.ExpiryBadge(c.NearestExpiry, today, t), SeaService: service.NewSpan(c.SeaServiceDays), GroupKey: c.GroupKey,
	}
}

// NewCrewList maps a datagrid page.
func NewCrewList(l service.CrewList) CrewList {
	items := make([]CrewRow, len(l.Items))
	for i, c := range l.Items {
		items[i] = NewCrewRow(c, l.Today, l.Thresholds)
	}
	var groups []CrewGroup
	for _, g := range l.Groups {
		groups = append(groups, CrewGroup{Key: g.Key, Name: g.Name, Count: g.Count})
	}
	return CrewList{Items: items, Groups: groups}
}
