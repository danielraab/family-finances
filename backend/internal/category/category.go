// Package category owns per-user, tree-structured categories that entries
// are classified by. Each owner fully self-serves their own tree — no
// admin involvement. It follows the repo's four-file shape (category.go,
// store.go, service.go, handler.go). It imports internal/auth only for
// auth.UserFromContext — never its Store or a database driver.
package category

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// DefaultCategory is one entry in the starter set seeded for every new user
// (Service.SeedDefaults, invoked as an internal/auth.NewUserHook). Icon and
// Color are opaque client presentation tokens — the same fixed assignment
// for every user, independent of language — so a fresh tree looks
// considered rather than blank; the user is free to change or clear them
// like any other category.
type DefaultCategory struct {
	Name  string
	Icon  string
	Color string
}

// DefaultCategories is the starter set of root categories, in the order
// they're assigned sort_order. The icon tokens come from the web client's
// curated set and the color tokens from its palette (see the
// account-category-icon-color change).
var DefaultCategories = []DefaultCategory{
	{"Salary", "hand-coins", "green"},
	{"Groceries", "shopping-cart", "orange"},
	{"Rent", "house", "blue"},
	{"Utilities", "zap", "yellow"},
	{"Transportation", "car", "aqua"},
	{"Entertainment", "gamepad-2", "magenta"},
	{"Health", "heart", "red"},
	{"Other", "circle-dashed", "violet"},
}

// DefaultNames lists the seeded category names in sort order. Derived from
// DefaultCategories, which is the single source of truth for the starter
// set.
var DefaultNames = func() []string {
	names := make([]string, len(DefaultCategories))
	for i, c := range DefaultCategories {
		names[i] = c.Name
	}
	return names
}()

// Category is one node in a category tree private to its owner. ParentID
// is nil for a root category.
type Category struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon,omitempty"`
	Color     string    `json:"color,omitempty"`
	SortOrder int       `json:"sort_order"`
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
	// EntryCount is the number of the owner's non-deleted entries directly
	// categorized under this category — direct references only, not rolled
	// up from descendant categories. Computed by the store, never persisted
	// directly; storage/memory has no visibility into entries and always
	// reports 0 (see its doc comment), mirroring tag.Tag.EntryCount.
	EntryCount int        `json:"entry_count"`
	OwnerID    string     `json:"-"`
	DeletedAt  *time.Time `json:"-"`
}

// OptionalID distinguishes a JSON key that is absent (Set is false) from
// one present as either null (Set true, Value nil — "make it a root
// category") or an id (Set true, Value set) — the same trick account.Date
// uses for closing_date.
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

// New is the input to creating a category. SortOrder and Disabled are
// never client-settable at creation — a new category always starts
// enabled and appended to the end of its sibling group (see Service.Create).
type New struct {
	ParentID *string
	Name     string
	Icon     string
	Color    string
}

// Update is a partial change; a nil Name leaves it untouched. ParentID uses
// OptionalID so a category can be explicitly reparented to root. Disabled
// and SortOrder are changed only through their own dedicated operations
// (Disable/Enable, MoveUp/MoveDown), never through Update.
type Update struct {
	Name     *string
	Icon     *string
	Color    *string
	ParentID OptionalID
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidValue
	}
	return nil
}

// presentationToken bounds an icon or color value: a short, opaque
// client-side token the backend stores and echoes but never interprets.
// An empty value means "unset" and is always allowed.
var presentationToken = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

// validPresentationToken reports whether s is an acceptable icon/color
// value — empty (unset) or matching presentationToken.
func validPresentationToken(s string) bool {
	return s == "" || presentationToken.MatchString(s)
}
