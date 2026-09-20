package postgres

import (
	"context"
	"errors"
	"strconv"
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

// recurringColsFor returns the SELECT column list every recurring
// transaction read query uses, correlating its subqueries against table —
// "recurring_transactions" for a single-row canonical fetch (Get, or
// Create/Update's post-write re-select) or "recurring_transaction_legs"
// for a listing. The legs view (migration 0031) presents every template
// once per account it touches, its account_id/to_account_id already
// reflecting the per-leg viewing perspective, so the same column shape
// works against either table unchanged — exactly the arrangement
// entryColsFor already uses for entries and entry_legs.
// account_currency/to_account_name/to_account_currency are resolved
// unconditionally, never gated by the reading caller's permission on the
// account named.
func recurringColsFor(table string) string {
	return `id::text, created_by::text, account_id::text, to_account_id::text, kind, title, COALESCE(description, ''),
	category_id::text, COALESCE(counterparty, ''), COALESCE(location, ''), amount, interval_unit, interval_count,
	starts_on, ends_on, created_at, updated_at,
	COALESCE((SELECT array_agg(tag_id::text) FROM recurring_transaction_tags WHERE recurring_transaction_id = ` + table + `.id), '{}'),
	COALESCE((SELECT COALESCE(display_name, email) FROM users WHERE users.id = ` + table + `.created_by), ''),
	COALESCE((SELECT currency FROM accounts WHERE accounts.id = ` + table + `.account_id), ''),
	COALESCE((SELECT title FROM accounts WHERE accounts.id = ` + table + `.to_account_id), ''),
	COALESCE((SELECT currency FROM accounts WHERE accounts.id = ` + table + `.to_account_id), '')`
}

// recurringCols is recurringColsFor("recurring_transactions") — the
// plain-table shape Get and the post-write re-selects use.
var recurringCols = recurringColsFor("recurring_transactions")

// scanRecurringRow scans one recurringColsFor row, plus any caller-supplied
// extra destinations appended after the fixed column list (List uses this
// for the legs view's trailing "native" column).
func scanRecurringRow(row pgx.Row, extra ...any) (rt.RecurringTransaction, error) {
	var r rt.RecurringTransaction
	var unit string
	var kind string
	var startsOn time.Time
	var endsOn *time.Time
	dest := []any{&r.ID, &r.CreatedBy, &r.AccountID, &r.ToAccountID, &kind, &r.Title, &r.Description,
		&r.CategoryID, &r.Counterparty, &r.Location, &r.Amount, &unit, &r.IntervalCount,
		&startsOn, &endsOn, &r.CreatedAt, &r.UpdatedAt,
		&r.TagIDs, &r.CreatedByName, &r.AccountCurrency, &r.ToAccountName, &r.ToAccountCurrency}
	err := row.Scan(append(dest, extra...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return rt.RecurringTransaction{}, rt.ErrNotFound
	}
	if err != nil {
		return rt.RecurringTransaction{}, err
	}
	r.Kind = rt.Kind(kind)
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

// scanRecurring scans a canonical, single-row fetch — always the template
// as stored, so Native is true.
func scanRecurring(row pgx.Row) (rt.RecurringTransaction, error) {
	r, err := scanRecurringRow(row)
	if err != nil {
		return rt.RecurringTransaction{}, err
	}
	r.Native = true
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
		INSERT INTO recurring_transactions (created_by, account_id, to_account_id, kind, title, description, category_id, counterparty, location, amount, interval_unit, interval_count, starts_on, ends_on)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12, $13, $14)
		RETURNING id::text`,
		createdBy, in.AccountID, in.ToAccountID, string(in.Kind), in.Title, in.Description, in.CategoryID, in.Counterparty, in.Location, in.Amount,
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

// List queries recurring_transaction_legs (migration 0031) rather than the
// table, which is what makes a self_transfer template appear once per
// account of its two that is in scope — or twice when both are. The three
// self-transfer modes are one extra WHERE clause each over that single
// view; account_id = ANY(...) applies unchanged in all three, so "once per
// account in scope" falls out of the same filter that already scopes an
// ordinary template. See design.md of add-recurring-self-transfers.
func (s *RecurringTransactionStore) List(ctx context.Context, f rt.Filter) ([]rt.RecurringTransaction, error) {
	if len(f.AccountIDs) == 0 {
		return nil, nil
	}
	args := []any{f.AccountIDs}
	// arg appends v and returns its positional placeholder, the same
	// accumulate-as-you-go shape EntryStore's own filter building uses.
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	where := `deleted_at IS NULL AND account_id = ANY($1::uuid[])`
	switch f.SelfTransfers {
	case rt.SelfTransferBothLegs:
		// Every leg of every kind — no extra clause.
	case rt.SelfTransferNative:
		where += ` AND native`
	default:
		where += ` AND native AND kind = 'transaction'`
	}
	// Tested for nil, never for length: an empty CategoryIDs is a category
	// filter that matches nothing (a category the caller holds no
	// permission on), not the absence of one — see rt.Filter's doc comment.
	if f.CategoryIDs != nil {
		where += ` AND category_id = ANY(` + arg(f.CategoryIDs) + `::uuid[])`
	}
	if f.TagID != nil {
		where += ` AND EXISTS (SELECT 1 FROM recurring_transaction_tags rtt
			WHERE rtt.recurring_transaction_id = recurring_transaction_legs.id AND rtt.tag_id = ` + arg(*f.TagID) + `::uuid)`
	}
	// native DESC puts a template's outgoing leg immediately before its
	// incoming one when both are listed; created_at/id keep the existing
	// ordering otherwise.
	rows, err := s.pool.Query(ctx,
		`SELECT `+recurringColsFor("recurring_transaction_legs")+`, native FROM recurring_transaction_legs
		WHERE `+where+`
		ORDER BY created_at, id, native DESC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []rt.RecurringTransaction
	for rows.Next() {
		var native bool
		r, err := scanRecurringRow(rows, &native)
		if err != nil {
			return nil, err
		}
		r.Native = native
		out = append(out, r)
	}
	return out, rows.Err()
}

// HasSelfTransferAccount reports whether accountID is named on either side
// of any non-deleted self_transfer template — the existence check backing
// internal/account's currency-immutability rule.
func (s *RecurringTransactionStore) HasSelfTransferAccount(ctx context.Context, accountID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM recurring_transactions
			WHERE kind = 'self_transfer' AND deleted_at IS NULL
				AND (account_id = $1 OR to_account_id = $1)
		)`, accountID).Scan(&exists)
	return exists, err
}
