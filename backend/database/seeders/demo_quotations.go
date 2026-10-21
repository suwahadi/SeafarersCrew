package seeders

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

type quotationLineSeed struct {
	rank        string
	crewCount   int
	days        int
	dailyRate   string
	description string
	lumpSum     string
}

type quotationSeed struct {
	client       string
	vessel       string
	currency     string
	daysAgo      int
	validityDays int
	taxPercent   string
	discount     string
	terms        string
	notes        string
	finalStatus  string
	lines        []quotationLineSeed
}

const quotationTerms = "Invoices are issued monthly in arrears and are payable within 30 days of the invoice date. Rates exclude port charges and visa costs unless stated."

var demoQuotations = []quotationSeed{
	{
		client: "SNO", vessel: "Nusantara Titan", currency: currencyUSD, daysAgo: 24, validityDays: 30, taxPercent: "11", discount: "1500",
		terms: quotationTerms, notes: "Crew for a 90-day offshore support charter.", finalStatus: model.QuotationAccepted,
		lines: []quotationLineSeed{
			{rank: "MST", crewCount: 1, days: 90, dailyRate: "280"},
			{rank: "CE", crewCount: 1, days: 90, dailyRate: "260"},
			{rank: "CO", crewCount: 1, days: 90, dailyRate: "200"},
			{rank: "AB", crewCount: 4, days: 90, dailyRate: "55"},
			{description: "Mobilisation and demobilisation (travel and visa)", lumpSum: "4800"},
		},
	},
	{
		client: "MMS", vessel: "Orion Star", currency: currencyUSD, daysAgo: 9, validityDays: 30, taxPercent: "0", discount: "0",
		terms: quotationTerms, notes: "Replacement officers and engine ratings for a tanker.", finalStatus: model.QuotationSent,
		lines: []quotationLineSeed{
			{rank: "MST", crewCount: 1, days: 60, dailyRate: "310"},
			{rank: "CE", crewCount: 1, days: 60, dailyRate: "290"},
			{rank: "2E", crewCount: 1, days: 60, dailyRate: "210"},
			{rank: "OLR", crewCount: 2, days: 60, dailyRate: "48"},
		},
	},
	{
		client: "KTL", vessel: "Madura Tug 3", currency: currencyIDR, daysAgo: 2, validityDays: 30, taxPercent: "11", discount: "2000000",
		terms: quotationTerms, notes: "Crew change for the harbour towage contract.", finalStatus: model.QuotationDraft,
		lines: []quotationLineSeed{
			{rank: "MST", crewCount: 1, days: 30, dailyRate: "2400000"},
			{rank: "CE", crewCount: 1, days: 30, dailyRate: "2200000"},
			{rank: "AB", crewCount: 3, days: 30, dailyRate: "650000"},
			{description: "Document and medical preparation", lumpSum: "7500000"},
		},
	},
}

func seedDemoQuotations(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	companies, err := d.loadCompanies(ctx)
	if err != nil {
		return err
	}
	vessels, err := d.loadVessels(ctx)
	if err != nil {
		return err
	}
	vesselByName := make(map[string]demoVessel, len(vessels))
	for _, v := range vessels {
		vesselByName[v.Name] = v
	}
	for i, seed := range demoQuotations {
		q, err := d.quotationModel(seed, companies, vesselByName)
		if err != nil {
			return fmt.Errorf("quotation %d: %w", i+1, err)
		}
		created, err := d.Quotations.Create(ctx, q)
		if err != nil {
			return fmt.Errorf("create quotation %d: %w", i+1, err)
		}
		for _, status := range quotationPath(seed.finalStatus) {
			if _, err := d.Quotations.ChangeStatus(ctx, created.PublicID, status); err != nil {
				return fmt.Errorf("move quotation %d to %s: %w", i+1, status, err)
			}
		}
	}
	return nil
}

func quotationPath(final string) []string {
	switch final {
	case model.QuotationSent:
		return []string{model.QuotationSent}
	case model.QuotationAccepted:
		return []string{model.QuotationSent, model.QuotationAccepted}
	}
	return nil
}

func (d *demoRun) quotationModel(seed quotationSeed, companies map[string]demoCompany, vessels map[string]demoVessel) (model.Quotation, error) {
	vessel := vessels[seed.vessel].ID
	date := d.today.AddDate(0, 0, -seed.daysAgo)
	tax, err := decimal.NewFromString(seed.taxPercent)
	if err != nil {
		return model.Quotation{}, fmt.Errorf("tax percent: %w", err)
	}
	discount, err := decimal.NewFromString(seed.discount)
	if err != nil {
		return model.Quotation{}, fmt.Errorf("discount: %w", err)
	}
	q := model.Quotation{
		ClientCompanyID: companies[seed.client].ID, VesselID: &vessel, CurrencyCode: seed.currency,
		QuotationDate: date, ValidUntil: date.AddDate(0, 0, seed.validityDays), TaxPercent: tax, DiscountAmount: discount,
		Terms: seed.terms, Notes: seed.notes,
	}
	for i, l := range seed.lines {
		line := model.QuotationLine{SortOrder: (i + 1) * 10}
		if l.rank != "" {
			rate, err := decimal.NewFromString(l.dailyRate)
			if err != nil {
				return model.Quotation{}, fmt.Errorf("daily rate: %w", err)
			}
			rankID, count, days := d.refs.ranks[l.rank].ID, l.crewCount, l.days
			line.LineType, line.RankID, line.CrewCount, line.DailyRate, line.Days = model.LineDailyRate, &rankID, &count, &rate, &days
		} else {
			amount, err := decimal.NewFromString(l.lumpSum)
			if err != nil {
				return model.Quotation{}, fmt.Errorf("lump sum: %w", err)
			}
			line.LineType, line.Description, line.Amount = model.LineLumpSum, l.description, amount
		}
		q.Lines = append(q.Lines, line)
	}
	return q, nil
}
