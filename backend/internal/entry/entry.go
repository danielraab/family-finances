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

// Entry is a transaction or balance adjustment recorded against exactly one
// account. It has exactly one owner (the account's owner at creation time,
// its own column — see design.md) and is visible only to them.
type Entry struct {
	ID               string     `json:"id"`
	AccountID        string     `json:"account_id"`
	Kind             Kind       `json:"kind"`
	Amount           int64      `json:"amount"`
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

// New is the input to creating an entry.
type New struct {
	AccountID        string
	Kind             Kind
	Amount           int64
	BookingTimestamp time.Time
	Title            string
	Description      string
	CategoryID       *string
	TagIDs           []string
}

// Update is a partial change to an entry. AccountID and Kind are
// deliberately absent — they are immutable after creation (see design.md);
// the handler's request body has no fields for them either, so
// DisallowUnknownFields rejects an attempt to set them. A nil field here
// leaves it untouched; TagIDs replaces the full set when non-nil (including
// an empty, non-nil slice, which clears every tag).
type Update struct {
	Amount           *int64
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
	AccountIDs  []string
	CategoryID  *string
	CategoryIDs []string
	TagID       *string
	Kind        *Kind
	From        *time.Time
	To          *time.Time
	Query       string
	Sort        SortField
	Dir         SortDir
	After       *Cursor
	Limit       int
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
	return nil
}
