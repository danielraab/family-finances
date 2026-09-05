package tag

import (
	"context"
	"errors"
)

// Service is the tag use-case layer. It depends only on the Store
// interface.
type Service struct {
	store Store
}

// NewService builds the tag service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns every tag ownerID owns.
func (s *Service) List(ctx context.Context, ownerID string) ([]Tag, error) {
	return s.store.List(ctx, ownerID)
}

// Get returns ownerID's tag with this id.
func (s *Service) Get(ctx context.Context, ownerID, id string) (Tag, error) {
	return s.store.Get(ctx, ownerID, id)
}

// Create validates name and creates a tag owned by ownerID, or
// ErrDuplicateName if they already have one with this name.
func (s *Service) Create(ctx context.Context, ownerID, name string) (Tag, error) {
	if err := validateName(name); err != nil {
		return Tag{}, err
	}
	return s.store.Create(ctx, ownerID, name)
}

// GetOrCreate returns ownerID's existing tag named name, creating it first
// if none exists yet — the inline "create a tag on the fly from the entry
// form" path, so the caller never has to handle ErrDuplicateName itself.
func (s *Service) GetOrCreate(ctx context.Context, ownerID, name string) (Tag, error) {
	if err := validateName(name); err != nil {
		return Tag{}, err
	}
	existing, err := s.store.ByName(ctx, ownerID, name)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Tag{}, err
	}
	created, err := s.store.Create(ctx, ownerID, name)
	if errors.Is(err, ErrDuplicateName) {
		// Lost a race with a concurrent create of the same name.
		return s.store.ByName(ctx, ownerID, name)
	}
	return created, err
}

// Update renames ownerID's tag.
func (s *Service) Update(ctx context.Context, ownerID, id, name string) (Tag, error) {
	if err := validateName(name); err != nil {
		return Tag{}, err
	}
	return s.store.Update(ctx, ownerID, id, name)
}

// Delete removes ownerID's tag, detaching it from every entry it was
// attached to (enforced by the real backend's ON DELETE CASCADE on
// entry_tags.tag_id) — always allowed, regardless of use.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.Delete(ctx, ownerID, id)
}

// OwnedBy satisfies internal/entry's TagLookup interface.
func (s *Service) OwnedBy(ctx context.Context, ownerID string, tagIDs []string) (bool, error) {
	return s.store.OwnedBy(ctx, ownerID, tagIDs)
}
