package memory

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"at.draab/familyfinances/internal/account"
)

// AccountStore is the in-memory implementation of account.Store — the
// default for domain and handler tests and for local runs without a
// database. Safe for concurrent use.
type AccountStore struct {
	mu       sync.Mutex
	accounts map[string]account.Account
	types    map[string]account.Type
	seq      int
}

// NewAccountStore returns an empty AccountStore.
func NewAccountStore() *AccountStore {
	return &AccountStore{
		accounts: map[string]account.Account{},
		types:    map[string]account.Type{},
	}
}

func (s *AccountStore) nextID(prefix string) string {
	s.seq++
	return prefix + strconv.Itoa(s.seq)
}

func (s *AccountStore) Create(_ context.Context, ownerID string, in account.New) (account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	acc := account.Account{
		ID:                 s.nextID("acct"),
		OwnerID:            ownerID,
		Title:              in.Title,
		Description:        in.Description,
		TypeID:             in.TypeID,
		Currency:           in.Currency,
		FinancialInstitute: in.FinancialInstitute,
		OpeningDate:        in.OpeningDate,
		ClosingDate:        in.ClosingDate,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.accounts[acc.ID] = acc
	return acc, nil
}

func (s *AccountStore) Get(_ context.Context, ownerID, id string) (account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.OwnerID != ownerID || acc.DeletedAt != nil {
		return account.Account{}, account.ErrNotFound
	}
	return acc, nil
}

func (s *AccountStore) List(_ context.Context, ownerID string) ([]account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []account.Account
	for _, acc := range s.accounts {
		if acc.OwnerID == ownerID && acc.DeletedAt == nil {
			out = append(out, acc)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *AccountStore) Update(_ context.Context, ownerID, id string, upd account.Update) (account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.OwnerID != ownerID || acc.DeletedAt != nil {
		return account.Account{}, account.ErrNotFound
	}
	if upd.Title != nil {
		acc.Title = *upd.Title
	}
	if upd.Description != nil {
		acc.Description = *upd.Description
	}
	if upd.TypeID != nil {
		acc.TypeID = *upd.TypeID
	}
	if upd.Currency != nil {
		acc.Currency = *upd.Currency
	}
	if upd.FinancialInstitute != nil {
		acc.FinancialInstitute = *upd.FinancialInstitute
	}
	if upd.OpeningDate != nil {
		acc.OpeningDate = *upd.OpeningDate
	}
	if upd.ClosingDate.Set {
		acc.ClosingDate = upd.ClosingDate.Value
	}
	acc.UpdatedAt = time.Now().UTC()
	s.accounts[id] = acc
	return acc, nil
}

func (s *AccountStore) SetDisabled(_ context.Context, ownerID, id string, disabled bool) (account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.OwnerID != ownerID || acc.DeletedAt != nil {
		return account.Account{}, account.ErrNotFound
	}
	acc.Disabled = disabled
	acc.UpdatedAt = time.Now().UTC()
	s.accounts[id] = acc
	return acc, nil
}

func (s *AccountStore) SoftDelete(_ context.Context, ownerID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.OwnerID != ownerID || acc.DeletedAt != nil {
		return account.ErrNotFound
	}
	now := time.Now().UTC()
	acc.DeletedAt = &now
	s.accounts[id] = acc
	return nil
}

func (s *AccountStore) Owner(_ context.Context, id string) (string, string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.DeletedAt != nil {
		return "", "", false, account.ErrNotFound
	}
	return acc.OwnerID, acc.Currency, acc.Disabled, nil
}

func (s *AccountStore) ListTypes(_ context.Context) ([]account.Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []account.Type
	for _, t := range s.types {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *AccountStore) CreateType(_ context.Context, name string) (account.Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.types {
		if t.Name == name {
			return account.Type{}, account.ErrInvalidValue
		}
	}
	t := account.Type{ID: s.nextID("atype"), Name: name, CreatedAt: time.Now().UTC()}
	s.types[t.ID] = t
	return t, nil
}

func (s *AccountStore) UpdateType(_ context.Context, id, name string) (account.Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.types[id]
	if !ok {
		return account.Type{}, account.ErrNotFound
	}
	t.Name = name
	s.types[id] = t
	return t, nil
}

func (s *AccountStore) DeleteType(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.types[id]; !ok {
		return account.ErrNotFound
	}
	for _, acc := range s.accounts {
		if acc.TypeID == id && acc.DeletedAt == nil {
			return account.ErrTypeInUse
		}
	}
	delete(s.types, id)
	return nil
}

func (s *AccountStore) TypeExists(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.types[id]
	return ok, nil
}
