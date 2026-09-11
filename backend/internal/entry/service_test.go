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
	// shares maps accountID -> userID -> permission, for tests exercising
	// account-sharing tiers without depending on the real internal/account
	// package.
	shares map[string]map[string]string
}

func newStubAccounts() *stubAccounts {
	return &stubAccounts{
		owner: map[string]string{}, currency: map[string]string{}, disabled: map[string]bool{},
		shares: map[string]map[string]string{},
	}
}

func (s *stubAccounts) add(id, ownerID, currency string) {
	s.owner[id] = ownerID
	s.currency[id] = currency
}

// share grants userID permission on accountID — mirrors account-sharing's
// account_shares, entirely independent of the real owner.
func (s *stubAccounts) share(accountID, userID, permission string) {
	if s.shares[accountID] == nil {
		s.shares[accountID] = map[string]string{}
	}
	s.shares[accountID][userID] = permission
}

func (s *stubAccounts) Access(_ context.Context, accountID, callerID string) (string, bool, string, error) {
	owner, ok := s.owner[accountID]
	if !ok {
		return "", false, "", errNotFound
	}
	permission := ""
	if owner == callerID {
		permission = "owner"
	} else if p, ok := s.shares[accountID][callerID]; ok {
		permission = p
	}
	return s.currency[accountID], s.disabled[accountID], permission, nil
}

func (s *stubAccounts) VisibleIDs(_ context.Context, callerID string) ([]string, error) {
	var out []string
	for id, owner := range s.owner {
		if owner == callerID {
			out = append(out, id)
			continue
		}
		if _, ok := s.shares[id][callerID]; ok {
			out = append(out, id)
		}
	}
	return out, nil
}

var errNotFound = errors.New("account not found")

// stubTimezones satisfies entry.TimezoneLookup — a minimal fake so
// FlowSummary tests can exercise a non-UTC caller timezone without
// depending on the real internal/settings package.
type stubTimezones struct {
	tz map[string]string
}

func newStubTimezones() *stubTimezones { return &stubTimezones{tz: map[string]string{}} }

func (s *stubTimezones) Timezone(_ context.Context, ownerID string) (string, error) {
	if tz, ok := s.tz[ownerID]; ok {
		return tz, nil
	}
	return "UTC", nil
}

type stubCategories struct {
	owner    map[string]string // categoryID -> ownerID
	disabled map[string]bool
	// children maps a category id to its direct children, for Subtree.
	children map[string][]string
	// shares maps categoryID -> userID -> true, granting that user
	// visibility of id alone (never cascading to children) — mirrors
	// category.Service.Subtree's real distinction between an owner's
	// full-subtree cascade and a share's single-id resolution.
	shares map[string]map[string]bool
}

func newStubCategories() *stubCategories {
	return &stubCategories{
		owner: map[string]string{}, disabled: map[string]bool{}, children: map[string][]string{},
		shares: map[string]map[string]bool{},
	}
}

// add registers id as owned by "u1", the fixture owner used throughout
// this package's tests.
func (c *stubCategories) add(id string) { c.owner[id] = "u1" }

func (c *stubCategories) addOwnedBy(id, owner string) { c.owner[id] = owner }

func (c *stubCategories) disable(id string) { c.disabled[id] = true }

// share grants userID view/append-equivalent visibility of id alone —
// entry.Service's CategoryLookup interface (Usable/Subtree) never
// distinguishes view from append itself (Usable's append-only rule is
// entirely real category.Service's own concern, tested there), so this
// stub only models "has some share" vs. "owns" vs. "no permission at all."
func (c *stubCategories) share(id, userID string) {
	if c.shares[id] == nil {
		c.shares[id] = map[string]bool{}
	}
	c.shares[id][userID] = true
}

func (c *stubCategories) Usable(_ context.Context, callerID, id string) (bool, error) {
	if c.disabled[id] {
		return false, nil
	}
	if owner, ok := c.owner[id]; ok && owner == callerID {
		return true, nil
	}
	return c.shares[id][callerID], nil
}

// Subtree mirrors category.Service.Subtree: the owner's full descendant
// subtree for a category callerID owns, id alone for one visible only via
// a share, and nil (no permission at all) otherwise — its non-emptiness is
// also the permission signal entry.Service.resolveFilter relies on to
// widen account scoping for a category-filtered query.
func (c *stubCategories) Subtree(_ context.Context, callerID, id string) ([]string, error) {
	owner, ok := c.owner[id]
	if !ok {
		return nil, nil
	}
	if owner != callerID {
		if c.shares[id][callerID] {
			return []string{id}, nil
		}
		return nil, nil
	}
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
	owned    map[string]string // tagID -> ownerID
	disabled map[string]bool
}

func newStubTags() *stubTags {
	return &stubTags{owned: map[string]string{}, disabled: map[string]bool{}}
}

func (t *stubTags) add(id, ownerID string) { t.owned[id] = ownerID }

func (t *stubTags) disable(id string) { t.disabled[id] = true }

func (t *stubTags) OwnedBy(_ context.Context, owner string, tagIDs []string) (bool, error) {
	for _, id := range tagIDs {
		if t.owned[id] != owner {
			return false, nil
		}
	}
	return true, nil
}

func (t *stubTags) Usable(_ context.Context, owner string, tagIDs []string) (bool, error) {
	for _, id := range tagIDs {
		if t.owned[id] != owner || t.disabled[id] {
			return false, nil
		}
	}
	return true, nil
}

func ptr[T any](v T) *T { return &v }
