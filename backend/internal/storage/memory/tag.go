package memory

import (
	"context"
	"sort"
	"strconv"
	"sync"
	"time"

	"at.draab/familyfinances/internal/tag"
)

// TagStore is the in-memory implementation of tag.Store — the default for
// domain and handler tests and for local runs without a database. Safe for
// concurrent use.
type TagStore struct {
	mu   sync.Mutex
	tags map[string]tag.Tag
	seq  int
}

// NewTagStore returns an empty TagStore.
func NewTagStore() *TagStore {
	return &TagStore{tags: map[string]tag.Tag{}}
}

func (s *TagStore) nextID() string {
	s.seq++
	return "tag" + strconv.Itoa(s.seq)
}

func (s *TagStore) List(_ context.Context, ownerID string) ([]tag.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []tag.Tag
	for _, t := range s.tags {
		if t.OwnerID == ownerID {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *TagStore) Get(_ context.Context, ownerID, id string) (tag.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tags[id]
	if !ok || t.OwnerID != ownerID {
		return tag.Tag{}, tag.ErrNotFound
	}
	return t, nil
}

func (s *TagStore) ByName(_ context.Context, ownerID, name string) (tag.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tags {
		if t.OwnerID == ownerID && t.Name == name {
			return t, nil
		}
	}
	return tag.Tag{}, tag.ErrNotFound
}

func (s *TagStore) Create(_ context.Context, ownerID, name string) (tag.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tags {
		if t.OwnerID == ownerID && t.Name == name {
			return tag.Tag{}, tag.ErrDuplicateName
		}
	}
	t := tag.Tag{ID: s.nextID(), OwnerID: ownerID, Name: name, CreatedAt: time.Now().UTC()}
	s.tags[t.ID] = t
	return t, nil
}

func (s *TagStore) Update(_ context.Context, ownerID, id, name string) (tag.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tags[id]
	if !ok || t.OwnerID != ownerID {
		return tag.Tag{}, tag.ErrNotFound
	}
	for otherID, other := range s.tags {
		if otherID != id && other.OwnerID == ownerID && other.Name == name {
			return tag.Tag{}, tag.ErrDuplicateName
		}
	}
	t.Name = name
	s.tags[id] = t
	return t, nil
}

func (s *TagStore) Delete(_ context.Context, ownerID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tags[id]
	if !ok || t.OwnerID != ownerID {
		return tag.ErrNotFound
	}
	delete(s.tags, id)
	return nil
}

func (s *TagStore) OwnedBy(_ context.Context, ownerID string, tagIDs []string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range tagIDs {
		t, ok := s.tags[id]
		if !ok || t.OwnerID != ownerID {
			return false, nil
		}
	}
	return true, nil
}
