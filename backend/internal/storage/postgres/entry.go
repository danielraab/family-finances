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
// (no alias), since the tag_ids subquery correlates against entries.id.
const entryCols = `id::text, owner_id::text, account_id::text, kind, amount, booking_timestamp, title,
	COALESCE(description, ''), category_id::text, created_at, updated_at,
	COALESCE((SELECT array_agg(tag_id::text) FROM entry_tags WHERE entry_id = entries.id), '{}')`

func scanEntry(row pgx.Row) (entry.Entry, error) {
	var e entry.Entry
	var kind string
	err := row.Scan(&e.ID, &e.OwnerID, &e.AccountID, &kind, &e.Amount, &e.BookingTimestamp, &e.Title,
		&e.Description, &e.CategoryID, &e.CreatedAt, &e.UpdatedAt, &e.TagIDs)
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

func (s *EntryStore) Create(ctx context.Context, ownerID string, in entry.New) (entry.Entry, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entry.Entry{}, err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `
		INSERT INTO entries (owner_id, account_id, kind, amount, booking_timestamp, title, description, category_id)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8)
		RETURNING id`,
		ownerID, in.AccountID, string(in.Kind), in.Amount, in.BookingTimestamp, in.Title, in.Description, in.CategoryID,
	).Scan(&id)
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

	e, err := scanEntry(tx.QueryRow(ctx, `SELECT `+entryCols+` FROM entries WHERE id = $1`, id))
	if err != nil {
		return entry.Entry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return entry.Entry{}, err
	}
	return e, nil
}

func (s *EntryStore) Get(ctx context.Context, ownerID, id string) (entry.Entry, error) {
	eid, err := parseEntryID(id)
	if err != nil {
		return entry.Entry{}, err
	}
	return scanEntry(s.pool.QueryRow(ctx,
		`SELECT `+entryCols+` FROM entries WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`,
		eid, ownerID,
	))
}

func (s *EntryStore) Update(ctx context.Context, ownerID, id string, upd entry.Update) (entry.Entry, error) {
	eid, err := parseEntryID(id)
	if err != nil {
		return entry.Entry{}, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return entry.Entry{}, err
	}
	defer tx.Rollback(ctx)

	var categorySet bool
	var categoryID *string
	if upd.CategoryID.Set {
		categorySet = true
		categoryID = upd.CategoryID.Value
	}
	tag, err := tx.Exec(ctx, `
		UPDATE entries SET
			amount            = COALESCE($3, amount),
			booking_timestamp = COALESCE($4, booking_timestamp),
			title             = COALESCE($5, title),
			description       = COALESCE($6, description),
			category_id       = CASE WHEN $7 THEN $8 ELSE category_id END,
			updated_at        = now()
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`,
		eid, ownerID, upd.Amount, upd.BookingTimestamp, upd.Title, upd.Description, categorySet, categoryID,
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

	e, err := scanEntry(tx.QueryRow(ctx, `SELECT `+entryCols+` FROM entries WHERE id = $1`, eid))
	if err != nil {
		return entry.Entry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return entry.Entry{}, err
	}
	return e, nil
}

func (s *EntryStore) SoftDelete(ctx context.Context, ownerID, id string) error {
	eid, err := parseEntryID(id)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE entries SET deleted_at = now() WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL`,
		eid, ownerID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entry.ErrNotFound
	}
	return nil
}

// List builds a dynamic keyset query: every optional filter appends a WHERE
// clause and a positional argument; sort/dir pick the ORDER BY column and
// direction; the keyset comparison on (sortColumn, id) implements the
// cursor. It fetches Limit+1 rows to know whether a next page exists.
func (s *EntryStore) List(ctx context.Context, ownerID string, f entry.Filter) ([]entry.Entry, *entry.Cursor, error) {
	if len(f.AccountIDs) == 0 {
		return nil, nil, nil
	}

	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}

	where := []string{
		"owner_id = " + arg(ownerID),
		"deleted_at IS NULL",
		"account_id = ANY(" + arg(f.AccountIDs) + "::uuid[])",
	}
	if f.CategoryID != nil {
		where = append(where, "category_id = ANY("+arg(f.CategoryIDs)+"::uuid[])")
	}
	if f.TagID != nil {
		where = append(where, "EXISTS (SELECT 1 FROM entry_tags et WHERE et.entry_id = entries.id AND et.tag_id = "+arg(*f.TagID)+"::uuid)")
	}
	if f.Kind != nil {
		where = append(where, "kind = "+arg(string(*f.Kind)))
	}
	if f.From != nil {
		where = append(where, "booking_timestamp >= "+arg(*f.From))
	}
	if f.To != nil {
		where = append(where, "booking_timestamp <= "+arg(*f.To))
	}
	if f.Query != "" {
		p := arg("%" + f.Query + "%")
		where = append(where, "(title ILIKE "+p+" OR description ILIKE "+p+")")
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

// Balance implements design.md's live computation directly in SQL: the
// latest non-deleted balance_adjustment at or before asOf (or 0 if none),
// plus every non-deleted transaction strictly after it (by the same
// (booking_timestamp, id) tie-break), up to asOf.
func (s *EntryStore) Balance(ctx context.Context, accountID string, asOf time.Time) (int64, error) {
	var balance int64
	err := s.pool.QueryRow(ctx, `
		WITH base AS (
			SELECT booking_timestamp, id, amount
			FROM entries
			WHERE account_id = $1 AND kind = 'balance_adjustment'
				AND deleted_at IS NULL AND booking_timestamp <= $2
			ORDER BY booking_timestamp DESC, id DESC
			LIMIT 1
		),
		txns AS (
			SELECT COALESCE(SUM(e.amount), 0) AS total
			FROM entries e
			LEFT JOIN base ON true
			WHERE e.account_id = $1 AND e.kind = 'transaction'
				AND e.deleted_at IS NULL AND e.booking_timestamp <= $2
				AND (
					base.id IS NULL
					OR (e.booking_timestamp, e.id) > (base.booking_timestamp, base.id)
				)
		)
		SELECT COALESCE((SELECT amount FROM base), 0) + (SELECT total FROM txns)`,
		accountID, asOf,
	).Scan(&balance)
	return balance, err
}

func isCheckViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514"
}
