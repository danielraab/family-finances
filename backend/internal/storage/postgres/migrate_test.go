package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrateFreshDatabase(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	want, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}
	if len(want) == 0 {
		t.Fatal("no embedded migrations found")
	}

	got := migrationVersions(t, ctx, pool)
	if len(got) != len(want) {
		t.Fatalf("schema_migrations has %d rows, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i].version {
			t.Errorf("row %d = %q, want %q (files must apply in filename order)", i, got[i], want[i].version)
		}
	}
}

func TestMigrate0002AuthSchema(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// 0002 builds on 0001: gen_random_uuid() and citext (from 0001) must both
	// be usable, which they are only if 0002 applied cleanly on top.
	for _, table := range []string{
		"users", "identities", "sessions", "magic_link_tokens", "invites", "oidc_login_state",
	} {
		if !tableExists(t, ctx, pool, table) {
			t.Errorf("table %q missing after Migrate", table)
		}
	}

	// A second Migrate must not choke on the already-applied 0002.
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM schema_migrations WHERE version = '0002'`,
	).Scan(&n); err != nil {
		t.Fatalf("count 0002 rows: %v", err)
	}
	if n != 1 {
		t.Fatalf("schema_migrations has %d rows for 0002, want 1", n)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	stamp1 := appliedStamps(t, ctx, pool)

	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	stamp2 := appliedStamps(t, ctx, pool)

	if stamp1 != stamp2 {
		t.Fatalf("second Migrate re-applied migrations: applied_at changed\n first: %s\nsecond: %s", stamp1, stamp2)
	}
}

func TestMigrateBrokenMigrationIsAtomic(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	migs := []migration{
		{version: "0001", name: "0001_good.sql", body: `CREATE TABLE good_t (id int);`},
		{version: "0002", name: "0002_bad.sql", body: `CREATE TABLE bad_t (id int); SELECT no_such_function();`},
	}

	if err := runMigrations(ctx, pool, migs); err == nil {
		t.Fatal("runMigrations succeeded on a broken migration; want error")
	}

	got := migrationVersions(t, ctx, pool)
	if len(got) != 1 || got[0] != "0001" {
		t.Fatalf("schema_migrations = %v, want [0001] only", got)
	}
	if !tableExists(t, ctx, pool, "good_t") {
		t.Error("good_t missing; 0001 should have committed")
	}
	if tableExists(t, ctx, pool, "bad_t") {
		t.Error("bad_t present; the failed 0002 transaction should have rolled back")
	}
}

func TestNewPoolUnreachable(t *testing.T) {
	// No DATABASE_URL needed: port 1 refuses the connection immediately.
	_, err := NewPool(context.Background(),
		"postgres://user:pass@127.0.0.1:1/none?sslmode=disable&connect_timeout=2")
	if err == nil {
		t.Fatal("NewPool returned nil error for an unreachable database")
	}
}

func migrationVersions(t *testing.T, ctx context.Context, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	return out
}

func appliedStamps(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(string_agg(applied_at::text, ',' ORDER BY version), '') FROM schema_migrations`,
	).Scan(&s); err != nil {
		t.Fatalf("read applied_at: %v", err)
	}
	return s
}

func tableExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, name).Scan(&exists); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return exists
}
