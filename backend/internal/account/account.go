// Package account owns bookkeeping accounts — each belonging to exactly one
// real owner, shareable with other users at one of four permission tiers
// (see Permission) — and the per-owner account_types lookup they reference.
// It follows the repo's four-file shape (account.go, store.go, service.go,
// handler.go). It imports internal/auth only for auth.UserFromContext and
// internal/settings only for settings.ValidateCurrency — never their Store
// or a database driver.
package account

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"at.draab/familyfinances/internal/settings"
)

// presentationToken bounds an icon or color value: a short, opaque
// client-side token the backend stores and echoes but never interprets.
// An empty value means "unset" and is always allowed.
var presentationToken = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

// validPresentationToken reports whether s is an acceptable icon/color
// value — empty (unset) or matching presentationToken.
func validPresentationToken(s string) bool {
	return s == "" || presentationToken.MatchString(s)
}

// dateLayout is the wire format for opening_date/closing_date — a calendar
// date with no time-of-day or zone, matching an HTML <input type="date">.
const dateLayout = "2006-01-02"

// Date is a calendar date (no time-of-day, no zone), marshaled as
// "YYYY-MM-DD".
type Date struct{ time.Time }

// NewDate builds a Date from y/m/d, useful for tests and comparisons.
func NewDate(t time.Time) Date { return Date{t.Truncate(24 * time.Hour)} }

// ParseDate parses a "YYYY-MM-DD" string.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return Date{}, ErrInvalidValue
	}
	return Date{t}, nil
}

func (d Date) String() string { return d.Format(dateLayout) }

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(dateLayout))
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("account: invalid date: %w", err)
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("account: invalid date %q: %w", s, err)
	}
	d.Time = t
	return nil
}

// OptionalDate distinguishes a JSON key that is absent (Set is false, and
// UnmarshalJSON is never called by encoding/json) from one present as either
// null (Set true, Value nil — "clear it") or a date (Set true, Value set).
// Used for closing_date, the one field an update needs to be able to clear.
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

