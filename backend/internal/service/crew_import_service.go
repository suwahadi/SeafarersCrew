package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/request"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/filecheck"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/publicid"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/token"
)

// CrewImportJob is the job kind for committing a validated import.
const CrewImportJob = "crew_import"

// Import limits.
const (
	MaxImportRows   = 5000
	importBatchSize = 100
	importTTL       = time.Hour
)

// ImportColumns is the header of the import template, in order.
var ImportColumns = []string{
	"full_name", "date_of_birth", "place_of_birth", "gender", "nationality", "religion", "marital_status", "applied_rank",
	"mobile_phone", "email", "address", "city", "province", "country", "postal_code",
	"passport_number", "passport_expiry", "seaman_book_number", "seaman_book_expiry",
}

// ImportStore is the persistence the importer needs.
type ImportStore interface {
	ImportLookups(ctx context.Context) (repository.ImportLookups, error)
	ExistingDocumentNumbers(ctx context.Context, typeID uuid.UUID, numbers []string) (map[string]bool, error)
	ImportCrews(ctx context.Context, rows []repository.ImportCrew, newID func() (string, error), attempts int) ([]repository.ImportOutcome, error)
}

// CrewImportService validates CSV files and imports crew in batches (PRD 6.9).
type CrewImportService struct {
	store ImportStore
	cache Cache
	jobs  Enqueuer
	newID func() (string, error)
}

// NewCrewImportService builds a CrewImportService.
func NewCrewImportService(store ImportStore, cache Cache, jobs Enqueuer) *CrewImportService {
	return &CrewImportService{store: store, cache: cache, jobs: jobs, newID: publicid.New}
}

// Import row states.
const (
	RowValid     = "valid"
	RowInvalid   = "invalid"
	RowDuplicate = "duplicate"
)

// ImportPreviewRow is one validated row.
type ImportPreviewRow struct {
	Line   int               `json:"line"`
	Status string            `json:"status"`
	Values map[string]string `json:"values"`
	Errors map[string]string `json:"errors,omitempty"`
}

// ImportPreview is returned by validation; Token commits the valid rows.
type ImportPreview struct {
	Token     string             `json:"token"`
	Total     int                `json:"total"`
	Valid     int                `json:"valid"`
	Invalid   int                `json:"invalid"`
	Duplicate int                `json:"duplicate"`
	Rows      []ImportPreviewRow `json:"rows"`
}

// ImportReport is the job result of a commit.
type ImportReport struct {
	Created  int                        `json:"created"`
	Skipped  int                        `json:"skipped"`
	Failed   int                        `json:"failed"`
	Outcomes []repository.ImportOutcome `json:"outcomes"`
}

type stagedImport struct {
	Rows    []repository.ImportCrew `json:"rows"`
	Skipped int                     `json:"skipped"`
}

// Template returns the CSV template with one example row.
func (s *CrewImportService) Template() []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(ImportColumns)
	_ = w.Write([]string{"Budi Santoso", "1988-04-12", "Surabaya", "male", "Indonesian", "Islam", "married", "CO",
		"+6281234567890", "budi@example.com", "Jl. Pelabuhan 1", "Surabaya", "Jawa Timur", "Indonesia", "60165",
		"X1234567", "2030-04-11", "F123456", "2029-01-31"})
	w.Flush()
	return buf.Bytes()
}

