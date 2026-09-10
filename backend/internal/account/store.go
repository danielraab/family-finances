package account

import (
	"context"
	"errors"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes in
// one place; domain and service code never mentions net/http.
var (
	// ErrNotFound: no such account (or it belongs to a different owner, or
	// is soft-deleted — those behave identically to nonexistent, see
	// design.md), or the caller has no permission on it at all (real
	// ownership or any share).
	ErrNotFound = errors.New("not found")
	// ErrInvalidValue: a field failed validation.
	ErrInvalidValue = errors.New("invalid value")
	// ErrForbidden: the caller has some permission on the account but not
	// enough for the attempted operation (e.g. view/append/entry_admin
	// attempting an owner-tier action). Distinct from ErrNotFound, which is
	// for no permission at all.
	ErrForbidden = errors.New("forbidden")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue, ErrForbidden}

// Store is the persistence contract account declares. internal/storage/memory
// and internal/storage/postgres implement it; package main injects one.
type Store interface {
	// --- accounts ---

	// Create creates an account owned by ownerID — the real owner, always
	// the caller (accounts are never created pre-shared).
	Create(ctx context.Context, ownerID string, in New) (Account, error)
	// Get returns the account by id, annotated with callerID's Permission/
	// Shared/OwnerName (Permission is "" when callerID has no access at
	// all — not itself an error; Service.Get is the one place that's
	// translated into ErrNotFound, see design.md). ErrNotFound only when
	// id names no account at all, or a soft-deleted one.
	Get(ctx context.Context, id, callerID string) (Account, error)
	// List returns every non-deleted account callerID owns or has any
	// share on.
	List(ctx context.Context, callerID string) ([]Account, error)
	// Update applies upd to id, unscoped by caller — the Service has
	// already authorized the caller via Access before calling this.
	Update(ctx context.Context, id string, upd Update) (Account, error)
	SetDisabled(ctx context.Context, id string, disabled bool) (Account, error)
	// SoftDelete sets deleted_at. One-way — no undelete.
	SoftDelete(ctx context.Context, id string) error

	// Access resolves what callerID may do on id: PermissionOwner when
	// callerID is the real owner, else a matching AccountShare's
	// permission, else an empty Permission (no access at all — the
	// caller-visible convention is to then treat the account as
	// ErrNotFound, never to return ErrNotFound from Access itself, except
	// when id names no account — or a soft-deleted one — at all).
	Access(ctx context.Context, id, callerID string) (Access, error)
	// VisibleIDs returns the id of every non-deleted account callerID
	// owns or has any share on. Satisfies internal/entry's AccountLookup
	// interface, used to scope an entry listing or balance query.
	VisibleIDs(ctx context.Context, callerID string) ([]string, error)

	// --- sharing ---

	// CreateOrUpdateShare grants userID permission on accountID, recording
	// grantedBy. An existing share for (accountID, userID) has its
	// permission overwritten in place rather than duplicating a row — see
	// design.md.
	CreateOrUpdateShare(ctx context.Context, accountID, userID string, permission Permission, grantedBy string) (AccountShare, error)
	// ListShares returns every current share on accountID — not including
	// the real owner, who carries no share row (Service composes that
	// separately for display).
	ListShares(ctx context.Context, accountID string) ([]AccountShare, error)
	// ShareByUser returns userID's own share on accountID, or ErrNotFound
	// if they have none (including when they are the real owner, who has
	// no share row).
	ShareByUser(ctx context.Context, accountID, userID string) (AccountShare, error)
	// UpdateSharePermission changes an existing share's permission.
	// ErrNotFound if no share exists for (accountID, userID).
	UpdateSharePermission(ctx context.Context, accountID, userID string, permission Permission) (AccountShare, error)
	// DeleteShare removes a share unconditionally (no soft delete — see
	// design.md). ErrNotFound if none exists.
	DeleteShare(ctx context.Context, accountID, userID string) error

	// --- account types ---

	// ListInUseTypes returns the distinct, non-empty type labels present
	// on ownerID's own non-deleted accounts, sorted case-insensitively
	// ascending. For client autocomplete only — a type has no existence
	// apart from being written on an account.
	ListInUseTypes(ctx context.Context, ownerID string) ([]string, error)
}
