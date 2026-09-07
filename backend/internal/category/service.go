package category

import (
	"context"
	"errors"
)

// Service is the category use-case layer. It depends only on the Store
// interface. Every method is scoped to ownerID — the authenticated caller —
// with no admin override; a category belonging to a different owner reads
// as ErrNotFound.
type Service struct {
	store Store
}

// NewService builds the category service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns ownerID's full category tree.
func (s *Service) List(ctx context.Context, ownerID string) ([]Category, error) {
	return s.store.List(ctx, ownerID)
}

// Get returns ownerID's category with this id.
func (s *Service) Get(ctx context.Context, ownerID, id string) (Category, error) {
	return s.store.Get(ctx, ownerID, id)
}

// Create validates in and, when it sets a ParentID, checks it exists among
// ownerID's own categories.
func (s *Service) Create(ctx context.Context, ownerID string, in New) (Category, error) {
	if err := validateName(in.Name); err != nil {
		return Category{}, err
	}
	if in.ParentID != nil {
		ok, err := s.store.Exists(ctx, ownerID, *in.ParentID)
		if err != nil {
			return Category{}, err
		}
		if !ok {
			return Category{}, ErrInvalidValue
		}
	}
	return s.store.Create(ctx, ownerID, in)
}

// Update validates upd, rejecting a reparent onto the category itself or
// one of its own descendants (ErrCycle), scoped to ownerID's own tree.
func (s *Service) Update(ctx context.Context, ownerID, id string, upd Update) (Category, error) {
	if upd.Name != nil {
		if err := validateName(*upd.Name); err != nil {
			return Category{}, err
		}
	}
	if upd.ParentID.Set && upd.ParentID.Value != nil {
		newParent := *upd.ParentID.Value
		if newParent == id {
			return Category{}, ErrCycle
		}
		ok, err := s.store.Exists(ctx, ownerID, newParent)
		if err != nil {
			return Category{}, err
		}
		if !ok {
			return Category{}, ErrInvalidValue
		}
		descendants, err := s.store.Subtree(ctx, ownerID, id)
		if err != nil {
			return Category{}, err
		}
		for _, d := range descendants {
			if d == newParent {
				return Category{}, ErrCycle
			}
		}
	}
	return s.store.Update(ctx, ownerID, id, upd)
}

// Delete soft-deletes ownerID's category, or ErrInUse if it has a
// non-deleted child or is referenced elsewhere.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.Delete(ctx, ownerID, id)
}

// Disable blocks the category from being newly selected on an entry,
// without affecting any entry or child category already referencing it.
// Reversible via Enable.
func (s *Service) Disable(ctx context.Context, ownerID, id string) (Category, error) {
	return s.store.SetDisabled(ctx, ownerID, id, true)
}

// Enable reverses Disable.
func (s *Service) Enable(ctx context.Context, ownerID, id string) (Category, error) {
	return s.store.SetDisabled(ctx, ownerID, id, false)
}

// MoveUp moves ownerID's category up one position among its current
// siblings. A no-op if it is already first.
func (s *Service) MoveUp(ctx context.Context, ownerID, id string) (Category, error) {
	return s.store.MoveUp(ctx, ownerID, id)
}

// MoveDown moves ownerID's category down one position among its current
// siblings. A no-op if it is already last.
func (s *Service) MoveDown(ctx context.Context, ownerID, id string) (Category, error) {
	return s.store.MoveDown(ctx, ownerID, id)
}

// Usable satisfies internal/entry's CategoryLookup interface: it reports
// whether id exists, is owned by ownerID, and is not disabled — the check
// for a category being newly set (created, or explicitly changed) on an
// entry. A disabled category already referenced by an entry stays valid
// there; this method is only consulted when the caller is newly setting
// the field, never merely carrying over an entry's existing value.
func (s *Service) Usable(ctx context.Context, ownerID, id string) (bool, error) {
	c, err := s.store.Get(ctx, ownerID, id)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !c.Disabled, nil
}

// Subtree satisfies internal/entry's CategoryLookup interface: it returns id
// and every descendant id within ownerID's own tree, so a category filter
// can be resolved to "this category or any of its descendants."
func (s *Service) Subtree(ctx context.Context, ownerID, id string) ([]string, error) {
	return s.store.Subtree(ctx, ownerID, id)
}

// SeedDefaults seeds ownerID — a brand-new user — with DefaultNames as root
// categories. It satisfies internal/auth's NewUserHook interface
// structurally, wired in by package main via auth.WithNewUserHooks.
func (s *Service) SeedDefaults(ctx context.Context, ownerID string) error {
	return s.store.SeedDefaults(ctx, ownerID)
}