// Validate parses and checks every row, stages the valid ones and returns a preview with per-row errors.
func (s *CrewImportService) Validate(ctx context.Context, data []byte) (ImportPreview, error) {
	if err := filecheck.CheckCSV(data); err != nil {
		return ImportPreview{}, err
	}
	records, err := readCSV(data)
	if err != nil {
		return ImportPreview{}, err
	}
	if len(records) < 2 {
		return ImportPreview{}, apperr.Validation(map[string]string{"file": "The file has no data rows."})
	}
	if len(records)-1 > MaxImportRows {
		return ImportPreview{}, apperr.Validation(map[string]string{"file": fmt.Sprintf("The file may contain at most %d rows.", MaxImportRows)})
	}
	index, err := headerIndex(records[0])
	if err != nil {
		return ImportPreview{}, err
	}
	lookups, err := s.store.ImportLookups(ctx)
	if err != nil {
		return ImportPreview{}, err
	}
	rows := make([]ImportPreviewRow, 0, len(records)-1)
	parsed := make([]*repository.ImportCrew, 0, len(records)-1)
	for i, rec := range records[1:] {
		values := map[string]string{}
		for col, pos := range index {
			if pos < len(rec) {
				values[col] = strings.TrimSpace(rec[pos])
			}
		}
		crew, errs := parseImportRow(values, lookups)
		row := ImportPreviewRow{Line: i + 2, Values: values, Status: RowValid}
		if len(errs) > 0 {
			row.Status, row.Errors = RowInvalid, errs
			crew = nil
		} else {
			crew.Line = row.Line
		}
		rows = append(rows, row)
		parsed = append(parsed, crew)
	}
	if err := s.markDuplicates(ctx, rows, parsed, lookups); err != nil {
		return ImportPreview{}, err
	}
	return s.stage(ctx, rows, parsed)
}

func (s *CrewImportService) markDuplicates(ctx context.Context, rows []ImportPreviewRow, parsed []*repository.ImportCrew, l repository.ImportLookups) error {
	var passports []string
	for _, p := range parsed {
		if p == nil {
			continue
		}
		for _, d := range p.Documents {
			if d.TypeID == l.PassportType {
				passports = append(passports, strings.ToUpper(d.Number))
			}
		}
	}
	existing, err := s.store.ExistingDocumentNumbers(ctx, l.PassportType, passports)
	if err != nil {
		return err
	}
	seen := map[string]int{}
	for i, p := range parsed {
		if p == nil {
			continue
		}
		for _, d := range p.Documents {
			if d.TypeID != l.PassportType {
				continue
			}
			n := strings.ToUpper(d.Number)
			switch {
			case existing[n]:
				rows[i].Status, rows[i].Errors = RowDuplicate, map[string]string{"passport_number": "A crew member with this passport number already exists."}
			case seen[n] > 0:
				rows[i].Status, rows[i].Errors = RowDuplicate, map[string]string{"passport_number": fmt.Sprintf("Duplicate of line %d in this file.", seen[n])}
			default:
				seen[n] = rows[i].Line
			}
		}
		if rows[i].Status == RowDuplicate {
			parsed[i] = nil
		}
	}
	return nil
}

func (s *CrewImportService) stage(ctx context.Context, rows []ImportPreviewRow, parsed []*repository.ImportCrew) (ImportPreview, error) {
	preview := ImportPreview{Total: len(rows), Rows: rows}
	staged := stagedImport{}
	for i, r := range rows {
		switch r.Status {
		case RowValid:
			preview.Valid++
			staged.Rows = append(staged.Rows, *parsed[i])
		case RowInvalid:
			preview.Invalid++
		case RowDuplicate:
			preview.Duplicate++
			staged.Skipped++
		}
	}
	tok, err := token.URLSafe(18)
	if err != nil {
		return ImportPreview{}, err
	}
	payload, err := json.Marshal(staged)
	if err != nil {
		return ImportPreview{}, fmt.Errorf("encode staged import: %w", err)
	}
	if err := s.cache.Set(ctx, importKey(tok), payload, importTTL); err != nil {
		return ImportPreview{}, err
	}
	preview.Token = tok
	return preview, nil
}

// Commit queues the import of the rows staged under tok.
func (s *CrewImportService) Commit(ctx context.Context, tok string) (model.Job, error) {
	if _, ok, err := s.cache.Get(ctx, importKey(tok)); err != nil {
		return model.Job{}, err
	} else if !ok {
		return model.Job{}, apperr.New(410, "import_expired", "The validated import has expired; upload the file again.")
	}
	return s.jobs.Enqueue(ctx, CrewImportJob, map[string]string{"token": tok})
}

