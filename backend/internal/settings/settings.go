// Package settings owns per-user preferences: display language, timezone,
// and default currency. It follows the repo's four-file shape (settings.go,
// store.go, service.go, handler.go). It imports internal/auth only for
// auth.UserFromContext — the request-scoped-user accessor every HTTP-facing
// domain package uses — never its Store or a database driver.
package settings

import (
	"regexp"
	"time"
)

// Hardcoded application defaults, substituted for a missing row or a NULL
// column. There is no per-instance override (Non-Goal: no instance_settings
// table) — these are Go constants.
const (
	DefaultLanguage               = "en"
	DefaultTimezone               = "UTC"
	DefaultDefaultCurrency        = "EUR"
	DefaultDisplayedDecimalPlaces = 2
	MinDisplayedDecimalPlaces     = 0
	MaxDisplayedDecimalPlaces     = 4
)

// SupportedLanguages mirrors the client's web-client-i18n language set.
var SupportedLanguages = map[string]bool{"en": true, "de": true}

var currencyShape = regexp.MustCompile(`^[A-Z]{3}$`)

// Row is a user's raw, possibly-partial stored preferences: nil means unset.
// A user with no user_settings row is the zero Row (every field nil).
type Row struct {
	Language               *string
	Timezone               *string
	DefaultCurrency        *string
	DisplayedDecimalPlaces *int
}

// Update is a partial change: only non-nil fields are applied.
type Update struct {
	Language               *string
	Timezone               *string
	DefaultCurrency        *string
	DisplayedDecimalPlaces *int
}

// Settings is a user's fully-resolved preferences — every field always
// populated, defaults already substituted.
type Settings struct {
	Language               string `json:"language"`
	Timezone               string `json:"timezone"`
	DefaultCurrency        string `json:"default_currency"`
	DisplayedDecimalPlaces int    `json:"displayed_decimal_places"`
}

// Resolve substitutes the hardcoded defaults for any unset field in row.
func Resolve(row Row) Settings {
	s := Settings{
		Language:               DefaultLanguage,
		Timezone:               DefaultTimezone,
		DefaultCurrency:        DefaultDefaultCurrency,
		DisplayedDecimalPlaces: DefaultDisplayedDecimalPlaces,
	}
	if row.Language != nil {
		s.Language = *row.Language
	}
	if row.Timezone != nil {
		s.Timezone = *row.Timezone
	}
	if row.DefaultCurrency != nil {
		s.DefaultCurrency = *row.DefaultCurrency
	}
	if row.DisplayedDecimalPlaces != nil {
		s.DisplayedDecimalPlaces = *row.DisplayedDecimalPlaces
	}
	return s
}

// ValidateLanguage accepts only the two supported language codes.
func ValidateLanguage(v string) error {
	if !SupportedLanguages[v] {
		return ErrInvalidValue
	}
	return nil
}

// ValidateTimezone accepts only a value the IANA tzdata resolves.
func ValidateTimezone(v string) error {
	if _, err := time.LoadLocation(v); err != nil {
		return ErrInvalidValue
	}
	return nil
}

// ValidateCurrency accepts the ISO-4217 shape (three uppercase letters) —
// not checked against a canonical currency list; see design.md.
func ValidateCurrency(v string) error {
	if !currencyShape.MatchString(v) {
		return ErrInvalidValue
	}
	return nil
}

// ValidateDisplayedDecimalPlaces accepts 0..4 inclusive — the upper bound
// matches account-entries' fixed storage precision, since displaying more
// decimal digits than are ever stored would be meaningless.
func ValidateDisplayedDecimalPlaces(v int) error {
	if v < MinDisplayedDecimalPlaces || v > MaxDisplayedDecimalPlaces {
		return ErrInvalidValue
	}
	return nil
}
