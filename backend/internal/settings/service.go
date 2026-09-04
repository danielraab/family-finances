package settings

import "context"

// Service is the settings use-case layer. It depends only on the Store
// interface.
type Service struct {
	store Store
}

// NewService builds the settings service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Get returns userID's fully-resolved settings.
func (s *Service) Get(ctx context.Context, userID string) (Settings, error) {
	row, err := s.store.Get(ctx, userID)
	if err != nil {
		return Settings{}, err
	}
	return Resolve(row), nil
}

// Update validates and applies a partial change, returning the resulting
// resolved settings. A field that fails validation rejects the whole update
// without changing anything.
func (s *Service) Update(ctx context.Context, userID string, upd Update) (Settings, error) {
	if upd.Language != nil {
		if err := ValidateLanguage(*upd.Language); err != nil {
			return Settings{}, err
		}
	}
	if upd.Timezone != nil {
		if err := ValidateTimezone(*upd.Timezone); err != nil {
			return Settings{}, err
		}
	}
	if upd.DefaultCurrency != nil {
		if err := ValidateCurrency(*upd.DefaultCurrency); err != nil {
			return Settings{}, err
		}
	}

	row, err := s.store.Upsert(ctx, userID, upd)
	if err != nil {
		return Settings{}, err
	}
	return Resolve(row), nil
}

// Language returns userID's raw (unresolved) language preference — nil when
// unset. It satisfies internal/auth's LanguageLookup interface, used to embed
// the raw preference on GET /api/auth/me for the client's i18n precedence
// (see design.md's "raw preference, not the resolved one" decision): unlike
// Get, this does not substitute the hardcoded default, since the client
// still needs to fall back to browser detection when nothing is set.
func (s *Service) Language(ctx context.Context, userID string) (*string, error) {
	row, err := s.store.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return row.Language, nil
}
