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

func ptrInt64(n int64) *int64 { return &n }

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
	typ, err := accStore.CreateType(ctx, owner.ID, "Checking-entry", "")
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
	cat, err := catStore.Create(ctx, owner.ID, category.New{Name: "Groceries-entry"})
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
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(1234),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Coffee", CategoryID: &f.catID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if e.Amount != 1234 || e.CreatedBy != f.owner {
		t.Fatalf("e = %+v", e)
	}

	got, err := f.entries.Get(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Coffee" {
		t.Fatalf("got = %+v", got)
	}

	newTitle := "Coffee and pastry"
	updated, err := f.entries.Update(ctx, e.ID, entry.Update{Title: &newTitle})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != newTitle {
		t.Fatalf("updated = %+v", updated)
	}

	if err := f.entries.SoftDelete(ctx, e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Get(ctx, e.ID); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGEntryUpdateMovesAccount(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	opening, _ := account.ParseDate("2024-01-01")
	acc2, err := f.accounts.Create(ctx, f.owner, account.New{
		Title: "Savings", TypeID: mustType(t, f.accounts, f.owner), Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	e, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(1234),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Coffee", CategoryID: &f.catID,
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := f.entries.Update(ctx, e.ID, entry.Update{AccountID: &acc2.ID})
	if err != nil {
		t.Fatal(err)
	}
	if updated.AccountID != acc2.ID {
		t.Fatalf("AccountID = %q, want %q", updated.AccountID, acc2.ID)
	}

	got, err := f.entries.Get(ctx, e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountID != acc2.ID {
		t.Fatalf("persisted AccountID = %q, want %q", got.AccountID, acc2.ID)
	}
}

func mustType(t *testing.T, accStore *AccountStore, owner string) string {
	t.Helper()
	typ, err := accStore.CreateType(context.Background(), owner, "Savings-entry-move", "")
	if err != nil {
		t.Fatal(err)
	}
	return typ.ID
}

func TestPGEntryTransactionWithoutCategoryViolatesCheck(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	_, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(100),
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
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(10000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Opening",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-500),
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
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(999999),
		BookingTimestamp: ts, Title: "Before adjustment", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(5000),
		BookingTimestamp: ts, Title: "Adjustment",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(100),
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
			AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(1),
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
	first, cursor1, err := f.entries.List(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || cursor1 == nil {
		t.Fatalf("first = %v, cursor = %v", first, cursor1)
	}

	filter.After = cursor1
	second, cursor2, err := f.entries.List(ctx, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 2 || cursor2 == nil {
		t.Fatalf("second = %v, cursor = %v", second, cursor2)
	}

	filter.After = cursor2
	third, cursor3, err := f.entries.List(ctx, filter)
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

	child, err := f.cats.Create(ctx, f.owner, category.New{ParentID: &f.catID, Name: "Snacks-entry"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := f.cats.Create(ctx, f.owner, category.New{Name: "Other-entry"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(1),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "parent-cat", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(2),
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "child-cat", CategoryID: &child.ID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(3),
		BookingTimestamp: at("2024-01-03T00:00:00Z"), Title: "other-cat", CategoryID: &other.ID,
	}); err != nil {
		t.Fatal(err)
	}

	subtree, err := f.cats.Subtree(ctx, f.owner, f.catID)
	if err != nil {
		t.Fatal(err)
	}
	items, _, err := f.entries.List(ctx, entry.Filter{
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

func TestPGEntrySumGroupsByAccountAndExcludesBalanceAdjustments(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	opening, _ := account.ParseDate("2024-01-01")
	acc2, err := f.accounts.Create(ctx, f.owner, account.New{
		Title: "Savings-sum", TypeID: mustType(t, f.accounts, f.owner), Currency: "USD", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-100),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "x", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-50),
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "y", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: acc2.ID, Kind: entry.KindTransaction, Amount: ptrInt64(-20),
		BookingTimestamp: at("2024-01-03T00:00:00Z"), Title: "z", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(99999),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "adj",
	}); err != nil {
		t.Fatal(err)
	}

	perAccount, count, err := f.entries.Sum(ctx, entry.Filter{
		AccountIDs: []string{f.accID, acc2.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3 (balance adjustment excluded)", count)
	}
	if perAccount[f.accID] != -150 {
		t.Fatalf("perAccount[accID] = %d, want -150", perAccount[f.accID])
	}
	if perAccount[acc2.ID] != -20 {
		t.Fatalf("perAccount[acc2.ID] = %d, want -20", perAccount[acc2.ID])
	}
}

func TestPGEntrySumExactModeExcludesDescendants(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	child, err := f.cats.Create(ctx, f.owner, category.New{ParentID: &f.catID, Name: "Snacks-entry-sum"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-10),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "parent-cat", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-5),
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "child-cat", CategoryID: &child.ID,
	}); err != nil {
		t.Fatal(err)
	}

	perAccount, count, err := f.entries.Sum(ctx, entry.Filter{
		AccountIDs: []string{f.accID}, CategoryID: &f.catID, CategoryIDs: []string{f.catID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || perAccount[f.accID] != -10 {
		t.Fatalf("perAccount = %v, count = %d, want {accID: -10}, 1 (exact category only)", perAccount, count)
	}
}

func TestPGEntrySoftDeletedExcludedFromBalance(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	e, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(500),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "x", CategoryID: &f.catID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.entries.SoftDelete(ctx, e.ID); err != nil {
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

// --- balance adjustment delta recompute (real SQL: findAdjustment,
// setAmount, recomputeFrom in entry.go) ---------------------------------

func TestPGEntryTransactionInsertedBeforeAdjustmentShiftsItsDelta(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	adj, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(20000),
		BookingTimestamp: at("2024-02-01T00:00:00Z"), Title: "Opening",
	})
	if err != nil {
		t.Fatal(err)
	}
	if adj.Amount != 20000 {
		t.Fatalf("initial Amount = %d, want 20000", adj.Amount)
	}

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(5000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Earlier", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := f.entries.Get(ctx, adj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount != 15000 {
		t.Fatalf("adjustment Amount after inserting an earlier transaction = %d, want 15000", got.Amount)
	}
	balance, err := f.entries.Balance(ctx, f.accID, at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balance != 20000 {
		t.Fatalf("balance = %d, want 20000 (the adjustment's reading, unaffected)", balance)
	}
}

func TestPGEntryEditingTransactionBetweenTwoAdjustmentsOnlyRecomputesTheFollowingOne(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	adjA, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(10000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "A",
	})
	if err != nil {
		t.Fatal(err)
	}
	txn, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-2000),
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "between", CategoryID: &f.catID,
	})
	if err != nil {
		t.Fatal(err)
	}
	adjB, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(20000),
		BookingTimestamp: at("2024-01-03T00:00:00Z"), Title: "B",
	})
	if err != nil {
		t.Fatal(err)
	}
	if adjB.Amount != 12000 {
		t.Fatalf("AdjB.Amount = %d, want 12000 (20000 - (10000-2000))", adjB.Amount)
	}

	newAmount := int64(-5000)
	if _, err := f.entries.Update(ctx, txn.ID, entry.Update{Amount: &newAmount}); err != nil {
		t.Fatal(err)
	}

	gotA, err := f.entries.Get(ctx, adjA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotA.Amount != 10000 {
		t.Fatalf("AdjA.Amount = %d, want unchanged 10000", gotA.Amount)
	}
	gotB, err := f.entries.Get(ctx, adjB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotB.Amount != 15000 {
		t.Fatalf("AdjB.Amount after editing the transaction between = %d, want 15000 (20000 - (10000-5000))", gotB.Amount)
	}
}

func TestPGEntryDeletingAnAdjustmentShiftsTheNextOnesBaseline(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(10000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "A",
	}); err != nil {
		t.Fatal(err)
	}
	adjB, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(15000),
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "B",
	})
	if err != nil {
		t.Fatal(err)
	}
	adjC, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(20000),
		BookingTimestamp: at("2024-01-03T00:00:00Z"), Title: "C",
	})
	if err != nil {
		t.Fatal(err)
	}
	if adjC.Amount != 5000 {
		t.Fatalf("AdjC.Amount before delete = %d, want 5000", adjC.Amount)
	}

	if err := f.entries.SoftDelete(ctx, adjB.ID); err != nil {
		t.Fatal(err)
	}

	gotC, err := f.entries.Get(ctx, adjC.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotC.Amount != 10000 {
		t.Fatalf("AdjC.Amount after deleting AdjB = %d, want 10000 (20000-10000, now against AdjA)", gotC.Amount)
	}
}

func TestPGEntryMovingAnAdjustmentPastAnotherRecomputesBothNeighbors(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(10000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "A",
	}); err != nil {
		t.Fatal(err)
	}
	adjB, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(15000),
		BookingTimestamp: at("2024-01-05T00:00:00Z"), Title: "B",
	})
	if err != nil {
		t.Fatal(err)
	}
	adjC, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(30000),
		BookingTimestamp: at("2024-01-10T00:00:00Z"), Title: "C",
	})
	if err != nil {
		t.Fatal(err)
	}

	newTS := at("2024-01-15T00:00:00Z")
	if _, err := f.entries.Update(ctx, adjB.ID, entry.Update{BookingTimestamp: &newTS}); err != nil {
		t.Fatal(err)
	}

	gotC, err := f.entries.Get(ctx, adjC.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotC.Amount != 20000 {
		t.Fatalf("AdjC.Amount after AdjB moved past it = %d, want 20000 (30000-10000, now against AdjA)", gotC.Amount)
	}
	gotB, err := f.entries.Get(ctx, adjB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotB.Amount != -15000 {
		t.Fatalf("AdjB.Amount after moving past AdjC = %d, want -15000 (15000-30000, now against AdjC)", gotB.Amount)
	}
}

func TestPGEntryMovingAcrossAccountsRecomputesBothAccountsAdjustments(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	typ, err := f.accounts.CreateType(ctx, f.owner, "Savings-entry", "")
	if err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	acc2, err := f.accounts.Create(ctx, f.owner, account.New{
		Title: "Second", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	// acc1: an adjustment, then a transaction that will move to acc2.
	adj1, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(10000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "acc1 opening",
	})
	if err != nil {
		t.Fatal(err)
	}
	txn, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(1000),
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "movable", CategoryID: &f.catID,
	})
	if err != nil {
		t.Fatal(err)
	}
	adj1b, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(20000),
		BookingTimestamp: at("2024-01-03T00:00:00Z"), Title: "acc1 later",
	})
	if err != nil {
		t.Fatal(err)
	}
	if adj1b.Amount != 9000 {
		t.Fatalf("adj1b.Amount before move = %d, want 9000 (20000 - (10000+1000))", adj1b.Amount)
	}

	// acc2: an adjustment whose delta will shift once the transaction lands
	// before it.
	adj2, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: acc2.ID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(5000),
		BookingTimestamp: at("2024-01-05T00:00:00Z"), Title: "acc2 opening",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Move the transaction from acc1 to acc2.
	if _, err := f.entries.Update(ctx, txn.ID, entry.Update{AccountID: &acc2.ID}); err != nil {
		t.Fatal(err)
	}

	// acc1's later adjustment loses the moved transaction's contribution.
	gotAdj1b, err := f.entries.Get(ctx, adj1b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAdj1b.Amount != 10000 {
		t.Fatalf("adj1b.Amount after the transaction moved away = %d, want 10000 (20000-10000)", gotAdj1b.Amount)
	}
	// acc2's adjustment now has to absorb the newly-arrived transaction.
	gotAdj2, err := f.entries.Get(ctx, adj2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAdj2.Amount != 4000 {
		t.Fatalf("adj2.Amount after the transaction moved in = %d, want 4000 (5000-1000)", gotAdj2.Amount)
	}
	_ = adj1
}

// TestPGEntryMigration0018BackfillPreservesBalances applies every migration
// up to (not including) 0018, inserts legacy-shape data directly (a
// balance_adjustment's amount holding the absolute reading, exactly as
// entries were stored before this change — no balance_reading column
// exists yet), then applies 0018 and confirms the new plain-SUM Balance()
// reproduces exactly what the old two-step formula computed by hand. It
// also confirms a post-migration mutation keeps working through the new
// recompute path.
func TestPGEntryMigration0018BackfillPreservesBalances(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	all, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}
	var pre, at0018, post []migration
	for _, m := range all {
		if m.version < "0018" {
			pre = append(pre, m)
		} else if m.version == "0018" {
			at0018 = append(at0018, m)
		} else {
			post = append(post, m)
		}
	}
	if len(at0018) != 1 {
		t.Fatalf("found %d migrations at version 0018, want 1", len(at0018))
	}
	// Apply everything except 0018: the pre-0018 schema, plus every later
	// migration (all of which only add accounts/categories columns and never
	// touch `entries`), so the account/category store fixtures below run
	// against the schema the current store code expects. `entries` still has
	// its pre-0018 shape — no balance_reading column — which is what this
	// test needs before it hand-inserts legacy-shape adjustment rows.
	if err := runMigrations(ctx, pool, pre); err != nil {
		t.Fatalf("apply pre-0018 migrations: %v", err)
	}
	if err := runMigrations(ctx, pool, post); err != nil {
		t.Fatalf("apply post-0018 migrations: %v", err)
	}

	authStore := NewAuthStore(pool)
	accStore := NewAccountStore(pool)
	catStore := NewCategoryStore(pool)
	owner := mustUser(t, authStore, "legacy@example.com")
	typ, err := accStore.CreateType(ctx, owner.ID, "Checking-legacy", "")
	if err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := accStore.Create(ctx, owner.ID, account.New{
		Title: "Legacy", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catStore.Create(ctx, owner.ID, category.New{Name: "Legacy cat"})
	if err != nil {
		t.Fatal(err)
	}

	insert := func(kind string, amount int64, ts string, catID *string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `
			INSERT INTO entries (created_by, account_id, kind, amount, booking_timestamp, title, category_id)
			VALUES ($1, $2, $3, $4, $5, 'legacy', $6)`,
			owner.ID, acc.ID, kind, amount, at(ts), catID,
		); err != nil {
			t.Fatalf("insert legacy %s entry: %v", kind, err)
		}
	}

	// Old-style history, mirroring design.md's worked example: a
	// transaction, an adjustment (absolute reading 10000), a transaction
	// between adjustments, a second adjustment (absolute reading 20000),
	// then one more transaction after it.
	insert("transaction", 999999, "2023-01-01T00:00:00Z", &cat.ID)
	insert("balance_adjustment", 10000, "2024-01-01T00:00:00Z", nil)
	insert("transaction", -2000, "2024-01-02T00:00:00Z", &cat.ID)
	insert("balance_adjustment", 20000, "2024-01-03T00:00:00Z", nil)
	insert("transaction", -500, "2024-01-04T00:00:00Z", &cat.ID)

	asOf := at("2024-06-01T00:00:00Z")
	wantBalance := int64(20000 - 500) // second adjustment + the one transaction strictly after it

	var preBalance int64
	if err := pool.QueryRow(ctx, `
		WITH base AS (
			SELECT booking_timestamp, id, amount FROM entries
			WHERE account_id = $1 AND kind = 'balance_adjustment' AND deleted_at IS NULL AND booking_timestamp <= $2
			ORDER BY booking_timestamp DESC, id DESC LIMIT 1
		), txns AS (
			SELECT COALESCE(SUM(e.amount), 0) AS total FROM entries e LEFT JOIN base ON true
			WHERE e.account_id = $1 AND e.kind = 'transaction' AND e.deleted_at IS NULL AND e.booking_timestamp <= $2
				AND (base.id IS NULL OR (e.booking_timestamp, e.id) > (base.booking_timestamp, base.id))
		)
		SELECT COALESCE((SELECT amount FROM base), 0) + (SELECT total FROM txns)`,
		acc.ID, asOf,
	).Scan(&preBalance); err != nil {
		t.Fatalf("pre-migration balance query: %v", err)
	}
	if preBalance != wantBalance {
		t.Fatalf("sanity check: pre-migration formula = %d, want %d", preBalance, wantBalance)
	}

	if err := runMigrations(ctx, pool, at0018); err != nil {
		t.Fatalf("apply 0018: %v", err)
	}

	entryStore := NewEntryStore(pool)
	gotBalance, err := entryStore.Balance(ctx, acc.ID, asOf)
	if err != nil {
		t.Fatal(err)
	}
	if gotBalance != wantBalance {
		t.Fatalf("post-migration Balance() = %d, want %d (unchanged by the backfill)", gotBalance, wantBalance)
	}

	// A post-migration mutation between the two (now-backfilled)
	// adjustments must still keep the second one's balance_reading exact —
	// proving the ordinary recompute path keeps working on backfilled data.
	if _, err := entryStore.Create(ctx, owner.ID, entry.New{
		AccountID: acc.ID, Kind: entry.KindTransaction, Amount: ptrInt64(-100),
		BookingTimestamp: at("2024-01-02T12:00:00Z"), Title: "post-migration", CategoryID: &cat.ID,
	}); err != nil {
		t.Fatal(err)
	}
	gotBalance2, err := entryStore.Balance(ctx, acc.ID, asOf)
	if err != nil {
		t.Fatal(err)
	}
	if gotBalance2 != wantBalance {
		t.Fatalf("Balance() after a post-migration transaction between the two adjustments = %d, want %d (unaffected — absorbed by the second adjustment's reset)", gotBalance2, wantBalance)
	}
}

// --- flow summary (real SQL: date_trunc/timezone/FILTER in entry.go) -----

func TestPGEntryFlowSummaryMonthlyBucketsWithAdjustmentDelta(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(1000),
		BookingTimestamp: at("2024-01-15T00:00:00Z"), Title: "income", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(-500),
		BookingTimestamp: at("2024-01-20T00:00:00Z"), Title: "correction",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(300),
		BookingTimestamp: at("2024-03-01T00:00:00Z"), Title: "other month", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := f.entries.FlowSummary(ctx, entry.FlowFilter{
		AccountIDs: []string{f.accID}, Unit: entry.FlowUnitMonth, Year: 2024, Timezone: "UTC",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %+v, want 2 (January and March)", rows)
	}
	byPeriod := map[string]entry.FlowRow{}
	for _, r := range rows {
		byPeriod[r.Period] = r
	}
	// The adjustment's delta is -500 (its reading) minus 1000 (the balance
	// strictly before it, from the earlier transaction) = -1500.
	jan, ok := byPeriod["2024-01-01"]
	if !ok || jan.Income != 1000 || jan.Outcome != 1500 {
		t.Fatalf("January row = %+v, want income 1000, outcome 1500 (adjustment's computed delta)", jan)
	}
	mar, ok := byPeriod["2024-03-01"]
	if !ok || mar.Income != 300 || mar.Outcome != 0 {
		t.Fatalf("March row = %+v, want income 300, outcome 0", mar)
	}
}

func TestPGEntryFlowSummaryDailyBuckets(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-250),
		BookingTimestamp: at("2024-02-05T00:00:00Z"), Title: "x", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := f.entries.FlowSummary(ctx, entry.FlowFilter{
		AccountIDs: []string{f.accID}, Unit: entry.FlowUnitDay, Year: 2024, Month: 2, Timezone: "UTC",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Period != "2024-02-05" || rows[0].Outcome != 250 {
		t.Fatalf("rows = %+v, want one row for 2024-02-05 with outcome 250", rows)
	}
}

func TestPGEntryFlowSummaryUsesGivenTimezone(t *testing.T) {
	f := newEntryFixture(t)
	ctx := context.Background()

	// 2024-02-01T02:00:00Z is 2024-01-31T21:00:00-05:00 in America/New_York
	// — January there, not February.
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(100),
		BookingTimestamp: at("2024-02-01T02:00:00Z"), Title: "x", CategoryID: &f.catID,
	}); err != nil {
		t.Fatal(err)
	}

	rows, err := f.entries.FlowSummary(ctx, entry.FlowFilter{
		AccountIDs: []string{f.accID}, Unit: entry.FlowUnitMonth, Year: 2024, Timezone: "America/New_York",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Period != "2024-01-01" {
		t.Fatalf("rows = %+v, want a single January row (America/New_York local date)", rows)
	}
}
