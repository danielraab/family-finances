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
// still referenced by one — see category.ErrInUse's doc comment — and it
// always reports Category.EntryCount as 0, mirroring memory.TagStore's
// equivalent gap.
//
// It also has no visibility into internal/auth's users, the same accepted
// gap memory.AccountStore has for AccountShare.Name/Email/GrantedByName:
// CategoryShare.Name/Email/GrantedByName and Category.OwnerName are always
// left empty here — fine, since storage/memory is test/local-dev
// infrastructure, never what ships.
type CategoryStore struct {
	mu     sync.Mutex
	cats   map[string]category.Category
	shares map[string]category.CategoryShare // key: categoryShareKey(categoryID, userID)
	seq    int
}

// NewCategoryStore returns an empty CategoryStore.
func NewCategoryStore() *CategoryStore {
	return &CategoryStore{
		cats:   map[string]category.Category{},
		shares: map[string]category.CategoryShare{},
	}
}

func (s *CategoryStore) nextID() string {
	s.seq++
	return "cat" + strconv.Itoa(s.seq)
}

func categoryShareKey(categoryID, userID string) string { return categoryID + "|" + userID }

// permissionLocked resolves callerID's Permission on c (id is c's own key,
// passed separately to avoid a second map lookup). Callers MUST hold s.mu.
func (s *CategoryStore) permissionLocked(id, callerID string, c category.Category) category.Permission {
	if c.OwnerID == callerID {
		return category.PermissionOwner
	}
	if sh, ok := s.shares[categoryShareKey(id, callerID)]; ok {
		return sh.Permission
	}
	return ""
}

// visible reports whether c belongs to ownerID and is not soft-deleted.
func visible(c category.Category, ownerID string) bool {
	return c.OwnerID == ownerID && c.DeletedAt == nil
}

func sameGroup(a, b category.Category) bool {
	if (a.ParentID == nil) != (b.ParentID == nil) {
		return false
	}
	return a.ParentID == nil || *a.ParentID == *b.ParentID
}

func (s *CategoryStore) List(_ context.Context, callerID string) ([]category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []category.Category
	for id, c := range s.cats {
		if c.DeletedAt != nil {
			continue
		}
		perm := s.permissionLocked(id, callerID, c)
		if perm == "" {
			continue
		}
		c.Permission = perm
		c.Shared = c.OwnerID != callerID
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (s *CategoryStore) Get(_ context.Context, ownerID, id string) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok || !visible(c, ownerID) {
		return category.Category{}, category.ErrNotFound
	}
	return c, nil
}

func (s *CategoryStore) GetForCaller(_ context.Context, callerID, id string) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok || c.DeletedAt != nil {
		return category.Category{}, category.ErrNotFound
	}
	c.Permission = s.permissionLocked(id, callerID, c)
	c.Shared = c.Permission != "" && c.OwnerID != callerID
	return c, nil
}

// nextSortOrder returns the sort_order that appends a new member to the
// sibling group sharing ownerID + parentID.
func (s *CategoryStore) nextSortOrder(ownerID string, parentID *string) int {
	max := -1
	for _, c := range s.cats {
		if c.OwnerID != ownerID || c.DeletedAt != nil {
			continue
		}
		if (c.ParentID == nil) != (parentID == nil) {
			continue
		}
		if c.ParentID != nil && *c.ParentID != *parentID {
			continue
		}
		if c.SortOrder > max {
			max = c.SortOrder
		}
	}
	return max + 1
}

func (s *CategoryStore) Create(_ context.Context, ownerID string, in category.New) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := category.Category{
		ID:        s.nextID(),
		OwnerID:   ownerID,
		ParentID:  in.ParentID,
		Name:      in.Name,
		Icon:      in.Icon,
		Color:     in.Color,
		SortOrder: s.nextSortOrder(ownerID, in.ParentID),
		CreatedAt: time.Now().UTC(),
	}
	s.cats[c.ID] = c
	return c, nil
}

func (s *CategoryStore) Update(_ context.Context, ownerID, id string, upd category.Update) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok || !visible(c, ownerID) {
		return category.Category{}, category.ErrNotFound
	}
	if upd.Name != nil {
		c.Name = *upd.Name
	}
	if upd.Icon != nil {
		c.Icon = *upd.Icon
	}
	if upd.Color != nil {
		c.Color = *upd.Color
	}
	if upd.ParentID.Set {
		c.ParentID = upd.ParentID.Value
		c.SortOrder = s.nextSortOrder(ownerID, upd.ParentID.Value)
	}
	s.cats[id] = c
	return c, nil
}

func (s *CategoryStore) Delete(_ context.Context, ownerID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok || !visible(c, ownerID) {
		return category.ErrNotFound
	}
	for _, other := range s.cats {
		if other.ParentID != nil && *other.ParentID == id && other.DeletedAt == nil {
			return category.ErrInUse
		}
	}
	now := time.Now().UTC()
	c.DeletedAt = &now
	s.cats[id] = c
	return nil
}

