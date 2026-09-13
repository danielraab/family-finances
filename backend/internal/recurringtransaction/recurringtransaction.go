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

// RecurringTransaction is a recurring transaction template: the same
// content fields as a transaction entry (AccountID, Title, Description,
// CategoryID, Counterparty, Location, TagIDs, Amount), plus a recurrence
// rule (Unit/IntervalCount), StartsOn, and an optional EndsOn.
//
// AccountCurrency and CreatedByName are resolved server-side by the
// Postgres store's query, the same way entry.Entry's equivalent fields
// are — never by Service. PerYearAmount, Ended, and NextSuggestedDate are
// computed by Service on every read, never stored (see design.md) —
// PerYearAmount and Ended are pure functions of Amount/IntervalUnit/
// IntervalCount/EndsOn; NextSuggestedDate additionally depends on the
// latest linked entry's booking date, resolved via EntryLookup.
type RecurringTransaction struct {
	ID                string    `json:"id"`
	AccountID         string    `json:"account_id"`
	AccountCurrency   string    `json:"account_currency,omitempty"`
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
	LinkedEntryCount int        `json:"linked_entry_count"`
	DeletedAt        *time.Time `json:"-"`
}

// New is the input to creating a recurring transaction.
type New struct {
	AccountID     string
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
// cleared.
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
}

func validateNew(in New) error {
	if strings.TrimSpace(in.AccountID) == "" {
		return ErrInvalidValue
	}
	if strings.TrimSpace(in.Title) == "" {
		return ErrInvalidValue
	}
	if in.CategoryID == nil || strings.TrimSpace(*in.CategoryID) == "" {
		return ErrInvalidValue
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
