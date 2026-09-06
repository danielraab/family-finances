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
	// ErrTypeDisabled: a create/update tried to (re)assign a disabled
	// account type — well-formed request, rejected by a business rule,
	// distinct from ErrInvalidValue (malformed/missing field).
	ErrTypeDisabled = errors.New("account type is disabled")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue, ErrTypeInUse, ErrTypeDisabled}

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

	// ListTypes returns every type owned by ownerID.
	ListTypes(ctx context.Context, ownerID string) ([]Type, error)
	// GetType returns ErrNotFound if id does not exist or belongs to a
	// different owner. Used both to resolve a type for display and to
	// check whether a type_id may be (re)assigned (see
	// Service.resolveAssignableType).
	GetType(ctx context.Context, ownerID, id string) (Type, error)
	CreateType(ctx context.Context, ownerID, title, description string) (Type, error)
	UpdateType(ctx context.Context, ownerID, id, title, description string) (Type, error)
	// SetTypeDisabled toggles whether a type may be newly (re)assigned.
	// It does not affect any account already carrying it.
	SetTypeDisabled(ctx context.Context, ownerID, id string, disabled bool) (Type, error)
	// DeleteType returns ErrTypeInUse if a non-deleted account still
	// references it, ErrNotFound if it does not exist or belongs to a
	// different owner.
	DeleteType(ctx context.Context, ownerID, id string) error
	// SeedDefaultTypes inserts DefaultTypeTitles for a brand-new ownerID.
	// Called unconditionally — there is nothing to collide with for a
	// fresh owner.
	SeedDefaultTypes(ctx context.Context, ownerID string) error
}
