package category

import (
	"context"
	"errors"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes in
// one place; domain and service code never mentions net/http.
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidValue = errors.New("invalid value")
	// ErrInUse: the category has at least one child category (checked here)
	// or is referenced by a non-soft-deleted entry (enforced by the real
	// backend's entries.category_id foreign key / an explicit check in
	// storage/postgres — internal/category itself never imports
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
	List(ctx context.Context) ([]Category, error)
	Get(ctx context.Context, id string) (Category, error)
	Create(ctx context.Context, in New) (Category, error)
	Update(ctx context.Context, id string, upd Update) (Category, error)
	// Delete returns ErrInUse if the category has children, or (on the
	// real backend) is referenced by a non-deleted entry.
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, id string) (bool, error)
	// Subtree returns id and every descendant id, used to resolve "this
	// category or any of its descendants" for internal/entry's category
	// filter, and to detect a reparent-into-own-subtree cycle.
	Subtree(ctx context.Context, id string) ([]string, error)
}
