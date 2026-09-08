// Package entry owns entries recorded against an account — relative
// transactions and absolute balance adjustments — their relationship to
// accounts/categories/tags, live balance computation, and a
// filterable/searchable/sortable, cursor-paginated listing. It follows the
// repo's four-file shape (entry.go, store.go, service.go, handler.go). It
// imports internal/auth only for auth.UserFromContext, and declares narrow
// AccountLookup/CategoryLookup/TagLookup interfaces — satisfied structurally
// by *account.Service, *category.Service, and *tag.Service — rather than
// importing those packages' Store or a database driver (see design.md's
// package-boundaries decision).
package entry

import (
	"encoding/json"
	"strings"
	"time"
)

// Kind is whether an entry is a relative delta or an absolute balance
// reading.
type Kind string

const (
	// KindTransaction is a relative amount applied to the account's
	// running balance.
	KindTransaction Kind = "transaction"
	// KindBalanceAdjustment is an absolute amount the account's balance is
	// set to at that point in time.
	KindBalanceAdjustment Kind = "balance_adjustment"
)

func (k Kind) valid() bool {
	return k == KindTransaction || k == KindBalanceAdjustment
}

// AmountScale is the fixed number of decimal places every stored amount is
// scaled by, instance-wide — a constant, not environment-configurable (see
// design.md). E.g. an amount of 105000 at AmountScale 4 represents 10.5000
// in the account's currency.
const AmountScale = 4

// SortField is a column entry listing can be ordered by.
type SortField string

const (
	SortBookingTimestamp SortField = "booking_timestamp"
	SortAmount           SortField = "amount"
)

func (f SortField) valid() bool {
	return f == SortBookingTimestamp || f == SortAmount
}

// SortDir is the direction of an entry listing's ordering.
type SortDir string

const (
	DirAsc  SortDir = "asc"
	DirDesc SortDir = "desc"
)

func (d SortDir) valid() bool {
	return d == DirAsc || d == DirDesc
}

// CategoryMode controls how Filter.CategoryID is resolved. It has no effect
// when CategoryID is nil.
type CategoryMode string

const (
	// ModeSubtree (the default) resolves CategoryID to itself plus every
	// descendant in the category tree.
	ModeSubtree CategoryMode = "subtree"
	// ModeExact resolves CategoryID to itself alone.
	ModeExact CategoryMode = "exact"
)

func (m CategoryMode) valid() bool {
	return m == "" || m == ModeSubtree || m == ModeExact
}

