package postgres

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"at.draab/familyfinances/internal/entry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EntryStore is the PostgreSQL implementation of entry.Store.
type EntryStore struct {
	pool *pgxpool.Pool
}

// NewEntryStore returns an EntryStore over pool.
func NewEntryStore(pool *pgxpool.Pool) *EntryStore { return &EntryStore{pool: pool} }

// entryCols always assumes the query's FROM clause is literally "entries"
// (no alias), since the tag_ids and created_by_name subqueries correlate
// against entries.id / entries.created_by. created_by_name reaches into
// internal/auth's users table by raw SQL — entry.Store doesn't import
// internal/auth, but its Postgres implementation may still name its
// tables, the same precedent internal/storage/postgres/tag.go's
// entry_count and category.go's Delete already establish for reaching
// into another domain's table.
const entryCols = `id::text, created_by::text, account_id::text, kind, amount, balance_reading, booking_timestamp, title,
	COALESCE(description, ''), category_id::text, created_at, updated_at,
	COALESCE((SELECT array_agg(tag_id::text) FROM entry_tags WHERE entry_id = entries.id), '{}'),
	COALESCE((SELECT COALESCE(display_name, email) FROM users WHERE users.id = entries.created_by), '')`

func scanEntry(row pgx.Row) (entry.Entry, error) {
	var e entry.Entry
	var kind string
	err := row.Scan(&e.ID, &e.CreatedBy, &e.AccountID, &kind, &e.Amount, &e.Balance, &e.BookingTimestamp, &e.Title,
		&e.Description, &e.CategoryID, &e.CreatedAt, &e.UpdatedAt, &e.TagIDs, &e.CreatedByName)
	if errors.Is(err, pgx.ErrNoRows) {
		return entry.Entry{}, entry.ErrNotFound
	}
	if err != nil {
		return entry.Entry{}, err
	}
	e.Kind = entry.Kind(kind)
	if e.TagIDs == nil {
		e.TagIDs = []string{}
	}
	return e, nil
}

// parseEntryID converts the domain's string id to the bigint entries.id
// really is. A malformed id can never match a row, so it maps to
// ErrNotFound rather than a query error.
func parseEntryID(id string) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, entry.ErrNotFound
	}
	return n, nil
}

func (s *EntryStore) Create(ctx context.Context, createdBy string, in entry.New) (entry.Entry, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entry.Entry{}, err
	}
	defer tx.Rollback(ctx)

	// A balance adjustment's amount is never client-supplied — it starts as
	// a 0 placeholder, immediately overwritten below by recomputeFrom (which
	// finds this very row, since nothing else can share its just-assigned
	// id) using its balance_reading. A transaction stores in.Amount as-is,
	// with balance_reading left NULL.
	var amount int64
	if in.Kind == entry.KindTransaction && in.Amount != nil {
		amount = *in.Amount
	}

	var id int64
	var bookingTS time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO entries (created_by, account_id, kind, amount, balance_reading, booking_timestamp, title, description, category_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9)
		RETURNING id, booking_timestamp`,
		createdBy, in.AccountID, string(in.Kind), amount, in.Balance, in.BookingTimestamp, in.Title, in.Description, in.CategoryID,
	).Scan(&id, &bookingTS)
	if isForeignKeyViolation(err) || isCheckViolation(err) {
		return entry.Entry{}, entry.ErrInvalidValue
	}
	if err != nil {
		return entry.Entry{}, err
	}

	for _, tagID := range in.TagIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO entry_tags (entry_id, tag_id) VALUES ($1, $2)`, id, tagID); err != nil {
			if isForeignKeyViolation(err) {
				return entry.Entry{}, entry.ErrInvalidValue
			}
			return entry.Entry{}, err
		}
	}

	if err := recomputeFrom(ctx, tx, in.AccountID, bookingTS, id, 0); err != nil {
		return entry.Entry{}, err
	}

	e, err := scanEntry(tx.QueryRow(ctx, `SELECT `+entryCols+` FROM entries WHERE id = $1`, id))
	if err != nil {
		return entry.Entry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return entry.Entry{}, err
	}
	return e, nil
}

