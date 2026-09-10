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
//
// It has no visibility into internal/auth's users, the same accepted gap
// internal/storage/memory's TagStore has for entries (see tag.go's
// EntryCount): AccountShare.Name/Email/GrantedByName and Account.OwnerName
// are always left empty here — fine, since storage/memory is test/local-dev
// infrastructure, never what ships.
type AccountStore struct {
	mu       sync.Mutex
	accounts map[string]account.Account
	shares   map[string]account.AccountShare // key: shareKey(accountID, userID)
	types    map[string]account.Type
	seq      int
}

// NewAccountStore returns an empty AccountStore.
func NewAccountStore() *AccountStore {
	return &AccountStore{
		accounts: map[string]account.Account{},
		shares:   map[string]account.AccountShare{},
		types:    map[string]account.Type{},
	}
}

func (s *AccountStore) nextID(prefix string) string {
	s.seq++
	return prefix + strconv.Itoa(s.seq)
}

func shareKey(accountID, userID string) string { return accountID + "|" + userID }

// permissionLocked resolves callerID's Permission on acc (id is acc's own
// key, passed separately to avoid a second map lookup). Callers MUST hold
// s.mu.
func (s *AccountStore) permissionLocked(id, callerID string, acc account.Account) account.Permission {
	if acc.OwnerID == callerID {
		return account.PermissionOwner
	}
	if sh, ok := s.shares[shareKey(id, callerID)]; ok {
		return sh.Permission
	}
	return ""
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
		Icon:               in.Icon,
		Color:              in.Color,
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

func (s *AccountStore) Get(_ context.Context, id, callerID string) (account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.DeletedAt != nil {
		return account.Account{}, account.ErrNotFound
	}
	acc.Permission = s.permissionLocked(id, callerID, acc)
	acc.Shared = acc.Permission != "" && acc.OwnerID != callerID
	return acc, nil
}

func (s *AccountStore) List(_ context.Context, callerID string) ([]account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []account.Account
	for id, acc := range s.accounts {
		if acc.DeletedAt != nil {
			continue
		}
		perm := s.permissionLocked(id, callerID, acc)
		if perm == "" {
			continue
		}
		acc.Permission = perm
		acc.Shared = acc.OwnerID != callerID
		out = append(out, acc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *AccountStore) Update(_ context.Context, id string, upd account.Update) (account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.DeletedAt != nil {
		return account.Account{}, account.ErrNotFound
	}
	if upd.Title != nil {
		acc.Title = *upd.Title
	}
	if upd.Description != nil {
		acc.Description = *upd.Description
	}
	if upd.Icon != nil {
		acc.Icon = *upd.Icon
	}
	if upd.Color != nil {
		acc.Color = *upd.Color
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

func (s *AccountStore) SetDisabled(_ context.Context, id string, disabled bool) (account.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.DeletedAt != nil {
		return account.Account{}, account.ErrNotFound
	}
	acc.Disabled = disabled
	acc.UpdatedAt = time.Now().UTC()
	s.accounts[id] = acc
	return acc, nil
}

func (s *AccountStore) SoftDelete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.DeletedAt != nil {
		return account.ErrNotFound
	}
	now := time.Now().UTC()
	acc.DeletedAt = &now
	s.accounts[id] = acc
	return nil
}

func (s *AccountStore) Access(_ context.Context, id, callerID string) (account.Access, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[id]
	if !ok || acc.DeletedAt != nil {
		return account.Access{}, account.ErrNotFound
	}
	return account.Access{
		Currency:   acc.Currency,
		Disabled:   acc.Disabled,
		Permission: s.permissionLocked(id, callerID, acc),
	}, nil
}

func (s *AccountStore) VisibleIDs(_ context.Context, callerID string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []string
	for id, acc := range s.accounts {
		if acc.DeletedAt != nil {
			continue
		}
		if s.permissionLocked(id, callerID, acc) != "" {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// --- sharing -----------------------------------------------------------

func (s *AccountStore) CreateOrUpdateShare(_ context.Context, accountID, userID string, permission account.Permission, grantedBy string) (account.AccountShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := shareKey(accountID, userID)
	now := time.Now().UTC()
	sh, exists := s.shares[key]
	if exists {
		sh.Permission = permission
		sh.GrantedBy = grantedBy
		sh.UpdatedAt = now
	} else {
		sh = account.AccountShare{
			AccountID:  accountID,
			UserID:     userID,
			Permission: permission,
			GrantedBy:  grantedBy,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	}
	s.shares[key] = sh
	return sh, nil
}

func (s *AccountStore) ListShares(_ context.Context, accountID string) ([]account.AccountShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []account.AccountShare
	for _, sh := range s.shares {
		if sh.AccountID == accountID {
			out = append(out, sh)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *AccountStore) ShareByUser(_ context.Context, accountID, userID string) (account.AccountShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, ok := s.shares[shareKey(accountID, userID)]
	if !ok {
		return account.AccountShare{}, account.ErrNotFound
	}
	return sh, nil
}

func (s *AccountStore) UpdateSharePermission(_ context.Context, accountID, userID string, permission account.Permission) (account.AccountShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := shareKey(accountID, userID)
	sh, ok := s.shares[key]
	if !ok {
		return account.AccountShare{}, account.ErrNotFound
	}
	sh.Permission = permission
	sh.UpdatedAt = time.Now().UTC()
	s.shares[key] = sh
	return sh, nil
}

func (s *AccountStore) DeleteShare(_ context.Context, accountID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := shareKey(accountID, userID)
	if _, ok := s.shares[key]; !ok {
		return account.ErrNotFound
	}
	delete(s.shares, key)
	return nil
}

// --- account types -------------------------------------------------------

func (s *AccountStore) ListTypes(_ context.Context, ownerID string) ([]account.Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []account.Type
	for _, t := range s.types {
		if t.OwnerID == ownerID {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *AccountStore) GetType(_ context.Context, ownerID, id string) (account.Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.types[id]
	if !ok || t.OwnerID != ownerID {
		return account.Type{}, account.ErrNotFound
	}
	return t, nil
}

func (s *AccountStore) CreateType(_ context.Context, ownerID, title, description string) (account.Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := account.Type{
		ID:          s.nextID("atype"),
		OwnerID:     ownerID,
		Title:       title,
		Description: description,
		CreatedAt:   time.Now().UTC(),
	}
	s.types[t.ID] = t
	return t, nil
}

func (s *AccountStore) UpdateType(_ context.Context, ownerID, id, title, description string) (account.Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.types[id]
	if !ok || t.OwnerID != ownerID {
		return account.Type{}, account.ErrNotFound
	}
	t.Title = title
	t.Description = description
	s.types[id] = t
	return t, nil
}

func (s *AccountStore) SetTypeDisabled(_ context.Context, ownerID, id string, disabled bool) (account.Type, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.types[id]
	if !ok || t.OwnerID != ownerID {
		return account.Type{}, account.ErrNotFound
	}
	t.Disabled = disabled
	s.types[id] = t
	return t, nil
}

func (s *AccountStore) DeleteType(_ context.Context, ownerID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.types[id]
	if !ok || t.OwnerID != ownerID {
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

func (s *AccountStore) SeedDefaultTypes(_ context.Context, ownerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for _, title := range account.DefaultTypeTitles {
		t := account.Type{ID: s.nextID("atype"), OwnerID: ownerID, Title: title, CreatedAt: now}
		s.types[t.ID] = t
	}
	return nil
}
