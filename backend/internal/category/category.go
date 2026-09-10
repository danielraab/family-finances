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
// is nil for a root category. It has exactly one real owner (OwnerID) but
// MAY be shared with other users at one of two permission tiers — see
// Permission and CategoryShare.
//
// Permission and OwnerName are populated by Service, never by Store — they
// depend on the viewing caller (Permission) or a cross-package user lookup
// (OwnerName), neither of which a Store implementation has access to.
// OwnerName is resolved for every category, real owner's own view included
// (mirroring account.Account.OwnerName) — only meaningful to render when
// Shared is true, so the client can show a "shared by X" badge without a
// second request.
type Category struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parent_id,omitempty"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon,omitempty"`
	Color     string    `json:"color,omitempty"`
	SortOrder int       `json:"sort_order"`
	Disabled  bool      `json:"disabled"`
	CreatedAt time.Time `json:"created_at"`
	// EntryCount is the number of the *viewing caller's own* non-deleted
	// entries directly categorized under this category — direct references
	// only, not rolled up from descendant categories, and scoped to the
	// caller even when they are viewing a category shared with them rather
	// than one they own. Computed by the store, never persisted directly;
	// storage/memory has no visibility into entries and always reports 0
	// (see its doc comment), mirroring tag.Tag.EntryCount.
	EntryCount int        `json:"entry_count"`
	Permission Permission `json:"permission"`
	Shared     bool       `json:"shared"`
	OwnerName  string     `json:"owner_name,omitempty"`
	OwnerID    string     `json:"-"`
	DeletedAt  *time.Time `json:"-"`
}

// Permission is the level of access a user has on a category: either the
// real owner (implicitly PermissionOwner, with no CategoryShare row) or a
// tier granted via a share. Unlike internal/account, there is no shareable
// "owner" tier — PermissionOwner is only ever the real owner's own,
// implicit permission, never granted through a share. append is a strict
// superset of view — see AtLeast.
type Permission string

const (
	// PermissionView grants seeing the category resolve wherever it's
	// referenced — entry-list/report category filters, and the name shown
	// on an entry already categorized under it — but not selecting it on a
	// new or edited entry.
	PermissionView Permission = "view"
	// PermissionAppend additionally grants selecting the category on a new
	// or edited entry.
	PermissionAppend Permission = "append"
	// PermissionOwner is the real owner's own, implicit permission. It is
	// never a valid value for a share (see (Permission).valid) — only the
	// real owner may ever edit a category's own metadata, lifecycle, or
	// shares.
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

// CategoryShare is one row granting a non-owner user a Permission on a
// category. The category's real owner (Category.OwnerID) never has a row
// here — see Service.ListShares, which synthesizes the real owner's entry
// separately for display.
type CategoryShare struct {
	CategoryID    string     `json:"-"`
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
// application invite — mirrors account.ShareResult.
type ShareResult struct {
	Matched       bool
	Share         *CategoryShare
	InviteAllowed bool
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
