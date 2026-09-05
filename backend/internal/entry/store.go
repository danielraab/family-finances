package entry

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes in
// one place; domain and service code never mentions net/http.
var (
	// ErrNotFound: no such entry (or it belongs to a different owner, or is
	// soft-deleted, or its account is soft-deleted — all behave identically
	// to nonexistent).
	ErrNotFound = errors.New("not found")
	// ErrInvalidValue: a field failed validation, including a category or
	// tag that does not exist or is not the caller's, or an account_id the
	// caller does not own.
	ErrInvalidValue = errors.New("invalid value")
	// ErrAccountDisabled: the target account has disabled = true; only
	// entry creation is rejected by it (see accounts and design.md).
	ErrAccountDisabled = errors.New("account is disabled")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue, ErrAccountDisabled}

// AccountLookup is the narrow view of internal/account that entry needs:
// confirming the caller owns the target account (and reading its currency
// and disabled flag), and resolving which of the caller's accounts are
// currently visible (non-deleted) to scope a listing or balance query.
// *account.Service satisfies this structurally.
type AccountLookup interface {
	// Owner returns accountID's owner, currency, and disabled flag, or
	// ErrNotFound (also returned for a soft-deleted account).
	Owner(ctx context.Context, accountID string) (ownerID string, currency string, disabled bool, err error)
	// VisibleIDs returns every non-deleted account id ownerID owns.
	VisibleIDs(ctx context.Context, ownerID string) ([]string, error)
}

// CategoryLookup is the narrow view of internal/category that entry needs.
// *category.Service satisfies this structurally.
type CategoryLookup interface {
	Exists(ctx context.Context, categoryID string) (bool, error)
	// Subtree returns categoryID and every descendant id, used to resolve
	// a category filter to "this category or any of its descendants."
	Subtree(ctx context.Context, categoryID string) ([]string, error)
}

// TagLookup is the narrow view of internal/tag that entry needs.
// *tag.Service satisfies this structurally.
type TagLookup interface {
	// OwnedBy reports whether every id in tagIDs exists and belongs to
	// owner.
	OwnedBy(ctx context.Context, owner string, tagIDs []string) (bool, error)
}

// Store is the persistence contract entry declares. internal/storage/memory
// and internal/storage/postgres implement it; package main injects one.
type Store interface {
	Create(ctx context.Context, ownerID string, in New) (Entry, error)
	// Get returns the entry, scoped to ownerID; ErrNotFound if it does not
	// exist, belongs to a different owner, or is soft-deleted.
	Get(ctx context.Context, ownerID, id string) (Entry, error)
	Update(ctx context.Context, ownerID, id string, upd Update) (Entry, error)
	// SoftDelete sets deleted_at. One-way — no undelete.
	SoftDelete(ctx context.Context, ownerID, id string) error

	// List returns a page of ownerID's entries matching filter (already
	// resolved by Service — AccountIDs and CategoryIDs are the effective
	// sets to filter by), plus the cursor for the next page, or nil once
	// there are no more.
	List(ctx context.Context, ownerID string, filter Filter) ([]Entry, *Cursor, error)

	// Balance computes accountID's balance as of asOf per design.md's live
	// computation: the latest non-deleted balance_adjustment at or before
	// asOf (or 0 if none), plus every non-deleted transaction after it, up
	// to asOf.
	Balance(ctx context.Context, accountID string, asOf time.Time) (int64, error)
}
