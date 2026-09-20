// Package recurringtransaction owns recurring transaction templates
// recorded against an account: their fields (the same content fields as a
// transaction entry), a recurrence rule (interval unit/count, replacing a
// growing list of named frequencies — see design.md), a computed per-year
// amount, an "ended" flag derived from an optional end date, and a computed
// next-suggested-booking-date derived from the entries linked to it. It
// follows the repo's four-file shape (recurringtransaction.go, service.go,
// store.go, handler.go), mirroring internal/entry's package-boundary
// pattern: narrow AccountLookup/CategoryLookup/TagLookup interfaces
// satisfied structurally by *account.Service/*category.Service/*tag.Service,
// plus a new EntryLookup interface satisfied structurally by *entry.Service.
package recurringtransaction

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// dateLayout is the wire format for starts_on/ends_on — a calendar date
// with no time-of-day or zone, matching an HTML <input type="date">, the
// same layout internal/account's Date type uses.
const dateLayout = "2006-01-02"

// Date is a calendar date (no time-of-day, no zone), marshaled as
// "YYYY-MM-DD". A small local copy of internal/account's Date type — domain
// packages don't share such value types across a package boundary (see
// backend/AGENTS.md's "dependencies flow one way" rule).
type Date struct{ time.Time }

// NewDate builds a Date from a time.Time, truncating to the calendar day.
func NewDate(t time.Time) Date { return Date{t.Truncate(24 * time.Hour)} }

func (d Date) String() string { return d.Format(dateLayout) }

// Before reports whether d is strictly before other, comparing only the
// calendar date component.
func (d Date) Before(other Date) bool { return d.Time.Before(other.Time) }

// After reports whether d is strictly after other, comparing only the
// calendar date component.
func (d Date) After(other Date) bool { return d.Time.After(other.Time) }

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(dateLayout))
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("recurringtransaction: invalid date: %w", err)
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("recurringtransaction: invalid date %q: %w", s, err)
	}
	d.Time = t
	return nil
}

// OptionalDate distinguishes a JSON key that is absent (Set is false) from
// one present as either null (Set true, Value nil — "clear it") or a date
// (Set true, Value set). Used for ends_on, the one field an update needs to
// be able to explicitly clear (e.g. re-activating a cancelled subscription).
type OptionalDate struct {
	Set   bool
	Value *Date
}

func (o *OptionalDate) UnmarshalJSON(b []byte) error {
	o.Set = true
	if string(b) == "null" {
		o.Value = nil
		return nil
	}
	var d Date
	if err := json.Unmarshal(b, &d); err != nil {
		return err
	}
	o.Value = &d
	return nil
}

// Unit is the recurrence rule's interval unit — combined with a positive
// IntervalCount to mean "every IntervalCount Units". There is no separate
// named frequency (yearly, quarterly, …) — see design.md: quarterly is
// {Unit: UnitMonth, IntervalCount: 3}, yearly is {Unit: UnitYear,
// IntervalCount: 1}.
type Unit string

const (
	UnitDay   Unit = "day"
	UnitWeek  Unit = "week"
	UnitMonth Unit = "month"
	UnitYear  Unit = "year"
)

func (u Unit) valid() bool {
	return u == UnitDay || u == UnitWeek || u == UnitMonth || u == UnitYear
}

// Kind is whether a recurring transaction templates an ordinary transaction
// or a self-transfer between two of the caller's own accounts. There is
// deliberately no balance_adjustment: an absolute balance reading is a
// correction, and correcting a balance on a schedule is meaningless — see
// design.md of add-recurring-self-transfers, which reopened only the
// self_transfer half of this package's original "no kind field" decision.
type Kind string

const (
	// KindTransaction templates an ordinary transaction entry — what every
	// recurring transaction was before ToAccountID existed.
	KindTransaction Kind = "transaction"
	// KindSelfTransfer templates a self_transfer entry: Amount moved from
	// AccountID to ToAccountID, required exactly for this kind.
	KindSelfTransfer Kind = "self_transfer"
)

