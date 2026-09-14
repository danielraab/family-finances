package dashboard_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/dashboard"
	"at.draab/familyfinances/internal/storage/memory"
)

// stubAccounts, stubCategories, stubTags are minimal fakes satisfying
// dashboard.AccountLookup/CategoryLookup/TagLookup — dashboard.Service is
// tested in isolation from the real account/category/tag packages,
// mirroring internal/entry's own stub pattern (service_test.go).

var errNotFound = errors.New("not found")

type stubAccounts struct {
	owner  map[string]string
	shares map[string]map[string]string // accountID -> userID -> permission
}

func newStubAccounts() *stubAccounts {
	return &stubAccounts{owner: map[string]string{}, shares: map[string]map[string]string{}}
}

func (s *stubAccounts) add(id, ownerID string) { s.owner[id] = ownerID }

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
	return "EUR", false, permission, nil
}

type stubCategories struct {
	visible map[string]map[string]bool // categoryID -> callerID -> visible
}

func newStubCategories() *stubCategories {
	return &stubCategories{visible: map[string]map[string]bool{}}
}

func (s *stubCategories) grant(categoryID, callerID string) {
	if s.visible[categoryID] == nil {
		s.visible[categoryID] = map[string]bool{}
	}
	s.visible[categoryID][callerID] = true
}

func (s *stubCategories) Visible(_ context.Context, callerID string, categoryIDs []string) (bool, error) {
	for _, id := range categoryIDs {
		if !s.visible[id][callerID] {
			return false, nil
		}
	}
	return true, nil
}

type stubTags struct {
	visible map[string]map[string]bool // tagID -> callerID -> visible
}

func newStubTags() *stubTags {
	return &stubTags{visible: map[string]map[string]bool{}}
}

func (s *stubTags) grant(tagID, callerID string) {
	if s.visible[tagID] == nil {
		s.visible[tagID] = map[string]bool{}
	}
	s.visible[tagID][callerID] = true
}

func (s *stubTags) OwnedBy(_ context.Context, callerID string, tagIDs []string) (bool, error) {
	for _, id := range tagIDs {
		if !s.visible[id][callerID] {
			return false, nil
		}
	}
	return true, nil
}

func newTestService() (*dashboard.Service, *stubAccounts, *stubCategories, *stubTags) {
	accounts := newStubAccounts()
	categories := newStubCategories()
	tags := newStubTags()
	svc := dashboard.NewService(memory.NewDashboardStore(), accounts, categories, tags)
	return svc, accounts, categories, tags
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }
func intPtr(n int) *int       { return &n }

func TestServiceCreateAccountStat(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u1")

	card, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: strPtr("acc1")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if card.SortOrder != 0 {
		t.Fatalf("sort_order = %d, want 0", card.SortOrder)
	}
}

func TestServiceCreateAccountStatRequiresAccountID(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeAccountStat, Config: dashboard.Config{}})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateAccountStatRejectsForeignField(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u1")
	_, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: strPtr("acc1"), TagID: strPtr("tag1")},
	})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateUnknownTypeRejected(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardType("something_else")})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateBarChartRequiresUnit(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeBarChart, Config: dashboard.Config{}})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateBarChartRejectsInvalidUnit(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeBarChart,
		Config: dashboard.Config{Unit: strPtr("week")},
	})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateBarChartRejectsRange(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeBarChart,
		Config: dashboard.Config{Unit: strPtr("month"), Range: &dashboard.Range{Preset: "this_month"}},
	})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateQueryStatWithNoFiltersSucceeds(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat, Config: dashboard.Config{}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestServiceCreateEntryListWithoutColumnsSucceeds(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeEntryList, Config: dashboard.Config{}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestServiceCreateEntryListAcceptsColumnsWithinRange(t *testing.T) {
	svc, _, _, _ := newTestService()
	for _, n := range []int{2, 3, 4} {
		_, err := svc.Create(t.Context(), "u1", dashboard.New{
			Type:   dashboard.CardTypeEntryList,
			Config: dashboard.Config{Columns: intPtr(n)},
		})
		if err != nil {
			t.Fatalf("columns=%d: %v", n, err)
		}
	}
}

func TestServiceCreateEntryListRejectsColumnsOutOfRange(t *testing.T) {
	svc, _, _, _ := newTestService()
	for _, n := range []int{0, 1, 5} {
		_, err := svc.Create(t.Context(), "u1", dashboard.New{
			Type:   dashboard.CardTypeEntryList,
			Config: dashboard.Config{Columns: intPtr(n)},
		})
		if !errors.Is(err, dashboard.ErrInvalidValue) {
			t.Fatalf("columns=%d: err = %v, want ErrInvalidValue", n, err)
		}
	}
}

func TestServiceCreateColumnsRejectedOnOtherTypes(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u1")

	if _, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: strPtr("acc1"), Columns: intPtr(2)},
	}); !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("account_stat: err = %v, want ErrInvalidValue", err)
	}

	if _, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeQueryStat,
		Config: dashboard.Config{Columns: intPtr(2)},
	}); !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("query_stat: err = %v, want ErrInvalidValue", err)
	}

	if _, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeBarChart,
		Config: dashboard.Config{Unit: strPtr("month"), Columns: intPtr(2)},
	}); !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("bar_chart: err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateTitleAllowedOnFilterBearingTypes(t *testing.T) {
	svc, _, _, _ := newTestService()

	if _, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeQueryStat,
		Config: dashboard.Config{Title: strPtr("Groceries this month")},
	}); err != nil {
		t.Fatalf("query_stat: %v", err)
	}
	if _, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeEntryList,
		Config: dashboard.Config{Title: strPtr("Recent groceries")},
	}); err != nil {
		t.Fatalf("entry_list: %v", err)
	}
	if _, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeBarChart,
		Config: dashboard.Config{Unit: strPtr("month"), Title: strPtr("Spending trend")},
	}); err != nil {
		t.Fatalf("bar_chart: %v", err)
	}
}

