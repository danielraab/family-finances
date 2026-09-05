package entry

import (
	"context"
	"strings"
	"time"
)

// Service is the entry use-case layer. It depends on Store plus the three
// narrow lookup interfaces it declares.
type Service struct {
	store      Store
	accounts   AccountLookup
	categories CategoryLookup
	tags       TagLookup
}

// NewService builds the entry service.
func NewService(store Store, accounts AccountLookup, categories CategoryLookup, tags TagLookup) *Service {
	return &Service{store: store, accounts: accounts, categories: categories, tags: tags}
}

// Create validates in, confirms ownerID owns the target account and that it
// is not disabled, confirms any category/tags are usable, and creates the
// entry.
func (s *Service) Create(ctx context.Context, ownerID string, in New) (Entry, error) {
	if err := validateNew(in); err != nil {
		return Entry{}, err
	}

	accOwner, _, disabled, err := s.accounts.Owner(ctx, in.AccountID)
	if err != nil || accOwner != ownerID {
		return Entry{}, ErrInvalidValue
	}
	if disabled {
		return Entry{}, ErrAccountDisabled
	}

	if in.CategoryID != nil {
		ok, err := s.categories.Exists(ctx, *in.CategoryID)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrInvalidValue
		}
	}
	if len(in.TagIDs) > 0 {
		ok, err := s.tags.OwnedBy(ctx, ownerID, in.TagIDs)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrInvalidValue
		}
	}

	return s.store.Create(ctx, ownerID, in)
}

// Get returns ownerID's entry with this id.
func (s *Service) Get(ctx context.Context, ownerID, id string) (Entry, error) {
	return s.store.Get(ctx, ownerID, id)
}

// Update validates and applies a partial change to ownerID's entry.
// AccountID and Kind cannot be changed — there is no field for them on
// Update at all (see its doc comment).
func (s *Service) Update(ctx context.Context, ownerID, id string, upd Update) (Entry, error) {
	current, err := s.store.Get(ctx, ownerID, id)
	if err != nil {
		return Entry{}, err
	}

	if upd.Title != nil && strings.TrimSpace(*upd.Title) == "" {
		return Entry{}, ErrInvalidValue
	}
	if upd.BookingTimestamp != nil && upd.BookingTimestamp.IsZero() {
		return Entry{}, ErrInvalidValue
	}

	newCategoryID := current.CategoryID
	if upd.CategoryID.Set {
		newCategoryID = upd.CategoryID.Value
	}
	if current.Kind == KindTransaction && newCategoryID == nil {
		return Entry{}, ErrInvalidValue
	}
	if newCategoryID != nil {
		ok, err := s.categories.Exists(ctx, *newCategoryID)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrInvalidValue
		}
	}

	if upd.TagIDs != nil && len(*upd.TagIDs) > 0 {
		ok, err := s.tags.OwnedBy(ctx, ownerID, *upd.TagIDs)
		if err != nil {
			return Entry{}, err
		}
		if !ok {
			return Entry{}, ErrInvalidValue
		}
	}

	return s.store.Update(ctx, ownerID, id, upd)
}

// Delete soft-deletes ownerID's entry.
func (s *Service) Delete(ctx context.Context, ownerID, id string) error {
	return s.store.SoftDelete(ctx, ownerID, id)
}

// List resolves f's caller-supplied AccountIDs/CategoryID against the
// caller's visible accounts and the category tree, applies defaults for
// Sort/Dir/Limit, and returns a page of ownerID's matching entries.
func (s *Service) List(ctx context.Context, ownerID string, f Filter) ([]Entry, *Cursor, error) {
	if f.Sort == "" {
		f.Sort = SortBookingTimestamp
	} else if !f.Sort.valid() {
		return nil, nil, ErrInvalidValue
	}
	if f.Dir == "" {
		f.Dir = DirDesc
	} else if !f.Dir.valid() {
		return nil, nil, ErrInvalidValue
	}
	if f.Limit <= 0 || f.Limit > maxPageSize {
		f.Limit = defaultPageSize
	}
	if f.Kind != nil && !f.Kind.valid() {
		return nil, nil, ErrInvalidValue
	}

	visible, err := s.accounts.VisibleIDs(ctx, ownerID)
	if err != nil {
		return nil, nil, err
	}
	if len(f.AccountIDs) > 0 {
		f.AccountIDs = intersect(f.AccountIDs, visible)
	} else {
		f.AccountIDs = visible
	}

	if f.CategoryID != nil {
		ids, err := s.categories.Subtree(ctx, *f.CategoryID)
		if err != nil {
			return nil, nil, err
		}
		f.CategoryIDs = ids
	}

	return s.store.List(ctx, ownerID, f)
}

// Balance confirms ownerID owns accountID, then returns its live balance as
// of asOf.
func (s *Service) Balance(ctx context.Context, ownerID, accountID string, asOf time.Time) (int64, error) {
	owner, _, _, err := s.accounts.Owner(ctx, accountID)
	if err != nil {
		return 0, err
	}
	if owner != ownerID {
		return 0, ErrNotFound
	}
	return s.store.Balance(ctx, accountID, asOf)
}

// intersect returns the elements of a that also appear in b.
func intersect(a, b []string) []string {
	set := make(map[string]bool, len(b))
	for _, id := range b {
		set[id] = true
	}
	var out []string
	for _, id := range a {
		if set[id] {
			out = append(out, id)
		}
	}
	return out
}