func (s *EntryStore) Get(ctx context.Context, id string) (entry.Entry, error) {
	eid, err := parseEntryID(id)
	if err != nil {
		return entry.Entry{}, err
	}
	return scanEntry(s.pool.QueryRow(ctx,
		`SELECT `+entryCols+` FROM entries WHERE id = $1 AND deleted_at IS NULL`,
		eid,
	))
}

func (s *EntryStore) Update(ctx context.Context, id string, upd entry.Update) (entry.Entry, error) {
	eid, err := parseEntryID(id)
	if err != nil {
		return entry.Entry{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entry.Entry{}, err
	}
	defer tx.Rollback(ctx)

	// Captured before the update so both the position this entry is
	// leaving and the one it lands on (if either its account or its
	// booking_timestamp changes) can have their nearest balance adjustment
	// recomputed — see design.md's algorithm.
	var oldAccountID string
	var oldTS time.Time
	if err := tx.QueryRow(ctx,
		`SELECT account_id::text, booking_timestamp FROM entries WHERE id = $1 AND deleted_at IS NULL`,
		eid,
	).Scan(&oldAccountID, &oldTS); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entry.Entry{}, entry.ErrNotFound
		}
		return entry.Entry{}, err
	}

	var categorySet bool
	var categoryID *string
	if upd.CategoryID.Set {
		categorySet = true
		categoryID = upd.CategoryID.Value
	}
	tag, err := tx.Exec(ctx, `
		UPDATE entries SET
			account_id        = COALESCE($2, account_id),
			amount            = COALESCE($3, amount),
			balance_reading   = COALESCE($4, balance_reading),
			booking_timestamp = COALESCE($5, booking_timestamp),
			title             = COALESCE($6, title),
			description       = COALESCE($7, description),
			category_id       = CASE WHEN $8 THEN $9 ELSE category_id END,
			updated_at        = now()
		WHERE id = $1 AND deleted_at IS NULL`,
		eid, upd.AccountID, upd.Amount, upd.Balance, upd.BookingTimestamp, upd.Title, upd.Description, categorySet, categoryID,
	)
	if isForeignKeyViolation(err) || isCheckViolation(err) {
		return entry.Entry{}, entry.ErrInvalidValue
	}
	if err != nil {
		return entry.Entry{}, err
	}
	if tag.RowsAffected() == 0 {
		return entry.Entry{}, entry.ErrNotFound
	}

	if upd.TagIDs != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM entry_tags WHERE entry_id = $1`, eid); err != nil {
			return entry.Entry{}, err
		}
		for _, tagID := range *upd.TagIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO entry_tags (entry_id, tag_id) VALUES ($1, $2)`, eid, tagID); err != nil {
				if isForeignKeyViolation(err) {
					return entry.Entry{}, entry.ErrInvalidValue
				}
				return entry.Entry{}, err
			}
		}
	}

	var newAccountID string
	var newTS time.Time
	if err := tx.QueryRow(ctx, `SELECT account_id::text, booking_timestamp FROM entries WHERE id = $1`, eid).
		Scan(&newAccountID, &newTS); err != nil {
		return entry.Entry{}, err
	}
	if newAccountID != oldAccountID || !newTS.Equal(oldTS) {
		// This row already carries its new account_id/booking_timestamp, so
		// the old-position search must exclude it explicitly (id=eid) —
		// otherwise, if only the timestamp moved later within the same
		// account, its own (already-updated) row could still satisfy the
		// old position's ">=" comparison and be mistaken for what's still
		// there.
		if err := recomputeFrom(ctx, tx, oldAccountID, oldTS, eid, eid); err != nil {
			return entry.Entry{}, err
		}
		if err := recomputeFrom(ctx, tx, newAccountID, newTS, eid, 0); err != nil {
			return entry.Entry{}, err
		}
	} else {
		if err := recomputeFrom(ctx, tx, oldAccountID, oldTS, eid, 0); err != nil {
			return entry.Entry{}, err
		}
	}

	e, err := scanEntry(tx.QueryRow(ctx, `SELECT `+entryCols+` FROM entries WHERE id = $1`, eid))
	if err != nil {
		return entry.Entry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return entry.Entry{}, err
	}
	return e, nil
}