func TestServiceCreateTitleRejectedOnAccountStat(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u1")

	_, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: strPtr("acc1"), Title: strPtr("My cash")},
	})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateRejectsInaccessibleAccount(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u2") // owned by someone else, no share

	_, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeQueryStat,
		Config: dashboard.Config{AccountID: strPtr("acc1")},
	})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateAcceptsCategorySharedAtViewTier(t *testing.T) {
	svc, _, categories, _ := newTestService()
	categories.grant("cat1", "u1")

	_, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeQueryStat,
		Config: dashboard.Config{CategoryID: strPtr("cat1"), IncludeSubcategories: boolPtr(true)},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestServiceCreateRejectsInvisibleCategory(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeEntryList,
		Config: dashboard.Config{CategoryID: strPtr("cat-nope")},
	})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceCreateRejectsInvisibleTag(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeEntryList,
		Config: dashboard.Config{TagID: strPtr("tag-nope")},
	})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestServiceUpdateRevalidatesAgainstExistingType(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u1")
	card, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: strPtr("acc1")},
	})
	if err != nil {
		t.Fatal(err)
	}

	// A type in the update body is ignored — the field doesn't even exist
	// on Update — so this only ever revalidates Config against the card's
	// original, immutable type (account_stat).
	if _, err := svc.Update(t.Context(), "u1", card.ID, dashboard.Update{Config: dashboard.Config{}}); !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue (account_stat still requires account_id)", err)
	}
}

func TestServiceUpdateRejectsNewlyInaccessibleAccount(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u1")
	accounts.add("acc2", "u2")
	card, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: strPtr("acc1")},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Update(t.Context(), "u1", card.ID, dashboard.Update{Config: dashboard.Config{AccountID: strPtr("acc2")}})
	if !errors.Is(err, dashboard.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}

	// Stored config is unchanged after the rejected update.
	list, err := svc.List(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Config.AccountID == nil || *list[0].Config.AccountID != "acc1" {
		t.Fatalf("list = %+v, want unchanged acc1", list)
	}
}

func TestServiceUpdateCrossOwnerNotFound(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u1")
	card, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: strPtr("acc1")},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Update(t.Context(), "u2", card.ID, dashboard.Update{Config: dashboard.Config{AccountID: strPtr("acc1")}}); !errors.Is(err, dashboard.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestServiceListOrdersBySortOrder(t *testing.T) {
	svc, _, _, _ := newTestService()
	a, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}

	list, err := svc.List(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != a.ID || list[1].ID != b.ID {
		t.Fatalf("list = %+v", list)
	}
}

func TestServiceListNeverReturnsAnotherUsersCards(t *testing.T) {
	svc, _, _, _ := newTestService()
	if _, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat}); err != nil {
		t.Fatal(err)
	}

	list, err := svc.List(t.Context(), "u2")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %+v, want empty", list)
	}
}

func TestServiceMoveUpSwapsWithPrevious(t *testing.T) {
	svc, _, _, _ := newTestService()
	a, _ := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})
	b, _ := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})

	if _, err := svc.MoveUp(t.Context(), "u1", b.ID); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != b.ID || list[1].ID != a.ID {
		t.Fatalf("list = %+v, want [b, a]", list)
	}
}

func TestServiceMoveUpFirstCardIsNoop(t *testing.T) {
	svc, _, _, _ := newTestService()
	a, _ := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})
	_, _ = svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})

	if _, err := svc.MoveUp(t.Context(), "u1", a.ID); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if list[0].ID != a.ID {
		t.Fatalf("list = %+v, want a first (unchanged)", list)
	}
}

func TestServiceMoveDownLastCardIsNoop(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, _ = svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})
	b, _ := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})

	if _, err := svc.MoveDown(t.Context(), "u1", b.ID); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if list[1].ID != b.ID {
		t.Fatalf("list = %+v, want b last (unchanged)", list)
	}
}

func TestServiceMoveDoesNotRevalidateStaleReference(t *testing.T) {
	svc, accounts, _, _ := newTestService()
	accounts.add("acc1", "u1")
	card, err := svc.Create(t.Context(), "u1", dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: strPtr("acc1")},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})

	// The account share is revoked entirely after the card was created —
	// account no longer resolves for u1 at all.
	accounts.owner["acc1"] = "someone-else"

	if _, err := svc.MoveDown(t.Context(), "u1", card.ID); err != nil {
		t.Fatalf("MoveDown returned an error for a stale reference: %v", err)
	}
}

func TestServiceDeleteClosesGapForSubsequentMove(t *testing.T) {
	svc, _, _, _ := newTestService()
	a, _ := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})
	b, _ := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})
	c, _ := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})

	if err := svc.Delete(t.Context(), "u1", b.ID); err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != a.ID || list[1].ID != c.ID {
		t.Fatalf("list = %+v, want [a, c]", list)
	}

	if _, err := svc.MoveUp(t.Context(), "u1", c.ID); err != nil {
		t.Fatal(err)
	}
	list, err = svc.List(t.Context(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if list[0].ID != c.ID {
		t.Fatalf("list = %+v, want c first after move", list)
	}
}

func TestServiceDeleteCrossOwnerNotFound(t *testing.T) {
	svc, _, _, _ := newTestService()
	a, err := svc.Create(t.Context(), "u1", dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(t.Context(), "u2", a.ID); !errors.Is(err, dashboard.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
