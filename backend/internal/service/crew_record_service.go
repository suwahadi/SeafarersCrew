package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/repository"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/datetime"
)

// CrewRecordStore persists the rows of the crew tabs.
type CrewRecordStore interface {
	ListBankAccounts(ctx context.Context, crewID uuid.UUID) ([]model.BankAccount, error)
	SaveBankAccount(ctx context.Context, b model.BankAccount) (model.BankAccount, error)
	DeleteBankAccount(ctx context.Context, crewID, id uuid.UUID) error
	ListNextOfKins(ctx context.Context, crewID uuid.UUID) ([]model.NextOfKin, error)
	SaveNextOfKin(ctx context.Context, k model.NextOfKin) (model.NextOfKin, error)
	DeleteNextOfKin(ctx context.Context, crewID, id uuid.UUID) error
	ListDocuments(ctx context.Context, k repository.DocumentKind, crewID uuid.UUID) ([]model.CrewDocument, error)
	GetDocument(ctx context.Context, k repository.DocumentKind, crewID, id uuid.UUID) (model.CrewDocument, error)
	SaveDocument(ctx context.Context, k repository.DocumentKind, d model.CrewDocument) (uuid.UUID, error)
	DeleteDocument(ctx context.Context, k repository.DocumentKind, crewID, id uuid.UUID) error
	TypeHasExpiry(ctx context.Context, k repository.DocumentKind, typeID uuid.UUID) (bool, error)
	ListExperiences(ctx context.Context, crewID uuid.UUID) ([]model.Experience, error)
	GetExperience(ctx context.Context, crewID, id uuid.UUID) (model.Experience, error)
	SaveExperience(ctx context.Context, x model.Experience) (uuid.UUID, error)
	DeleteExperience(ctx context.Context, crewID, id uuid.UUID) error
}

// DocumentView is a document or certificate with its expiry badge.
type DocumentView struct {
	model.CrewDocument
	Badge    string `json:"badge"`
	DaysLeft *int   `json:"days_left"`
}

// ListBankAccounts returns the Bank Information tab.
func (s *CrewService) ListBankAccounts(ctx context.Context, publicID string) ([]model.BankAccount, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return nil, err
	}
	return s.records.ListBankAccounts(ctx, c.ID)
}

// SaveBankAccount creates (zero ID) or updates a bank account.
func (s *CrewService) SaveBankAccount(ctx context.Context, publicID string, b model.BankAccount) (model.BankAccount, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return model.BankAccount{}, err
	}
	b.CrewID = c.ID
	out, err := s.records.SaveBankAccount(ctx, b)
	return out, recordWriteError(err, "crew_bank_accounts", "Bank account")
}

// DeleteBankAccount removes a bank account.
func (s *CrewService) DeleteBankAccount(ctx context.Context, publicID string, id uuid.UUID) error {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	return recordWriteError(s.records.DeleteBankAccount(ctx, c.ID, id), "crew_bank_accounts", "Bank account")
}

// ListNextOfKins returns the Next of Kin tab.
func (s *CrewService) ListNextOfKins(ctx context.Context, publicID string) ([]model.NextOfKin, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return nil, err
	}
	return s.records.ListNextOfKins(ctx, c.ID)
}

// SaveNextOfKin creates (zero ID) or updates a next of kin.
func (s *CrewService) SaveNextOfKin(ctx context.Context, publicID string, k model.NextOfKin) (model.NextOfKin, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return model.NextOfKin{}, err
	}
	k.CrewID = c.ID
	out, err := s.records.SaveNextOfKin(ctx, k)
	return out, recordWriteError(err, "crew_next_of_kins", "Next of kin")
}

// DeleteNextOfKin removes a next of kin.
func (s *CrewService) DeleteNextOfKin(ctx context.Context, publicID string, id uuid.UUID) error {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	return recordWriteError(s.records.DeleteNextOfKin(ctx, c.ID, id), "crew_next_of_kins", "Next of kin")
}

// ListDocuments returns the Documents or Certificates tab with expiry badges.
func (s *CrewService) ListDocuments(ctx context.Context, k repository.DocumentKind, publicID string) ([]DocumentView, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return nil, err
	}
	docs, err := s.records.ListDocuments(ctx, k, c.ID)
	if err != nil {
		return nil, err
	}
	today, thresholds, err := s.todayAndThresholds(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]DocumentView, len(docs))
	for i, d := range docs {
		out[i] = documentView(d, today, thresholds)
	}
	return out, nil
}