func (s *EntryStore) SoftDelete(ctx context.Context, id string) error {
	eid, err := parseEntryID(id)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var accountID string
	var ts time.Time
	if err := tx.QueryRow(ctx,
		`SELECT account_id::text, booking_timestamp FROM entries WHERE id = $1 AND deleted_at IS NULL`,
		eid,
	).Scan(&accountID, &ts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entry.ErrNotFound
		}
		return err
	}

	if _, err := tx.Exec(ctx, `UPDATE entries SET deleted_at = now() WHERE id = $1`, eid); err != nil {
		return err
	}

	// deleted_at IS NULL already excludes this row from every recompute
	// query below, so no explicit exclusion is needed the way a moved (but
	// still live) entry needs in Update.
	if err := recomputeFrom(ctx, tx, accountID, ts, eid, 0); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// buildWhere returns the WHERE clauses and positional args shared by List
// and Sum, scoping to non-deleted entries matching f's
// account/category/tag/kind/date-range/query filters — every column is
// qualified with "entries." so the same clauses work whether or not a
// caller's query joins another table. f.AccountIDs is always the sole
// caller-scoping mechanism (already narrowed by entry.Service to the
// caller's visible, owned-or-shared accounts before it reaches here — see
// design.md) rather than an owner/creator equality clause, so a viewer
// sees every entry on a visible account regardless of who created it.
// Callers append their own additional clauses (and, via the same
// numbering, args) as needed.
func buildWhere(f entry.Filter) (where []string, args []any) {
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	where = []string{
		"entries.deleted_at IS NULL",
		"entries.account_id = ANY(" + arg(f.AccountIDs) + "::uuid[])",
	}
	if f.CategoryID != nil {
		where = append(where, "entries.category_id = ANY("+arg(f.CategoryIDs)+"::uuid[])")
	}
	if f.TagID != nil {
		where = append(where, "EXISTS (SELECT 1 FROM entry_tags et WHERE et.entry_id = entries.id AND et.tag_id = "+arg(*f.TagID)+"::uuid)")
	}
	if f.Kind != nil {
		where = append(where, "entries.kind = "+arg(string(*f.Kind)))
	}
	if f.From != nil {
		where = append(where, "entries.booking_timestamp >= "+arg(*f.From))
	}
	if f.To != nil {
		where = append(where, "entries.booking_timestamp <= "+arg(*f.To))
	}
	if f.Query != "" {
		p := arg("%" + f.Query + "%")
		where = append(where, "(entries.title ILIKE "+p+" OR entries.description ILIKE "+p+")")
	}
	return where, args
}

// List builds a dynamic keyset query: every optional filter appends a WHERE
// clause and a positional argument; sort/dir pick the ORDER BY column and
// direction; the keyset comparison on (sortColumn, id) implements the
// cursor. It fetches Limit+1 rows to know whether a next page exists.
func (s *EntryStore) List(ctx context.Context, f entry.Filter) ([]entry.Entry, *entry.Cursor, error) {
	if len(f.AccountIDs) == 0 {
		return nil, nil, nil
	}

	where, args := buildWhere(f)
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	sortCol := "booking_timestamp"
	if f.Sort == entry.SortAmount {
		sortCol = "amount"
	}
	op, orderDir := ">", "ASC"
	if f.Dir == entry.DirDesc {
		op, orderDir = "<", "DESC"
	}

	if f.After != nil {
		afterID, err := parseEntryID(f.After.ID)
		if err != nil {
			return nil, nil, nil // an unparseable cursor simply yields no further rows
		}
		var sortVal any = f.After.BookingTimestamp
		if f.Sort == entry.SortAmount {
			sortVal = f.After.Amount
		}
		where = append(where, "("+sortCol+", id) "+op+" ("+arg(sortVal)+", "+arg(afterID)+")")
	}

	limitPlaceholder := arg(f.Limit + 1)
	query := `SELECT ` + entryCols + ` FROM entries WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY ` + sortCol + ` ` + orderDir + `, id ` + orderDir +
		` LIMIT ` + limitPlaceholder

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var items []entry.Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var next *entry.Cursor
	if len(items) > f.Limit {
		last := items[f.Limit-1]
		next = &entry.Cursor{BookingTimestamp: last.BookingTimestamp, Amount: last.Amount, ID: last.ID}
		items = items[:f.Limit]
	}
	return items, next, nil
}

