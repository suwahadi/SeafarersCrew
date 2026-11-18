package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

// PayrollExportJob is the job kind that writes a period's lines to a workbook.
const PayrollExportJob = "payroll_export"

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// PayrollExportStore is the persistence the payroll exporter needs.
type PayrollExportStore interface {
	GetPeriod(ctx context.Context, publicID string) (model.PayrollPeriod, error)
	AllLines(ctx context.Context, periodID uuid.UUID, f model.PayrollLineFilter) ([]model.PayrollLine, error)
}

// PayrollExportPayload is the job payload of a payroll export.
type PayrollExportPayload struct {
	Period string                  `json:"period"`
	Filter model.PayrollLineFilter `json:"filter"`
}

// PayrollExportResult is the job result of a payroll export.
type PayrollExportResult struct {
	FileID   uuid.UUID `json:"file_id"`
	Filename string    `json:"filename"`
	Rows     int       `json:"rows"`
}

// PayrollExportService writes payroll periods to bank-transfer workbooks.
type PayrollExportService struct {
	store    PayrollExportStore
	jobs     Enqueuer
	files    GeneratedFileStore
	logos    FileReader
	settings SettingsReader
	now      func() time.Time
}

// NewPayrollExportService builds a PayrollExportService.
func NewPayrollExportService(store PayrollExportStore, jobs Enqueuer, files GeneratedFileStore, logos FileReader, settings SettingsReader) *PayrollExportService {
	return &PayrollExportService{store: store, jobs: jobs, files: files, logos: logos, settings: settings, now: time.Now}
}

func (s *PayrollExportService) period(ctx context.Context, publicID string) (model.PayrollPeriod, error) {
	p, err := s.store.GetPeriod(ctx, publicID)
	if errors.Is(err, repository.ErrNotFound) {
		return p, apperr.NotFound("Payroll period")
	}
	return p, err
}

// Enqueue queues an export of the period's lines matching f.
func (s *PayrollExportService) Enqueue(ctx context.Context, periodPublicID string, f model.PayrollLineFilter) (model.Job, error) {
	if _, err := s.period(ctx, periodPublicID); err != nil {
		return model.Job{}, err
	}
	return s.jobs.Enqueue(ctx, PayrollExportJob, PayrollExportPayload{Period: periodPublicID, Filter: f})
}

// Handle runs an export job and stores the workbook.
func (s *PayrollExportService) Handle(ctx context.Context, j model.Job) (any, error) {
	var p PayrollExportPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode payroll export payload: %w", err)
	}
	period, err := s.period(ctx, p.Period)
	if err != nil {
		return nil, err
	}
	lines, err := s.store.AllLines(ctx, period.ID, p.Filter)
	if err != nil {
		return nil, err
	}
	head, err := s.head(ctx, period)
	if err != nil {
		return nil, err
	}
	data, err := WritePayrollWorkbook(head, lines)
	if err != nil {
		return nil, err
	}
	name := "payroll-" + period.PublicID + "-" + s.now().Format("20060102") + ".xlsx"
	f, err := s.files.StoreGenerated(ctx, "exports", name, xlsxMime, "xlsx", data, nil)
	if err != nil {
		return nil, err
	}
	return PayrollExportResult{FileID: f.ID, Filename: name, Rows: len(lines)}, nil
}

func (s *PayrollExportService) head(ctx context.Context, p model.PayrollPeriod) (PayrollSheetHead, error) {
	company, err := s.settings.Values(ctx, "company")
	if err != nil {
		return PayrollSheetHead{}, err
	}
	str := func(k string) string {
		v, _ := company[k].(string)
		return v
	}
	head := PayrollSheetHead{
		Company: str("legal_name"), Address: str("address"), Phone: str("phone"), Email: str("email"),
		Title: p.Title, Range: FormatPayrollRange(p.PeriodStart, p.PeriodEnd),
	}
	if id, err := uuid.Parse(str("logo_light_file_id")); err == nil && s.logos != nil {
		if data, _, err := s.logos.Content(ctx, id); err == nil {
			head.Logo = data
		}
	}
	return head, nil
}