// Account is a bookkeeping account: title, description, per-owner type,
// currency, financial institute, and opening/closing dates. It has exactly
// one real owner (OwnerID) but MAY be shared with other users at one of
// four permission tiers — see Permission and AccountShare. Disabled blocks
// creating new entries against it without hiding it or affecting existing
// entries; it is independent of ClosingDate (informational only) and of
// soft delete.
//
// Permission and OwnerName are populated by Service, never by Store — they
// depend on the viewing caller (Permission) or a cross-package user lookup
// (OwnerName), neither of which a Store implementation has access to.
// OwnerName is set only when the viewing caller is not the real owner, so
// the client can render a "shared by X" badge without a second request.
type Account struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Description        string     `json:"description,omitempty"`
	Icon               string     `json:"icon,omitempty"`
	Color              string     `json:"color,omitempty"`
	TypeID             string     `json:"type_id"`
	Currency           string     `json:"currency"`
	FinancialInstitute string     `json:"financial_institute,omitempty"`
	OpeningDate        Date       `json:"opening_date"`
	ClosingDate        *Date      `json:"closing_date,omitempty"`
	Disabled           bool       `json:"disabled"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	Permission         Permission `json:"permission"`
	Shared             bool       `json:"shared"`
	OwnerName          string     `json:"owner_name,omitempty"`
	OwnerID            string     `json:"-"`
	DeletedAt          *time.Time `json:"-"`
}

// Permission is the level of access a user has on an account: either the
// real owner (implicitly PermissionOwner, with no AccountShare row) or a
// tier granted via a share. Each tier is a strict superset of the ones
// before it — see AtLeast.
type Permission string

const (
	// PermissionView grants reading the account's entries and balance.
	PermissionView Permission = "view"
	// PermissionAppend additionally grants creating entries and
	// editing/deleting only entries the same user created.
	PermissionAppend Permission = "append"
	// PermissionEntryAdmin additionally grants editing/deleting any entry
	// on the account, not only ones that user created.
	PermissionEntryAdmin Permission = "entry_admin"
	// PermissionOwner additionally grants editing the account's own
	// metadata (all fields except TypeID — see Service.Update), disabling/
	// enabling/soft-deleting the account, and managing shares. Identical in
	// capability to the real owner, whether held via OwnerID or a share.
	PermissionOwner Permission = "owner"
)

// permissionRank orders the four tiers for AtLeast; not exported — callers
// compare via AtLeast, never the raw rank.
var permissionRank = map[Permission]int{
	PermissionView:       1,
	PermissionAppend:     2,
	PermissionEntryAdmin: 3,
	PermissionOwner:      4,
}

func (p Permission) valid() bool {
	_, ok := permissionRank[p]
	return ok
}

// AtLeast reports whether p is other or a stronger tier. An empty
// Permission (no access at all) is never AtLeast anything, including
// PermissionView.
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

// Access is what a specific caller may do on a specific account, resolved
// by Service.Access (internal/entry's AccountLookup interface consumes this
// shape too, under its own local type — see that package's design note).
// Permission is "" when the caller has no access at all — real ownership
// and every AccountShare checked in one place, so a consumer never needs to
// know shares exist as a separate concept.
type Access struct {
	Currency   string
	Disabled   bool
	Permission Permission
}

// AccountShare is one row granting a non-owner user a Permission on an
// account. The account's real owner (Account.OwnerID) never has a row here
// — see Service.ListShares, which synthesizes the real owner's entry
// separately for display.
type AccountShare struct {
	AccountID     string     `json:"-"`
	UserID        string     `json:"user_id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	Permission    Permission `json:"permission"`
	GrantedBy     string     `json:"granted_by"`
	GrantedByName string     `json:"granted_by_name"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ShareResult is the outcome of Service.InviteShare: either a created/
// updated share (Matched), or a report that the email matched no user
// (!Matched), including whether the instance currently allows sending an
// application invite — see design.md's "sharing by email is a synchronous,
// revealing lookup" decision.
type ShareResult struct {
	Matched       bool
	Share         *AccountShare
	InviteAllowed bool
}

// Type is one row of the account_types lookup, private to the user who
// owns it. Disabled blocks it from being (re)assigned to an account — see
// Service.resolveAssignableType — without affecting any account already
// carrying it.
type Type struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Disabled    bool      `json:"disabled"`
	CreatedAt   time.Time `json:"created_at"`
	OwnerID     string    `json:"-"`
}

// DefaultTypeTitles is the starter set of account types seeded for every
// new user (Service.SeedDefaults, invoked as an internal/auth.NewUserHook)
// — the same set regardless of the user's language, since a seeded type is
// immediately theirs to rename like any other.
var DefaultTypeTitles = []string{
	"Checking",
	"Savings",
	"Cash",
	"Credit Card",
	"Loan",
	"Investment",
}

// New is the input to creating an account.
type New struct {
	Title              string
	Description        string
	Icon               string
	Color              string
	TypeID             string
	Currency           string
	FinancialInstitute string
	OpeningDate        Date
	ClosingDate        *Date
}

// Update is a partial change to an account: a nil field is left untouched.
// ClosingDate uses OptionalDate so it can also be explicitly cleared
// (re-opening an account), distinct from "not provided."
type Update struct {
	Title              *string
	Description        *string
	Icon               *string
	Color              *string
	TypeID             *string
	Currency           *string
	FinancialInstitute *string
	OpeningDate        *Date
	ClosingDate        OptionalDate
}

// validateNew rejects a New whose required fields are missing or whose
// values fail validation.
func validateNew(in New) error {
	if strings.TrimSpace(in.Title) == "" {
		return ErrInvalidValue
	}
	if strings.TrimSpace(in.TypeID) == "" {
		return ErrInvalidValue
	}
	if err := settings.ValidateCurrency(in.Currency); err != nil {
		return ErrInvalidValue
	}
	if !validPresentationToken(in.Icon) || !validPresentationToken(in.Color) {
		return ErrInvalidValue
	}
	if in.ClosingDate != nil && in.ClosingDate.Time.Before(in.OpeningDate.Time) {
		return ErrInvalidValue
	}
	return nil
}

// validateUpdate rejects an Update whose provided fields fail validation,
// resolving against current to check the closing/opening relationship even
// when only one side of it is being changed.
func validateUpdate(current Account, upd Update) error {
	if upd.Title != nil && strings.TrimSpace(*upd.Title) == "" {
		return ErrInvalidValue
	}
	if upd.TypeID != nil && strings.TrimSpace(*upd.TypeID) == "" {
		return ErrInvalidValue
	}
	if upd.Currency != nil {
		if err := settings.ValidateCurrency(*upd.Currency); err != nil {
			return ErrInvalidValue
		}
	}
	if upd.Icon != nil && !validPresentationToken(*upd.Icon) {
		return ErrInvalidValue
	}
	if upd.Color != nil && !validPresentationToken(*upd.Color) {
		return ErrInvalidValue
	}

	opening := current.OpeningDate
	if upd.OpeningDate != nil {
		opening = *upd.OpeningDate
	}
	var closing *Date
	if upd.ClosingDate.Set {
		closing = upd.ClosingDate.Value
	} else {
		closing = current.ClosingDate
	}
	if closing != nil && closing.Time.Before(opening.Time) {
		return ErrInvalidValue
	}
	return nil
}