func (s *CategoryStore) SetDisabled(_ context.Context, ownerID, id string, disabled bool) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok || !visible(c, ownerID) {
		return category.Category{}, category.ErrNotFound
	}
	c.Disabled = disabled
	s.cats[id] = c
	return c, nil
}

// siblings returns c's siblings (same owner + parent, visible, excluding c
// itself), ordered by sort_order then id.
func (s *CategoryStore) siblings(c category.Category) []category.Category {
	var out []category.Category
	for _, other := range s.cats {
		if other.ID == c.ID || !visible(other, c.OwnerID) || !sameGroup(c, other) {
			continue
		}
		out = append(out, other)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *CategoryStore) MoveUp(_ context.Context, ownerID, id string) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok || !visible(c, ownerID) {
		return category.Category{}, category.ErrNotFound
	}
	var prev *category.Category
	for _, sib := range s.siblings(c) {
		if sib.SortOrder < c.SortOrder || (sib.SortOrder == c.SortOrder && sib.ID < c.ID) {
			sib := sib
			if prev == nil || sib.SortOrder > prev.SortOrder || (sib.SortOrder == prev.SortOrder && sib.ID > prev.ID) {
				prev = &sib
			}
		}
	}
	if prev == nil {
		return c, nil
	}
	c.SortOrder, prev.SortOrder = prev.SortOrder, c.SortOrder
	s.cats[c.ID] = c
	s.cats[prev.ID] = *prev
	return c, nil
}

func (s *CategoryStore) MoveDown(_ context.Context, ownerID, id string) (category.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	if !ok || !visible(c, ownerID) {
		return category.Category{}, category.ErrNotFound
	}
	var next *category.Category
	for _, sib := range s.siblings(c) {
		if sib.SortOrder > c.SortOrder || (sib.SortOrder == c.SortOrder && sib.ID > c.ID) {
			sib := sib
			if next == nil || sib.SortOrder < next.SortOrder || (sib.SortOrder == next.SortOrder && sib.ID < next.ID) {
				next = &sib
			}
		}
	}
	if next == nil {
		return c, nil
	}
	c.SortOrder, next.SortOrder = next.SortOrder, c.SortOrder
	s.cats[c.ID] = c
	s.cats[next.ID] = *next
	return c, nil
}

func (s *CategoryStore) Exists(_ context.Context, ownerID, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.cats[id]
	return ok && visible(c, ownerID), nil
}

func (s *CategoryStore) SeedDefaults(_ context.Context, ownerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for i, dc := range category.DefaultCategories {
		c := category.Category{
			ID:        s.nextID(),
			OwnerID:   ownerID,
			Name:      dc.Name,
			Icon:      dc.Icon,
			Color:     dc.Color,
			SortOrder: i,
			CreatedAt: now,
		}
		s.cats[c.ID] = c
	}
	return nil
}

func (s *CategoryStore) Subtree(_ context.Context, ownerID, id string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	children := map[string][]string{}
	for _, c := range s.cats {
		if c.OwnerID != ownerID || c.DeletedAt != nil {
			continue
		}
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

// --- sharing -----------------------------------------------------------

func (s *CategoryStore) CreateOrUpdateShare(_ context.Context, categoryID, userID string, permission category.Permission, grantedBy string) (category.CategoryShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := categoryShareKey(categoryID, userID)
	now := time.Now().UTC()
	sh, exists := s.shares[key]
	if exists {
		sh.Permission = permission
		sh.GrantedBy = grantedBy
		sh.UpdatedAt = now
	} else {
		sh = category.CategoryShare{
			CategoryID: categoryID,
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

func (s *CategoryStore) ListShares(_ context.Context, categoryID string) ([]category.CategoryShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []category.CategoryShare
	for _, sh := range s.shares {
		if sh.CategoryID == categoryID {
			out = append(out, sh)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (s *CategoryStore) ShareByUser(_ context.Context, categoryID, userID string) (category.CategoryShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sh, ok := s.shares[categoryShareKey(categoryID, userID)]
	if !ok {
		return category.CategoryShare{}, category.ErrNotFound
	}
	return sh, nil
}

func (s *CategoryStore) UpdateSharePermission(_ context.Context, categoryID, userID string, permission category.Permission) (category.CategoryShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := categoryShareKey(categoryID, userID)
	sh, ok := s.shares[key]
	if !ok {
		return category.CategoryShare{}, category.ErrNotFound
	}
	sh.Permission = permission
	sh.UpdatedAt = time.Now().UTC()
	s.shares[key] = sh
	return sh, nil
}

func (s *CategoryStore) DeleteShare(_ context.Context, categoryID, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := categoryShareKey(categoryID, userID)
	if _, ok := s.shares[key]; !ok {
		return category.ErrNotFound
	}
	delete(s.shares, key)
	return nil
}