// SaveDocument creates (zero ID) or updates a document or certificate.
func (s *CrewService) SaveDocument(ctx context.Context, k repository.DocumentKind, publicID string, d model.CrewDocument) (DocumentView, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return DocumentView{}, err
	}
	hasExpiry, err := s.records.TypeHasExpiry(ctx, k, d.TypeID)
	if errors.Is(err, repository.ErrNotFound) {
		return DocumentView{}, apperr.Validation(map[string]string{"type_id": "The selected type does not exist."})
	}
	if err != nil {
		return DocumentView{}, err
	}
	if hasExpiry && d.ExpiredDate == nil {
		return DocumentView{}, apperr.Validation(map[string]string{"expired_date": "This type requires an expiry date."})
	}
	if !hasExpiry {
		d.ExpiredDate = nil
	}
	d.CrewID = c.ID
	id, err := s.records.SaveDocument(ctx, k, d)
	if err != nil {
		return DocumentView{}, recordWriteError(err, k.Table, "Document")
	}
	saved, err := s.records.GetDocument(ctx, k, c.ID, id)
	if err != nil {
		return DocumentView{}, err
	}
	today, thresholds, err := s.todayAndThresholds(ctx)
	if err != nil {
		return DocumentView{}, err
	}
	return documentView(saved, today, thresholds), nil
}

// DeleteDocument removes a document or certificate.
func (s *CrewService) DeleteDocument(ctx context.Context, k repository.DocumentKind, publicID string, id uuid.UUID) error {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	return recordWriteError(s.records.DeleteDocument(ctx, k, c.ID, id), k.Table, "Document")
}

// SaveExperience creates (zero ID) or updates a manual sea service row; contract rows are read-only.
func (s *CrewService) SaveExperience(ctx context.Context, publicID string, x model.Experience) (model.Experience, error) {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return model.Experience{}, err
	}
	if x.ID != uuid.Nil {
		if err := s.ensureManualExperience(ctx, c.ID, x.ID); err != nil {
			return model.Experience{}, err
		}
	}
	x.CrewID = c.ID
	id, err := s.records.SaveExperience(ctx, x)
	if err != nil {
		return model.Experience{}, recordWriteError(err, "experiences", "Experience")
	}
	return s.records.GetExperience(ctx, c.ID, id)
}

// DeleteExperience removes a manual sea service row.
func (s *CrewService) DeleteExperience(ctx context.Context, publicID string, id uuid.UUID) error {
	c, err := s.Get(ctx, publicID)
	if err != nil {
		return err
	}
	if err := s.ensureManualExperience(ctx, c.ID, id); err != nil {
		return err
	}
	return recordWriteError(s.records.DeleteExperience(ctx, c.ID, id), "experiences", "Experience")
}

func (s *CrewService) ensureManualExperience(ctx context.Context, crewID, id uuid.UUID) error {
	cur, err := s.records.GetExperience(ctx, crewID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound("Experience")
	}
	if err != nil {
		return err
	}
	if cur.Source != "manual" {
		return apperr.Conflict("experience_from_contract", "Sea service created from a contract cannot be edited or deleted here.")
	}
	return nil
}

func (s *CrewService) todayAndThresholds(ctx context.Context) (time.Time, Thresholds, error) {
	loc, t, err := s.displaySettings(ctx)
	if err != nil {
		return time.Time{}, Thresholds{}, err
	}
	return datetime.Today(s.now(), loc), t, nil
}

func documentView(d model.CrewDocument, today time.Time, t Thresholds) DocumentView {
	expiry := d.ExpiredDate
	if !d.HasExpiry {
		expiry = nil
	}
	return DocumentView{CrewDocument: d, Badge: ExpiryBadge(expiry, today, t), DaysLeft: DaysToExpiry(expiry, today)}
}

var recordFieldByColumn = map[string]string{
	"currency_code":       "currency_code",
	"document_type_id":    "type_id",
	"certificate_type_id": "type_id",
	"file_id":             "file_id",
	"vessel_id":           "vessel_id",
	"rank_id":             "rank_id",
	"vessel_type_id":      "vessel_type_id",
	"company_id":          "company_id",
	"principal_id":        "principal_id",
}

func recordWriteError(err error, table, resource string) error {
	if err == nil {
		return nil
	}
	if col, ok := postgres.ForeignKeyColumn(err, table); ok {
		if field, known := recordFieldByColumn[col]; known {
			return apperr.Validation(map[string]string{field: "The selected value does not exist."})
		}
	}
	if errors.Is(err, repository.ErrNotFound) {
		return apperr.NotFound(resource)
	}
	return err
}
