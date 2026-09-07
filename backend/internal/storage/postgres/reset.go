package postgres

import (
	"context"
	"fmt"
	"strings"
)

// TableRowCount is one table's name and current row count — used by the
// `seed` CLI's bare (no --yes) mode to show what a reset would wipe before
// anything is touched.
type TableRowCount struct {
	Table string
	Rows  int64
}

// dataTables returns every table name in the public schema except
// schema_migrations — read at runtime rather than hardcoded, so a future
// migration that adds a table is picked up automatically. schema_migrations
// is never part of a data reset: this is a data reset, not a request to
// re-run migrations from an empty tracking table.
func dataTables(ctx context.Context, pool *Pool) ([]string, error) {
	rows, err := pool.Query(ctx,
		`SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'schema_migrations' ORDER BY tablename`,
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

// TableCounts reports every data table's current row count.
func TableCounts(ctx context.Context, pool *Pool) ([]TableRowCount, error) {
	names, err := dataTables(ctx, pool)
	if err != nil {
		return nil, err
	}
	out := make([]TableRowCount, 0, len(names))
	for _, name := range names {
		var n int64
		if err := pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %q", name)).Scan(&n); err != nil {
			return nil, err
		}
		out = append(out, TableRowCount{Table: name, Rows: n})
	}
	return out, nil
}

// ResetAll truncates every data table (every table except
// schema_migrations) in one statement, resetting the database to empty
// without touching migration history. Used only by the `seed` CLI's --yes
// path — never reachable over HTTP.
func ResetAll(ctx context.Context, pool *Pool) error {
	names, err := dataTables(ctx, pool)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return nil
	}
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = fmt.Sprintf("%q", n)
	}
	_, err = pool.Exec(ctx, "TRUNCATE "+strings.Join(quoted, ", ")+" RESTART IDENTITY CASCADE")
	return err
}
