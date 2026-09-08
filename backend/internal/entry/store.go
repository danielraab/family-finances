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
	// Usable reports whether categoryID exists, is owned by ownerID, and
	// is not disabled. Consulted only when a category is being newly set
	// on an entry (creation, or an update that explicitly touches
	// category_id) — never for a value merely carried over unchanged, so
	// disabling a category never disturbs entries already referencing it.
	Usable(ctx context.Context, ownerID, categoryID string) (bool, error)
	// Subtree returns categoryID and every descendant id within ownerID's
	// own tree, used to resolve a category filter to "this category or
	// any of its descendants."
	Subtree(ctx context.Context, ownerID, categoryID string) ([]string, error)
}

// TagLookup is the narrow view of internal/tag that entry needs.
// *tag.Service satisfies this structurally.
type TagLookup interface {
	// OwnedBy reports whether every id in tagIDs exists and belongs to
	// owner.
	OwnedBy(ctx context.Context, owner string, tagIDs []string) (bool, error)
	// Usable reports whether every id in tagIDs exists, belongs to owner,
	// and is not disabled. Consulted only for tag ids newly appearing on an
	// entry (creation, or the ids an update's tag_ids adds beyond what the
	// entry already carried) — never for a tag merely carried over
	// unchanged, so disabling a tag never disturbs entries already
	// referencing it.
	Usable(ctx context.Context, owner string, tagIDs []string) (bool, error)
}

// TimezoneLookup resolves an authenticated caller's resolved timezone
// setting (default "UTC" when unset), used to bucket
// GET /api/entries/flow-summary by the caller's own calendar rather than
// UTC unconditionally. internal/settings' Service satisfies this
// structurally via its own Timezone method; wiring it via
// WithTimezoneLookup is optional (a nil lookup means every caller buckets
// in UTC) — the same optional-dependency shape internal/auth's
// LanguageLookup uses.
type TimezoneLookup interface {
	Timezone(ctx context.Context, ownerID string) (string, error)
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

	// Balance computes accountID's balance as of asOf: the sum of every
	// non-deleted entry's Amount at or before asOf. A balance adjustment's
	// Amount is itself always kept, by the recompute algorithm every
	// Create/Update/SoftDelete runs (see design.md), equal to its Balance
	// reading minus the balance strictly before it — so this plain sum
	// reproduces exactly the same result as always resetting to the latest
	// balance adjustment and summing only the transactions after it.
	Balance(ctx context.Context, accountID string, asOf time.Time) (int64, error)

	// Sum computes, for ownerID's entries matching filter (already resolved
	// by Service exactly as List's is — AccountIDs and CategoryIDs are the
	// effective sets to filter by) and restricted to Kind ==
	// KindTransaction regardless of filter.Kind, the total amount per
	// account id, plus the total number of matching entries across every
	// account. Service.Sum groups the per-account totals by currency —
	// Store has no notion of an account's currency.
	Sum(ctx context.Context, ownerID string, filter Filter) (perAccount map[string]int64, count int, err error)

	// FlowSummary buckets ownerID's entries matching filter (already
	// resolved by Service — AccountIDs is the effective set to filter by,
	// Timezone is already resolved) by booking_timestamp in filter.Timezone,
	// one row per (account, period) combination that has at least one
	// matching entry. Service.FlowSummary fills in periods with no matching
	// entries and resolves each account id to its currency.
	FlowSummary(ctx context.Context, ownerID string, filter FlowFilter) ([]FlowRow, error)
}
