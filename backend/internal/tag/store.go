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
	// ErrInUse: the tag currently has at least one active share — deleting
	// it is blocked until every share on it is revoked or left, so a
	// delete never silently un-tags another user's entries. Unlike
	// category.ErrInUse, this is never triggered by the owner's own
	// entries still carrying the tag — only by sharing.
	ErrInUse = errors.New("tag is in use")
	// ErrForbidden: the caller has some permission on the tag (a share)
	// but not enough for the attempted operation — every share-management
	// action (invite, change permission, revoke someone else's) requires
	// being the real owner. Distinct from ErrNotFound, which is for no
	// permission at all.
	ErrForbidden = errors.New("forbidden")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue, ErrDuplicateName, ErrInUse, ErrForbidden}

// Store is the persistence contract tag declares. internal/storage/memory
// and internal/storage/postgres implement it; package main injects one.
type Store interface {
	// List returns every tag callerID owns (Permission "owner") union
	// every tag shared with them (Permission "view"/"append", Shared
	// true, OwnerName set).
	List(ctx context.Context, callerID string) ([]Tag, error)
	// Get returns the tag, scoped to ownerID; ErrNotFound if it does not
	// exist or belongs to a different owner. Used only for the strictly
	// owner-scoped operations (rename, disable, delete) — never for a
	// caller-facing read, which goes through GetForCaller instead.
	Get(ctx context.Context, ownerID, id string) (Tag, error)
	// GetForCaller returns the tag by id, annotated with callerID's
	// Permission/Shared/OwnerName (Permission is "" when callerID has no
	// access at all — not itself an error; Service.Get is the one place
	// that's translated into ErrNotFound, mirroring category.Store.Get's
	// convention). ErrNotFound only when id names no tag at all.
	GetForCaller(ctx context.Context, callerID, id string) (Tag, error)
	// ByName returns ownerID's tag named name, or ErrNotFound.
	ByName(ctx context.Context, ownerID, name string) (Tag, error)
	Create(ctx context.Context, ownerID, name string) (Tag, error)
	Update(ctx context.Context, ownerID, id, name string) (Tag, error)
	// Delete removes the tag, detaching it from every entry it was
	// attached to. Returns ErrInUse if the tag currently has at least one
	// active share.
	Delete(ctx context.Context, ownerID, id string) error
	// SetDisabled sets ownerID's tag id's disabled flag, reversibly.
	SetDisabled(ctx context.Context, ownerID, id string, disabled bool) (Tag, error)

	// OwnedBy reports whether every id in tagIDs exists and is visible to
	// callerID — owned by them, or shared with them at any tier (at least
	// view) — satisfies internal/entry's TagLookup interface. This is the
	// looser of the two checks: it validates a reference that may simply
	// be carried over unchanged from a prior request, not one newly being
	// added, so it does not require append tier or check disabled.
	OwnedBy(ctx context.Context, callerID string, tagIDs []string) (bool, error)
	// Usable reports whether every id in tagIDs exists, is usable by
	// callerID — owned by them, or shared with them at append tier — and
	// is not disabled — satisfies internal/entry's TagLookup interface.
	// An empty tagIDs is trivially true, matching OwnedBy.
	Usable(ctx context.Context, callerID string, tagIDs []string) (bool, error)

	// --- sharing ---

	// CreateOrUpdateShare grants userID permission on tagID, recording
	// grantedBy. An existing share for (tagID, userID) has its permission
	// overwritten in place rather than duplicating a row.
	CreateOrUpdateShare(ctx context.Context, tagID, userID string, permission Permission, grantedBy string) (TagShare, error)
	// ListShares returns every current share on tagID — not including the
	// real owner, who carries no share row (Service composes that
	// separately for display).
	ListShares(ctx context.Context, tagID string) ([]TagShare, error)
	// ShareByUser returns userID's own share on tagID, or ErrNotFound if
	// they have none (including when they are the real owner, who has no
	// share row).
	ShareByUser(ctx context.Context, tagID, userID string) (TagShare, error)
	// UpdateSharePermission changes an existing share's permission.
	// ErrNotFound if no share exists for (tagID, userID).
	UpdateSharePermission(ctx context.Context, tagID, userID string, permission Permission) (TagShare, error)
	// DeleteShare removes a share unconditionally (no soft delete).
	// ErrNotFound if none exists.
	DeleteShare(ctx context.Context, tagID, userID string) error
}
