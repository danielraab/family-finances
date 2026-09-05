package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/entry"
)

type entryFixture struct {
	entries  *EntryStore
	accounts *AccountStore
	cats     *CategoryStore
	owner    string
	accID    string
	catID    string
}

func newEntryFixture(t *testing.T) entryFixture {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	ctx := context.Background()
	authStore := NewAuthStore(pool)
	accStore := NewAccountStore(pool)
	catStore := NewCategoryStore(pool)
	entryStore := NewEntryStore(pool)

	owner := mustUser(t, authStore, "entryowner@example.com")
	typ, err := accStore.CreateType(ctx, "Checking-entry")
	if err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := accStore.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catStore.Create(ctx, category.New{Name: "Groceries-entry"})
	if err != nil {
		t.Fatal(err)
	}

	return entryFixture{
		entries: entryStore, accounts: accStore, cats: catStore,
		owner: owner.ID, accID: acc.ID, catID: cat.ID,
	}
}

func at(s string) time.Time {
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return tm
}

func TestPGEntryCreateGetUpdateDelete(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	e, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: 1234,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Coffee", CategoryID: &f.catID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.Amount != 1234 || e.OwnerID != f.owner {
		t.Fatalf("e = %+v", e)
	}

	got, err := f.entries.Get(ctx, f.owner, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Coffee" {
		t.Fatalf("got = %+v", got)
	}

	newTitle := "Coffee and pastry"
	updated, err := f.entries.Update(ctx, f.owner, e.ID, entry.Update{Title: &newTitle})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != newTitle {
		t.Fatalf("updated = %+v", updated)
	}

	if err := f.entries.SoftDelete(ctx, f.owner, e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Get(ctx, f.owner, e.ID); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGEntryTransactionWithoutCategoryViolatesCheck(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	_, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: 100,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "No category",
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue (CHECK constraint)", err)
	}
}

func TestPGEntryBalanceLiveComputation(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Amount: 10000,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Opening",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: -500,
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "Spend", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}

	balance, err := f.entries.Balance(ctx, f.accID, at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balance != 9500 {
		t.Fatalf("balance = %d, want 9500", balance)
	}

	// As of before the adjustment: base is 0, no transactions counted.
	early, err := f.entries.Balance(ctx, f.accID, at("2023-01-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if early != 0 {
		t.Fatalf("early balance = %d, want 0", early)
	}
}

func TestPGEntryBalanceSameMillisecondTieBreak(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()
	ts := at("2024-01-01T00:00:00.500Z")

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: 999999,
		BookingTimestamp: ts, Title: "Before adjustment", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Amount: 5000,
		BookingTimestamp: ts, Title: "Adjustment",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: 100,
		BookingTimestamp: ts, Title: "After adjustment", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}

	balance, err := f.entries.Balance(ctx, f.accID, at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balance != 5100 {
		t.Fatalf("balance = %d, want 5100 (insertion-order tie-break)", balance)
	}
}

func TestPGEntryListCursorPaginationRoundTrips(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	days := []string{"2024-01-01", "2024-01-02", "2024-01-03", "2024-01-04", "2024-01-05"}
	for _, d := range days {
		if _, err := f.entries.Create(ctx, f.owner, entry.New{
			AccountID: f.accID, Kind: entry.KindTransaction, Amount: 1,
			BookingTimestamp: at(d + "T00:00:00Z"), Title: "x", CategoryID: &f.catID,
		}); err != nil {
			t.Fatal(err)
		}
	}

	filter := entry.Filter{
		AccountIDs: []string{f.accID},
		Sort:       entry.SortBookingTimestamp,
		Dir:        entry.DirAsc,
		Limit:      2,
	}
	first, cursor1, err := f.entries.List(ctx, f.owner, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || cursor1 == nil {
		t.Fatalf("first = %v, cursor = %v", first, cursor1)
	}

	filter.After = cursor1
	second, cursor2, err := f.entries.List(ctx, f.owner, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 2 || cursor2 == nil {
		t.Fatalf("second = %v, cursor = %v", second, cursor2)
	}

	filter.After = cursor2
	third, cursor3, err := f.entries.List(ctx, f.owner, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(third) != 1 || cursor3 != nil {
		t.Fatalf("third = %v, cursor = %v, want 1 item and nil cursor", third, cursor3)
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
		t.Fatalf("saw %d distinct entries, want 5", len(seen))
	}
}

func TestPGEntryListFiltersByCategorySubtree(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	child, err := f.cats.Create(ctx, category.New{ParentID: &f.catID, Name: "Snacks-entry"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.cats.Create(ctx, category.New{Name: "Other-entry"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: 1,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "parent-cat", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: 2,
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "child-cat", CategoryID: &child.ID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: 3,
		BookingTimestamp: at("2024-01-03T00:00:00Z"), Title: "other-cat", CategoryID: &other.ID,
	}); err != nil {
		t.Fatal(err)
	}

	subtree, err := f.cats.Subtree(ctx, f.catID)
	if err != nil {
		t.Fatal(err)
	}
	items, _, err := f.entries.List(ctx, f.owner, entry.Filter{
		AccountIDs: []string{f.accID}, CategoryIDs: subtree,
		CategoryID: &f.catID, Sort: entry.SortBookingTimestamp, Dir: entry.DirAsc, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %v, want 2 (parent-cat + child-cat)", items)
	}
}

func TestPGEntrySoftDeletedExcludedFromBalance(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	e, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: 500,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "x", CategoryID: &f.catID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.entries.SoftDelete(ctx, f.owner, e.ID); err != nil {
		t.Fatal(err)
	}
	balance, err := f.entries.Balance(ctx, f.accID, at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balance != 0 {
		t.Fatalf("balance = %d, want 0 (deleted entry excluded)", balance)
	}
}
