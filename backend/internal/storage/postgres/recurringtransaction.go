package postgres

import (
	"context"
	"errors"
	"time"

	rt "at.draab/familyfinances/internal/recurringtransaction"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecurringTransactionStore is the PostgreSQL implementation of
// recurringtransaction.Store. Mirrors EntryStore's shape: account_currency
// and created_by_name are resolved server-side by this query, the same
// resolve-it-in-SQL precedent entryCols establishes, rather than by
// Service reaching back into internal/account or internal/auth.
type RecurringTransactionStore struct {
	pool *pgxpool.Pool
}

// NewRecurringTransactionStore returns a RecurringTransactionStore over pool.
func NewRecurringTransactionStore(pool *pgxpool.Pool) *RecurringTransactionStore {
	return &RecurringTransactionStore{pool: pool}
}

const recurringCols = `id::text, created_by::text, account_id::text, title, COALESCE(description, ''),
	category_id::text, COALESCE(counterparty, ''), COALESCE(location, ''), amount, interval_unit, interval_count,
	starts_on, ends_on, created_at, updated_at,
	COALESCE((SELECT array_agg(tag_id::text) FROM recurring_transaction_tags WHERE recurring_transaction_id = recurring_transactions.id), '{}'),
	COALESCE((SELECT COALESCE(display_name, email) FROM users WHERE users.id = recurring_transactions.created_by), ''),
	COALESCE((SELECT currency FROM accounts WHERE accounts.id = recurring_transactions.account_id), '')`

func scanRecurring(row pgx.Row) (rt.RecurringTransaction, error) {
	var r rt.RecurringTransaction
	var unit string
	var startsOn time.Time
	var endsOn *time.Time
	err := row.Scan(&r.ID, &r.CreatedBy, &r.AccountID, &r.Title, &r.Description,
		&r.CategoryID, &r.Counterparty, &r.Location, &r.Amount, &unit, &r.IntervalCount,
		&startsOn, &endsOn, &r.CreatedAt, &r.UpdatedAt,
		&r.TagIDs, &r.CreatedByName, &r.AccountCurrency)
	if errors.Is(err, pgx.ErrNoRows) {
		return rt.RecurringTransaction{}, rt.ErrNotFound
	}
	if err != nil {
		return rt.RecurringTransaction{}, err
	}
	r.IntervalUnit = rt.Unit(unit)
	r.StartsOn = rt.NewDate(startsOn)
	if endsOn != nil {
		d := rt.NewDate(*endsOn)
		r.EndsOn = &d
	}
	if r.TagIDs == nil {
		r.TagIDs = []string{}
	}
	return r, nil
}

func (s *RecurringTransactionStore) Create(ctx context.Context, createdBy string, in rt.New) (rt.RecurringTransaction, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return rt.RecurringTransaction{}, err
	}
	defer tx.Rollback(ctx)

	var endsOn *time.Time
	if in.EndsOn != nil {
		t := in.EndsOn.Time
		endsOn = &t
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO recurring_transactions (created_by, account_id, title, description, category_id, counterparty, location, amount, interval_unit, interval_count, starts_on, ends_on)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, NULLIF($6, ''), NULLIF($7, ''), $8, $9, $10, $11, $12)
		RETURNING id::text`,
		createdBy, in.AccountID, in.Title, in.Description, in.CategoryID, in.Counterparty, in.Location, in.Amount,
		string(in.IntervalUnit), in.IntervalCount, in.StartsOn.Time, endsOn,
	).Scan(&id)
	if isForeignKeyViolation(err) || isCheckViolation(err) {
		return rt.RecurringTransaction{}, rt.ErrInvalidValue
	}
	if err != nil {
		return rt.RecurringTransaction{}, err
	}

	for _, tagID := range in.TagIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO recurring_transaction_tags (recurring_transaction_id, tag_id) VALUES ($1, $2)`, id, tagID); err != nil {
			if isForeignKeyViolation(err) {
				return rt.RecurringTransaction{}, rt.ErrInvalidValue
			}
			return rt.RecurringTransaction{}, err
		}
	}

	r, err := scanRecurring(tx.QueryRow(ctx, `SELECT `+recurringCols+` FROM recurring_transactions WHERE id = $1`, id))
	if err != nil {
		return rt.RecurringTransaction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return rt.RecurringTransaction{}, err
	}
	return r, nil
}

