package tag

import (
	"context"
	"errors"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes in
// one place; domain and service code never mentions net/http.
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidValue = errors.New("invalid value")
	// ErrDuplicateName: the owner already has a tag with this name.
	ErrDuplicateName = errors.New("tag name already in use")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue, ErrDuplicateName}

// Store is the persistence contract tag declares. internal/storage/memory
// and internal/storage/postgres implement it; package main injects one.
type Store interface {
	List(ctx context.Context, ownerID string) ([]Tag, error)
	Get(ctx context.Context, ownerID, id string) (Tag, error)
	// ByName returns ownerID's tag named name, or ErrNotFound.
	ByName(ctx context.Context, ownerID, name string) (Tag, error)
	Create(ctx context.Context, ownerID, name string) (Tag, error)
	Update(ctx context.Context, ownerID, id, name string) (Tag, error)
	Delete(ctx context.Context, ownerID, id string) error

	// OwnedBy reports whether every id in tagIDs exists and belongs to
	// ownerID — satisfies internal/entry's TagLookup interface.
	OwnedBy(ctx context.Context, ownerID string, tagIDs []string) (bool, error)
}
