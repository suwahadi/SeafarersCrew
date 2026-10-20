package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// CrewExportJob is the job kind for datagrid exports.
const CrewExportJob = "crew_export"

const exportBatch = 500

const utf8BOM = "\xEF\xBB\xBF"

// Enqueuer queues background jobs.
type Enqueuer interface {
	Enqueue(ctx context.Context, kind string, payload any) (model.Job, error)
}

// GeneratedFileStore saves server-generated files.
type GeneratedFileStore interface {
	StoreGenerated(ctx context.Context, entity, filename, mime, ext string, data []byte, ownerID *uuid.UUID) (model.File, error)
}

// CrewExportService exports the filtered datagrid to CSV or Excel.
type CrewExportService struct {
	crews    *CrewService
	jobs     Enqueuer
	files    GeneratedFileStore
	settings SettingsReader
}

// NewCrewExportService builds a CrewExportService.
func NewCrewExportService(crews *CrewService, jobs Enqueuer, files GeneratedFileStore, settings SettingsReader) *CrewExportService {
	return &CrewExportService{crews: crews, jobs: jobs, files: files, settings: settings}
}

// CrewExportPayload is stored with the export job.
type CrewExportPayload struct {
	Format string           `json:"format"`
	Filter model.CrewFilter `json:"filter"`
}

// CrewExportResult is the job result.
type CrewExportResult struct {
	FileID   uuid.UUID `json:"file_id"`
	Filename string    `json:"filename"`
	Rows     int       `json:"rows"`
}

// Enqueue validates the format and queues an export of exactly the filtered rows.
func (s *CrewExportService) Enqueue(ctx context.Context, format string, f model.CrewFilter) (model.Job, error) {
	if format != "csv" && format != "xlsx" {
		return model.Job{}, apperr.Validation(map[string]string{"format": "The format must be csv or xlsx."})
	}
	if !slices.Contains(f.StatusCodes, StatusBlacklisted) {
		f.ExcludeStatuses = []string{StatusBlacklisted}
	}
	f.GroupBy = ""
	return s.jobs.Enqueue(ctx, CrewExportJob, CrewExportPayload{Format: format, Filter: f})
}

// Handle runs an export job.
func (s *CrewExportService) Handle(ctx context.Context, j model.Job) (any, error) {
	var p CrewExportPayload
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode export payload: %w", err)
	}
	loc, thresholds, err := LoadDisplaySettings(ctx, s.settings)
	if err != nil {
		return nil, err
	}
	pattern := s.datePattern(ctx)
	p.Filter.Today = datetime.Today(s.crews.now(), loc)
	rows := [][]string{exportHeader}
	err = s.crews.crews.StreamCrews(ctx, p.Filter, exportBatch, func(c model.CrewRow) error {
		rows = append(rows, exportRecord(c, p.Filter.Today, thresholds, pattern))
		return nil
	})
	if err != nil {
		return nil, err
	}
	data, mime, err := encodeExport(p.Format, rows)
	if err != nil {
		return nil, err
	}
	name := "crew-export-" + p.Filter.Today.Format("20060102") + "." + p.Format
	f, err := s.files.StoreGenerated(ctx, "exports", name, mime, p.Format, data, nil)
	if err != nil {
		return nil, err
	}
	return CrewExportResult{FileID: f.ID, Filename: name, Rows: len(rows) - 1}, nil
}

func (s *CrewExportService) datePattern(ctx context.Context) string {
	general, err := s.settings.Values(ctx, "general")
	if err != nil {
		return datetime.DefaultDisplayFormat
	}
	if p, ok := general["date_format"].(string); ok && p != "" {
		return p
	}
	return datetime.DefaultDisplayFormat
}

var exportHeader = []string{
	"Crew ID", "Full Name", "Rank", "Department", "Status", "Nationality", "Date of Birth", "Age", "Mobile Phone",
	"Email", "Passport No", "Seaman Book No", "Current Vessel", "Vessel Type", "Principal", "Compliance",
	"Nearest Expiry", "Expiry Badge", "Sea Service",
}

func exportRecord(c model.CrewRow, today time.Time, t Thresholds, pattern string) []string {
	formatDate := func(d *time.Time) string {
		if d == nil {
			return ""
		}
		return datetime.FormatDate(*d, pattern)
	}
	age := ""
	if a := datetime.Age(c.DateOfBirth, today); a != nil {
		age = fmt.Sprint(*a)
	}
	compliance := "Not Compliant"
	if c.Compliant {
		compliance = "Compliant"
	}
	return []string{
		c.PublicID, c.FullName, c.RankName, departmentLabel(c.Department), c.StatusName, c.Nationality,
		formatDate(c.DateOfBirth), age, c.MobilePhone, c.Email, c.PassportNumber, c.SeamanBook, c.VesselName,
		c.VesselType, c.PrincipalName, compliance, formatDate(c.NearestExpiry), ExpiryBadge(c.NearestExpiry, today, t),
		NewSpan(c.SeaServiceDays).String(),
	}
}

func departmentLabel(d string) string {
	switch d {
	case "deck":
		return "Deck"
	case "engine":
		return "Engine"
	case "catering_other":
		return "Catering/Other"
	}
	return ""
}

// SafeCell neutralises spreadsheet formula injection by prefixing risky leading characters.
func SafeCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

func encodeExport(format string, rows [][]string) ([]byte, string, error) {
	if format == "csv" {
		var buf bytes.Buffer
		buf.WriteString(utf8BOM)
		w := csv.NewWriter(&buf)
		for _, r := range rows {
			safe := make([]string, len(r))
			for i, v := range r {
				safe[i] = SafeCell(v)
			}
			if err := w.Write(safe); err != nil {
				return nil, "", fmt.Errorf("write csv: %w", err)
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return nil, "", fmt.Errorf("write csv: %w", err)
		}
		return buf.Bytes(), "text/csv; charset=utf-8", nil
	}
	return encodeXLSX(rows)
}

func encodeXLSX(rows [][]string) ([]byte, string, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	const sheet = "Crew"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, "", fmt.Errorf("name sheet: %w", err)
	}
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return nil, "", fmt.Errorf("open stream writer: %w", err)
	}
	for i, r := range rows {
		cells := make([]any, len(r))
		for j, v := range r {
			cells[j] = excelize.Cell{Value: v}
		}
		addr, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return nil, "", fmt.Errorf("cell name: %w", err)
		}
		if err := sw.SetRow(addr, cells); err != nil {
			return nil, "", fmt.Errorf("write row: %w", err)
		}
	}
	if err := sw.Flush(); err != nil {
		return nil, "", fmt.Errorf("flush sheet: %w", err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", fmt.Errorf("encode xlsx: %w", err)
	}
	return buf.Bytes(), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
}
