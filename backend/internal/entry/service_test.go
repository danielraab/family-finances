package entry_test

import (
	"context"
	"errors"
)

// stubAccounts, stubCategories, stubTags are minimal fakes satisfying
// entry.AccountLookup/CategoryLookup/TagLookup — entry.Service is tested in
// isolation from the real account/category/tag packages, matching the
// package-boundaries decision in design.md.

type stubAccounts struct {
	owner    map[string]string
	currency map[string]string
	disabled map[string]bool
}

func newStubAccounts() *stubAccounts {
	return &stubAccounts{owner: map[string]string{}, currency: map[string]string{}, disabled: map[string]bool{}}
}

func (s *stubAccounts) add(id, ownerID, currency string) {
	s.owner[id] = ownerID
	s.currency[id] = currency
}

func (s *stubAccounts) Owner(_ context.Context, accountID string) (string, string, bool, error) {
	owner, ok := s.owner[accountID]
	if !ok {
		return "", "", false, errNotFound
	}
	return owner, s.currency[accountID], s.disabled[accountID], nil
}

func (s *stubAccounts) VisibleIDs(_ context.Context, ownerID string) ([]string, error) {
	var out []string
	for id, owner := range s.owner {
		if owner == ownerID {
			out = append(out, id)
		}
	}
	return out, nil
}

var errNotFound = errors.New("account not found")

type stubCategories struct {
	exists map[string]bool
	// children maps a category id to its direct children, for Subtree.
	children map[string][]string
}

func newStubCategories() *stubCategories {
	return &stubCategories{exists: map[string]bool{}, children: map[string][]string{}}
}

func (c *stubCategories) add(id string) { c.exists[id] = true }

func (c *stubCategories) Exists(_ context.Context, id string) (bool, error) {
	return c.exists[id], nil
}

func (c *stubCategories) Subtree(_ context.Context, id string) ([]string, error) {
	out := []string{id}
	queue := []string{id}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, child := range c.children[cur] {
			out = append(out, child)
			queue = append(queue, child)
		}
	}
	return out, nil
}

type stubTags struct {
	owned map[string]string // tagID -> ownerID
}

func newStubTags() *stubTags { return &stubTags{owned: map[string]string{}} }

func (t *stubTags) add(id, ownerID string) { t.owned[id] = ownerID }

func (t *stubTags) OwnedBy(_ context.Context, owner string, tagIDs []string) (bool, error) {
	for _, id := range tagIDs {
		if t.owned[id] != owner {
			return false, nil
		}
	}
	return true, nil
}

func ptr[T any](v T) *T { return &v }
