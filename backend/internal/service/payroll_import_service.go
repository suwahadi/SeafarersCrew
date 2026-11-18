package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// PayrollImportJob is the job kind that replaces a period's lines with a validated workbook.
const PayrollImportJob = "payroll_import"

// Payroll import row states.
const (
	PayrollRowValid      = "valid"
	PayrollRowNewAccount = "new_account"
	PayrollRowError      = "error"
)

const (
	warnNoContract  = "No contract covers this period; rank and vessel are left empty."
	warnLeadingZero = "The account number was stored as a number in the file; check that no leading zero is missing."
)

// PayrollPreviewRow is one checked workbook row.
type PayrollPreviewRow struct {
	Line          int               `json:"line"`
	Status        string            `json:"status"`
	EmployeeNo    string            `json:"employee_no"`
	CrewName      string            `json:"crew_name"`
	BankName      string            `json:"bank_name"`
	AccountName   string            `json:"account_name"`
	AccountNumber string            `json:"account_number"`
	AmountUSD     *string           `json:"amount_usd"`
	AmountIDR     string            `json:"amount_idr"`
	TransferDate  string            `json:"transfer_date"`
	Remark        string            `json:"remark"`
	Warnings      []string          `json:"warnings,omitempty"`
	Errors        map[string]string `json:"errors,omitempty"`
}

// PayrollPreview summarises a checked workbook; Token is set only when no row has an error.
type PayrollPreview struct {
	Token        string              `json:"token"`
	Total        int                 `json:"total"`
	Valid        int                 `json:"valid"`
	NewAccounts  int                 `json:"new_accounts"`
	Errors       int                 `json:"errors"`
	WithWarnings int                 `json:"with_warnings"`
	WillReplace  int                 `json:"will_replace"`
	Rows         []PayrollPreviewRow `json:"rows"`
}

// PayrollImportReport is the job result of a committed import.
type PayrollImportReport struct {
	Lines         int `json:"lines"`
	AccountsAdded int `json:"accounts_added"`
}

// PayrollImportStore is the persistence the payroll importer needs.
type PayrollImportStore interface {
	GetPeriod(ctx context.Context, publicID string) (model.PayrollPeriod, error)
	CrewsByEmployeeNo(ctx context.Context, normalized []string) (map[string]model.PayrollCrew, error)
	BankAccountsByCrew(ctx context.Context, crewIDs []uuid.UUID) (map[uuid.UUID][]model.BankAccount, error)
	CoveringContracts(ctx context.Context, crewIDs []uuid.UUID, start, end time.Time) (map[uuid.UUID]model.PayrollPlacement, error)
	ReplaceLines(ctx context.Context, periodID uuid.UUID, accounts []repository.PayrollNewAccount, lines []repository.PayrollStagedLine) (int, error)
}

// PayrollImportService checks payroll workbooks and replaces a period's lines with them.
type PayrollImportService struct {
	store PayrollImportStore
	cache Cache
	jobs  Enqueuer
}

// NewPayrollImportService builds a PayrollImportService.
func NewPayrollImportService(store PayrollImportStore, cache Cache, jobs Enqueuer) *PayrollImportService {
	return &PayrollImportService{store: store, cache: cache, jobs: jobs}
}

type stagedPayrollImport struct {
	PeriodID       uuid.UUID                      `json:"period_id"`
	PeriodPublicID string                         `json:"period_public_id"`
	Accounts       []repository.PayrollNewAccount `json:"accounts"`
	Lines          []repository.PayrollStagedLine `json:"lines"`
}

func payrollImportKey(tok string) string { return "payroll_import:" + tok }

var errPayrollImportExpired = apperr.New(http.StatusGone, "import_expired", "The validated import has expired; upload the file again.")

func (s *PayrollImportService) period(ctx context.Context, publicID string) (model.PayrollPeriod, error) {
	p, err := s.store.GetPeriod(ctx, publicID)
	if errors.Is(err, repository.ErrNotFound) {
		return p, apperr.NotFound("Payroll period")
	}
	return p, err
}

type payrollLookups struct {
	crews      map[string]model.PayrollCrew
	accounts   map[uuid.UUID][]model.BankAccount
	placements map[uuid.UUID]model.PayrollPlacement
}

func (s *PayrollImportService) lookups(ctx context.Context, p model.PayrollPeriod, rows []WorkbookRow) (payrollLookups, error) {
	seen := map[string]bool{}
	var numbers []string
	for _, r := range rows {
		if n := model.NormalizeEmployeeNo(r.EmployeeNo); n != "" && !seen[n] {
			seen[n] = true
			numbers = append(numbers, n)
		}
	}
	l := payrollLookups{}
	var err error
	if l.crews, err = s.store.CrewsByEmployeeNo(ctx, numbers); err != nil {
		return l, err
	}
	ids := make([]uuid.UUID, 0, len(l.crews))
	for _, c := range l.crews {
		ids = append(ids, c.ID)
	}
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		l.accounts, err = s.store.BankAccountsByCrew(gctx, ids)
		return err
	})
	g.Go(func() error {
		var err error
		l.placements, err = s.store.CoveringContracts(gctx, ids, p.PeriodStart, p.PeriodEnd)
		return err
	})
	return l, g.Wait()
}

