package account

import (
	"context"
	"strings"
)

// Service is the account use-case layer. It depends only on the Store
// interface.
type Service struct {
	store Store
}

// NewService builds the account service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create validates in and creates an account owned by ownerID.
func (s *Service) Create(ctx context.Context, ownerID string, in New) (Account, error) {
	if err := validateNew(in); err != nil {
		return Account{}, err
	}
	ok, err := s.store.TypeExists(ctx, in.TypeID)
	if err != nil {
		return Account{}, err
	}
	if !ok {
		return Account{}, ErrInvalidValue
	}
	return s.store.Create(ctx, ownerID, in)
}

// Get returns ownerID's account with this id.
func (s *Service) Get(ctx context.Context, ownerID, id string) (Account, error) {
	return s.store.Get(ctx, ownerID, id)
}

// List returns every non-deleted account ownerID owns.
func (s *Service) List(ctx context.Context, ownerID string) ([]Account, error) {
	return s.store.List(ctx, ownerID)
}

// VisibleIDs returns the id of every non-deleted account ownerID owns.
// Satisfies internal/entry's AccountLookup interface, used to scope an
// entry listing or balance query to accounts that still exist.
func (s *Service) VisibleIDs(ctx context.Context, ownerID string) ([]string, error) {
	accounts, err := s.store.List(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(accounts))
	for i, a := range accounts {
		ids[i] = a.ID
	}
	return ids, nil
}

// Update validates and applies a partial change to ownerID's account.
func (s *Service) Update(ctx context.Context, ownerID, id string, upd Update) (Account, error) {
	current, err := s.store.Get(ctx, ownerID, id)
	if err != nil {
		return Account{}, err
	}
	if err := validateUpdate(current, upd); err != nil {
		return Account{}, err
	}
	if upd.TypeID != nil {
		ok, err := s.store.TypeExists(ctx, *upd.TypeID)
		if err != nil {
			return Account{}, err
		}
		if !ok {
			return Account{}, ErrInvalidValue
		}
	}
	return s.store.Update(ctx, ownerID, id, upd)
}

// Disable blocks creating new entries against the account without hiding it
// or affecting its existing entries. Reversible via Enable.
func (s *Service) Disable(ctx context.Context, ownerID, id string) (Account, error) {
	return s.store.SetDisabled(ctx, ownerID, id, true)
}

// Enable reverses Disable.
func (s *Service) Enable(ctx context.Context, ownerID, id string) (Account, error) {
	return s.store.SetDisabled(ctx, ownerID, id, false)
}

// Delete soft-deletes ownerID's account.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.SoftDelete(ctx, ownerID, id)
}

// Owner satisfies internal/entry's AccountLookup interface: it returns id's
// owner, currency, and disabled flag so entry.Service can check the caller
// owns it and whether entry creation is currently blocked.
func (s *Service) Owner(ctx context.Context, id string) (ownerID string, currency string, disabled bool, err error) {
	return s.store.Owner(ctx, id)
}

// ListTypes returns every account type.
func (s *Service) ListTypes(ctx context.Context) ([]Type, error) {
	return s.store.ListTypes(ctx)
}

// CreateType creates a new account type. Authorization (admin-only) is the
// handler's responsibility, matching internal/auth's convention.
func (s *Service) CreateType(ctx context.Context, name string) (Type, error) {
	if strings.TrimSpace(name) == "" {
		return Type{}, ErrInvalidValue
	}
	return s.store.CreateType(ctx, name)
}

// UpdateType renames an account type.
func (s *Service) UpdateType(ctx context.Context, id, name string) (Type, error) {
	if strings.TrimSpace(name) == "" {
		return Type{}, ErrInvalidValue
	}
	return s.store.UpdateType(ctx, id, name)
}

// DeleteType deletes an account type, or ErrTypeInUse if a non-deleted
// account still references it.
func (s *Service) DeleteType(ctx context.Context, id string) error {
	return s.store.DeleteType(ctx, id)
}
