package account

import (
	"context"
	"errors"
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

// resolveAssignableType checks that id may be (re)assigned as an account's
// type_id: it must exist, belong to ownerID, and not be disabled. A missing
// or cross-owner type_id is ErrInvalidValue rather than ErrNotFound — it's
// caller-supplied input, the same contract the old TypeExists-based check
// had.
func (s *Service) resolveAssignableType(ctx context.Context, ownerID, id string) error {
	t, err := s.store.GetType(ctx, ownerID, id)
	if errors.Is(err, ErrNotFound) {
		return ErrInvalidValue
	}
	if err != nil {
		return err
	}
	if t.Disabled {
		return ErrTypeDisabled
	}
	return nil
}

// Create validates in and creates an account owned by ownerID.
func (s *Service) Create(ctx context.Context, ownerID string, in New) (Account, error) {
	if err := validateNew(in); err != nil {
		return Account{}, err
	}
	if err := s.resolveAssignableType(ctx, ownerID, in.TypeID); err != nil {
		return Account{}, err
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

// Update validates and applies a partial change to ownerID's account. The
// account's *effective* type_id — upd.TypeID if provided, else its current
// one — must resolve to a non-disabled type: an account whose current type
// has since been disabled therefore rejects every update (whatever else it
// changes) until that same request also supplies a type_id for a
// different, live type.
func (s *Service) Update(ctx context.Context, ownerID, id string, upd Update) (Account, error) {
	current, err := s.store.Get(ctx, ownerID, id)
	if err != nil {
		return Account{}, err
	}
	if err := validateUpdate(current, upd); err != nil {
		return Account{}, err
	}
	effectiveTypeID := current.TypeID
	if upd.TypeID != nil {
		effectiveTypeID = *upd.TypeID
	}
	if err := s.resolveAssignableType(ctx, ownerID, effectiveTypeID); err != nil {
		return Account{}, err
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

// ListTypes returns every account type ownerID owns, including disabled
// ones.
func (s *Service) ListTypes(ctx context.Context, ownerID string) ([]Type, error) {
	return s.store.ListTypes(ctx, ownerID)
}

// CreateType creates a new account type owned by ownerID. Any authenticated
// user may create their own types — there is no admin gate.
func (s *Service) CreateType(ctx context.Context, ownerID, title, description string) (Type, error) {
	if strings.TrimSpace(title) == "" {
		return Type{}, ErrInvalidValue
	}
	return s.store.CreateType(ctx, ownerID, title, description)
}

// UpdateType changes ownerID's account type's title and description.
func (s *Service) UpdateType(ctx context.Context, ownerID, id, title, description string) (Type, error) {
	if strings.TrimSpace(title) == "" {
		return Type{}, ErrInvalidValue
	}
	return s.store.UpdateType(ctx, ownerID, id, title, description)
}

// DisableType blocks a type from being (re)assigned to an account, without
// affecting any account already carrying it. Reversible via EnableType.
func (s *Service) DisableType(ctx context.Context, ownerID, id string) (Type, error) {
	return s.store.SetTypeDisabled(ctx, ownerID, id, true)
}

// EnableType reverses DisableType.
func (s *Service) EnableType(ctx context.Context, ownerID, id string) (Type, error) {
	return s.store.SetTypeDisabled(ctx, ownerID, id, false)
}

// DeleteType deletes ownerID's account type, or ErrTypeInUse if a
// non-deleted account still references it — regardless of whether it is
// disabled.
func (s *Service) DeleteType(ctx context.Context, ownerID, id string) error {
	return s.store.DeleteType(ctx, ownerID, id)
}

// SeedDefaults seeds ownerID — a brand-new user — with DefaultTypeTitles.
// It satisfies internal/auth's NewUserHook interface structurally, wired in
// by package main via auth.WithNewUserHooks.
func (s *Service) SeedDefaults(ctx context.Context, ownerID string) error {
	return s.store.SeedDefaultTypes(ctx, ownerID)
}
