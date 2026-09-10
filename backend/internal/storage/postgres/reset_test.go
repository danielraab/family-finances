package postgres

import (
	"context"
	"testing"
)

func TestResetAllTruncatesEveryDataTable(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	authStore := NewAuthStore(pool)
	owner := mustUser(t, authStore, "reset-check@example.com")
	accStore := NewAccountStore(pool)
	mustAccount(t, accStore, owner.ID, "Checking", "Checking")

	before, err := TableCounts(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	var totalBefore int64
	for _, c := range before {
		totalBefore += c.Rows
	}
	if totalBefore == 0 {
		t.Fatal("expected some rows before ResetAll — fixture setup didn't take")
	}

	if err := ResetAll(ctx, pool); err != nil {
		t.Fatalf("ResetAll: %v", err)
	}

	after, err := TableCounts(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	if len(after) == 0 {
		t.Fatal("TableCounts returned no tables — did dataTables() break?")
	}
	for _, c := range after {
		if c.Rows != 0 {
			t.Fatalf("table %s has %d rows after ResetAll, want 0", c.Table, c.Rows)
		}
	}
}

func TestTableCountsExcludesSchemaMigrations(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	counts, err := TableCounts(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range counts {
		if c.Table == "schema_migrations" {
			t.Fatalf("TableCounts included schema_migrations, want it excluded")
		}
	}
}
