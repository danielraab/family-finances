package category

import (
	"context"
	"errors"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes in
// one place; domain and service code never mentions net/http.
var (
	// ErrNotFound: no such category (or it belongs to a different owner,
	// or is soft-deleted — those behave identically to nonexistent).
	ErrNotFound     = errors.New("not found")
	ErrInvalidValue = errors.New("invalid value")
	// ErrInUse: the category has at least one non-deleted child category
	// (checked here) or is referenced by a non-soft-deleted entry (enforced
	// by the real backend's entries.category_id foreign key / an explicit
	// check in storage/postgres — internal/category itself never imports
	// internal/entry, so internal/storage/memory's Store cannot enforce
	// the entry side of this for tests).
	ErrInUse = errors.New("category is in use")
	// ErrCycle: an update would make a category its own ancestor.
	ErrCycle = errors.New("category cannot be its own ancestor")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue, ErrInUse, ErrCycle}

// Store is the persistence contract category declares. internal/storage/memory
// and internal/storage/postgres implement it; package main injects one.
type Store interface {
	// List returns every non-deleted category ownerID owns.
	List(ctx context.Context, ownerID string) ([]Category, error)
	// Get returns the category, scoped to ownerID; ErrNotFound if it does
	// not exist, belongs to a different owner, or is soft-deleted.
	Get(ctx context.Context, ownerID, id string) (Category, error)
	// Create appends the new category to the end of its sibling group
	// (same ownerID + parent_id).
	Create(ctx context.Context, ownerID string, in New) (Category, error)
	// Update applies upd. A reparent (ParentID.Set with a non-nil value)
	// appends the category to the end of its new parent's sibling group.
	Update(ctx context.Context, ownerID, id string, upd Update) (Category, error)
	// Delete soft-deletes (sets deleted_at). Returns ErrInUse if the
	// category has a non-deleted child, or (on the real backend) is
	// referenced by a non-deleted entry. One-way — no undelete.
	Delete(ctx context.Context, ownerID, id string) error
	// SetDisabled toggles whether the category may be newly (re)selected
	// on an entry. It does not affect any entry or child category already
	// referencing it.
	SetDisabled(ctx context.Context, ownerID, id string, disabled bool) (Category, error)
	// MoveUp swaps the category's sort_order with its immediate previous
	// sibling. A no-op (category unchanged) if it is already first among
	// its siblings.
	MoveUp(ctx context.Context, ownerID, id string) (Category, error)
	// MoveDown swaps the category's sort_order with its immediate next
	// sibling. A no-op (category unchanged) if it is already last among
	// its siblings.
	MoveDown(ctx context.Context, ownerID, id string) (Category, error)

	// Exists reports whether id exists, belongs to ownerID, and is not
	// soft-deleted — used both to validate a ParentID and to satisfy
	// internal/entry's CategoryLookup.
	Exists(ctx context.Context, ownerID, id string) (bool, error)
	// Subtree returns id and every descendant id within ownerID's own
	// tree, used to detect a reparent-into-own-subtree cycle and to
	// resolve internal/entry's "this category or any of its descendants"
	// filter.
	Subtree(ctx context.Context, ownerID, id string) ([]string, error)

	// SeedDefaults inserts DefaultNames, as root categories in that order,
	// for ownerID. Called unconditionally for a brand-new user — there is
	// nothing to collide with yet (the migration's existing-user backfill,
	// pure SQL, is the only caller that needs to guard against a non-empty
	// tree, since it isn't calling this method).
	SeedDefaults(ctx context.Context, ownerID string) error
}
