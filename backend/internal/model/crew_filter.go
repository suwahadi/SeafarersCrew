package model

import (
	"time"

	"github.com/google/uuid"
)

// CrewFilter holds the datagrid search, filters, sort and grouping (PRD 6.9).
type CrewFilter struct {
	Q               string
	StatusCodes     []string
	RankIDs         []uuid.UUID
	Departments     []string
	VesselTypeIDs   []uuid.UUID
	VesselIDs       []uuid.UUID
	PrincipalIDs    []uuid.UUID
	AgeMin          *int
	AgeMax          *int
	Nationalities   []string
	Compliance      string
	DocValidity     string
	DocDays         int
	SeaRankID       *uuid.UUID
	SeaMinDays      int
	GroupBy         string
	Sort            string
	Desc            bool
	Today           time.Time
	ExcludeStatuses []string
}

// CrewRow is one datagrid row.
type CrewRow struct {
	PublicID       string
	FullName       string
	DateOfBirth    *time.Time
	Nationality    string
	MobilePhone    string
	Email          string
	StatusCode     string
	StatusName     string
	StatusColor    string
	RankID         *uuid.UUID
	RankName       string
	Department     string
	ContractID     string
	VesselID       *uuid.UUID
	VesselName     string
	VesselType     string
	PrincipalID    *uuid.UUID
	PrincipalName  string
	EmployerName   string
	PassportNumber string
	SeamanBook     string
	Compliant      bool
	NearestExpiry  *time.Time
	SeaServiceDays int
	GroupKey       string
	CreatedAt      time.Time
}

// CrewGroup is one group of the grouped datagrid.
type CrewGroup struct {
	Key   string
	Name  string
	Count int
}
