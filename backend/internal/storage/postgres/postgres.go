// Package postgres holds the PostgreSQL implementations of the Store interfaces
// that the domain packages (internal/auth, …) declare. It is the real
// persistence layer: package main builds a *pgxpool.Pool with NewPool, applies
// schema with Migrate, and injects a Store backed by that pool into each
// domain service. internal/storage/memory remains the default for unit tests.
//
// This package is intentionally free of Store implementations until the first
// domain package declares a Store — today it provides only the pool
// constructor and the migration runner.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the connection pool type, aliased so callers (package main, the CLI)
// can name it without importing the pgx module directly.
type Pool = pgxpool.Pool

// pingTimeout bounds the connectivity check NewPool performs before returning.
const pingTimeout = 5 * time.Second

// NewPool parses databaseURL, opens a pgx connection pool, and verifies
// connectivity with a bounded Ping. A non-nil error means the backend must not
// start: the DSN is invalid or no database answered.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
