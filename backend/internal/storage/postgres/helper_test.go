package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"
	"time"

	"at.draab/familyfinances/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newTestPool provisions an isolated, freshly-created database for a single
// test and returns a pool connected to it. The test is skipped when
// DATABASE_URL is unset, so `go test ./...` still passes on a bare checkout.
// The throwaway database is dropped on test cleanup.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	adminURL := cfg.DatabaseURL
	if adminURL == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres integration test")
	}

	ctx := context.Background()

	adminCfg, err := pgxpool.ParseConfig(adminURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL: %v", err)
	}
	admin, err := pgxpool.NewWithConfig(ctx, adminCfg)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}
	defer admin.Close()

	// dbName uses only [a-z0-9_] so it needs no identifier quoting.
	dbName := "ff_test_" + randSuffix(t)
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+dbName); err != nil {
		t.Fatalf("create test database %s: %v", dbName, err)
	}

	testCfg, err := pgxpool.ParseConfig(adminURL)
	if err != nil {
		t.Fatalf("parse DATABASE_URL (test): %v", err)
	}
	testCfg.ConnConfig.Database = dbName
	pool, err := pgxpool.NewWithConfig(ctx, testCfg)
	if err != nil {
		t.Fatalf("connect test pool: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()

		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		down, err := pgxpool.NewWithConfig(cleanupCtx, adminCfg)
		if err != nil {
			t.Logf("cleanup: reconnect admin: %v", err)
			return
		}
		defer down.Close()
		if _, err := down.Exec(cleanupCtx, "DROP DATABASE IF EXISTS "+dbName+" WITH (FORCE)"); err != nil {
			t.Logf("cleanup: drop database %s: %v", dbName, err)
		}
	})

	return pool
}

func randSuffix(t *testing.T) string {
	t.Helper()
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return hex.EncodeToString(b[:])
}
