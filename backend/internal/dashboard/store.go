package dashboard

import (
	"context"
	"errors"
)

// Sentinel errors. internal/httpapi/respond.go maps these to status codes
// in one place; domain and service code never mentions net/http.
var (
	// ErrNotFound: no such card, or it belongs to a different owner — the
	// two behave identically, mirroring category/tag's "wrong owner reads
	// as not found, never forbidden" convention, since a dashboard has no
	// sharing at all.
	ErrNotFound = errors.New("not found")
	// ErrInvalidValue: an unrecognized type, a config missing a field its
	// type requires, a config field foreign to its type, or an
	// account_id/category_id/tag_id the caller cannot see.
	ErrInvalidValue = errors.New("invalid value")
)

// Sentinels is every error above, for the httpapi mapping.
var Sentinels = []error{ErrNotFound, ErrInvalidValue}

// Store is the persistence contract dashboard declares. internal/storage/memory
// and internal/storage/postgres implement it; package main injects one.
type Store interface {
	// List returns every one of ownerID's own cards, ordered by
	// sort_order.
	List(ctx context.Context, ownerID string) ([]Card, error)
	// Get returns ownerID's own card by id. ErrNotFound if it does not
	// exist or belongs to a different owner. Used only by Service.Update
	// to learn the card's immutable type before validating a replacement
	// config against it.
	Get(ctx context.Context, ownerID, id string) (Card, error)
	// Create appends the new card to the end of ownerID's own list.
	Create(ctx context.Context, ownerID string, in New) (Card, error)
	// Update replaces id's config. ErrNotFound if id does not exist or
	// belongs to a different owner.
	Update(ctx context.Context, ownerID, id string, upd Update) (Card, error)
	// Delete removes id. ErrNotFound if id does not exist or belongs to a
	// different owner. Deleting a card closes the gap in sort_order for
	// the rest of ownerID's list, so a later move never gets stuck on it.
	Delete(ctx context.Context, ownerID, id string) error
	// MoveUp swaps id's sort_order with its immediately preceding card in
	// ownerID's own list. A no-op (card unchanged) if it is already
	// first.
	MoveUp(ctx context.Context, ownerID, id string) (Card, error)
	// MoveDown swaps id's sort_order with its immediately following card
	// in ownerID's own list. A no-op (card unchanged) if it is already
	// last.
	MoveDown(ctx context.Context, ownerID, id string) (Card, error)
}

// AccountLookup is the narrow, flat-value view of internal/account that
// dashboard needs to validate a config's account_id — the same
// import-cycle-avoiding pattern internal/entry's AccountLookup uses.
// *account.Service satisfies this structurally (its own Access method
// already has this exact signature).
type AccountLookup interface {
	// Access resolves callerID's effective permission on accountID — ""
	// when they have no access at all (real ownership or any share).
	// ErrNotFound only when accountID itself does not exist or is
	// soft-deleted; ignored here beyond "not found means invalid."
	Access(ctx context.Context, accountID, callerID string) (currency string, disabled bool, permission string, err error)
}

// CategoryLookup is the narrow view of internal/category that dashboard
// needs to validate a config's category_id. *category.Service satisfies
// this structurally (its Visible method exists specifically for this).
type CategoryLookup interface {
	// Visible reports whether every id in categoryIDs exists and is
	// visible to callerID (owned, or shared at view tier or above).
	Visible(ctx context.Context, callerID string, categoryIDs []string) (bool, error)
}

// TagLookup is the narrow view of internal/tag that dashboard needs to
// validate a config's tag_id. *tag.Service satisfies this structurally —
// OwnedBy already has exactly this "visible at view tier or above"
// contract for internal/entry's own TagLookup.
type TagLookup interface {
	OwnedBy(ctx context.Context, callerID string, tagIDs []string) (bool, error)
}
