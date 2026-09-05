package postgres

import (
	"context"
	"errors"
	"time"

	"at.draab/familyfinances/internal/account"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AccountStore is the PostgreSQL implementation of account.Store.
type AccountStore struct {
	pool *pgxpool.Pool
}

// NewAccountStore returns an AccountStore over pool.
func NewAccountStore(pool *pgxpool.Pool) *AccountStore { return &AccountStore{pool: pool} }

const accountCols = `id::text, owner_id::text, title, COALESCE(description, ''), type_id::text,
	currency, COALESCE(financial_institute, ''), opening_date, closing_date, disabled,
	created_at, updated_at`

func scanAccount(row pgx.Row) (account.Account, error) {
	var acc account.Account
	var opening time.Time
	var closing *time.Time
	err := row.Scan(&acc.ID, &acc.OwnerID, &acc.Title, &acc.Description, &acc.TypeID,
		&acc.Currency, &acc.FinancialInstitute, &opening, &closing, &acc.Disabled,
		&acc.CreatedAt, &acc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return account.Account{}, account.ErrNotFound
	}
	if err != nil {
		return account.Account{}, err
	}
	acc.OpeningDate = account.NewDate(opening)
	if closing != nil {
		d := account.NewDate(*closing)
		acc.ClosingDate = &d
	}
	return acc, nil
}

func (s *AccountStore) Create(ctx context.Context, ownerID string, in account.New) (account.Account, error) {
	var closing *time.Time
	if in.ClosingDate != nil {
		t := in.ClosingDate.Time
		closing = &t
	}
	acc, err := scanAccount(s.pool.QueryRow(ctx, `
		INSERT INTO accounts (owner_id, title, description, type_id, currency, financial_institute, opening_date, closing_date)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''), $7, $8)
		RETURNING `+accountCols,
		ownerID, in.Title, in.Description, in.TypeID, in.Currency, in.FinancialInstitute,
		in.OpeningDate.Time, closing,
	))
	if isForeignKeyViolation(err) {
		return account.Account{}, account.ErrInvalidValue
	}
	return acc, err
}

func (s *AccountStore) Get(ctx context.Context, ownerID, id string) (account.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx,
		`SELECT `+accountCols+` FROM accounts WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`,
		id, ownerID,
	))
}

func (s *AccountStore) List(ctx context.Context, ownerID string) ([]account.Account, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+accountCols+` FROM accounts WHERE owner_id = $1 AND deleted_at IS NULL ORDER BY created_at`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []account.Account
	for rows.Next() {
		acc, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, acc)
	}
	return out, rows.Err()
}

func (s *AccountStore) Update(ctx context.Context, ownerID, id string, upd account.Update) (account.Account, error) {
	var opening *time.Time
	if upd.OpeningDate != nil {
		t := upd.OpeningDate.Time
		opening = &t
	}
	var closing *time.Time
	if upd.ClosingDate.Set && upd.ClosingDate.Value != nil {
		t := upd.ClosingDate.Value.Time
		closing = &t
	}
	acc, err := scanAccount(s.pool.QueryRow(ctx, `
		UPDATE accounts SET
			title               = COALESCE($3, title),
			description         = COALESCE($4, description),
			type_id             = COALESCE($5, type_id),
			currency            = COALESCE($6, currency),
			financial_institute = COALESCE($7, financial_institute),
			opening_date        = COALESCE($8, opening_date),
			closing_date        = CASE WHEN $9 THEN $10 ELSE closing_date END,
			updated_at          = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		RETURNING `+accountCols,
		id, ownerID, upd.Title, upd.Description, upd.TypeID, upd.Currency,
		upd.FinancialInstitute, opening, upd.ClosingDate.Set, closing,
	))
	if isForeignKeyViolation(err) {
		return account.Account{}, account.ErrInvalidValue
	}
	return acc, err
}

func (s *AccountStore) SetDisabled(ctx context.Context, ownerID, id string, disabled bool) (account.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx, `
		UPDATE accounts SET disabled = $3, updated_at = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
		RETURNING `+accountCols,
		id, ownerID, disabled,
	))
}

func (s *AccountStore) SoftDelete(ctx context.Context, ownerID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET deleted_at = now() WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`,
		id, ownerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return account.ErrNotFound
	}
	return nil
}

func (s *AccountStore) Owner(ctx context.Context, id string) (string, string, bool, error) {
	var ownerID, currency string
	var disabled bool
	err := s.pool.QueryRow(ctx,
		`SELECT owner_id::text, currency, disabled FROM accounts WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&ownerID, &currency, &disabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, account.ErrNotFound
	}
	return ownerID, currency, disabled, err
}

// --- account types ---------------------------------------------------

func scanType(row pgx.Row) (account.Type, error) {
	var t account.Type
	err := row.Scan(&t.ID, &t.Name, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return account.Type{}, account.ErrNotFound
	}
	return t, err
}

func (s *AccountStore) ListTypes(ctx context.Context) ([]account.Type, error) {
	rows, err := s.pool.Query(ctx, `SELECT id::text, name, created_at FROM account_types ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []account.Type
	for rows.Next() {
		t, err := scanType(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *AccountStore) CreateType(ctx context.Context, name string) (account.Type, error) {
	t, err := scanType(s.pool.QueryRow(ctx,
		`INSERT INTO account_types (name) VALUES ($1) RETURNING id::text, name, created_at`, name,
	))
	if isUniqueViolation(err) {
		return account.Type{}, account.ErrInvalidValue
	}
	return t, err
}

func (s *AccountStore) UpdateType(ctx context.Context, id, name string) (account.Type, error) {
	t, err := scanType(s.pool.QueryRow(ctx,
		`UPDATE account_types SET name = $2 WHERE id = $1 RETURNING id::text, name, created_at`, id, name,
	))
	if isUniqueViolation(err) {
		return account.Type{}, account.ErrInvalidValue
	}
	return t, err
}

func (s *AccountStore) DeleteType(ctx context.Context, id string) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM account_types WHERE id = $1)`, id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return account.ErrNotFound
	}
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM account_types
		WHERE id = $1 AND NOT EXISTS (
			SELECT 1 FROM accounts WHERE type_id = $1 AND deleted_at IS NULL
		)`, id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return account.ErrTypeInUse
	}
	return nil
}

func (s *AccountStore) TypeExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM account_types WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}
