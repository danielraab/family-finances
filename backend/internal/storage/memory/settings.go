package memory

import (
	"context"
	"sync"

	"at.draab/familyfinances/internal/settings"
)

// SettingsStore is the in-memory implementation of settings.Store — the
// default for tests and local runs without a database. Safe for concurrent
// use.
type SettingsStore struct {
	mu   sync.Mutex
	rows map[string]settings.Row
}

// NewSettingsStore returns an empty SettingsStore.
func NewSettingsStore() *SettingsStore {
	return &SettingsStore{rows: map[string]settings.Row{}}
}

func (s *SettingsStore) Get(_ context.Context, userID string) (settings.Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rows[userID], nil
}

func (s *SettingsStore) Upsert(_ context.Context, userID string, upd settings.Update) (settings.Row, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.rows[userID]
	if upd.Language != nil {
		row.Language = upd.Language
	}
	if upd.Timezone != nil {
		row.Timezone = upd.Timezone
	}
	if upd.DefaultCurrency != nil {
		row.DefaultCurrency = upd.DefaultCurrency
	}
	if upd.DisplayedDecimalPlaces != nil {
		row.DisplayedDecimalPlaces = upd.DisplayedDecimalPlaces
	}
	s.rows[userID] = row
	return row, nil
}
