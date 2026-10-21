package seeders

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/shopspring/decimal"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/service"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

const (
	currencyIDR = "IDR"
	currencyUSD = "USD"
	idrPerUSD   = 15000
)

var usdSalaryByRank = map[string]int64{
	"MST": 6500, "CO": 4800, "2O": 3200, "3O": 2800, "CE": 6200, "2E": 4500, "3E": 3300, "4E": 2700,
	"BSN": 1900, "AB": 1500, "OS": 1100, "OLR": 1350, "WPR": 1050, "EC": 900, "CK": 1400, "STW": 1100,
}

var portsByVessel = []string{"Balikpapan", "Balikpapan", "Singapore", "Batam", "Batam", "Surabaya", "Surabaya", "Singapore", "Singapore", "Surabaya"}

func contractSignOn(today time.Time, p contractPlan) time.Time {
	return datetime.AddMonths(today.AddDate(0, 0, p.remaining), -p.months)
}

func contractCurrency(rank demoRank, vessel demoVessel) string {
	if rank.IsOfficer || vessel.VesselType == "Tanker" || vessel.VesselType == "Bulk Carrier" {
		return currencyUSD
	}
	return currencyIDR
}

func monthlySalary(rankCode, currency string) decimal.Decimal {
	usd := decimal.NewFromInt(usdSalaryByRank[rankCode])
	if currency == currencyIDR {
		return usd.Mul(decimal.NewFromInt(idrPerUSD))
	}
	return usd
}

type allowanceDraft struct {
	name     string
	usd, idr int64
}

func allowanceDrafts(index int) []allowanceDraft {
	out := []allowanceDraft{{"Offshore Allowance", 8, 100_000}}
	if index%2 == 1 {
		out = append(out, allowanceDraft{"Overtime", 250, 3_500_000})
	}
	if index%3 == 0 {
		out = append(out, allowanceDraft{"Leave Pay", 150, 2_000_000})
	}
	if index%4 == 2 {
		out = append(out, allowanceDraft{"Standby Pay", 5, 70_000})
	}
	if index%5 == 0 {
		out = append(out, allowanceDraft{"Bonus", 200, 3_000_000})
	}
	return out
}

func (d *demoRun) contractFor(c demoCrew, vessel demoVessel, companies map[string]demoCompany, months int, signOn time.Time, index int) model.Contract {
	rank := d.refs.ranks[c.spec.rank]
	currency := contractCurrency(rank, vessel)
	hours, probation := 48, 1
	leave := decimal.NewFromInt(5)
	var allowances []model.ContractAllowance
	for i, a := range allowanceDrafts(index) {
		value := decimal.NewFromInt(a.usd)
		if currency == currencyIDR {
			value = decimal.NewFromInt(a.idr)
		}
		allowances = append(allowances, model.ContractAllowance{AllowanceTypeID: d.refs.allowanceTypes[a.name].ID, Value: value, SortOrder: (i + 1) * 10})
	}
	port := portsByVessel[vesselIndex(vessel.Name)]
	return model.Contract{
		CrewPublicID: c.publicID, VesselID: vessel.ID, RankID: rank.ID,
		PrincipalID: companies[vessel.Principal].ID, EmployerCompanyID: companies[vessel.OwnerCode].ID,
		SignOnDate: signOn, DurationMonths: months, PointOfHire: c.spec.person.City, SignOnPort: port, SignOffPort: port,
		CurrencyCode: currency, BasicSalary: monthlySalary(c.spec.rank, currency), WorkingHoursWeek: &hours,
		LeaveDaysMonth: &leave, ProbationMonths: &probation, Allowances: allowances,
	}
}

func vesselIndex(name string) int {
	for i, v := range demoVessels {
		if v.name == name {
			return i
		}
	}
	return 0
}

