package recurringtransaction_test

import (
	"context"
	"errors"
	"testing"
	"time"

	rt "at.draab/familyfinances/internal/recurringtransaction"
	"at.draab/familyfinances/internal/storage/memory"
)

// stubAccounts/stubCategories/stubTags mirror internal/entry's own test
// stubs — recurringtransaction.Service is tested in isolation from the
// real account/category/tag/entry packages, matching internal/entry's own
// package-boundaries test pattern.

type stubAccounts struct {
	owner    map[string]string
	currency map[string]string
	disabled map[string]bool
	shares   map[string]map[string]string
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

func (s *stubAccounts) share(accountID, userID, permission string) {
	if s.shares[accountID] == nil {
		s.shares[accountID] = map[string]string{}
	}
	s.shares[accountID][userID] = permission
}

var errAccountNotFound = errors.New("account not found")

func (s *stubAccounts) Access(_ context.Context, accountID, callerID string) (string, bool, string, error) {
	owner, ok := s.owner[accountID]
	if !ok {
		return "", false, "", errAccountNotFound
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

type stubCategories struct {
	usable map[string]bool
	// children maps a category id to its direct children, for Subtree —
	// mirrors internal/entry's own stubCategories.
	children map[string][]string
}

func newStubCategories() *stubCategories {
	return &stubCategories{usable: map[string]bool{}, children: map[string][]string{}}
}

func (c *stubCategories) add(id string) { c.usable[id] = true }

// addChild registers child as a direct child of parent, for Subtree.
func (c *stubCategories) addChild(parent, child string) {
	c.usable[child] = true
	c.children[parent] = append(c.children[parent], child)
}

func (c *stubCategories) Usable(_ context.Context, _, id string) (bool, error) {
	return c.usable[id], nil
}

// Subtree mirrors category.Service.Subtree: id plus every registered
// descendant, when id is usable by the caller at all; nil otherwise.
func (c *stubCategories) Subtree(_ context.Context, _, id string) ([]string, error) {
	if !c.usable[id] {
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
	owned    map[string]string
	disabled map[string]bool
}

func newStubTags() *stubTags {
	return &stubTags{owned: map[string]string{}, disabled: map[string]bool{}}
}

func (tg *stubTags) add(id, owner string) { tg.owned[id] = owner }

func (tg *stubTags) OwnedBy(_ context.Context, owner string, tagIDs []string) (bool, error) {
	for _, id := range tagIDs {
		if tg.owned[id] != owner {
			return false, nil
		}
	}
	return true, nil
}

func (tg *stubTags) Usable(_ context.Context, owner string, tagIDs []string) (bool, error) {
	ok, err := tg.OwnedBy(context.Background(), owner, tagIDs)
	if err != nil || !ok {
		return false, err
	}
	for _, id := range tagIDs {
		if tg.disabled[id] {
			return false, nil
		}
	}
	return true, nil
}

// stubEntries satisfies rt.EntryLookup for testing NextSuggestedDate/delete
// blocking without depending on the real internal/entry package.
type stubEntries struct {
	latest map[string]time.Time
	count  map[string]int
}

func newStubEntries() *stubEntries {
	return &stubEntries{latest: map[string]time.Time{}, count: map[string]int{}}
}

func (e *stubEntries) LatestLinkedBookingTime(_ context.Context, id string) (*time.Time, error) {
	t, ok := e.latest[id]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

func (e *stubEntries) LinkedCount(_ context.Context, id string) (int, error) {
	return e.count[id], nil
}

func newService() (*rt.Service, *stubAccounts, *stubCategories, *stubTags, *stubEntries) {
	accounts := newStubAccounts()
	categories := newStubCategories()
	tags := newStubTags()
	entries := newStubEntries()
	svc := rt.NewService(memory.NewRecurringTransactionStore(), accounts, categories, tags)
	svc.SetEntryLookup(entries)
	return svc, accounts, categories, tags, entries
}

func baseNew(accountID, categoryID string) rt.New {
	return rt.New{
		AccountID:     accountID,
		Title:         "Netflix",
		CategoryID:    &categoryID,
		Amount:        -1500,
		IntervalUnit:  rt.UnitMonth,
		IntervalCount: 1,
		StartsOn:      mustDateT(2026, 1, 1),
	}
}

func mustDateT(y int, m time.Month, d int) rt.Date {
	return rt.NewDate(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
}

func TestCreateRequiresAppendPermission(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	// Owner can create.
	if _, err := svc.Create(context.Background(), "u1", baseNew("acc1", "cat1")); err != nil {
		t.Fatalf("owner create: %v", err)
	}

	// A view-tier share cannot create.
	accounts.share("acc1", "u2", "view")
	if _, err := svc.Create(context.Background(), "u2", baseNew("acc1", "cat1")); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("view-tier create: err = %v, want ErrInvalidValue", err)
	}

	// An append-tier share can create.
	accounts.share("acc1", "u3", "append")
	if _, err := svc.Create(context.Background(), "u3", baseNew("acc1", "cat1")); err != nil {
		t.Fatalf("append-tier create: %v", err)
	}
}

func TestCreateRejectedOnDisabledAccount(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	accounts.disabled["acc1"] = true
	categories.add("cat1")

	_, err := svc.Create(context.Background(), "u1", baseNew("acc1", "cat1"))
	if !errors.Is(err, rt.ErrAccountDisabled) {
		t.Fatalf("err = %v, want ErrAccountDisabled", err)
	}
}

func TestCreateRequiresCategory(t *testing.T) {
	svc, accounts, _, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")

	in := baseNew("acc1", "")
	in.CategoryID = nil
	if _, err := svc.Create(context.Background(), "u1", in); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdatePermissionTiers(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	accounts.share("acc1", "append-user", "append")
	accounts.share("acc1", "other-append-user", "append")
	accounts.share("acc1", "admin-user", "entry_admin")

	created, err := svc.Create(context.Background(), "append-user", baseNew("acc1", "cat1"))
	if err != nil {
		t.Fatal(err)
	}

	newTitle := "Netflix Premium"
	// The creator (append tier) can edit their own.
	if _, err := svc.Update(context.Background(), "append-user", created.ID, rt.Update{Title: &newTitle}); err != nil {
		t.Fatalf("creator update: %v", err)
	}
	// A different append-tier user cannot edit someone else's.
	if _, err := svc.Update(context.Background(), "other-append-user", created.ID, rt.Update{Title: &newTitle}); !errors.Is(err, rt.ErrForbidden) {
		t.Fatalf("other append-tier update: err = %v, want ErrForbidden", err)
	}
	// entry_admin can edit anyone's.
	if _, err := svc.Update(context.Background(), "admin-user", created.ID, rt.Update{Title: &newTitle}); err != nil {
		t.Fatalf("admin update: %v", err)
	}
}

func TestDeleteBlockedWhileLinked(t *testing.T) {
	svc, accounts, categories, _, entries := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	created, err := svc.Create(context.Background(), "u1", baseNew("acc1", "cat1"))
	if err != nil {
		t.Fatal(err)
	}

	entries.count[created.ID] = 1
	if err := svc.Delete(context.Background(), "u1", created.ID); !errors.Is(err, rt.ErrInUse) {
		t.Fatalf("err = %v, want ErrInUse", err)
	}

	entries.count[created.ID] = 0
	if err := svc.Delete(context.Background(), "u1", created.ID); err != nil {
		t.Fatalf("delete once unlinked: %v", err)
	}
}

func TestNextSuggestedDateNoLinkedEntries(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	created, err := svc.Create(context.Background(), "u1", baseNew("acc1", "cat1"))
	if err != nil {
		t.Fatal(err)
	}
	if created.NextSuggestedDate.String() != "2026-01-01" {
		t.Errorf("next_suggested_date = %s, want 2026-01-01 (starts_on)", created.NextSuggestedDate)
	}
}

func TestNextSuggestedDateFromLatestLinkedEntry(t *testing.T) {
	svc, accounts, categories, _, entries := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	created, err := svc.Create(context.Background(), "u1", baseNew("acc1", "cat1"))
	if err != nil {
		t.Fatal(err)
	}

	entries.latest[created.ID] = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	got, err := svc.Get(context.Background(), "u1", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.NextSuggestedDate.String() != "2026-04-01" {
		t.Errorf("next_suggested_date = %s, want 2026-04-01 (advanced one month)", got.NextSuggestedDate)
	}
}

func TestSummaryExcludesEndedAndGroupsByCurrency(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc-eur", "u1", "EUR")
	accounts.add("acc-usd", "u1", "USD")
	categories.add("cat1")

	active := baseNew("acc-eur", "cat1")
	active.Amount = -100000
	if _, err := svc.Create(context.Background(), "u1", active); err != nil {
		t.Fatal(err)
	}

	ended := baseNew("acc-eur", "cat1")
	ended.Amount = -50000
	ended.StartsOn = mustDateT(2019, 1, 1)
	past := mustDateT(2020, 1, 1)
	ended.EndsOn = &past
	if _, err := svc.Create(context.Background(), "u1", ended); err != nil {
		t.Fatal(err)
	}

	usd := baseNew("acc-usd", "cat1")
	usd.Amount = 200000
	usd.IntervalUnit = rt.UnitYear
	usd.IntervalCount = 1
	if _, err := svc.Create(context.Background(), "u1", usd); err != nil {
		t.Fatal(err)
	}

	sum, err := svc.Summary(context.Background(), "u1", rt.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Count != 2 {
		t.Fatalf("count = %d, want 2 (ended one excluded)", sum.Count)
	}
	got := map[string]int64{}
	for _, cs := range sum.Sums {
		got[cs.Currency] = cs.Amount
	}
	if got["EUR"] != -1200000 {
		t.Errorf("EUR total = %d, want -1200000 (active only, -100000*12)", got["EUR"])
	}
	if got["USD"] != 200000 {
		t.Errorf("USD total = %d, want 200000", got["USD"])
	}
}

func TestPreviewRequiresCutoff(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	if _, err := svc.Create(context.Background(), "u1", baseNew("acc1", "cat1")); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{}); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestPreviewBasicProjection(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	in := baseNew("acc1", "cat1")
	in.IntervalUnit = rt.UnitDay
	in.IntervalCount = 10
	in.StartsOn = rt.NewDate(time.Now().AddDate(0, 0, 5))
	if _, err := svc.Create(context.Background(), "u1", in); err != nil {
		t.Fatal(err)
	}

	// starts +5, then +15, +25, +35 fall within the cutoff; +45 doesn't.
	cutoff := time.Now().AddDate(0, 0, 35)
	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{To: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatalf("len(items) = %d, want 4", len(items))
	}
	for _, it := range items {
		if it.Overdue {
			t.Errorf("item dated %s marked overdue unexpectedly", it.BookingTimestamp)
		}
	}
	if items[0].AccountCurrency != "EUR" || items[0].RecurringTransactionID == "" {
		t.Errorf("unexpected item: %+v", items[0])
	}
}

func TestPreviewSingleOverdueRow(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	in := baseNew("acc1", "cat1")
	in.IntervalUnit = rt.UnitDay
	in.IntervalCount = 1
	anchor := rt.NewDate(time.Now().AddDate(0, 0, -10))
	in.StartsOn = anchor
	if _, err := svc.Create(context.Background(), "u1", in); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now().AddDate(0, 0, 5)
	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{To: cutoff})
	if err != nil {
		t.Fatal(err)
	}

	overdueCount := 0
	for _, it := range items {
		if it.Overdue {
			overdueCount++
		}
	}
	if overdueCount != 1 {
		t.Fatalf("overdue rows = %d, want exactly 1 (%d total items)", overdueCount, len(items))
	}
	if len(items) == 0 || !items[0].Overdue {
		t.Fatalf("first item should be the overdue anchor row, got %+v", items)
	}
	if items[0].BookingTimestamp.String() != anchor.String() {
		t.Errorf("overdue row date = %s, want anchor date %s", items[0].BookingTimestamp, anchor)
	}
}

func TestPreviewEndsOnClampsTighterThanCutoff(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	in := baseNew("acc1", "cat1")
	in.IntervalUnit = rt.UnitDay
	in.IntervalCount = 1
	in.StartsOn = rt.NewDate(time.Now().AddDate(0, 0, 1))
	endsOn := rt.NewDate(time.Now().AddDate(0, 0, 3))
	in.EndsOn = &endsOn
	if _, err := svc.Create(context.Background(), "u1", in); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now().AddDate(0, 0, 30)
	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{To: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("expected at least the first occurrence")
	}
	for _, it := range items {
		if it.BookingTimestamp.After(endsOn) {
			t.Errorf("item dated %s is after ends_on %s, despite a later cutoff", it.BookingTimestamp, endsOn)
		}
	}
}

func TestPreviewEndedTemplateContributesNothing(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	in := baseNew("acc1", "cat1")
	in.StartsOn = mustDateT(2020, 1, 1)
	past := mustDateT(2020, 6, 1)
	in.EndsOn = &past
	if _, err := svc.Create(context.Background(), "u1", in); err != nil {
		t.Fatal(err)
	}

	cutoff := time.Now().AddDate(1, 0, 0)
	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{To: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0 (template already past its end date)", len(items))
	}
}

func TestPreviewFiltersByAccountCategoryAndTag(t *testing.T) {
	svc, accounts, categories, tags, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "EUR")
	categories.add("cat1")
	categories.add("cat2")
	tags.add("tag1", "u1")

	future := rt.NewDate(time.Now().AddDate(0, 0, 1))
	cutoff := time.Now().AddDate(0, 1, 0)

	inAcc1 := baseNew("acc1", "cat1")
	inAcc1.StartsOn = future
	if _, err := svc.Create(context.Background(), "u1", inAcc1); err != nil {
		t.Fatal(err)
	}

	inAcc2Cat2 := baseNew("acc2", "cat2")
	inAcc2Cat2.StartsOn = future
	inAcc2Cat2.TagIDs = []string{"tag1"}
	if _, err := svc.Create(context.Background(), "u1", inAcc2Cat2); err != nil {
		t.Fatal(err)
	}

	// Narrow by account.
	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{AccountIDs: []string{"acc1"}, To: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.AccountID != "acc1" {
			t.Errorf("account_id filter leaked item from %s", it.AccountID)
		}
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item for acc1")
	}

	// Narrow by category.
	cat2 := "cat2"
	items, err = svc.Preview(context.Background(), "u1", rt.PreviewFilter{CategoryID: &cat2, To: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.AccountID != "acc2" {
			t.Errorf("category filter should only match the acc2 template, got %s", it.AccountID)
		}
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item for cat2")
	}

	// Narrow by tag.
	tag1 := "tag1"
	items, err = svc.Preview(context.Background(), "u1", rt.PreviewFilter{TagID: &tag1, To: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.AccountID != "acc2" {
			t.Errorf("tag filter should only match the acc2 template, got %s", it.AccountID)
		}
	}
	if len(items) == 0 {
		t.Fatal("expected at least one item for tag1")
	}
}

func TestPreviewNoVisibleTemplatesReturnsEmpty(t *testing.T) {
	svc, _, _, _, _ := newService()
	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{To: time.Now().AddDate(0, 1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0", len(items))
	}
}

func TestPreviewCapsOccurrencesPerTemplate(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	in := baseNew("acc1", "cat1")
	in.IntervalUnit = rt.UnitDay
	in.IntervalCount = 1
	in.StartsOn = rt.NewDate(time.Now().AddDate(0, 0, 1))
	if _, err := svc.Create(context.Background(), "u1", in); err != nil {
		t.Fatal(err)
	}

	// A cutoff far beyond the 366-occurrence cap for a daily template.
	cutoff := time.Now().AddDate(3, 0, 0)
	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{To: cutoff})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 366 {
		t.Fatalf("len(items) = %d, want 366 (defensive cap)", len(items))
	}
}

func TestPreviewNoAccessibleAccountsReturnsEmpty(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u2", "EUR")
	categories.add("cat1")
	if _, err := svc.Create(context.Background(), "u2", baseNew("acc1", "cat1")); err != nil {
		t.Fatal(err)
	}

	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{To: time.Now().AddDate(0, 1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("len(items) = %d, want 0 (u1 has no permission on u2's account)", len(items))
	}
}