func (s *RecurringTransactionStore) Get(ctx context.Context, id string) (rt.RecurringTransaction, error) {
	return scanRecurring(s.pool.QueryRow(ctx,
		`SELECT `+recurringCols+` FROM recurring_transactions WHERE id = $1 AND deleted_at IS NULL`,
		id,
	))
}

func (s *RecurringTransactionStore) Update(ctx context.Context, id string, upd rt.Update) (rt.RecurringTransaction, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return rt.RecurringTransaction{}, err
	}
	defer tx.Rollback(ctx)

	var intervalUnit *string
	if upd.IntervalUnit != nil {
		u := string(*upd.IntervalUnit)
		intervalUnit = &u
	}
	var endsOnSet bool
	var endsOn *time.Time
	if upd.EndsOn.Set {
		endsOnSet = true
		if upd.EndsOn.Value != nil {
			t := upd.EndsOn.Value.Time
			endsOn = &t
		}
	}
	var startsOn *time.Time
	if upd.StartsOn != nil {
		t := upd.StartsOn.Time
		startsOn = &t
	}

	tag, err := tx.Exec(ctx, `
		UPDATE recurring_transactions SET
			account_id     = COALESCE($2, account_id),
			title          = COALESCE($3, title),
			description    = COALESCE($4, description),
			category_id    = COALESCE($5, category_id),
			counterparty   = COALESCE($6, counterparty),
			location       = COALESCE($7, location),
			amount         = COALESCE($8, amount),
			interval_unit  = COALESCE($9, interval_unit),
			interval_count = COALESCE($10, interval_count),
			starts_on      = COALESCE($11, starts_on),
			ends_on        = CASE WHEN $12 THEN $13 ELSE ends_on END,
			updated_at     = now()
		WHERE id = $1 AND deleted_at IS NULL`,
		id, upd.AccountID, upd.Title, upd.Description, upd.CategoryID, upd.Counterparty, upd.Location, upd.Amount,
		intervalUnit, upd.IntervalCount, startsOn, endsOnSet, endsOn,
	)
	if isForeignKeyViolation(err) || isCheckViolation(err) {
		return rt.RecurringTransaction{}, rt.ErrInvalidValue
	}
	if err != nil {
		return rt.RecurringTransaction{}, err
	}
	if tag.RowsAffected() == 0 {
		return rt.RecurringTransaction{}, rt.ErrNotFound
	}

	if upd.TagIDs != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM recurring_transaction_tags WHERE recurring_transaction_id = $1`, id); err != nil {
			return rt.RecurringTransaction{}, err
		}
		for _, tagID := range *upd.TagIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO recurring_transaction_tags (recurring_transaction_id, tag_id) VALUES ($1, $2)`, id, tagID); err != nil {
				if isForeignKeyViolation(err) {
					return rt.RecurringTransaction{}, rt.ErrInvalidValue
				}
				return rt.RecurringTransaction{}, err
			}
		}
	}

	r, err := scanRecurring(tx.QueryRow(ctx, `SELECT `+recurringCols+` FROM recurring_transactions WHERE id = $1`, id))
	if err != nil {
		return rt.RecurringTransaction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return rt.RecurringTransaction{}, err
	}
	return r, nil
}

func (s *RecurringTransactionStore) SoftDelete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE recurring_transactions SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return rt.ErrNotFound
	}
	return nil
}

func (s *RecurringTransactionStore) List(ctx context.Context, f rt.Filter) ([]rt.RecurringTransaction, error) {
	if len(f.AccountIDs) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+recurringCols+` FROM recurring_transactions
		WHERE deleted_at IS NULL AND account_id = ANY($1::uuid[])
		ORDER BY created_at, id`,
		f.AccountIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []rt.RecurringTransaction
	for rows.Next() {
		r, err := scanRecurring(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