func seedDemoContracts(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	crew, err := d.loadCrew(ctx)
	if err != nil {
		return err
	}
	companies, err := d.loadCompanies(ctx)
	if err != nil {
		return err
	}
	vessels, err := d.loadVessels(ctx)
	if err != nil {
		return err
	}
	signedOff := 0
	for i, c := range crew {
		switch c.role.role {
		case roleOnBoard:
			plan := contractPlans[c.role.contract]
			contract := d.contractFor(c, vessels[plan.vessel], companies, plan.months, contractSignOn(d.today, plan), c.role.contract)
			if err := d.createActiveContract(ctx, contract); err != nil {
				return fmt.Errorf("contract of crew %d: %w", i+1, err)
			}
		case roleSignedOff:
			if err := d.createCompletedContract(ctx, c, vessels, companies, signedOff); err != nil {
				return fmt.Errorf("completed contract of crew %d: %w", i+1, err)
			}
			signedOff++
		}
	}
	return nil
}

func (d *demoRun) createActiveContract(ctx context.Context, contract model.Contract) error {
	created, err := d.Contracts.Create(ctx, contract)
	if err != nil {
		return err
	}
	_, err = d.Contracts.Activate(ctx, created.PublicID)
	return err
}

func (d *demoRun) createCompletedContract(ctx context.Context, c demoCrew, vessels []demoVessel, companies map[string]demoCompany, order int) error {
	months := completedMonthsFor[c.spec.rank]
	signOff := d.today.AddDate(0, 0, -(35 + 10*order))
	signOn := datetime.AddMonths(signOff, -months)
	contract := d.contractFor(c, vessels[(order*3+1)%len(vessels)], companies, months, signOn, order+20)
	created, err := d.Contracts.Create(ctx, contract)
	if err != nil {
		return err
	}
	if _, err := d.Contracts.Activate(ctx, created.PublicID); err != nil {
		return err
	}
	_, err = d.Contracts.Complete(ctx, created.PublicID, signOff)
	return err
}

func seedDemoReliefs(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	crew, err := d.loadCrew(ctx)
	if err != nil {
		return err
	}
	onBoard := map[int]demoCrew{}
	relievers := map[int]demoCrew{}
	for _, c := range crew {
		switch c.role.role {
		case roleOnBoard:
			onBoard[c.role.contract] = c
		case roleReliever:
			relievers[c.role.contract] = c
		}
	}
	for i, plan := range contractPlans {
		if plan.reliefStatus == reliefNone {
			continue
		}
		assignment, err := d.Contracts.CurrentAssignment(ctx, onBoard[i].publicID)
		if err != nil {
			return fmt.Errorf("current assignment for relief %d: %w", i+1, err)
		}
		if assignment == nil {
			return fmt.Errorf("crew for relief %d is not on board", i+1)
		}
		_, err = d.Rotation.Create(ctx, service.ReliefInput{
			ContractPublicID:  assignment.Contract.PublicID,
			RelieverPublicID:  relievers[i].publicID,
			PlannedReliefDate: d.today.AddDate(0, 0, plan.reliefDays),
			JoiningPort:       assignment.Contract.SignOnPort,
			Status:            plan.reliefStatus,
			Notes:             "Reliever briefed and documents under review.",
		})
		if err != nil {
			return fmt.Errorf("create relief %d: %w", i+1, err)
		}
	}
	return nil
}

func seedDemoStatuses(ctx context.Context, d *demoRun, _ *rand.Rand) error {
	crew, err := d.loadCrew(ctx)
	if err != nil {
		return err
	}
	for i, c := range crew {
		switch c.role.role {
		case roleMedicalLeave:
			if _, err := d.Crews.ChangeStatus(ctx, c.publicID, service.ActionMedicalLeave, "Recovering after a minor injury; fit-for-duty review pending."); err != nil {
				return fmt.Errorf("medical leave of crew %d: %w", i+1, err)
			}
		case roleBlacklisted:
			if _, err := d.Crews.ChangeStatus(ctx, c.publicID, service.ActionBlacklist, "Left the vessel without notice during the previous contract."); err != nil {
				return fmt.Errorf("blacklist of crew %d: %w", i+1, err)
			}
		}
	}
	return nil
}
