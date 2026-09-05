package entry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"at.draab/familyfinances/internal/entry"
	"at.draab/familyfinances/internal/storage/memory"
)

func newFixture() (*entry.Service, *stubAccounts, *stubCategories, *stubTags) {
	accounts := newStubAccounts()
	categories := newStubCategories()
	tags := newStubTags()
	svc := entry.NewService(memory.NewEntryStore(), accounts, categories, tags)
	return svc, accounts, categories, tags
}

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestCreateTransactionWithoutCategoryRejected(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: 100,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Coffee",
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateBalanceAdjustmentWithoutCategoryAccepted(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindBalanceAdjustment, Amount: 10000,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Opening balance",
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.CategoryID != nil {
		t.Fatalf("CategoryID = %v, want nil", e.CategoryID)
	}
}

func TestCreateAgainstOtherOwnersAccountRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u2", "EUR")
	categories.add("cat1")

	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: 100,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateAgainstDisabledAccountRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.disabled["acc1"] = true
	categories.add("cat1")

	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: 100,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if !errors.Is(err, entry.ErrAccountDisabled) {
		t.Fatalf("err = %v, want ErrAccountDisabled", err)
	}
}

func TestCreateWithForeignTagRejected(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	tags.add("tag1", "u2")

	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: 100,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
		TagIDs: []string{"tag1"},
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateClearingCategoryOnTransactionRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: 100,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Update(context.Background(), "u1", e.ID, entry.Update{
		CategoryID: entry.OptionalID{Set: true, Value: nil},
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestSoftDeleteExcludesFromGetAndListing(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: 100,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(context.Background(), "u1", e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "u1", e.ID); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
	}
	items, _, err := svc.List(context.Background(), "u1", entry.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("List after delete = %v, want empty", items)
	}
}

// --- balance ---------------------------------------------------------

func TestBalanceStartsFromZeroWithNoAdjustment(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 500, "2024-01-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -200, "2024-01-02T00:00:00Z", ptr("cat1"))

	bal, err := svc.Balance(context.Background(), "u1", "acc1", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if bal != 300 {
		t.Fatalf("balance = %d, want 300", bal)
	}
}

func TestBalanceUsesLatestAdjustmentAsBaseline(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 999999, "2023-01-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 10000, "2024-01-01T00:00:00Z", nil)
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -500, "2024-01-02T00:00:00Z", ptr("cat1"))

	bal, err := svc.Balance(context.Background(), "u1", "acc1", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if bal != 9500 {
		t.Fatalf("balance = %d, want 9500", bal)
	}
}

func TestBalanceAsOfPastPointIgnoresLaterEntries(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 1000, "2024-01-01T00:00:00Z", nil)
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 500, "2024-02-01T00:00:00Z", ptr("cat1"))

	bal, err := svc.Balance(context.Background(), "u1", "acc1", at("2024-01-15T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if bal != 1000 {
		t.Fatalf("balance as of before the transaction = %d, want 1000", bal)
	}
}

func TestBalanceSameMillisecondTieBreaksByInsertionOrder(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	// Both at the exact same timestamp: the transaction inserted before the
	// adjustment should NOT count (adjustment, inserted later, wins the
	// tie-break and becomes the baseline); a transaction inserted after
	// the adjustment at the same timestamp SHOULD count.
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 999999, "2024-01-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 5000, "2024-01-01T00:00:00Z", nil)
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-01-01T00:00:00Z", ptr("cat1"))

	bal, err := svc.Balance(context.Background(), "u1", "acc1", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if bal != 5100 {
		t.Fatalf("balance = %d, want 5100", bal)
	}
}

func TestBalanceCrossOwnerAccountNotFound(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u2", "EUR")
	if _, err := svc.Balance(context.Background(), "u1", "acc1", time.Now()); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func mustCreate(t *testing.T, svc *entry.Service, owner, accountID string, kind entry.Kind, amount int64, ts string, categoryID *string) entry.Entry {
	t.Helper()
	e, err := svc.Create(context.Background(), owner, entry.New{
		AccountID: accountID, Kind: kind, Amount: amount,
		BookingTimestamp: at(ts), Title: "x", CategoryID: categoryID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return e
}

// --- listing: filters, search, sort, cursor pagination ----------------

func TestListFiltersByCategoryIncludingDescendants(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("parent")
	categories.add("child")
	categories.children["parent"] = []string{"child"}

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 1, "2024-01-01T00:00:00Z", ptr("child"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 2, "2024-01-02T00:00:00Z", ptr("parent"))
	categories.add("other")
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 3, "2024-01-03T00:00:00Z", ptr("other"))

	items, _, err := svc.List(context.Background(), "u1", entry.Filter{CategoryID: ptr("parent")})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %v, want 2 (parent + child)", items)
	}
}

func TestListSearchMatchesTitleOrDescription(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	if _, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: 1,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Morning coffee", CategoryID: ptr("cat1"),
	}); err != nil {
		t.Fatal(err)
	}
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 2, "2024-01-02T00:00:00Z", ptr("cat1"))

	items, _, err := svc.List(context.Background(), "u1", entry.Filter{Query: "coffee"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Morning coffee" {
		t.Fatalf("items = %v", items)
	}
}

func TestListSortByAmount(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 300, "2024-01-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-01-02T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 200, "2024-01-03T00:00:00Z", ptr("cat1"))

	items, _, err := svc.List(context.Background(), "u1", entry.Filter{Sort: entry.SortAmount, Dir: entry.DirAsc})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].Amount != 100 || items[1].Amount != 200 || items[2].Amount != 300 {
		t.Fatalf("items = %+v", items)
	}
}

func TestListCursorPaginationRoundTrips(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	days := []string{"2024-01-01", "2024-01-02", "2024-01-03", "2024-01-04", "2024-01-05"}
	for _, d := range days {
		mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 1, d+"T00:00:00Z", ptr("cat1"))
	}

	first, cursor1, err := svc.List(context.Background(), "u1", entry.Filter{Sort: entry.SortBookingTimestamp, Dir: entry.DirAsc, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || cursor1 == nil {
		t.Fatalf("first page = %v, cursor = %v", first, cursor1)
	}

	second, cursor2, err := svc.List(context.Background(), "u1", entry.Filter{Sort: entry.SortBookingTimestamp, Dir: entry.DirAsc, Limit: 2, After: cursor1})
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 2 || cursor2 == nil {
		t.Fatalf("second page = %v, cursor = %v", second, cursor2)
	}

	third, cursor3, err := svc.List(context.Background(), "u1", entry.Filter{Sort: entry.SortBookingTimestamp, Dir: entry.DirAsc, Limit: 2, After: cursor2})
	if err != nil {
		t.Fatal(err)
	}
	if len(third) != 1 || cursor3 != nil {
		t.Fatalf("third page = %v, cursor = %v, want 1 item and nil cursor", third, cursor3)
	}

	seen := map[string]bool{}
	for _, page := range [][]entry.Entry{first, second, third} {
		for _, e := range page {
			if seen[e.ID] {
				t.Fatalf("id %s seen twice across pages", e.ID)
			}
			seen[e.ID] = true
		}
	}
	if len(seen) != 5 {
		t.Fatalf("saw %d distinct entries across pages, want 5", len(seen))
	}
}