// Handle runs an import job in batches and returns the report.
func (s *CrewImportService) Handle(ctx context.Context, j model.Job) (any, error) {
	var p struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(j.Payload, &p); err != nil {
		return nil, fmt.Errorf("decode import payload: %w", err)
	}
	raw, ok, err := s.cache.Get(ctx, importKey(p.Token))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("staged import expired before it ran")
	}
	var staged stagedImport
	if err := json.Unmarshal(raw, &staged); err != nil {
		return nil, fmt.Errorf("decode staged import: %w", err)
	}
	report := ImportReport{Skipped: staged.Skipped, Outcomes: []repository.ImportOutcome{}}
	for start := 0; start < len(staged.Rows); start += importBatchSize {
		batch := staged.Rows[start:min(start+importBatchSize, len(staged.Rows))]
		outcomes, err := s.store.ImportCrews(ctx, batch, s.newID, publicid.MaxAttempts)
		if err != nil {
			return nil, err
		}
		for _, o := range outcomes {
			if o.Error != "" {
				report.Failed++
			} else {
				report.Created++
			}
		}
		report.Outcomes = append(report.Outcomes, outcomes...)
	}
	if err := s.cache.Delete(ctx, importKey(p.Token)); err != nil {
		return nil, err
	}
	return report, nil
}

func importKey(tok string) string { return "crew_import:" + tok }

func readCSV(data []byte) ([][]string, error) {
	r := csv.NewReader(bytes.NewReader(bytes.TrimPrefix(data, []byte(utf8BOM))))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	var out [][]string
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return nil, apperr.Validation(map[string]string{"file": "The CSV could not be parsed: " + err.Error()})
		}
		if len(rec) == 1 && strings.TrimSpace(rec[0]) == "" {
			continue
		}
		out = append(out, rec)
	}
}

func headerIndex(header []string) (map[string]int, error) {
	index := map[string]int{}
	var unknown []string
	for i, h := range header {
		name := strings.ToLower(strings.TrimSpace(h))
		if !slices.Contains(ImportColumns, name) {
			unknown = append(unknown, h)
			continue
		}
		index[name] = i
	}
	if len(unknown) > 0 {
		return nil, apperr.Validation(map[string]string{"file": "Unknown columns: " + strings.Join(unknown, ", ") + ". Use the import template."})
	}
	if _, ok := index["full_name"]; !ok {
		return nil, apperr.Validation(map[string]string{"file": "The full_name column is required. Use the import template."})
	}
	return index, nil
}

func parseImportRow(v map[string]string, l repository.ImportLookups) (*repository.ImportCrew, map[string]string) {
	in := request.Crew{
		FullName: v["full_name"], DateOfBirth: v["date_of_birth"], PlaceOfBirth: v["place_of_birth"],
		Gender: strings.ToLower(v["gender"]), Nationality: v["nationality"], Religion: v["religion"],
		MaritalStatus: strings.ToLower(v["marital_status"]), MobilePhone: v["mobile_phone"], Email: v["email"],
		Address: v["address"], City: v["city"], Province: v["province"], Country: v["country"], PostalCode: v["postal_code"],
	}
	errs := request.Errors{}
	if rank := v["applied_rank"]; rank != "" {
		id, ok := l.Ranks[strings.ToUpper(rank)]
		if !ok {
			errs["applied_rank"] = "Unknown rank; use a rank short code or name."
		} else {
			in.AppliedRankID = id.String()
		}
	}
	for field, msg := range in.Validate() {
		errs[field] = msg
	}
	crew := &repository.ImportCrew{Crew: in.Model()}
	addDoc := func(numberCol, expiryCol string, typeID uuid.UUID) {
		number := v[numberCol]
		if number == "" {
			if v[expiryCol] != "" {
				errs[numberCol] = "The number is required when an expiry date is given."
			}
			return
		}
		errs.MaxLen(numberCol, number, 60)
		if typeID == uuid.Nil {
			errs[numberCol] = "This document type is not configured."
			return
		}
		exp := errs.RequiredDate(expiryCol, v[expiryCol])
		crew.Documents = append(crew.Documents, repository.ImportDocument{TypeID: typeID, Number: number, ExpiredDate: &exp})
	}
	addDoc("passport_number", "passport_expiry", l.PassportType)
	addDoc("seaman_book_number", "seaman_book_expiry", l.SeamanBook)
	if len(errs) > 0 {
		return nil, errs
	}
	return crew, nil
}
