package model

import (
	"time"

	"github.com/google/uuid"
)

// Crew is a seafarer record.
type Crew struct {
	ID             uuid.UUID
	PublicID       string
	FullName       string
	PlaceOfBirth   string
	DateOfBirth    *time.Time
	Gender         *string
	Nationality    string
	Religion       string
	MaritalStatus  *string
	HeightCM       *int16
	WeightKG       *int16
	ShoeSize       string
	CoverallSize   string
	AppliedRankID  *uuid.UUID
	CrewStatusID   uuid.UUID
	StatusReason   string
	PhotoFileID    *uuid.UUID
	CVFileID       *uuid.UUID
	Address        string
	City           string
	Province       string
	Country        string
	PostalCode     string
	HomePhone      string
	MobilePhone    string
	Email          string
	NearestAirport string
	CreatedAt      time.Time
	UpdatedAt      time.Time

	AppliedRankName string
	StatusCode      string
	StatusName      string
	StatusColor     string
}

// BankAccount is one crew bank account.
type BankAccount struct {
	ID            uuid.UUID
	CrewID        uuid.UUID
	BankName      string
	AccountName   string
	AccountNumber string
	CurrencyCode  string
	CoverBank     string
	Remarks       string
	IsPrimary     bool
}

// NextOfKin is one emergency contact.
type NextOfKin struct {
	ID           uuid.UUID
	CrewID       uuid.UUID
	Name         string
	Relationship string
	Phone        string
	Address      string
	IsPrimary    bool
}

// CrewDocument is a travel document or an STCW certificate row.
type CrewDocument struct {
	ID               uuid.UUID
	CrewID           uuid.UUID
	TypeID           uuid.UUID
	TypeName         string
	HasExpiry        bool
	Number           string
	IssuedDate       *time.Time
	ExpiredDate      *time.Time
	IssuingAuthority string
	PlaceOfIssue     string
	FileID           *uuid.UUID
}

// Experience is one sea service row.
type Experience struct {
	ID            uuid.UUID
	CrewID        uuid.UUID
	VesselName    string
	VesselID      *uuid.UUID
	RankID        uuid.UUID
	RankName      string
	VesselTypeID  *uuid.UUID
	VesselType    string
	GRTHP         string
	CompanyID     *uuid.UUID
	CompanyName   string
	PrincipalID   *uuid.UUID
	PrincipalName string
	SignOnDate    time.Time
	SignOffDate   *time.Time
	Status        string
	Remarks       string
	Source        string
	ContractID    *uuid.UUID
}
