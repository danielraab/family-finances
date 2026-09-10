package postgres

import (
	"context"
	"errors"
	"fmt"
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

// accountCols is the plain column list — no caller-dependent fields — used
// by Create/Update/SetDisabled, whose Service callers already have the
// caller's Permission/Shared/OwnerName in hand from a preceding Get and
// copy it onto the returned row themselves (see account.Service.Update).
const accountCols = `id::text, owner_id::text, title, COALESCE(description, ''),
	COALESCE(icon, ''), COALESCE(color, ''), type,
	currency, COALESCE(financial_institute, ''), opening_date, closing_date, disabled,
	created_at, updated_at`

// accountViewCols is accountCols plus three caller-dependent columns —
// permission, shared, owner_name — used by Get/List, the two methods whose
// result becomes a caller-facing "what may I do here" answer directly.
// callerParam is the 1-based positional parameter index the caller's id is
// bound to elsewhere in the same query.
func accountViewCols(callerParam int) string {
	return accountCols + fmt.Sprintf(`,
	CASE WHEN owner_id = $%[1]d THEN 'owner' ELSE COALESCE(
		(SELECT permission FROM account_shares WHERE account_id = accounts.id AND user_id = $%[1]d), ''
	) END,
	(owner_id != $%[1]d),
	COALESCE((SELECT COALESCE(display_name, email) FROM users WHERE users.id = accounts.owner_id), '')`, callerParam)
}

func scanAccount(row pgx.Row) (account.Account, error) {
	var acc account.Account
	var opening time.Time
	var closing *time.Time
	err := row.Scan(&acc.ID, &acc.OwnerID, &acc.Title, &acc.Description,
		&acc.Icon, &acc.Color, &acc.Type,
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

func scanAccountView(row pgx.Row) (account.Account, error) {
	var acc account.Account
	var opening time.Time
	var closing *time.Time
	var permission string
	err := row.Scan(&acc.ID, &acc.OwnerID, &acc.Title, &acc.Description,
		&acc.Icon, &acc.Color, &acc.Type,
		&acc.Currency, &acc.FinancialInstitute, &opening, &closing, &acc.Disabled,
		&acc.CreatedAt, &acc.UpdatedAt, &permission, &acc.Shared, &acc.OwnerName)
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
	acc.Permission = account.Permission(permission)
	return acc, nil
}

func (s *AccountStore) Create(ctx context.Context, ownerID string, in account.New) (account.Account, error) {
	var closing *time.Time
	if in.ClosingDate != nil {
		t := in.ClosingDate.Time
		closing = &t
	}
	acc, err := scanAccount(s.pool.QueryRow(ctx, `
		INSERT INTO accounts (owner_id, title, description, icon, color, type, currency, financial_institute, opening_date, closing_date)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6, $7, NULLIF($8, ''), $9, $10)
		RETURNING `+accountCols,
		ownerID, in.Title, in.Description, in.Icon, in.Color, in.Type, in.Currency, in.FinancialInstitute,
		in.OpeningDate.Time, closing,
	))
	if isForeignKeyViolation(err) {
		return account.Account{}, account.ErrInvalidValue
	}
	return acc, err
}

func (s *AccountStore) Get(ctx context.Context, id, callerID string) (account.Account, error) {
	return scanAccountView(s.pool.QueryRow(ctx,
		`SELECT `+accountViewCols(2)+` FROM accounts WHERE id = $1 AND deleted_at IS NULL`,
		id, callerID,
	))
}

func (s *AccountStore) List(ctx context.Context, callerID string) ([]account.Account, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+accountViewCols(1)+` FROM accounts
		 WHERE deleted_at IS NULL
		   AND (owner_id = $1 OR EXISTS (
		       SELECT 1 FROM account_shares WHERE account_id = accounts.id AND user_id = $1
		   ))
		 ORDER BY created_at`,
		callerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []account.Account
	for rows.Next() {
		acc, err := scanAccountView(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, acc)
	}
	return out, rows.Err()
}

func (s *AccountStore) Update(ctx context.Context, id string, upd account.Update) (account.Account, error) {
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
			title               = COALESCE($2, title),
			description         = COALESCE($3, description),
			type                = COALESCE($4, type),
			currency            = COALESCE($5, currency),
			financial_institute = COALESCE($6, financial_institute),
			opening_date        = COALESCE($7, opening_date),
			closing_date        = CASE WHEN $8 THEN $9 ELSE closing_date END,
			icon                = CASE WHEN $10::text IS NULL THEN icon ELSE NULLIF($10, '') END,
			color               = CASE WHEN $11::text IS NULL THEN color ELSE NULLIF($11, '') END,
			updated_at          = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+accountCols,
		id, upd.Title, upd.Description, upd.Type, upd.Currency,
		upd.FinancialInstitute, opening, upd.ClosingDate.Set, closing,
		upd.Icon, upd.Color,
	))
	if isForeignKeyViolation(err) {
		return account.Account{}, account.ErrInvalidValue
	}
	return acc, err
}

func (s *AccountStore) SetDisabled(ctx context.Context, id string, disabled bool) (account.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx, `
		UPDATE accounts SET disabled = $2, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+accountCols,
		id, disabled,
	))
}

func (s *AccountStore) SoftDelete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return account.ErrNotFound
	}
	return nil
}

func (s *AccountStore) Access(ctx context.Context, id, callerID string) (account.Access, error) {
	var currency, permission string
	var disabled bool
	err := s.pool.QueryRow(ctx, `
		SELECT currency, disabled,
			CASE WHEN owner_id = $2 THEN 'owner' ELSE COALESCE(
				(SELECT permission FROM account_shares WHERE account_id = accounts.id AND user_id = $2), ''
			) END
		FROM accounts WHERE id = $1 AND deleted_at IS NULL`,
		id, callerID,
	).Scan(&currency, &disabled, &permission)
	if errors.Is(err, pgx.ErrNoRows) {
		return account.Access{}, account.ErrNotFound
	}
	if err != nil {
		return account.Access{}, err
	}
	return account.Access{Currency: currency, Disabled: disabled, Permission: account.Permission(permission)}, nil
}

func (s *AccountStore) VisibleIDs(ctx context.Context, callerID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text FROM accounts
		WHERE deleted_at IS NULL
		  AND (owner_id = $1 OR EXISTS (
		      SELECT 1 FROM account_shares WHERE account_id = accounts.id AND user_id = $1
		  ))`,
		callerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// --- sharing -------------------------------------------------------------

const shareCols = `account_shares.user_id::text, COALESCE(u.display_name, u.email), u.email,
	permission, granted_by::text, COALESCE(gu.display_name, gu.email), account_shares.created_at, account_shares.updated_at`

const shareJoins = `FROM account_shares
	JOIN users u ON u.id = account_shares.user_id
	JOIN users gu ON gu.id = account_shares.granted_by`

func scanShare(row pgx.Row, accountID string) (account.AccountShare, error) {
	var sh account.AccountShare
	err := row.Scan(&sh.UserID, &sh.Name, &sh.Email, &sh.Permission, &sh.GrantedBy, &sh.GrantedByName, &sh.CreatedAt, &sh.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return account.AccountShare{}, account.ErrNotFound
	}
	sh.AccountID = accountID
	return sh, err
}

func (s *AccountStore) CreateOrUpdateShare(ctx context.Context, accountID, userID string, permission account.Permission, grantedBy string) (account.AccountShare, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO account_shares (account_id, user_id, permission, granted_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (account_id, user_id)
		DO UPDATE SET permission = $3, granted_by = $4, updated_at = now()`,
		accountID, userID, permission, grantedBy,
	)
	if isForeignKeyViolation(err) {
		return account.AccountShare{}, account.ErrInvalidValue
	}
	if err != nil {
		return account.AccountShare{}, err
	}
	return scanShare(s.pool.QueryRow(ctx,
		`SELECT `+shareCols+` `+shareJoins+` WHERE account_shares.account_id = $1 AND account_shares.user_id = $2`,
		accountID, userID,
	), accountID)
}

func (s *AccountStore) ListShares(ctx context.Context, accountID string) ([]account.AccountShare, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+shareCols+` `+shareJoins+` WHERE account_shares.account_id = $1 ORDER BY account_shares.created_at`,
		accountID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []account.AccountShare
	for rows.Next() {
		sh, err := scanShare(rows, accountID)
		if err != nil {
			return nil, err
		}
		out = append(out, sh)
	}
	return out, rows.Err()
}

func (s *AccountStore) ShareByUser(ctx context.Context, accountID, userID string) (account.AccountShare, error) {
	return scanShare(s.pool.QueryRow(ctx,
		`SELECT `+shareCols+` `+shareJoins+` WHERE account_shares.account_id = $1 AND account_shares.user_id = $2`,
		accountID, userID,
	), accountID)
}

func (s *AccountStore) UpdateSharePermission(ctx context.Context, accountID, userID string, permission account.Permission) (account.AccountShare, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE account_shares SET permission = $3, updated_at = now() WHERE account_id = $1 AND user_id = $2`,
		accountID, userID, permission,
	)
	if err != nil {
		return account.AccountShare{}, err
	}
	if tag.RowsAffected() == 0 {
		return account.AccountShare{}, account.ErrNotFound
	}
	return s.ShareByUser(ctx, accountID, userID)
}

func (s *AccountStore) DeleteShare(ctx context.Context, accountID, userID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM account_shares WHERE account_id = $1 AND user_id = $2`,
		accountID, userID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return account.ErrNotFound
	}
	return nil
}

// --- account types ---------------------------------------------------

// ListInUseTypes returns the distinct, non-empty type labels on ownerID's
// own non-deleted accounts, sorted case-insensitively ascending — for the
// account form's autocomplete. A type has no row of its own; it exists
// only as text written on an account.
func (s *AccountStore) ListInUseTypes(ctx context.Context, ownerID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT type FROM (
			SELECT DISTINCT type FROM accounts
			WHERE owner_id = $1 AND deleted_at IS NULL AND btrim(type) <> ''
		) t
		ORDER BY lower(type), type`,
		ownerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
