package memory

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"at.draab/familyfinances/internal/category"
)

// CategoryStore is the in-memory implementation of category.Store — the
// default for domain and handler tests and for local runs without a
// database. Safe for concurrent use. It has no visibility into entries, so
// unlike the real backend it cannot reject deleting a category that is
// still referenced by one — see category.ErrInUse's doc comment.
type CategoryStore struct {
	mu   sync.Mutex
	cats map[string]category.Category
	seq  int
}

// NewCategoryStore returns an empty CategoryStore.
func NewCategoryStore() *CategoryStore {
	return &CategoryStore{cats: map[string]category.Category{}}
}

func (s *CategoryStore) nextID() string {
	s.seq++
	return "cat" + strconv.Itoa(s.seq)
}

func (s *CategoryStore) List(_ context.Context) ([]category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []category.Category
	for _, c := range s.cats {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *CategoryStore) Get(_ context.Context, id string) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok {
		return category.Category{}, category.ErrNotFound
	}
	return c, nil
}

func (s *CategoryStore) Create(_ context.Context, in category.New) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := category.Category{ID: s.nextID(), ParentID: in.ParentID, Name: in.Name, CreatedAt: time.Now().UTC()}
	s.cats[c.ID] = c
	return c, nil
}

func (s *CategoryStore) Update(_ context.Context, id string, upd category.Update) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok {
		return category.Category{}, category.ErrNotFound
	}
	if upd.Name != nil {
		c.Name = *upd.Name
	}
	if upd.ParentID.Set {
		c.ParentID = upd.ParentID.Value
	}
	s.cats[id] = c
	return c, nil
}

func (s *CategoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cats[id]; !ok {
		return category.ErrNotFound
	}
	for _, c := range s.cats {
		if c.ParentID != nil && *c.ParentID == id {
			return category.ErrInUse
		}
	}
	delete(s.cats, id)
	return nil
}

func (s *CategoryStore) Exists(_ context.Context, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.cats[id]
	return ok, nil
}

func (s *CategoryStore) Subtree(_ context.Context, id string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	children := map[string][]string{}
	for _, c := range s.cats {
		if c.ParentID != nil {
			children[*c.ParentID] = append(children[*c.ParentID], c.ID)
		}
	}
	out := []string{id}
	queue := []string{id}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range children[cur] {
			out = append(out, child)
			queue = append(queue, child)
		}
	}
	return out, nil
}
