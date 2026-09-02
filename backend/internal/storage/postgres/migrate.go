package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// migration is one embedded .sql file, identified by the numeric prefix of its
// filename (e.g. "0001" for "0001_init.sql").
type migration struct {
	version string
	name    string
	body    string
}

// Migrate applies every embedded migration not yet recorded in
// schema_migrations, in ascending version order, each inside its own
// transaction. It is idempotent: a database already at the latest version is
// left untouched. Any failure aborts before committing the offending
// migration, leaving schema_migrations without a row for it.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	return runMigrations(ctx, pool, migrations)
}

// runMigrations is the engine behind Migrate, split out so tests can drive it
// with a hand-built migration set (including a deliberately broken one).
func runMigrations(ctx context.Context, pool *pgxpool.Pool, migrations []migration) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		if err := applyOne(ctx, pool, m); err != nil {
			return fmt.Errorf("migration %s (%s): %w", m.version, m.name, err)
		}
	}
	return nil
}

// applyOne runs a single migration and records it, atomically.
func applyOne(ctx context.Context, pool *pgxpool.Pool, m migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, m.body); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, m.version,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

// loadMigrations reads and validates the embedded migration files, returning
// them sorted by version. It rejects a filename without an "NNNN_name.sql"
// shape and any two files that share a version.
func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, fmt.Errorf("read embedded migrations: %w", err)
	}

	var migrations []migration
	seen := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		version, ok := parseVersion(e.Name())
		if !ok {
			return nil, fmt.Errorf("migration %q: name must be NNNN_description.sql", e.Name())
		}
		if other, dup := seen[version]; dup {
			return nil, fmt.Errorf("migrations %q and %q share version %s", other, e.Name(), version)
		}
		seen[version] = e.Name()

		body, err := fs.ReadFile(migrationFS, "migrations/"+e.Name())
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, migration{
			version: version,
			name:    e.Name(),
			body:    string(body),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	return migrations, nil
}

// parseVersion extracts the digit prefix before the first underscore, e.g.
// "0001" from "0001_init.sql". It requires at least one leading digit and an
// underscore separator.
func parseVersion(filename string) (string, bool) {
	i := strings.IndexByte(filename, '_')
	if i <= 0 {
		return "", false
	}
	prefix := filename[:i]
	for _, r := range prefix {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return prefix, true
}
