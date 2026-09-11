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
// concurrent use. It has no visibility into entries, so it always reports
// EntryCount as 0 — mirroring memory.CategoryStore's equivalent gap.
//
// It also has no visibility into internal/auth's users, the same accepted
// gap memory.CategoryStore has for CategoryShare.Name/Email/GrantedByName:
// TagShare.Name/Email/GrantedByName and Tag.OwnerName are always left
// empty here — fine, since storage/memory is test/local-dev
// infrastructure, never what ships.
type TagStore struct {
	mu     sync.Mutex
	tags   map[string]tag.Tag
	shares map[string]tag.TagShare // key: tagShareKey(tagID, userID)
	seq    int
}

// NewTagStore returns an empty TagStore.
func NewTagStore() *TagStore {
	return &TagStore{tags: map[string]tag.Tag{}, shares: map[string]tag.TagShare{}}
}

func (s *TagStore) nextID() string {
	s.seq++
	return "tag" + strconv.Itoa(s.seq)
}

func tagShareKey(tagID, userID string) string { return tagID + "|" + userID }

// permissionLocked resolves callerID's Permission on t (id is t's own key,
// passed separately to avoid a second map lookup). Callers MUST hold s.mu.
func (s *TagStore) permissionLocked(id, callerID string, t tag.Tag) tag.Permission {
	if t.OwnerID == callerID {
		return tag.PermissionOwner
	}
	if sh, ok := s.shares[tagShareKey(id, callerID)]; ok {
		return sh.Permission
	}
	return ""
}

func (s *TagStore) List(_ context.Context, callerID string) ([]tag.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []tag.Tag
	for id, t := range s.tags {
		perm := s.permissionLocked(id, callerID, t)
		if perm == "" {
			continue
		}
		t.Permission = perm
		t.Shared = t.OwnerID != callerID
		out = append(out, t)
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

func (s *TagStore) GetForCaller(_ context.Context, callerID, id string) (tag.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tags[id]
	if !ok {
		return tag.Tag{}, tag.ErrNotFound
	}
	t.Permission = s.permissionLocked(id, callerID, t)
	t.Shared = t.Permission != "" && t.OwnerID != callerID
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
	for _, sh := range s.shares {
		if sh.TagID == id {
			return tag.ErrInUse
		}
	}
	delete(s.tags, id)
	return nil
}

func (s *TagStore) SetDisabled(_ context.Context, ownerID, id string, disabled bool) (tag.Tag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tags[id]
	if !ok || t.OwnerID != ownerID {
		return tag.Tag{}, tag.ErrNotFound
	}
	t.Disabled = disabled
	s.tags[id] = t
	return t, nil
}

func (s *TagStore) OwnedBy(_ context.Context, callerID string, tagIDs []string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range tagIDs {
		t, ok := s.tags[id]
		if !ok || s.permissionLocked(id, callerID, t) == "" {
			return false, nil
		}
	}
	return true, nil
}

func (s *TagStore) Usable(_ context.Context, callerID string, tagIDs []string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range tagIDs {
		t, ok := s.tags[id]
		if !ok || t.Disabled || !s.permissionLocked(id, callerID, t).AtLeast(tag.PermissionAppend) {
			return false, nil
		}
	}
	return true, nil
}

// --- sharing -----------------------------------------------------------

func (s *TagStore) CreateOrUpdateShare(_ context.Context, tagID, userID string, permission tag.Permission, grantedBy string) (tag.TagShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tagShareKey(tagID, userID)
	now := time.Now().UTC()
	sh, exists := s.shares[key]
	if exists {
		sh.Permission = permission
		sh.GrantedBy = grantedBy
		sh.UpdatedAt = now
	} else {
		sh = tag.TagShare{
			TagID:      tagID,
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

func (s *TagStore) ListShares(_ context.Context, tagID string) ([]tag.TagShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []tag.TagShare
	for _, sh := range s.shares {
		if sh.TagID == tagID {
			out = append(out, sh)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *TagStore) ShareByUser(_ context.Context, tagID, userID string) (tag.TagShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, ok := s.shares[tagShareKey(tagID, userID)]
	if !ok {
		return tag.TagShare{}, tag.ErrNotFound
	}
	return sh, nil
}

func (s *TagStore) UpdateSharePermission(_ context.Context, tagID, userID string, permission tag.Permission) (tag.TagShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tagShareKey(tagID, userID)
	sh, ok := s.shares[key]
	if !ok {
		return tag.TagShare{}, tag.ErrNotFound
	}
	sh.Permission = permission
	sh.UpdatedAt = time.Now().UTC()
	s.shares[key] = sh
	return sh, nil
}

func (s *TagStore) DeleteShare(_ context.Context, tagID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tagShareKey(tagID, userID)
	if _, ok := s.shares[key]; !ok {
		return tag.ErrNotFound
	}
	delete(s.shares, key)
	return nil
}
