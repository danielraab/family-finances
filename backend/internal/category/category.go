// Package category owns the global, tree-structured, admin-managed
// category lookup that every transaction entry references. It follows the
// repo's four-file shape (category.go, store.go, service.go, handler.go).
// It imports internal/auth only for auth.UserFromContext — never its Store
// or a database driver.
package category

import (
	"encoding/json"
	"strings"
	"time"
)

// Category is one node in the global category tree. ParentID is nil for a
// root category.
type Category struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
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

// New is the input to creating a category.
type New struct {
	ParentID *string
	Name     string
}

// Update is a partial change; a nil Name leaves it untouched. ParentID uses
// OptionalID so a category can be explicitly reparented to root.
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
