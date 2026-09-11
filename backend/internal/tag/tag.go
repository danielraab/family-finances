// Package tag owns per-user tags: labels an owner attaches to their own
// entries, full CRUD restricted to that owner, but shareable with other
// users at one of two permission tiers — mirroring internal/category's
// sharing model. It follows the repo's four-file shape (tag.go, store.go,
// service.go, handler.go). It imports internal/auth only for
// auth.UserFromContext — never its Store or a database driver.
package tag

import (
	"strings"
	"time"
)

// Tag is a per-user label. It has exactly one real owner (OwnerID) but MAY
// be shared with other users at one of two permission tiers — see
// Permission and TagShare.
//
// Permission and OwnerName are populated by Service, never by Store — they
// depend on the viewing caller (Permission) or a cross-package user lookup
// (OwnerName), neither of which a Store implementation has access to.
type Tag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
	// EntryCount is the number of the *viewing caller's own* non-deleted
	// entries currently carrying this tag — scoped to the caller even when
	// they are viewing a tag shared with them rather than one they own.
	// Computed by the store, never persisted directly; storage/memory has
	// no visibility into entries and always reports 0 (see its doc
	// comment), mirroring category.Category.EntryCount.
	EntryCount int        `json:"entry_count"`
	Permission Permission `json:"permission"`
	Shared     bool       `json:"shared"`
	OwnerName  string     `json:"owner_name,omitempty"`
	OwnerID    string     `json:"-"`
}

// Permission is the level of access a user has on a tag: either the real
// owner (implicitly PermissionOwner, with no TagShare row) or a tier
// granted via a share. There is no shareable "owner" tier —
// PermissionOwner is only ever the real owner's own, implicit permission,
// never granted through a share. append is a strict superset of view — see
// AtLeast. Mirrors category.Permission exactly.
type Permission string

const (
	// PermissionView grants seeing the tag resolve wherever it's
	// referenced — entry-list filters, and the name shown on an entry
	// already carrying it — but not selecting it on a new or edited entry.
	PermissionView Permission = "view"
	// PermissionAppend additionally grants selecting the tag on a new or
	// edited entry.
	PermissionAppend Permission = "append"
	// PermissionOwner is the real owner's own, implicit permission. It is
	// never a valid value for a share (see (Permission).valid) — only the
	// real owner may ever rename, disable/enable, delete a tag, or manage
	// its shares.
	PermissionOwner Permission = "owner"
)

// permissionRank orders the three tiers for AtLeast; not exported —
// callers compare via AtLeast, never the raw rank.
var permissionRank = map[Permission]int{
	PermissionView:   1,
	PermissionAppend: 2,
	PermissionOwner:  3,
}

// valid reports whether p is a value a share may be created or updated at
// (view or append) — never PermissionOwner, which a client can never
// request.
func (p Permission) valid() bool {
	return p == PermissionView || p == PermissionAppend
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

// TagShare is one row granting a non-owner user a Permission on a tag. The
// tag's real owner (Tag.OwnerID) never has a row here — see
// Service.ListShares, which synthesizes the real owner's entry separately
// for display.
type TagShare struct {
	TagID         string     `json:"-"`
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
// application invite — mirrors category.ShareResult.
type ShareResult struct {
	Matched       bool
	Share         *TagShare
	InviteAllowed bool
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidValue
	}
	return nil
}
