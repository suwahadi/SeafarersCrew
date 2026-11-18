package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/suwahadi/seafarerscrew/backend/internal/model"
)

// PayrollNewAccount is a bank account an import adds to a crew profile, referenced by Key from staged lines.
type PayrollNewAccount struct {
	Key     string
	Account model.BankAccount
}

// PayrollStagedLine is an imported line; NewAccountKey points at a PayrollNewAccount when the account is new.
type PayrollStagedLine struct {
	Line          model.PayrollLine
	NewAccountKey string
}

// CrewsByEmployeeNo returns the crew members with the given normalized Employee IDs, keyed by that form.
func (r *PayrollRepository) CrewsByEmployeeNo(ctx context.Context, normalized []string) (map[string]model.PayrollCrew, error) {
	out := map[string]model.PayrollCrew{}
	if len(normalized) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT upper(ltrim(employee_no, '0')), id, public_id, full_name, employee_no FROM crews
		WHERE deleted_at IS NULL AND employee_no <> '' AND upper(ltrim(employee_no, '0')) = ANY($1)`, normalized)
	if err != nil {
		return nil, fmt.Errorf("find crews by employee no: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var c model.PayrollCrew
		if err := rows.Scan(&key, &c.ID, &c.PublicID, &c.FullName, &c.EmployeeNo); err != nil {
			return nil, fmt.Errorf("scan crew by employee no: %w", err)
		}
		out[key] = c
	}
	return out, rows.Err()
}

// BankAccountsByCrew returns the bank accounts of the given crew members, primary first.
func (r *PayrollRepository) BankAccountsByCrew(ctx context.Context, crewIDs []uuid.UUID) (map[uuid.UUID][]model.BankAccount, error) {
	out := map[uuid.UUID][]model.BankAccount{}
	if len(crewIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT `+bankColumns+` FROM crew_bank_accounts WHERE crew_id = ANY($1)
		ORDER BY crew_id, is_primary DESC, created_at`, crewIDs)
	if err != nil {
		return nil, fmt.Errorf("list bank accounts by crew: %w", err)
	}
	accounts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (model.BankAccount, error) { return scanBank(row) })
	if err != nil {
		return nil, fmt.Errorf("scan bank accounts by crew: %w", err)
	}
	for _, a := range accounts {
		out[a.CrewID] = append(out[a.CrewID], a)
	}
	return out, nil
}

// CoveringContracts returns, per crew member, the rank code and vessel of the latest non-draft contract that overlaps start..end.
func (r *PayrollRepository) CoveringContracts(ctx context.Context, crewIDs []uuid.UUID, start, end time.Time) (map[uuid.UUID]model.PayrollPlacement, error) {
	out := map[uuid.UUID]model.PayrollPlacement{}
	if len(crewIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT DISTINCT ON (k.crew_id) k.crew_id, r.short_code, v.name
		FROM contracts k
		JOIN ranks r ON r.id = k.rank_id
		JOIN vessels v ON v.id = k.vessel_id
		WHERE k.crew_id = ANY($1) AND k.deleted_at IS NULL AND k.status IN ('active', 'completed', 'terminated')
			AND k.sign_on_date <= $3 AND coalesce(k.actual_sign_off_date, k.planned_sign_off_date) >= $2
		ORDER BY k.crew_id, k.sign_on_date DESC`, crewIDs, start, end)
	if err != nil {
		return nil, fmt.Errorf("find covering contracts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var p model.PayrollPlacement
		if err := rows.Scan(&id, &p.RankCode, &p.VesselName); err != nil {
			return nil, fmt.Errorf("scan covering contract: %w", err)
		}
		out[id] = p
	}
	return out, rows.Err()
}

// ReplaceLines swaps all lines of a period for the staged ones and adds the new bank accounts, in one transaction.
func (r *PayrollRepository) ReplaceLines(ctx context.Context, periodID uuid.UUID, accounts []PayrollNewAccount, lines []PayrollStagedLine) (int, error) {
	added := 0
	err := InTx(ctx, r.pool, func(tx pgx.Tx) error {
		var locked uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM payroll_periods WHERE id = $1 FOR UPDATE`, periodID).Scan(&locked); err != nil {
			return fmt.Errorf("lock payroll period: %w", notFound(err))
		}
		ids := make(map[string]uuid.UUID, len(accounts))
		for _, a := range accounts {
			id, inserted, err := upsertImportedAccount(ctx, tx, a.Account)
			if err != nil {
				return err
			}
			ids[a.Key] = id
			if inserted {
				added++
			}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM payroll_lines WHERE period_id = $1`, periodID); err != nil {
			return fmt.Errorf("clear payroll lines: %w", err)
		}
		batch := &pgx.Batch{}
		for i, s := range lines {
			l := s.Line
			account := l.BankAccountID
			if s.NewAccountKey != "" {
				id, ok := ids[s.NewAccountKey]
				if !ok {
					return fmt.Errorf("staged line %d references an unknown account", i+1)
				}
				account = &id
			}
			batch.Queue(`INSERT INTO payroll_lines (period_id, line_no, crew_id, bank_account_id, employee_no, crew_name, rank_code,
				vessel_name, bank_name, account_name, account_number, amount_usd, amount_idr, transfer_date, remark)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
				periodID, i+1, l.CrewID, account, l.EmployeeNo, l.CrewName, l.RankCode, l.VesselName, l.BankName, l.AccountName,
				l.AccountNumber, l.AmountUSD, l.AmountIDR, l.TransferDate, l.Remark)
		}
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return fmt.Errorf("insert payroll lines: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return added, nil
}

func upsertImportedAccount(ctx context.Context, tx pgx.Tx, b model.BankAccount) (uuid.UUID, bool, error) {
	compact := strings.NewReplacer(" ", "", "-", "").Replace(b.AccountNumber)
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM crew_bank_accounts
		WHERE crew_id = $1 AND replace(replace(account_number, ' ', ''), '-', '') = $2 ORDER BY created_at LIMIT 1`, b.CrewID, compact).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, fmt.Errorf("find imported bank account: %w", err)
	}
	err = tx.QueryRow(ctx, `INSERT INTO crew_bank_accounts (crew_id, bank_name, account_name, account_number, currency_code, cover_bank, remarks, is_primary)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8 AND NOT EXISTS (SELECT 1 FROM crew_bank_accounts WHERE crew_id = $1 AND is_primary)
		RETURNING id`, b.CrewID, b.BankName, b.AccountName, b.AccountNumber, b.CurrencyCode, b.CoverBank, b.Remarks, b.IsPrimary).Scan(&id)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("insert imported bank account: %w", err)
	}
	return id, true, nil
}
