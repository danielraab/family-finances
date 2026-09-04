package settings

import (
	"context"
	"errors"
)

// ErrInvalidValue: a field in an update failed validation. internal/httpapi
// maps it to a status code (see registerErrStatus).
var ErrInvalidValue = errors.New("invalid settings value")

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrInvalidValue}

// Store is the persistence contract settings declares. internal/storage/memory
// and internal/storage/postgres implement it; package main injects one.
type Store interface {
	// Get returns the raw stored row for userID. A user with no row yields
	// the zero Row (every field nil), not an error.
	Get(ctx context.Context, userID string) (Row, error)
	// Upsert merges upd into the user's row — only its non-nil fields
	// change — creating the row if it does not exist yet, and returns the
	// resulting raw row.
	Upsert(ctx context.Context, userID string, upd Update) (Row, error)
}
