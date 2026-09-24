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
	// KindSelfTransfer is a relative amount moved from AccountID to
	// ToAccountID — one entry, two accounts. See Entry's doc comment.
	KindSelfTransfer Kind = "self_transfer"
)

func (k Kind) valid() bool {
	return k == KindTransaction || k == KindBalanceAdjustment || k == KindSelfTransfer
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

// OriginalAccountRole is the caller's choice, on
// POST /api/entries/{id}/self-transfer, of which role the entry being
// converted's own account plays in the resulting self-transfer — see
// Service.ConvertToSelfTransfer. It is never inferred from the entry's
// existing amount sign, since a self_transfer's amount is valid either sign
// on account_id (see design.md of add-self-transfer-conversion).
type OriginalAccountRole string

const (
	// RoleSender: the entry's own account keeps the sender role
	// (account_id, amount unchanged); the caller-chosen counterparty
	// account becomes to_account_id.
	RoleSender OriginalAccountRole = "sender"
	// RoleReceiver: the entry's own account becomes the receiver
	// (to_account_id); the caller-chosen counterparty account becomes
	// account_id, and the amount is negated so the entry's own account's
	// real economic effect is unchanged either way.
	RoleReceiver OriginalAccountRole = "receiver"
)

func (r OriginalAccountRole) valid() bool {
	return r == RoleSender || r == RoleReceiver
}

// Entry is a transaction or balance adjustment recorded against exactly one
// account. CreatedBy is the user who logged it — not necessarily the
// account's real owner, once account-sharing lets any permitted user
// create one — and is immutable after creation. Visibility is scoped to
// every user holding any permission on the entry's parent account (see
// account-sharing), not to CreatedBy; CreatedBy matters only for the
// append tier's "edit only what I created" rule — see design.md.
//
// Amount is always a signed delta applied to the account's running balance —
// for a transaction, exactly what the caller supplied; for a balance
// adjustment, computed automatically (see design.md's recompute algorithm)
// as the change from the balance immediately before it. Balance is the
// absolute reading the caller supplied for a balance adjustment (nil for a
// transaction) — it is what a balance adjustment's amount is defined
// against, never computed itself.
//
// CreatedByName is resolved server-side (never by the client joining
// against a shares list) so a viewer can see who logged an entry across a
// cursor-paginated, filtered list with no second request — see design.md.
//
// AccountCurrency is resolved server-side the same way, regardless of the
// caller's account-level access to AccountID — so a client can always
// render Amount correctly, including for an entry surfaced only via a
// category-permission filter with no account access at all (see
// category-sharing). Like CreatedByName, it's omitempty: the memory Store
// (domain/handler unit tests only, never production) never populates it —
// see design.md.
//
// Counterparty and Location are free-text, transaction-only fields (see
// validateNew) — rejected on a balance_adjustment or self_transfer.
// Location is never parsed or validated here beyond that: it may hold a
// typed address or a JSON-encoded {"lat":…,"lng":…} coordinate string, and
// deciding which is a frontend concern (see design.md of
// add-entry-counterparty-location).
//
// ToAccountID is set only for a self_transfer — the receiving account,
// required exactly then (see validateNew) and immutable after creation,
// like AccountID/Kind for that kind (see design.md of add-self-transfer).
// Amount stays signed from AccountID's perspective; the receiving side's
// effective delta is -Amount (see Store.Balance and the entry_legs view in
// storage/postgres). ToAccountName/ToAccountCurrency are resolved
// server-side unconditionally, the same reasoning AccountCurrency/
// CreatedByName already follow — a caller who can only see AccountID still
// needs to know where the money went and in what currency.
type Entry struct {
	ID                string  `json:"id"`
	AccountID         string  `json:"account_id"`
	AccountCurrency   string  `json:"account_currency,omitempty"`
	ToAccountID       *string `json:"to_account_id,omitempty"`
	ToAccountName     string  `json:"to_account_name,omitempty"`
	ToAccountCurrency string  `json:"to_account_currency,omitempty"`
	Kind              Kind    `json:"kind"`
	Amount            int64   `json:"amount"`
	// AfterBalance is the account's running balance immediately after this
	// entry's leg has been applied. For a self-transfer it is expressed from
	// the listed AccountID's perspective, just like Amount.
	AfterBalance     int64     `json:"after_balance"`
	Balance          *int64    `json:"balance,omitempty"`
	BookingTimestamp time.Time `json:"booking_timestamp"`
	Title            string    `json:"title"`
	Description      string    `json:"description,omitempty"`
	CategoryID       *string   `json:"category_id,omitempty"`
	Counterparty     string    `json:"counterparty,omitempty"`
	Location         string    `json:"location,omitempty"`
	TagIDs           []string  `json:"tag_ids"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	CreatedBy        string    `json:"created_by"`
	CreatedByName    string    `json:"created_by_name,omitempty"`
	// RecurringTransactionID is the recurring transaction this entry was
	// created from or has been linked to, or nil. See design.md of
	// add-recurring-transactions: settable at creation or via update
	// (RecurringTransactionLookup.SameAccount is consulted whenever it is
	// newly set, requiring it to name a recurring transaction on this
	// entry's own AccountID), and never touched by an update that doesn't
	// supply it.
	RecurringTransactionID *string    `json:"recurring_transaction_id,omitempty"`
	DeletedAt              *time.Time `json:"-"`
}

// Permission mirrors internal/account's four-tier model as plain values so
// entry.AccountLookup can be satisfied structurally by *account.Service
// without entry importing internal/account (see design.md's
// package-boundaries decision — the same reason AccountLookup.Access
// returns plain strings/bools rather than a shared struct type). Entry
// never grants or revokes a permission itself, only compares tiers.
type Permission string

const (
	PermissionView       Permission = "view"
	PermissionAppend     Permission = "append"
	PermissionEntryAdmin Permission = "entry_admin"
	PermissionOwner      Permission = "owner"
)

var permissionRank = map[Permission]int{
	PermissionView:       1,
	PermissionAppend:     2,
	PermissionEntryAdmin: 3,
	PermissionOwner:      4,
}

// AtLeast reports whether p is other or a stronger tier. An empty
// Permission (no access at all) is never AtLeast anything.
func (p Permission) AtLeast(other Permission) bool {
	pr, ok := permissionRank[p]
	if !ok {
		return false
	}
	or, ok := permissionRank[other]
	if !ok {
		return false
	}
	return pr >= or
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
// KindTransaction/KindSelfTransfer) or Balance (for KindBalanceAdjustment)
// SHALL be set — see validateNew. For a balance adjustment, Amount is never
// client-supplied: the store computes and stores it as part of Create (see
// design.md). ToAccountID is required exactly for KindSelfTransfer.
type New struct {
	AccountID        string
	ToAccountID      *string
	Kind             Kind
	Amount           *int64
	Balance          *int64
	BookingTimestamp time.Time
	Title            string
	Description      string
	CategoryID       *string
	Counterparty     string
	Location         string
	TagIDs           []string
	// RecurringTransactionID, when set, must name a recurring transaction
	// on this same AccountID — see RecurringTransactionLookup.SameAccount.
	RecurringTransactionID *string
}

// Update is a partial change to an entry. Kind is deliberately absent — it
// is immutable after creation (see design.md); the handler's request body
// has no field for it either, so DisallowUnknownFields rejects an attempt
// to set it. ToAccountID is likewise absent — a self_transfer's two accounts
// are fixed at creation, never individually reassigned. A nil field here
// leaves it untouched; TagIDs replaces the full set when non-nil (including
// an empty, non-nil slice, which clears every tag). A non-nil AccountID
// moves the entry to a different account, subject to the same ownership/
// disabled-account checks Create applies — rejected outright for a
// self_transfer (see design.md of add-self-transfer). Amount is settable on
// a transaction or self_transfer, Balance only on a balance adjustment (see
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
	Counterparty     *string
	Location         *string
	TagIDs           *[]string
	// RecurringTransactionID uses the same OptionalID trick as CategoryID:
	// absent (Set false) leaves the link untouched; explicit null clears
	// it; a value links to (or re-links) a recurring transaction on this
	// entry's (possibly also-updated) AccountID.
	RecurringTransactionID OptionalID
}

// Cursor is the keyset position of the last row of a previous page: the
// sorted column's value plus id, for a stable tie-break — see design.md.
// Native distinguishes a self_transfer's two same-id rows in a listing
// (true for the row matched via AccountID, false for the one matched via
// ToAccountID — see design.md of add-self-transfer); it is always true for
// every other kind, which only ever produces one row. Opaque to the
// client, like every other Cursor field.
type Cursor struct {
	BookingTimestamp time.Time
	Amount           int64
	ID               string
	Native           bool
}

// Filter narrows and orders a List call. CategoryID is the caller-supplied
// filter value; Service.List resolves it (including descendants) into
// CategoryIDs before the Store sees it — Store implementations read only
// CategoryIDs. Likewise AccountIDs is always resolved by Service.List to the
// caller's own visible accounts (optionally narrowed further by the
// caller-supplied AccountIDs) before reaching Store. AmountFrom and AmountTo
// are inclusive non-negative magnitude bounds in AmountScale units, applied to
// the listed account-oriented amount (including each self-transfer leg).
type Filter struct {
	AccountIDs []string
	// AllAccounts, when true, tells Store to ignore AccountIDs entirely —
	// no account_id restriction for this query. Set only by
	// Service.resolveFilter, when CategoryID names a category the caller
	// holds real permission on (ownership or a share) and no explicit
	// AccountIDs filter was also supplied: that category permission alone
	// authorizes seeing its entries, regardless of account access. Every
	// other filter (CategoryIDs, TagID, Kind, date range, Query) still
	// applies as usual — this only lifts the account restriction.
	AllAccounts            bool
	CategoryID             *string
	CategoryMode           CategoryMode
	CategoryIDs            []string
	TagID                  *string
	Kind                   *Kind
	RecurringTransactionID *string
	From                   *time.Time
	To                     *time.Time
	AmountFrom             *int64
	AmountTo               *int64
	Query                  string
	Sort                   SortField
	Dir                    SortDir
	After                  *Cursor
	Limit                  int
}

// CurrencySum is one currency's total within a Summary.
type CurrencySum struct {
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
}

// Summary is the result of summing a Filter's matching transaction entries,
// grouped by their account's currency — see Service.Sum. Income and Outcome
// split Sums by sign per currency: Income is the sum of positive amounts,
// Outcome the sum of the absolute value of negative amounts. A self-transfer
// entry counted on both its accounts contributes to both Income and Outcome
// even though its two legs cancel out in Sums.
type Summary struct {
	Sums    []CurrencySum `json:"sums"`
	Income  []CurrencySum `json:"income"`
	Outcome []CurrencySum `json:"outcome"`
	Count   int           `json:"count"`
}

// AccountSum is one account's totals within a Sum call — the net amount
// (Sums' contribution) plus its income/outcome split, before Service.Sum
// groups accounts by currency.
type AccountSum struct {
	Amount  int64
	Income  int64
	Outcome int64
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
// unconditionally. AllAccounts/CategoryIDs are likewise resolved by
// Service.FlowSummary (never caller-supplied) — see Filter's identical
// fields, whose category/tag resolution semantics FlowFilter shares
// exactly (Service.resolveCategoryAndTag backs both).
type FlowFilter struct {
	AccountIDs   []string
	AllAccounts  bool
	CategoryID   *string
	CategoryMode CategoryMode
	CategoryIDs  []string
	TagID        *string
	Unit         FlowUnit
	Year         int
	Month        int // 1-12; required when Unit == FlowUnitDay, must be 0 otherwise
	Timezone     string
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

// BalanceFilter narrows a BalanceSeries call. AccountIDs is resolved by
// Service.BalanceSeries to the caller's own visible accounts the same way
// FlowSummary resolves its own, before anything else runs. Timezone is
// never caller-supplied — Service.BalanceSeries fills it in from the
// caller's resolved settings (default UTC), so the midnight sample
// boundaries reflect the viewer's own calendar. Unit is accepted only as
// FlowUnitDay.
type BalanceFilter struct {
	AccountIDs []string
	Unit       FlowUnit
	Year       int
	Month      int // 1-12, required
	Timezone   string
}

// BalancePoint is the running account balance sampled at one local
// midnight — see Service.BalanceSeries. Period is the point's local
// calendar day, "YYYY-MM-DD" (the closing point carries the first day of
// the following month). Balances lists one entry per currency present in
// the selected accounts, always including that currency even when its
// amount is 0 — unlike FlowBucket, a balance line needs a value at every
// point.
type BalancePoint struct {
	Period   string        `json:"period"`
	Balances []CurrencySum `json:"balances"`
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
	if in.Kind != KindTransaction && (in.Counterparty != "" || in.Location != "") {
		return ErrInvalidValue
	}
	if in.Kind == KindSelfTransfer {
		if in.ToAccountID == nil || strings.TrimSpace(*in.ToAccountID) == "" {
			return ErrInvalidValue
		}
		if *in.ToAccountID == in.AccountID {
			return ErrInvalidValue
		}
	} else if in.ToAccountID != nil {
		return ErrInvalidValue
	}
	switch in.Kind {
	case KindTransaction, KindSelfTransfer:
		if in.Amount == nil || in.Balance != nil {
			return ErrInvalidValue
		}
	case KindBalanceAdjustment:
		if in.Balance == nil || in.Amount != nil {
			return ErrInvalidValue
		}
	}
	return nil
}
