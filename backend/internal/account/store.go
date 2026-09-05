package account

import (
	"context"
	"errors"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes in
// one place; domain and service code never mentions net/http.
var (
	// ErrNotFound: no such account or account type (or it belongs to a
	// different owner, or is soft-deleted — those behave identically to
	// nonexistent, see design.md).
	ErrNotFound = errors.New("not found")
	// ErrInvalidValue: a field failed validation.
	ErrInvalidValue = errors.New("invalid value")
	// ErrTypeInUse: an account type cannot be deleted because a non-deleted
	// account still references it.
	ErrTypeInUse = errors.New("account type is in use")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue, ErrTypeInUse}

// Store is the persistence contract account declares. internal/storage/memory
// and internal/storage/postgres implement it; package main injects one.
type Store interface {
	// --- accounts ---

	Create(ctx context.Context, ownerID string, in New) (Account, error)
	// Get returns the account, scoped to ownerID; ErrNotFound if it does
	// not exist, belongs to a different owner, or is soft-deleted.
	Get(ctx context.Context, ownerID, id string) (Account, error)
	// List returns every non-deleted account owned by ownerID.
	List(ctx context.Context, ownerID string) ([]Account, error)
	Update(ctx context.Context, ownerID, id string, upd Update) (Account, error)
	SetDisabled(ctx context.Context, ownerID, id string, disabled bool) (Account, error)
	// SoftDelete sets deleted_at. One-way — no undelete.
	SoftDelete(ctx context.Context, ownerID, id string) error

	// Owner returns id's owner, currency, and disabled flag, unscoped by
	// caller — used by internal/entry's AccountLookup, which checks the
	// returned owner against the calling user itself. ErrNotFound for a
	// missing or soft-deleted account.
	Owner(ctx context.Context, id string) (ownerID string, currency string, disabled bool, err error)

	// --- account types ---

	ListTypes(ctx context.Context) ([]Type, error)
	CreateType(ctx context.Context, name string) (Type, error)
	UpdateType(ctx context.Context, id, name string) (Type, error)
	// DeleteType returns ErrTypeInUse if a non-deleted account still
	// references it, ErrNotFound if it does not exist.
	DeleteType(ctx context.Context, id string) error
	TypeExists(ctx context.Context, id string) (bool, error)
}
