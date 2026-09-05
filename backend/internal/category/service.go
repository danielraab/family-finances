package category

import "context"

// Service is the category use-case layer. It depends only on the Store
// interface.
type Service struct {
	store Store
}

// NewService builds the category service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List returns the full category tree.
func (s *Service) List(ctx context.Context) ([]Category, error) {
	return s.store.List(ctx)
}

// Get returns a single category.
func (s *Service) Get(ctx context.Context, id string) (Category, error) {
	return s.store.Get(ctx, id)
}

// Create validates in and, when it sets a ParentID, checks it exists.
// Authorization (admin-only) is the handler's responsibility.
func (s *Service) Create(ctx context.Context, in New) (Category, error) {
	if err := validateName(in.Name); err != nil {
		return Category{}, err
	}
	if in.ParentID != nil {
		ok, err := s.store.Exists(ctx, *in.ParentID)
		if err != nil {
			return Category{}, err
		}
		if !ok {
			return Category{}, ErrInvalidValue
		}
	}
	return s.store.Create(ctx, in)
}

// Update validates upd, rejecting a reparent onto the category itself or
// one of its own descendants (ErrCycle).
func (s *Service) Update(ctx context.Context, id string, upd Update) (Category, error) {
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
		ok, err := s.store.Exists(ctx, newParent)
		if err != nil {
			return Category{}, err
		}
		if !ok {
			return Category{}, ErrInvalidValue
		}
		descendants, err := s.store.Subtree(ctx, id)
		if err != nil {
			return Category{}, err
		}
		for _, d := range descendants {
			if d == newParent {
				return Category{}, ErrCycle
			}
		}
	}
	return s.store.Update(ctx, id, upd)
}

// Delete removes a category, or ErrInUse if it has children or is
// referenced elsewhere.
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// Exists satisfies internal/entry's CategoryLookup interface.
func (s *Service) Exists(ctx context.Context, id string) (bool, error) {
	return s.store.Exists(ctx, id)
}

// Subtree satisfies internal/entry's CategoryLookup interface: it returns id
// and every descendant id, so a category filter can be resolved to "this
// category or any of its descendants."
func (s *Service) Subtree(ctx context.Context, id string) ([]string, error) {
	return s.store.Subtree(ctx, id)
}