// Validate checks every workbook row against the crew records and stages the import when nothing is in error.
func (s *PayrollImportService) Validate(ctx context.Context, periodPublicID string, data []byte) (PayrollPreview, error) {
	p, err := s.period(ctx, periodPublicID)
	if err != nil {
		return PayrollPreview{}, err
	}
	if err := filecheck.CheckXLSX(data); err != nil {
		return PayrollPreview{}, err
	}
	rows, err := ParsePayrollWorkbook(data)
	if err != nil {
		return PayrollPreview{}, err
	}
	l, err := s.lookups(ctx, p, rows)
	if err != nil {
		return PayrollPreview{}, err
	}
	b := payrollBuilder{period: p, lookups: l, newKeys: map[string]bool{}, newPrimary: map[uuid.UUID]bool{}}
	preview := PayrollPreview{Total: len(rows), WillReplace: p.LineCount, Rows: make([]PayrollPreviewRow, 0, len(rows))}
	staged := stagedPayrollImport{PeriodID: p.ID, PeriodPublicID: p.PublicID}
	for _, r := range rows {
		row, line := b.check(r)
		switch row.Status {
		case PayrollRowError:
			preview.Errors++
		case PayrollRowNewAccount:
			preview.NewAccounts++
		default:
			preview.Valid++
		}
		if len(row.Warnings) > 0 {
			preview.WithWarnings++
		}
		preview.Rows = append(preview.Rows, row)
		staged.Lines = append(staged.Lines, line)
	}
	staged.Accounts = b.accounts
	if preview.Errors > 0 {
		return preview, nil
	}
	tok, err := token.URLSafe(18)
	if err != nil {
		return PayrollPreview{}, err
	}
	payload, err := json.Marshal(staged)
	if err != nil {
		return PayrollPreview{}, fmt.Errorf("encode staged payroll import: %w", err)
	}
	if err := s.cache.Set(ctx, payrollImportKey(tok), payload, importTTL); err != nil {
		return PayrollPreview{}, err
	}
	preview.Token = tok
	return preview, nil
}

type payrollBuilder struct {
	period     model.PayrollPeriod
	lookups    payrollLookups
	accounts   []repository.PayrollNewAccount
	newKeys    map[string]bool
	newPrimary map[uuid.UUID]bool
}

func compactAccount(s string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(s)
}

func matchAccount(accounts []model.BankAccount, number string, numeric bool) *model.BankAccount {
	want := compactAccount(number)
	for i, a := range accounts {
		have := compactAccount(a.AccountNumber)
		if have == want || (numeric && strings.TrimLeft(have, "0") == strings.TrimLeft(want, "0") && strings.TrimLeft(want, "0") != "") {
			return &accounts[i]
		}
	}
	return nil
}

func (b *payrollBuilder) check(r WorkbookRow) (PayrollPreviewRow, repository.PayrollStagedLine) {
	row := PayrollPreviewRow{
		Line: r.Line, EmployeeNo: r.EmployeeNo, BankName: r.BankName, AccountName: r.AccountName, AccountNumber: r.AccountNumber,
		Remark: r.Remark,
	}
	errs := map[string]string{}
	for k, v := range r.Errors {
		errs[k] = v
	}
	line := model.PayrollLine{PeriodID: b.period.ID, Remark: r.Remark}
	b.amounts(r, &row, &line, errs)
	b.date(r, &row, &line, errs)
	if r.AccountNumber == "" {
		errs["account_number"] = "The account number is required."
	}
	staged := repository.PayrollStagedLine{}
	crew, found := b.lookups.crews[model.NormalizeEmployeeNo(r.EmployeeNo)]
	switch {
	case r.EmployeeNo == "":
		errs["employee_no"] = "The Employee ID is required."
	case !found:
		errs["employee_no"] = "No crew member has this Employee ID."
	default:
		row.CrewName = crew.FullName
		line.CrewID, line.CrewName, line.EmployeeNo = crew.ID, crew.FullName, crew.EmployeeNo
		if placement, ok := b.lookups.placements[crew.ID]; ok {
			line.RankCode, line.VesselName = placement.RankCode, placement.VesselName
		} else {
			row.Warnings = append(row.Warnings, warnNoContract)
		}
		if r.AccountNumber != "" {
			staged.NewAccountKey = b.account(crew, r, &row, &line, errs)
		}
	}
	staged.Line = line
	switch {
	case len(errs) > 0:
		row.Status, row.Errors = PayrollRowError, errs
	case staged.NewAccountKey != "":
		row.Status = PayrollRowNewAccount
	default:
		row.Status = PayrollRowValid
	}
	return row, staged
}