// Entry is a transaction or balance adjustment recorded against exactly one
// account. It has exactly one owner (the account's owner at creation time,
// its own column — see design.md) and is visible only to them.
//
// Amount is always a signed delta applied to the account's running balance —
// for a transaction, exactly what the caller supplied; for a balance
// adjustment, computed automatically (see design.md's recompute algorithm)
// as the change from the balance immediately before it. Balance is the
// absolute reading the caller supplied for a balance adjustment (nil for a
// transaction) — it is what a balance adjustment's amount is defined
// against, never computed itself.
type Entry struct {
	ID               string     `json:"id"`
	AccountID        string     `json:"account_id"`
	Kind             Kind       `json:"kind"`
	Amount           int64      `json:"amount"`
	Balance          *int64     `json:"balance,omitempty"`
	BookingTimestamp time.Time  `json:"booking_timestamp"`
	Title            string     `json:"title"`
	Description      string     `json:"description,omitempty"`
	CategoryID       *string    `json:"category_id,omitempty"`
	TagIDs           []string   `json:"tag_ids"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	OwnerID          string     `json:"-"`
	DeletedAt        *time.Time `json:"-"`
}

// OptionalID distinguishes a JSON key that is absent (Set is false) from one
// present as either null (Set true, Value nil) or an id (Set true, Value
// set) — the same trick account.Date and category.Category use for their
// clearable optional fields. Used here for category_id, which a
// balance_adjustment entry can explicitly clear.
type OptionalID struct {
	Set   bool
	Value *string
}

func (o *OptionalID) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	o.Value = &s
	return nil
}

// New is the input to creating an entry. Exactly one of Amount (for
// KindTransaction) or Balance (for KindBalanceAdjustment) SHALL be set — see
// validateNew. For a balance adjustment, Amount is never client-supplied:
// the store computes and stores it as part of Create (see design.md).
type New struct {
	AccountID        string
	Kind             Kind
	Amount           *int64
	Balance          *int64
	BookingTimestamp time.Time
	Title            string
	Description      string
	CategoryID       *string
	TagIDs           []string
}

// Update is a partial change to an entry. Kind is deliberately absent — it
// is immutable after creation (see design.md); the handler's request body
// has no field for it either, so DisallowUnknownFields rejects an attempt
// to set it. A nil field here leaves it untouched; TagIDs replaces the full
// set when non-nil (including an empty, non-nil slice, which clears every
// tag). A non-nil AccountID moves the entry to a different account, subject
// to the same ownership/disabled-account checks Create applies. Amount is
// only settable on a transaction, Balance only on a balance adjustment (see
// design.md) — Service.Update rejects the other one being non-nil for the
// entry's (immutable) kind.
type Update struct {
	AccountID        *string
	Amount           *int64
	Balance          *int64
	BookingTimestamp *time.Time
	Title            *string
	Description      *string
	CategoryID       OptionalID
	TagIDs           *[]string
}

// Cursor is the keyset position of the last row of a previous page: the
// sorted column's value plus id, for a stable tie-break — see design.md.
type Cursor struct {
	BookingTimestamp time.Time
	Amount           int64
	ID               string
}

// Filter narrows and orders a List call. CategoryID is the caller-supplied
// filter value; Service.List resolves it (including descendants) into
// CategoryIDs before the Store sees it — Store implementations read only
// CategoryIDs. Likewise AccountIDs is always resolved by Service.List to the
// caller's own visible accounts (optionally narrowed further by the
// caller-supplied AccountIDs) before reaching Store.
type Filter struct {
	AccountIDs   []string
	CategoryID   *string
	CategoryMode CategoryMode
	CategoryIDs  []string
	TagID        *string
	Kind         *Kind
	From         *time.Time
	To           *time.Time
	Query        string
	Sort         SortField
	Dir          SortDir
	After        *Cursor
	Limit        int
}

// CurrencySum is one currency's total within a Summary.
type CurrencySum struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

// Summary is the result of summing a Filter's matching transaction entries,
// grouped by their account's currency — see Service.Sum.
type Summary struct {
	Sums  []CurrencySum `json:"sums"`
	Count int           `json:"count"`
}

// FlowUnit is the bucket granularity FlowSummary groups entries by.
type FlowUnit string

const (
	FlowUnitMonth FlowUnit = "month"
	FlowUnitDay   FlowUnit = "day"
)

func (u FlowUnit) valid() bool { return u == FlowUnitMonth || u == FlowUnitDay }

// FlowFilter narrows and buckets a FlowSummary call. AccountIDs is resolved
// by Service.FlowSummary to the caller's own visible accounts the same way
// List/Sum resolve theirs, before Store sees it. Timezone is never
// caller-supplied — Service.FlowSummary fills it in from the caller's
// resolved settings (default UTC) before calling Store, so bucket
// boundaries always reflect the viewer's own calendar, not UTC
// unconditionally.
type FlowFilter struct {
	AccountIDs []string
	Unit       FlowUnit
	Year       int
	Month      int // 1-12; required when Unit == FlowUnitDay, must be 0 otherwise
	Timezone   string
}

// FlowRow is one (account, period)'s income/outcome totals, as Store
// computes them — Service.FlowSummary resolves each account's currency and
// merges same-currency, same-period rows into a FlowBucket, since Store has
// no notion of an account's currency (mirrors Sum/Service.Sum). Period is
// the bucket's first calendar day, "YYYY-MM-DD", regardless of Unit. Income
// is the sum of positive amounts, Outcome the sum of |negative amounts| —
// both non-negative.
type FlowRow struct {
	AccountID string
	Period    string
	Income    int64
	Outcome   int64
}

// FlowBucket is one period's income/outcome totals, grouped per currency —
// see Service.FlowSummary. A period with no matching entries at all still
// appears, with empty Income/Outcome.
type FlowBucket struct {
	Period  string        `json:"period"`
	Income  []CurrencySum `json:"income"`
	Outcome []CurrencySum `json:"outcome"`
}

const (
	defaultPageSize = 50
	maxPageSize     = 200
)

func validateNew(in New) error {
	if strings.TrimSpace(in.AccountID) == "" {
		return ErrInvalidValue
	}
	if !in.Kind.valid() {
		return ErrInvalidValue
	}
	if strings.TrimSpace(in.Title) == "" {
		return ErrInvalidValue
	}
	if in.BookingTimestamp.IsZero() {
		return ErrInvalidValue
	}
	if in.Kind == KindTransaction && in.CategoryID == nil {
		return ErrInvalidValue
	}
	if in.Kind == KindTransaction {
		if in.Amount == nil || in.Balance != nil {
			return ErrInvalidValue
		}
	} else {
		if in.Balance == nil || in.Amount != nil {
			return ErrInvalidValue
		}
	}
	return nil
}
