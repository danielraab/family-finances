// Package account owns bookkeeping accounts — each belonging to exactly one
// owner and visible only to them — and the admin-managed, instance-global
// account_types lookup they reference. It follows the repo's four-file shape
// (account.go, store.go, service.go, handler.go). It imports internal/auth
// only for auth.UserFromContext and internal/settings only for
// settings.ValidateCurrency — never their Store or a database driver.
package account

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"at.draab/familyfinances/internal/settings"
)

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

// Account is a bookkeeping account: title, description, admin-managed type,
// currency, financial institute, and opening/closing dates. It has exactly
// one owner and is visible only to them (no sharing in this change — see
// design.md). Disabled blocks creating new entries against it without
// hiding it or affecting existing entries; it is independent of ClosingDate
// (informational only) and of soft delete.
type Account struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Description        string     `json:"description,omitempty"`
	TypeID             string     `json:"type_id"`
	Currency           string     `json:"currency"`
	FinancialInstitute string     `json:"financial_institute,omitempty"`
	OpeningDate        Date       `json:"opening_date"`
	ClosingDate        *Date      `json:"closing_date,omitempty"`
	Disabled           bool       `json:"disabled"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	OwnerID            string     `json:"-"`
	DeletedAt          *time.Time `json:"-"`
}

// Type is one row of the admin-managed, instance-global account_types
// lookup.
type Type struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// New is the input to creating an account.
type New struct {
	Title              string
	Description        string
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
