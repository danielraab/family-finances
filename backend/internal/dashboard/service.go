package dashboard

import "context"

// Service is the dashboard use-case layer. It depends on the Store
// interface plus the three lookup interfaces used to validate a config's
// account_id/category_id/tag_id against the caller's real access. Every
// method is scoped to ownerID — there is no sharing and no admin
// override; acting on another user's card id reads as ErrNotFound.
type Service struct {
	store      Store
	accounts   AccountLookup
	categories CategoryLookup
	tags       TagLookup
}

// NewService builds the dashboard service. All three lookups are
// required — unlike category/tag's optional UserLookup, there is no
// useful degraded mode for a card whose references can never be
// validated.
func NewService(store Store, accounts AccountLookup, categories CategoryLookup, tags TagLookup) *Service {
	return &Service{store: store, accounts: accounts, categories: categories, tags: tags}
}

// List returns every one of ownerID's own cards, in display order.
func (s *Service) List(ctx context.Context, ownerID string) ([]Card, error) {
	return s.store.List(ctx, ownerID)
}

// Create validates in's shape and references, then appends the card to
// the end of ownerID's own list.
func (s *Service) Create(ctx context.Context, ownerID string, in New) (Card, error) {
	if err := validateShape(in.Type, in.Config); err != nil {
		return Card{}, err
	}
	if err := s.validateReferences(ctx, ownerID, in.Config); err != nil {
		return Card{}, err
	}
	return s.store.Create(ctx, ownerID, in)
}

// Update validates upd's config against the card's existing, immutable
// type, then replaces it. A type field the caller may have sent alongside
// config is never consulted — only Config is ever applied.
func (s *Service) Update(ctx context.Context, ownerID, id string, upd Update) (Card, error) {
	current, err := s.store.Get(ctx, ownerID, id)
	if err != nil {
		return Card{}, err
	}
	if err := validateShape(current.Type, upd.Config); err != nil {
		return Card{}, err
	}
	if err := s.validateReferences(ctx, ownerID, upd.Config); err != nil {
		return Card{}, err
	}
	return s.store.Update(ctx, ownerID, id, upd)
}

// Delete removes ownerID's own card.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.Delete(ctx, ownerID, id)
}

// MoveUp swaps id's position with its immediately preceding card. Never
// re-validates the card's config — a reference gone stale since creation
// does not block reordering.
func (s *Service) MoveUp(ctx context.Context, ownerID, id string) (Card, error) {
	return s.store.MoveUp(ctx, ownerID, id)
}

// MoveDown swaps id's position with its immediately following card. Same
// no-re-validation rule as MoveUp.
func (s *Service) MoveDown(ctx context.Context, ownerID, id string) (Card, error) {
	return s.store.MoveDown(ctx, ownerID, id)
}

// validateReferences confirms every account_id/category_id/tag_id in c is
// visible to callerID — any lookup error (including the referenced entity
// simply not existing) collapses to ErrInvalidValue rather than
// propagating, mirroring internal/entry's checkAccount convention: a
// filter reference either resolves for this caller or it doesn't, and
// callerID never gets a 500 for pointing at someone else's private data.
func (s *Service) validateReferences(ctx context.Context, callerID string, c Config) error {
	if c.AccountID != nil {
		_, _, permission, err := s.accounts.Access(ctx, *c.AccountID, callerID)
		if err != nil || permission == "" {
			return ErrInvalidValue
		}
	}
	if c.CategoryID != nil {
		ok, err := s.categories.Visible(ctx, callerID, []string{*c.CategoryID})
		if err != nil || !ok {
			return ErrInvalidValue
		}
	}
	if c.TagID != nil {
		ok, err := s.tags.OwnedBy(ctx, callerID, []string{*c.TagID})
		if err != nil || !ok {
			return ErrInvalidValue
		}
	}
	return nil
}