func (k Kind) valid() bool { return k == KindTransaction || k == KindSelfTransfer }

// SelfTransferMode is how a List/Summary call treats self_transfer
// templates — the resolved form of the API's include_self_transfer and
// self_transfer_both_legs pair, so Store sees one value rather than two
// booleans whose fourth combination means nothing (see design.md).
type SelfTransferMode string

const (
	// SelfTransferExclude (the default) leaves self_transfer templates out
	// of the result entirely.
	SelfTransferExclude SelfTransferMode = "exclude"
	// SelfTransferNative includes each self_transfer template at most once,
	// from its sending account's side — the row as stored.
	SelfTransferNative SelfTransferMode = "native"
	// SelfTransferBothLegs includes each self_transfer template once per
	// account of its two that is in scope: income and outcome.
	SelfTransferBothLegs SelfTransferMode = "both_legs"
)

// ResolveSelfTransferMode maps the two API booleans onto a mode.
// bothLegs without include is ignored, per the spec.
func ResolveSelfTransferMode(include, bothLegs bool) SelfTransferMode {
	switch {
	case !include:
		return SelfTransferExclude
	case bothLegs:
		return SelfTransferBothLegs
	default:
		return SelfTransferNative
	}
}

// RecurringTransaction is a recurring transaction template: the same
// content fields as a transaction entry (AccountID, Title, Description,
// CategoryID, Counterparty, Location, TagIDs, Amount), plus a recurrence
// rule (Unit/IntervalCount), StartsOn, and an optional EndsOn.
//
// AccountCurrency, CreatedByName and — for a self_transfer —
// ToAccountName/ToAccountCurrency are resolved server-side by the Postgres
// store's query, the same way entry.Entry's equivalent fields are — never
// by Service, and regardless of the reading caller's own permission on the
// account named, since a reader who can see only one side of a transfer
// still needs to know where the money goes and in what currency. PerYearAmount, Ended, and NextSuggestedDate are
// computed by Service on every read, never stored (see design.md) —
// PerYearAmount and Ended are pure functions of Amount/IntervalUnit/
// IntervalCount/EndsOn; NextSuggestedDate additionally depends on the
// latest linked entry's booking date, resolved via EntryLookup.
type RecurringTransaction struct {
	ID                string    `json:"id"`
	AccountID         string    `json:"account_id"`
	AccountCurrency   string    `json:"account_currency,omitempty"`
	ToAccountID       *string   `json:"to_account_id,omitempty"`
	ToAccountName     string    `json:"to_account_name,omitempty"`
	ToAccountCurrency string    `json:"to_account_currency,omitempty"`
	Kind              Kind      `json:"kind"`
	Title             string    `json:"title"`
	Description       string    `json:"description,omitempty"`
	CategoryID        *string   `json:"category_id,omitempty"`
	Counterparty      string    `json:"counterparty,omitempty"`
	Location          string    `json:"location,omitempty"`
	TagIDs            []string  `json:"tag_ids"`
	Amount            int64     `json:"amount"`
	IntervalUnit      Unit      `json:"interval_unit"`
	IntervalCount     int       `json:"interval_count"`
	StartsOn          Date      `json:"starts_on"`
	EndsOn            *Date     `json:"ends_on,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	CreatedBy         string    `json:"created_by"`
	CreatedByName     string    `json:"created_by_name,omitempty"`
	PerYearAmount     int64     `json:"per_year_amount"`
	Ended             bool      `json:"ended"`
	NextSuggestedDate Date      `json:"next_suggested_date"`
	// LinkedEntryCount is the number of non-deleted entries currently
	// linked to this recurring transaction — computed by Service.decorate
	// via EntryLookup, never stored. Lets the client disable the delete
	// action before ever attempting it; a 409 is still the authoritative
	// guard server-side (see design.md).
	LinkedEntryCount int `json:"linked_entry_count"`
	// Native reports whether this row is the template as stored (true) or
	// the flipped, receiving-side leg of a self_transfer (false) — see the
	// recurring_transaction_legs view. Always true outside a both-legs
	// listing, and for every KindTransaction template.
	Native    bool       `json:"native"`
	DeletedAt *time.Time `json:"-"`
}

// New is the input to creating a recurring transaction.
type New struct {
	AccountID     string
	ToAccountID   *string
	Kind          Kind
	Title         string
	Description   string
	CategoryID    *string
	Counterparty  string
	Location      string
	TagIDs        []string
	Amount        int64
	IntervalUnit  Unit
	IntervalCount int
	StartsOn      Date
	EndsOn        *Date
}

// Update is a partial change to a recurring transaction. A nil field is
// left untouched; EndsOn uses OptionalDate so it can also be explicitly
// cleared. Kind and ToAccountID are deliberately absent — both are
// immutable after creation, mirroring entry.Kind/entry.Entry.ToAccountID,
// and the handler's request body has no field for either, so
// DisallowUnknownFields rejects an attempt to set them.
type Update struct {
	AccountID     *string
	Title         *string
	Description   *string
	CategoryID    *string
	Counterparty  *string
	Location      *string
	TagIDs        *[]string
	Amount        *int64
	IntervalUnit  *Unit
	IntervalCount *int
	StartsOn      *Date
	EndsOn        OptionalDate
}

// Filter narrows a List/Summary call. AccountIDs is always resolved by
// Service to the caller's own visible accounts (optionally narrowed
// further by a caller-supplied filter) before reaching Store.
type Filter struct {
	AccountIDs []string
	// SelfTransfers is how self_transfer templates are treated — resolved
	// by Service from the caller's two booleans before Store sees it. The
	// zero value is not a valid mode; Service always sets one.
	SelfTransfers SelfTransferMode
}

// CategoryMode controls how PreviewFilter.CategoryID is resolved — a local
// mirror of internal/entry's identical CategoryMode, kept package-local per
// this repo's "domain packages don't share value types across a package
// boundary" rule (see the local Date type above for the same reasoning).
type CategoryMode string

const (
	// CategoryModeSubtree (the default) resolves CategoryID to itself plus
	// every descendant the caller can see.
	CategoryModeSubtree CategoryMode = "subtree"
	// CategoryModeExact resolves CategoryID to itself alone.
	CategoryModeExact CategoryMode = "exact"
)

func (m CategoryMode) valid() bool {
	return m == "" || m == CategoryModeSubtree || m == CategoryModeExact
}

// PreviewFilter narrows a Preview call. AccountIDs is resolved the same way
// Filter's is; CategoryID/CategoryMode/TagID are resolved in-process against
// each candidate recurring transaction (see Service.Preview) rather than
// pushed down to Store, since Preview's result set is always small (bounded
// by To and the per-template occurrence cap) and never paginated. To is
// required — Preview never computes an unbounded result.
type PreviewFilter struct {
	AccountIDs   []string
	CategoryID   *string
	CategoryMode CategoryMode
	TagID        *string
	To           time.Time
}

// PreviewItem is one projected future occurrence of a recurring
// transaction — never persisted, computed fresh on every Preview call. It
// carries the same content fields a materialized entry from that template
// would, plus which template it came from, the projected date, and whether
// that date is already in the past.
//
// A self_transfer template projects one item per account of its two that is
// within the caller's scope — twice when both are, the receiving side with
// AccountID/ToAccountID swapped and Amount negated. Preview does this
// unconditionally, with no equivalent of List/Summary's SelfTransferMode:
// the surfaces it feeds sit beside real self_transfer entries that are
// already two-sided, and a projected balance that omitted the receiving
// side would be wrong rather than merely terse (see design.md).
type PreviewItem struct {
	RecurringTransactionID string   `json:"recurring_transaction_id"`
	AccountID              string   `json:"account_id"`
	AccountCurrency        string   `json:"account_currency,omitempty"`
	ToAccountID            *string  `json:"to_account_id,omitempty"`
	ToAccountName          string   `json:"to_account_name,omitempty"`
	Kind                   Kind     `json:"kind"`
	Title                  string   `json:"title"`
	Description            string   `json:"description,omitempty"`
	CategoryID             *string  `json:"category_id,omitempty"`
	Counterparty           string   `json:"counterparty,omitempty"`
	Location               string   `json:"location,omitempty"`
	TagIDs                 []string `json:"tag_ids"`
	Amount                 int64    `json:"amount"`
	BookingTimestamp       Date     `json:"booking_timestamp"`
	Overdue                bool     `json:"overdue"`
}

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
	// The per-kind field rules below are exactly internal/entry's, because
	// what they govern is the entry this template materializes: a template
	// must never be able to describe something POST /api/entries refuses.
	if in.Kind == KindSelfTransfer {
		if in.ToAccountID == nil || strings.TrimSpace(*in.ToAccountID) == "" {
			return ErrInvalidValue
		}
		if *in.ToAccountID == in.AccountID {
			return ErrInvalidValue
		}
		if in.Counterparty != "" || in.Location != "" {
			return ErrInvalidValue
		}
	} else {
		if in.ToAccountID != nil {
			return ErrInvalidValue
		}
		if in.CategoryID == nil || strings.TrimSpace(*in.CategoryID) == "" {
			return ErrInvalidValue
		}
	}
	if !in.IntervalUnit.valid() {
		return ErrInvalidValue
	}
	if in.IntervalCount <= 0 {
		return ErrInvalidValue
	}
	if in.StartsOn.Time.IsZero() {
		return ErrInvalidValue
	}
	if in.EndsOn != nil && in.EndsOn.Before(in.StartsOn) {
		return ErrInvalidValue
	}
	return nil
}

// validateUpdate rejects an Update whose provided fields fail validation,
// resolving against current to check the starts_on/ends_on relationship
// even when only one side of it is being changed.
func validateUpdate(current RecurringTransaction, upd Update) error {
	if upd.Title != nil && strings.TrimSpace(*upd.Title) == "" {
		return ErrInvalidValue
	}
	if upd.CategoryID != nil && strings.TrimSpace(*upd.CategoryID) == "" {
		return ErrInvalidValue
	}
	if current.Kind == KindSelfTransfer {
		if (upd.Counterparty != nil && *upd.Counterparty != "") ||
			(upd.Location != nil && *upd.Location != "") {
			return ErrInvalidValue
		}
	}
	if upd.IntervalUnit != nil && !upd.IntervalUnit.valid() {
		return ErrInvalidValue
	}
	if upd.IntervalCount != nil && *upd.IntervalCount <= 0 {
		return ErrInvalidValue
	}
	if upd.StartsOn != nil && upd.StartsOn.Time.IsZero() {
		return ErrInvalidValue
	}

	starts := current.StartsOn
	if upd.StartsOn != nil {
		starts = *upd.StartsOn
	}
	var ends *Date
	if upd.EndsOn.Set {
		ends = upd.EndsOn.Value
	} else {
		ends = current.EndsOn
	}
	if ends != nil && ends.Before(starts) {
		return ErrInvalidValue
	}
	return nil
}

// PerYearAmount computes amount's annualized total from a recurrence rule.
// month/year units are exact, calendar-based multipliers (12/count,
// 1/count); week/day units use the standard 365.25-day-year average, since
// no exact answer exists for a fixed day-count interval — see design.md.
// The result is rounded to the nearest integer (never stored — Service
// recomputes this on every read).
func PerYearAmount(amount int64, unit Unit, count int) int64 {
	if count <= 0 {
		return 0
	}
	var periodsPerYear float64
	switch unit {
	case UnitMonth:
		periodsPerYear = 12 / float64(count)
	case UnitYear:
		periodsPerYear = 1 / float64(count)
	case UnitWeek:
		periodsPerYear = 365.25 / (7 * float64(count))
	case UnitDay:
		periodsPerYear = 365.25 / float64(count)
	default:
		return 0
	}
	product := float64(amount) * periodsPerYear
	return int64(product + sign(product)*0.5)
}

// sign returns 1 for a positive f, -1 for negative, 0 for zero — used by
// PerYearAmount to round-half-away-from-zero rather than truncate, so a
// negative (expense) amount rounds symmetrically to a positive one.
func sign(f float64) float64 {
	switch {
	case f > 0:
		return 1
	case f < 0:
		return -1
	default:
		return 0
	}
}

// Ended reports whether endsOn is at or before today — both calendar
// dates, no time-of-day. A nil endsOn is never ended.
func Ended(endsOn *Date, today Date) bool {
	if endsOn == nil {
		return false
	}
	return !endsOn.After(today)
}

// daysInMonth returns the number of days in the given calendar month.
func daysInMonth(year int, month time.Month, loc *time.Location) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
}

// addMonthsClamped advances t by the given number of calendar months,
// clamping the day-of-month to the target month's last day when the
// original day doesn't exist there (e.g. Jan 31 + 1 month -> Feb 28/29) —
// unlike time.Time.AddDate, which would overflow into March.
func addMonthsClamped(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	totalMonths := int(m) - 1 + months
	newYear := y + totalMonths/12
	newMonthIndex := totalMonths % 12
	if newMonthIndex < 0 {
		newMonthIndex += 12
		newYear--
	}
	newMonth := time.Month(newMonthIndex + 1)
	if last := daysInMonth(newYear, newMonth, t.Location()); d > last {
		d = last
	}
	return time.Date(newYear, newMonth, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// Advance steps t forward by exactly one recurrence interval of unit/count.
// month and year use calendar-aware stepping (addMonthsClamped); week and
// day use a fixed day-count (time.Time.AddDate) since those units are
// already exact day multiples.
func Advance(t time.Time, unit Unit, count int) time.Time {
	switch unit {
	case UnitMonth:
		return addMonthsClamped(t, count)
	case UnitYear:
		return addMonthsClamped(t, count*12)
	case UnitWeek:
		return t.AddDate(0, 0, 7*count)
	case UnitDay:
		return t.AddDate(0, 0, count)
	default:
		return t
	}
}

// maxPreviewOccurrences defensively bounds how many occurrences Preview
// generates for a single recurring transaction, regardless of how far its
// cutoff or ends_on reaches — see design.md's "defensive per-template
// occurrence cap" decision.
const maxPreviewOccurrences = 366

// previewOccurrences generates anchor plus every subsequent occurrence of a
// unit/count recurrence rule up to cutoff (inclusive) and endsOn (inclusive,
// when set), implementing the single-overdue-row rule: anchor is always
// included, however far before today it is; every later candidate that is
// still before today is advanced past without being included, so at most
// one returned date is ever before today. Generation stops after
// maxPreviewOccurrences dates regardless of cutoff/endsOn.
func previewOccurrences(anchor Date, unit Unit, count int, cutoff, today Date, endsOn *Date) []Date {
	if anchor.After(cutoff) {
		return nil
	}
	if endsOn != nil && anchor.After(*endsOn) {
		return nil
	}

	dates := []Date{anchor}
	current := anchor
	for len(dates) < maxPreviewOccurrences {
		next := NewDate(Advance(current.Time, unit, count))
		if next.After(cutoff) {
			break
		}
		if endsOn != nil && next.After(*endsOn) {
			break
		}
		current = next
		if next.Before(today) {
			// Still overdue — advance past it without emitting another
			// overdue row.
			continue
		}
		dates = append(dates, next)
	}
	return dates
}