// Balance implements design.md's live computation: the sum of every
// non-deleted entry's amount up to asOf. A balance adjustment's amount is
// kept, by the recompute helpers below (run inside Create/Update/
// SoftDelete), equal to its own balance_reading minus the balance strictly
// before it — so this plain sum reproduces exactly the same result as
// always resetting to the latest adjustment and summing only the
// transactions after it, with no kind-branching needed here.
func (s *EntryStore) Balance(ctx context.Context, accountID string, asOf time.Time) (int64, error) {
	var balance int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM entries
		WHERE account_id = $1 AND deleted_at IS NULL AND booking_timestamp <= $2`,
		accountID, asOf,
	).Scan(&balance)
	return balance, err
}

// findAdjustment locates the earliest non-deleted balance_adjustment on
// accountID at (inclusive=true) or strictly after (inclusive=false)
// position (ts, id), locking it (FOR UPDATE) against concurrent recompute.
// excludeID, when non-zero, filters out that one row id — used by
// recomputeFrom for a position an entry has just moved away from, so its
// own already-updated row (which may otherwise still satisfy the position
// comparison from its new location) is never mistaken for what's still at
// the old one.
func findAdjustment(ctx context.Context, tx pgx.Tx, accountID string, ts time.Time, id int64, inclusive bool, excludeID int64) (rowID int64, rowTS time.Time, reading int64, ok bool, err error) {
	op := ">"
	if inclusive {
		op = ">="
	}
	err = tx.QueryRow(ctx, `
		SELECT id, booking_timestamp, balance_reading FROM entries
		WHERE account_id = $1 AND kind = 'balance_adjustment' AND deleted_at IS NULL
			AND (booking_timestamp, id) `+op+` ($2, $3)
			AND ($4 = 0 OR id != $4)
		ORDER BY booking_timestamp, id
		LIMIT 1
		FOR UPDATE`,
		accountID, ts, id, excludeID,
	).Scan(&rowID, &rowTS, &reading)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, time.Time{}, 0, false, nil
	}
	if err != nil {
		return 0, time.Time{}, 0, false, err
	}
	return rowID, rowTS, reading, true, nil
}

// setAmount recomputes and stores the amount for the balance adjustment
// (id, ts) on accountID: its balance_reading minus the balance strictly
// before its own position.
func setAmount(ctx context.Context, tx pgx.Tx, accountID string, id int64, ts time.Time, reading int64) error {
	var before int64
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0) FROM entries
		WHERE account_id = $1 AND deleted_at IS NULL AND (booking_timestamp, id) < ($2, $3)`,
		accountID, ts, id,
	).Scan(&before); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE entries SET amount = $2 WHERE id = $1`, id, reading-before)
	return err
}

// recomputeFrom keeps every balance adjustment's stored amount correct
// after a mutation at (accountID, ts, id) — see design.md's algorithm and
// worked example: only the earliest non-deleted adjustment at or after
// that position (A1), and the one immediately after it (A2), can possibly
// need a new amount; nothing further downstream ever changes. excludeID (0
// = none) is forwarded to the A1 search only — see findAdjustment.
func recomputeFrom(ctx context.Context, tx pgx.Tx, accountID string, ts time.Time, id int64, excludeID int64) error {
	a1ID, a1TS, a1Reading, ok, err := findAdjustment(ctx, tx, accountID, ts, id, true, excludeID)
	if err != nil || !ok {
		return err
	}
	if err := setAmount(ctx, tx, accountID, a1ID, a1TS, a1Reading); err != nil {
		return err
	}

	a2ID, a2TS, a2Reading, ok, err := findAdjustment(ctx, tx, accountID, a1TS, a1ID, false, 0)
	if err != nil || !ok {
		return err
	}
	return setAmount(ctx, tx, accountID, a2ID, a2TS, a2Reading)
}

// Sum implements entry.Store's Sum: the same WHERE clauses List uses, forced
// to kind = 'transaction' regardless of f.Kind, aggregated per account id in
// SQL. It returns no currency — Service.Sum resolves each account's currency
// via AccountLookup and groups by it there, since this package has no
// business joining into a currency concept that belongs to internal/account.
func (s *EntryStore) Sum(ctx context.Context, f entry.Filter) (map[string]int64, int, error) {
	if len(f.AccountIDs) == 0 {
		return map[string]int64{}, 0, nil
	}

	// Clear f.Kind before buildWhere so it never contributes its own kind
	// clause — this forces kind = 'transaction' below regardless of what
	// the caller's filter asked for, rather than ANDing the two together
	// (which would wrongly return zero rows whenever f.Kind was set to
	// anything but transaction).
	f.Kind = nil
	where, args := buildWhere(f)
	where = append(where, "entries.kind = 'transaction'")

	query := `SELECT entries.account_id::text, SUM(entries.amount), COUNT(*)
		FROM entries WHERE ` + strings.Join(where, " AND ") + ` GROUP BY entries.account_id`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	perAccount := make(map[string]int64)
	var total int
	for rows.Next() {
		var accountID string
		var sum int64
		var count int
		if err := rows.Scan(&accountID, &sum, &count); err != nil {
			return nil, 0, err
		}
		perAccount[accountID] = sum
		total += count
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return perAccount, total, nil
}

// FlowSummary implements entry.Store's FlowSummary: one row per (account,
// period) combination with at least one matching entry, bucketed by
// booking_timestamp converted into f.Timezone (Service.FlowSummary already
// resolved it) and truncated to f.Unit ('month' or 'day' — usable directly
// as date_trunc's field argument, since FlowUnit's only two values are
// exactly those words). income/outcome split by amount's sign via FILTER,
// so a currency needs no CASE-WHEN gymnastics. f.AccountIDs is the sole
// caller-scoping mechanism, already narrowed by Service to the caller's
// visible accounts.
func (s *EntryStore) FlowSummary(ctx context.Context, f entry.FlowFilter) ([]entry.FlowRow, error) {
	if len(f.AccountIDs) == 0 {
		return nil, nil
	}

	where := []string{
		"account_id = ANY($1::uuid[])",
		"deleted_at IS NULL",
		"EXTRACT(year FROM timezone($2, booking_timestamp)) = $4",
	}
	args := []any{f.AccountIDs, f.Timezone, string(f.Unit), f.Year}
	if f.Unit == entry.FlowUnitDay {
		where = append(where, "EXTRACT(month FROM timezone($2, booking_timestamp)) = $5")
		args = append(args, f.Month)
	}

	query := `
		SELECT date_trunc($3, timezone($2, booking_timestamp))::date AS period,
		       account_id::text,
		       COALESCE(SUM(amount) FILTER (WHERE amount > 0), 0) AS income,
		       COALESCE(-SUM(amount) FILTER (WHERE amount < 0), 0) AS outcome
		FROM entries
		WHERE ` + strings.Join(where, " AND ") + `
		GROUP BY period, account_id`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []entry.FlowRow
	for rows.Next() {
		var period time.Time
		var r entry.FlowRow
		if err := rows.Scan(&period, &r.AccountID, &r.Income, &r.Outcome); err != nil {
			return nil, err
		}
		r.Period = period.Format("2006-01-02")
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
