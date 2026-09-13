package recurringtransaction

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes in
// one place; domain and service code never mentions net/http.
var (
	// ErrNotFound: no such recurring transaction (or it is soft-deleted, or
	// its account is soft-deleted or the caller has no permission on it at
	// all — all behave identically to nonexistent).
	ErrNotFound = errors.New("not found")
	// ErrInvalidValue: a field failed validation, including a category or
	// tag that does not exist or is not usable by the caller, or an
	// account_id the caller does not have at least append permission on.
	ErrInvalidValue = errors.New("invalid value")
	// ErrAccountDisabled: the target account has disabled = true; only
	// creation (or moving to a new account) is rejected by it.
	ErrAccountDisabled = errors.New("account is disabled")
	// ErrForbidden: the caller has some permission on the account but not
	// enough for the attempted operation.
	ErrForbidden = errors.New("forbidden")
	// ErrInUse: the recurring transaction has one or more non-deleted
	// linked entries and cannot be deleted until they are unlinked.
	ErrInUse = errors.New("in use")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue, ErrAccountDisabled, ErrForbidden, ErrInUse}

// AccountLookup is the narrow view of internal/account that
// recurringtransaction needs — identical in shape to internal/entry's own
// AccountLookup. *account.Service satisfies both structurally.
type AccountLookup interface {
	Access(ctx context.Context, accountID, callerID string) (currency string, disabled bool, permission string, err error)
	VisibleIDs(ctx context.Context, callerID string) ([]string, error)
}

// CategoryLookup is the narrow view of internal/category that
// recurringtransaction needs. *category.Service satisfies this
// structurally.
type CategoryLookup interface {
	// Usable reports whether categoryID exists, is usable by callerID
	// (owned, or shared at append tier), and is not disabled.
	Usable(ctx context.Context, callerID, categoryID string) (bool, error)
}

// TagLookup is the narrow view of internal/tag that recurringtransaction
// needs — identical in shape to internal/entry's own TagLookup.
// *tag.Service satisfies this structurally.
type TagLookup interface {
	OwnedBy(ctx context.Context, callerID string, tagIDs []string) (bool, error)
	Usable(ctx context.Context, callerID string, tagIDs []string) (bool, error)
}

// EntryLookup is the narrow, read-only view of internal/entry that
// recurringtransaction needs: resolving the latest linked entry's booking
// date (for NextSuggestedDate) and whether any linked entry exists at all
// (for the delete-blocked-while-linked rule). *entry.Service satisfies this
// structurally — recurringtransaction depends on entry, never the other
// way around (see design.md's package-boundaries decision).
type EntryLookup interface {
	// LatestLinkedBookingTime returns the highest booking_timestamp among
	// recurringTransactionID's non-deleted linked entries, or nil if it has
	// none.
	LatestLinkedBookingTime(ctx context.Context, recurringTransactionID string) (*time.Time, error)
	// LinkedCount returns the number of recurringTransactionID's
	// non-deleted linked entries.
	LinkedCount(ctx context.Context, recurringTransactionID string) (int, error)
}

// TimezoneLookup resolves an authenticated caller's resolved timezone
// setting (default "UTC" when unset) — used to decide "today" for the
// Ended computation and Summary's ended-exclusion, the same optional
// dependency shape internal/entry's own TimezoneLookup uses.
type TimezoneLookup interface {
	Timezone(ctx context.Context, ownerID string) (string, error)
}

// Store is the persistence contract recurringtransaction declares.
// internal/storage/memory and internal/storage/postgres implement it;
// package main injects one. Every method is deliberately unscoped by
// caller — Service resolves the caller's permission via
// AccountLookup.Access before calling any of these, mirroring
// internal/entry's Store.
type Store interface {
	Create(ctx context.Context, createdBy string, in New) (RecurringTransaction, error)
	// Get returns the recurring transaction by id alone, excluding
	// soft-deleted rows.
	Get(ctx context.Context, id string) (RecurringTransaction, error)
	Update(ctx context.Context, id string, upd Update) (RecurringTransaction, error)
	// SoftDelete sets deleted_at. One-way — no undelete. It does not itself
	// enforce the delete-blocked-while-linked rule — Service checks that via
	// EntryLookup.LinkedCount before ever calling this.
	SoftDelete(ctx context.Context, id string) error

	// List returns every non-deleted recurring transaction matching filter
	// (already resolved by Service — AccountIDs is the effective set to
	// filter by, already narrowed to the caller's visible accounts).
	List(ctx context.Context, filter Filter) ([]RecurringTransaction, error)
}