func (b *payrollBuilder) amounts(r WorkbookRow, row *PayrollPreviewRow, line *model.PayrollLine, errs map[string]string) {
	if r.USD != nil {
		usd := r.USD.StringFixed(2)
		row.AmountUSD, line.AmountUSD = &usd, r.USD
		if r.USD.IsNegative() {
			errs["usd"] = "USD cannot be negative."
		}
	}
	switch {
	case r.IDR != nil:
		row.AmountIDR, line.AmountIDR = r.IDR.StringFixed(2), *r.IDR
		if r.IDR.IsNegative() {
			errs["idr"] = "IDR cannot be negative."
		}
	case errs["idr"] == "":
		errs["idr"] = "IDR is required."
	}
}

func (b *payrollBuilder) date(r WorkbookRow, row *PayrollPreviewRow, line *model.PayrollLine, errs map[string]string) {
	day := r.Date
	if day == nil && errs["date"] == "" {
		day = b.period.TransferDate
		if day == nil {
			errs["date"] = "A date is required when the period has no transfer date."
		}
	}
	if day != nil {
		line.TransferDate = *day
		row.TransferDate = day.Format(time.DateOnly)
	}
}

func (b *payrollBuilder) account(crew model.PayrollCrew, r WorkbookRow, row *PayrollPreviewRow, line *model.PayrollLine, errs map[string]string) string {
	if match := matchAccount(b.lookups.accounts[crew.ID], r.AccountNumber, r.AccountNumeric); match != nil {
		line.BankAccountID = &match.ID
		line.BankName, line.AccountName, line.AccountNumber = match.BankName, match.AccountName, match.AccountNumber
		row.BankName, row.AccountName, row.AccountNumber = match.BankName, match.AccountName, match.AccountNumber
		return ""
	}
	if r.BankName == "" {
		errs["bank_name"] = "Bank name is required for a new account."
	}
	if r.AccountName == "" {
		errs["account_name"] = "Account name is required for a new account."
	}
	if r.AccountNumeric {
		row.Warnings = append(row.Warnings, warnLeadingZero)
	}
	line.BankName, line.AccountName, line.AccountNumber = r.BankName, r.AccountName, r.AccountNumber
	key := crew.ID.String() + "|" + compactAccount(r.AccountNumber)
	if !b.newKeys[key] {
		b.newKeys[key] = true
		primary := len(b.lookups.accounts[crew.ID]) == 0 && !b.newPrimary[crew.ID]
		b.newPrimary[crew.ID] = b.newPrimary[crew.ID] || primary
		b.accounts = append(b.accounts, repository.PayrollNewAccount{Key: key, Account: model.BankAccount{
			CrewID: crew.ID, BankName: r.BankName, AccountName: r.AccountName, AccountNumber: r.AccountNumber,
			CurrencyCode: "IDR", IsPrimary: primary,
		}})
	}
	return key
}

func (s *PayrollImportService) staged(ctx context.Context, tok string) (stagedPayrollImport, bool, error) {
	var staged stagedPayrollImport
	if tok == "" {
		return staged, false, nil
	}
	raw, ok, err := s.cache.Get(ctx, payrollImportKey(tok))
	if err != nil || !ok {
		return staged, false, err
	}
	if err := json.Unmarshal(raw, &staged); err != nil {
		return staged, false, fmt.Errorf("decode staged payroll import: %w", err)
	}
	return staged, true, nil
}

// Commit queues the replacement of the period's lines with the import staged under tok.
func (s *PayrollImportService) Commit(ctx context.Context, periodPublicID, tok string) (model.Job, error) {
	staged, ok, err := s.staged(ctx, tok)
	if err != nil {
		return model.Job{}, err
	}
	if !ok || staged.PeriodPublicID != periodPublicID {
		return model.Job{}, errPayrollImportExpired
	}
	if _, err := s.period(ctx, periodPublicID); err != nil {
		return model.Job{}, err
	}
	return s.jobs.Enqueue(ctx, PayrollImportJob, map[string]string{"token": tok})
}

// Handle runs a committed import job.
func (s *PayrollImportService) Handle(ctx context.Context, j model.Job) (any, error) {
	var p struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode payroll import payload: %w", err)
	}
	staged, ok, err := s.staged(ctx, p.Token)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("staged payroll import expired before it ran")
	}
	added, err := s.store.ReplaceLines(ctx, staged.PeriodID, staged.Accounts, staged.Lines)
	if err != nil {
		return nil, err
	}
	if err := s.cache.Delete(ctx, payrollImportKey(p.Token)); err != nil {
		return nil, err
	}
	return PayrollImportReport{Lines: len(staged.Lines), AccountsAdded: added}, nil
}
