// Package category owns per-user, tree-structured categories that entries
// are classified by. Each owner fully self-serves their own tree — no
// admin involvement. It follows the repo's four-file shape (category.go,
// store.go, service.go, handler.go). It imports internal/auth only for
// auth.UserFromContext — never its Store or a database driver.
package category

import (
	"encoding/json"
	"strings"
	"time"
)

// DefaultNames is the starter set of root categories seeded for every new
// user (Service.SeedDefaults, invoked as an internal/auth.NewUserHook), in
// the order they're assigned sort_order — the same set regardless of the
// user's language, since a seeded category is immediately theirs to rename
// like any other.
var DefaultNames = []string{
	"Salary",
	"Groceries",
	"Rent",
	"Utilities",
	"Transportation",
	"Entertainment",
	"Health",
	"Other",
}

// Category is one node in a category tree private to its owner. ParentID
// is nil for a root category.
type Category struct {
	ID        string     `json:"id"`
	ParentID  *string    `json:"parent_id,omitempty"`
	Name      string     `json:"name"`
	SortOrder int        `json:"sort_order"`
	Disabled  bool       `json:"disabled"`
	CreatedAt time.Time  `json:"created_at"`
	OwnerID   string     `json:"-"`
	DeletedAt *time.Time `json:"-"`
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
}

// Update is a partial change; a nil Name leaves it untouched. ParentID uses
// OptionalID so a category can be explicitly reparented to root. Disabled
// and SortOrder are changed only through their own dedicated operations
// (Disable/Enable, MoveUp/MoveDown), never through Update.
type Update struct {
	Name     *string
	ParentID OptionalID
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidValue
	}
	return nil
}
