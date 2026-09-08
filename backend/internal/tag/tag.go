// Package tag owns per-user tags: private labels an owner attaches to their
// own entries, full CRUD restricted to that owner, never visible to anyone
// else. It follows the repo's four-file shape (tag.go, store.go, service.go,
// handler.go). It imports internal/auth only for auth.UserFromContext —
// never its Store or a database driver.
package tag

import (
	"strings"
	"time"
)

// Tag is a per-user label.
type Tag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
	// EntryCount is the number of the owner's non-deleted entries currently
	// carrying this tag. Computed by the store, never persisted directly;
	// storage/memory has no visibility into entries and always reports 0
	// (see its doc comment).
	EntryCount int    `json:"entry_count"`
	OwnerID    string `json:"-"`
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidValue
	}
	return nil
}
