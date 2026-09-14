package memory

import (
	"context"
	"sort"
	"strconv"
	"sync"

	"at.draab/familyfinances/internal/dashboard"
)

// DashboardStore is the in-memory implementation of dashboard.Store — the
// default for domain and handler tests and for local runs without a
// database. Safe for concurrent use. Unlike category/tag, there is no
// sharing and no tree — every card belongs to exactly one owner and
// sort_order orders that owner's own flat list.
type DashboardStore struct {
	mu    sync.Mutex
	cards map[string]dashboard.Card
	seq   int
}

// NewDashboardStore returns an empty DashboardStore.
func NewDashboardStore() *DashboardStore {
	return &DashboardStore{cards: map[string]dashboard.Card{}}
}

func (s *DashboardStore) nextID() string {
	s.seq++
	return "card" + strconv.Itoa(s.seq)
}

// nextSortOrder returns one past ownerID's current highest sort_order.
// Callers MUST hold s.mu.
func (s *DashboardStore) nextSortOrder(ownerID string) int {
	max := -1
	for _, c := range s.cards {
		if c.OwnerID == ownerID && c.SortOrder > max {
			max = c.SortOrder
		}
	}
	return max + 1
}

// ownedSorted returns ownerID's own cards, ordered by sort_order then id
// (a stable tiebreaker, mirroring memory.CategoryStore.siblings). Callers
// MUST hold s.mu.
func (s *DashboardStore) ownedSorted(ownerID string) []dashboard.Card {
	var out []dashboard.Card
	for _, c := range s.cards {
		if c.OwnerID == ownerID {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *DashboardStore) List(_ context.Context, ownerID string) ([]dashboard.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ownedSorted(ownerID), nil
}

func (s *DashboardStore) Get(_ context.Context, ownerID, id string) (dashboard.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[id]
	if !ok || c.OwnerID != ownerID {
		return dashboard.Card{}, dashboard.ErrNotFound
	}
	return c, nil
}

func (s *DashboardStore) Create(_ context.Context, ownerID string, in dashboard.New) (dashboard.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := dashboard.Card{
		ID:        s.nextID(),
		OwnerID:   ownerID,
		Type:      in.Type,
		Config:    in.Config,
		SortOrder: s.nextSortOrder(ownerID),
	}
	s.cards[c.ID] = c
	return c, nil
}

func (s *DashboardStore) Update(_ context.Context, ownerID, id string, upd dashboard.Update) (dashboard.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[id]
	if !ok || c.OwnerID != ownerID {
		return dashboard.Card{}, dashboard.ErrNotFound
	}
	c.Config = upd.Config
	s.cards[id] = c
	return c, nil
}

func (s *DashboardStore) Delete(_ context.Context, ownerID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[id]
	if !ok || c.OwnerID != ownerID {
		return dashboard.ErrNotFound
	}
	delete(s.cards, id)
	return nil
}

func (s *DashboardStore) MoveUp(_ context.Context, ownerID, id string) (dashboard.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[id]
	if !ok || c.OwnerID != ownerID {
		return dashboard.Card{}, dashboard.ErrNotFound
	}
	ordered := s.ownedSorted(ownerID)
	idx := indexOfCard(ordered, id)
	if idx <= 0 {
		return c, nil
	}
	prev := ordered[idx-1]
	c.SortOrder, prev.SortOrder = prev.SortOrder, c.SortOrder
	s.cards[c.ID] = c
	s.cards[prev.ID] = prev
	return c, nil
}

func (s *DashboardStore) MoveDown(_ context.Context, ownerID, id string) (dashboard.Card, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cards[id]
	if !ok || c.OwnerID != ownerID {
		return dashboard.Card{}, dashboard.ErrNotFound
	}
	ordered := s.ownedSorted(ownerID)
	idx := indexOfCard(ordered, id)
	if idx == -1 || idx >= len(ordered)-1 {
		return c, nil
	}
	next := ordered[idx+1]
	c.SortOrder, next.SortOrder = next.SortOrder, c.SortOrder
	s.cards[c.ID] = c
	s.cards[next.ID] = next
	return c, nil
}

func indexOfCard(cards []dashboard.Card, id string) int {
	for i, c := range cards {
		if c.ID == id {
			return i
		}
	}
	return -1
}
