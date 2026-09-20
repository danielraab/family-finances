package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/entry"
	rt "at.draab/familyfinances/internal/recurringtransaction"
)

type recurringFixture struct {
	recurring  *RecurringTransactionStore
	entries    *EntryStore
	tags       *TagStore
	categories *CategoryStore
	owner      string
	accID      string
	catID      string
}

func newRecurringFixture(t *testing.T) recurringFixture {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	ctx := context.Background()
	authStore := NewAuthStore(pool)
	accStore := NewAccountStore(pool)
	catStore := NewCategoryStore(pool)

	owner := mustUser(t, authStore, "recurringowner@example.com")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := accStore.Create(ctx, owner.ID, account.New{
		Title: "Main", Type: "Checking", Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catStore.Create(ctx, owner.ID, category.New{Name: "Subscriptions"})
	if err != nil {
		t.Fatal(err)
	}

	return recurringFixture{
		recurring:  NewRecurringTransactionStore(pool),
		entries:    NewEntryStore(pool),
		tags:       NewTagStore(pool),
		categories: catStore,
		owner:      owner.ID,
		accID:      acc.ID,
		catID:      cat.ID,
	}
}

func TestPGRecurringTransactionCreateGetUpdateDelete(t *testing.T) {
	f := newRecurringFixture(t)
	ctx := context.Background()

	tg, err := f.tags.Create(ctx, f.owner, "streaming")
	if err != nil {
		t.Fatal(err)
	}

	starts, _ := account.ParseDate("2026-01-01")
	created, err := f.recurring.Create(ctx, f.owner, rt.New{
		Kind:          rt.KindTransaction,
		AccountID:     f.accID,
		Title:         "Netflix",
		CategoryID:    &f.catID,
		Amount:        -1500,
		IntervalUnit:  rt.UnitMonth,
		IntervalCount: 1,
		StartsOn:      rt.NewDate(starts.Time),
		TagIDs:        []string{tg.ID},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Title != "Netflix" || created.Amount != -1500 || created.IntervalUnit != rt.UnitMonth {
		t.Fatalf("created = %+v", created)
	}
	if len(created.TagIDs) != 1 || created.TagIDs[0] != tg.ID {
		t.Fatalf("created.TagIDs = %v, want [%s]", created.TagIDs, tg.ID)
	}
	if created.StartsOn.String() != "2026-01-01" {
		t.Fatalf("StartsOn = %s, want 2026-01-01", created.StartsOn)
	}
	if created.AccountCurrency != "EUR" {
		t.Fatalf("AccountCurrency = %q, want EUR", created.AccountCurrency)
	}

	got, err := f.recurring.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("got = %+v", got)
	}

	newTitle := "Netflix Premium"
	newCount := 3
	updated, err := f.recurring.Update(ctx, created.ID, rt.Update{
		Title:         &newTitle,
		IntervalCount: &newCount,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != newTitle || updated.IntervalCount != 3 {
		t.Fatalf("updated = %+v", updated)
	}

	if err := f.recurring.SoftDelete(ctx, created.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if _, err := f.recurring.Get(ctx, created.ID); !errors.Is(err, rt.ErrNotFound) {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}
}

func TestPGRecurringTransactionEndsOnClearable(t *testing.T) {
	f := newRecurringFixture(t)
	ctx := context.Background()

	starts, _ := account.ParseDate("2026-01-01")
	ends, _ := account.ParseDate("2026-12-31")
	created, err := f.recurring.Create(ctx, f.owner, rt.New{
		Kind:          rt.KindTransaction,
		AccountID:     f.accID,
		Title:         "Gym",
		CategoryID:    &f.catID,
		Amount:        -3000,
		IntervalUnit:  rt.UnitMonth,
		IntervalCount: 1,
		StartsOn:      rt.NewDate(starts.Time),
		EndsOn:        ptrDate(rt.NewDate(ends.Time)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.EndsOn == nil || created.EndsOn.String() != "2026-12-31" {
		t.Fatalf("created.EndsOn = %v, want 2026-12-31", created.EndsOn)
	}

	cleared, err := f.recurring.Update(ctx, created.ID, rt.Update{
		EndsOn: rt.OptionalDate{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cleared.EndsOn != nil {
		t.Fatalf("EndsOn after clear = %v, want nil", cleared.EndsOn)
	}
}

func TestPGRecurringTransactionListScopedToAccounts(t *testing.T) {
	f := newRecurringFixture(t)
	ctx := context.Background()

	starts, _ := account.ParseDate("2026-01-01")
	if _, err := f.recurring.Create(ctx, f.owner, rt.New{
		Kind:      rt.KindTransaction,
		AccountID: f.accID, Title: "Rent", CategoryID: &f.catID, Amount: -80000,
		IntervalUnit: rt.UnitMonth, IntervalCount: 1, StartsOn: rt.NewDate(starts.Time),
	}); err != nil {
		t.Fatal(err)
	}

	list, err := f.recurring.List(ctx, rt.Filter{AccountIDs: []string{f.accID}})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("List = %d items, want 1", len(list))
	}

	empty, err := f.recurring.List(ctx, rt.Filter{AccountIDs: nil})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("List with no account ids = %d items, want 0", len(empty))
	}
}

// ptrDate is a small helper so a struct literal above can take a *rt.Date
// from a value.
func ptrDate(d rt.Date) *rt.Date { return &d }

func TestPGEntryLinkedToRecurringTransaction(t *testing.T) {
	f := newRecurringFixture(t)
	ctx := context.Background()

	starts, _ := account.ParseDate("2026-01-01")
	created, err := f.recurring.Create(ctx, f.owner, rt.New{
		Kind:      rt.KindTransaction,
		AccountID: f.accID, Title: "Rent", CategoryID: &f.catID, Amount: -80000,
		IntervalUnit: rt.UnitMonth, IntervalCount: 1, StartsOn: rt.NewDate(starts.Time),
	})
	if err != nil {
		t.Fatal(err)
	}

	if count, err := f.entries.CountByRecurringTransaction(ctx, created.ID); err != nil || count != 0 {
		t.Fatalf("CountByRecurringTransaction before linking = %d, %v", count, err)
	}
	if latest, err := f.entries.LatestBookingTimeByRecurringTransaction(ctx, created.ID); err != nil || latest != nil {
		t.Fatalf("LatestBookingTimeByRecurringTransaction before linking = %v, %v", latest, err)
	}

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	e1, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-80000),
		BookingTimestamp: ts, Title: "Rent Jan", CategoryID: &f.catID,
		RecurringTransactionID: &created.ID,
	})
	if err != nil {
		t.Fatalf("Create linked entry: %v", err)
	}
	if e1.RecurringTransactionID == nil || *e1.RecurringTransactionID != created.ID {
		t.Fatalf("e1.RecurringTransactionID = %v, want %s", e1.RecurringTransactionID, created.ID)
	}

	ts2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-80000),
		BookingTimestamp: ts2, Title: "Rent Feb", CategoryID: &f.catID,
		RecurringTransactionID: &created.ID,
	}); err != nil {
		t.Fatalf("Create second linked entry: %v", err)
	}

	if count, err := f.entries.CountByRecurringTransaction(ctx, created.ID); err != nil || count != 2 {
		t.Fatalf("CountByRecurringTransaction = %d, %v, want 2", count, err)
	}
	latest, err := f.entries.LatestBookingTimeByRecurringTransaction(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if latest == nil || !latest.Equal(ts2) {
		t.Fatalf("latest = %v, want %v", latest, ts2)
	}

	// Unlinking (explicit null) removes it from the count.
	if _, err := f.entries.Update(ctx, e1.ID, entry.Update{
		RecurringTransactionID: entry.OptionalID{Set: true, Value: nil},
	}); err != nil {
		t.Fatalf("Unlink: %v", err)
	}
	if count, err := f.entries.CountByRecurringTransaction(ctx, created.ID); err != nil || count != 1 {
		t.Fatalf("CountByRecurringTransaction after unlink = %d, %v, want 1", count, err)
	}
}

// TestPGRecurringTransactionListFiltersByCategoryAndTag covers the two
// clauses List builds beyond the account scope. CategoryIDs is read for
// nil, never for length, so an empty-but-non-nil set has to match nothing
// — that is the case a caller filtering by a category they cannot see
// resolves to (see rt.Filter's doc comment).
func TestPGRecurringTransactionListFiltersByCategoryAndTag(t *testing.T) {
	f := newRecurringFixture(t)
	ctx := context.Background()

	other, err := f.categories.Create(ctx, f.owner, category.New{Name: "Utilities"})
	if err != nil {
		t.Fatal(err)
	}
	tg, err := f.tags.Create(ctx, f.owner, "fixed")
	if err != nil {
		t.Fatal(err)
	}

	starts, _ := account.ParseDate("2026-01-01")
	create := func(title, categoryID string, tagIDs ...string) {
		t.Helper()
		if _, err := f.recurring.Create(ctx, f.owner, rt.New{
			Kind:      rt.KindTransaction,
			AccountID: f.accID, Title: title, CategoryID: &categoryID, Amount: -80000,
			IntervalUnit: rt.UnitMonth, IntervalCount: 1, StartsOn: rt.NewDate(starts.Time),
			TagIDs: tagIDs,
		}); err != nil {
			t.Fatal(err)
		}
	}
	create("Netflix", f.catID)
	create("Electricity", other.ID, tg.ID)

	accounts := []string{f.accID}
	titlesOf := func(list []rt.RecurringTransaction) []string {
		out := make([]string, 0, len(list))
		for _, item := range list {
			out = append(out, item.Title)
		}
		return out
	}

	cases := []struct {
		name   string
		filter rt.Filter
		want   []string
	}{
		{"no content filter", rt.Filter{AccountIDs: accounts}, []string{"Netflix", "Electricity"}},
		{"by category", rt.Filter{AccountIDs: accounts, CategoryIDs: []string{other.ID}}, []string{"Electricity"}},
		{"by tag", rt.Filter{AccountIDs: accounts, TagID: &tg.ID}, []string{"Electricity"}},
		{"category and tag combined", rt.Filter{AccountIDs: accounts, CategoryIDs: []string{f.catID}, TagID: &tg.ID}, nil},
		{"empty category set matches nothing", rt.Filter{AccountIDs: accounts, CategoryIDs: []string{}}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list, err := f.recurring.List(ctx, tc.filter)
			if err != nil {
				t.Fatal(err)
			}
			got := titlesOf(list)
			if len(got) != len(tc.want) {
				t.Fatalf("List = %v, want %v", got, tc.want)
			}
			for i, title := range tc.want {
				if got[i] != title {
					t.Fatalf("List = %v, want %v", got, tc.want)
				}
			}
		})
	}
}
