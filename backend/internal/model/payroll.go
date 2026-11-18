package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PayrollPeriod is one salary run with the totals of its transfer lines.
type PayrollPeriod struct {
	ID                     uuid.UUID
	PublicID               string
	Title                  string
	PeriodStart, PeriodEnd time.Time
	TransferDate           *time.Time
	LineCount              int
	TotalUSD, TotalIDR     decimal.Decimal
	CreatedAt, UpdatedAt   time.Time
}

// PayrollLine is one bank transfer in a period, with crew, contract and bank data copied at the time it was saved.
type PayrollLine struct {
	ID            uuid.UUID
	PeriodID      uuid.UUID
	LineNo        int
	CrewID        uuid.UUID
	CrewPublicID  string
	BankAccountID *uuid.UUID
	EmployeeNo    string
	CrewName      string
	RankCode      string
	VesselName    string
	BankName      string
	AccountName   string
	AccountNumber string
	AmountUSD     *decimal.Decimal
	AmountIDR     decimal.Decimal
	TransferDate  time.Time
	Remark        string
}

// PayrollLineFilter narrows and orders the lines of a period.
type PayrollLineFilter struct {
	Q                     string
	Banks, Vessels, Ranks []string
	HasRemark, NoUSD      bool
	Sort                  string
	Desc                  bool
}

// PayrollTotals sums the lines matching a filter.
type PayrollTotals struct {
	Count    int
	USD, IDR decimal.Decimal
}

// PayrollFacets lists the distinct filter values present in a period.
type PayrollFacets struct {
	Banks, Vessels, Ranks []string
}

// PayrollPlacement is the rank and vessel of the contract that covers a period.
type PayrollPlacement struct {
	RankCode, VesselName string
}

// PayrollCrew is the crew data a payroll line copies.
type PayrollCrew struct {
	ID                             uuid.UUID
	PublicID, FullName, EmployeeNo string
}

// PayrollLineInput is what an administrator enters for one line; the rest is copied from the crew records.
type PayrollLineInput struct {
	CrewPublicID  string
	BankAccountID uuid.UUID
	AmountUSD     *decimal.Decimal
	AmountIDR     decimal.Decimal
	TransferDate  *time.Time
	Remark        string
}
