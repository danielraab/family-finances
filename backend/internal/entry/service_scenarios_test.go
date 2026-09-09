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
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
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
		AccountID: "acc1", Kind: entry.KindBalanceAdjustment, Balance: ptr(int64(10000)),
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
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
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
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if !errors.Is(err, entry.ErrAccountDisabled) {
		t.Fatalf("err = %v, want ErrAccountDisabled", err)
	}
}

func TestCreateWithForeignCategoryRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.addOwnedBy("cat1", "u2")

	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateWithDisabledCategoryRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	categories.disable("cat1")

	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateUnrelatedFieldKeepsSinceDisabledCategory(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if err != nil {
		t.Fatal(err)
	}

	categories.disable("cat1")

	got, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{Title: ptr("Y")})
	if err != nil {
		t.Fatalf("Update with an unrelated field on a since-disabled category: %v", err)
	}
	if got.Title != "Y" || got.CategoryID == nil || *got.CategoryID != "cat1" {
		t.Fatalf("got = %+v", got)
	}
}

func TestUpdateExplicitlySettingDisabledCategoryRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	categories.add("cat2")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if err != nil {
		t.Fatal(err)
	}
	categories.disable("cat2")

	_, err = svc.Update(context.Background(), "u1", e.ID, entry.Update{
		CategoryID: entry.OptionalID{Set: true, Value: ptr("cat2")},
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateWithForeignTagRejected(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	tags.add("tag1", "u2")

	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
		TagIDs: []string{"tag1"},
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateWithDisabledTagRejected(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	tags.add("tag1", "u1")
	tags.disable("tag1")

	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
		TagIDs: []string{"tag1"},
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateUnrelatedFieldKeepsSinceDisabledTag(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	tags.add("tag1", "u1")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
		TagIDs: []string{"tag1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	tags.disable("tag1")

	got, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{Title: ptr("Y")})
	if err != nil {
		t.Fatalf("Update with an unrelated field on a since-disabled tag: %v", err)
	}
	if got.Title != "Y" || len(got.TagIDs) != 1 || got.TagIDs[0] != "tag1" {
		t.Fatalf("got = %+v", got)
	}
}

func TestUpdateResubmittingUnchangedSinceDisabledTagSucceeds(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	tags.add("tag1", "u1")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
		TagIDs: []string{"tag1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	tags.disable("tag1")

	got, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{
		TagIDs: ptr([]string{"tag1"}),
	})
	if err != nil {
		t.Fatalf("Update resubmitting the entry's own since-disabled tag: %v", err)
	}
	if len(got.TagIDs) != 1 || got.TagIDs[0] != "tag1" {
		t.Fatalf("got.TagIDs = %v", got.TagIDs)
	}
}

func TestUpdateAddingDisabledTagAlongsideUntouchedExistingOneRejected(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	tags.add("tag1", "u1")
	tags.add("tag2", "u1")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
		TagIDs: []string{"tag1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	tags.disable("tag1")
	tags.disable("tag2")

	_, err = svc.Update(context.Background(), "u1", e.ID, entry.Update{
		TagIDs: ptr([]string{"tag1", "tag2"}),
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}

	got, err := svc.Get(context.Background(), "u1", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.TagIDs) != 1 || got.TagIDs[0] != "tag1" {
		t.Fatalf("rejected update must not change tags: got.TagIDs = %v", got.TagIDs)
	}
}

func TestUpdateClearingCategoryOnTransactionRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
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

// --- moving an entry between accounts ---------------------------------

func TestUpdateMovesEntryToAnotherOwnedAccount(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "EUR")
	categories.add("cat1")
	e := mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-01-01T00:00:00Z", ptr("cat1"))

	got, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{AccountID: ptr("acc2")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.AccountID != "acc2" {
		t.Fatalf("AccountID = %q, want acc2", got.AccountID)
	}
}

func TestUpdateMoveToOtherOwnersAccountRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u2", "EUR")
	categories.add("cat1")
	e := mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-01-01T00:00:00Z", ptr("cat1"))

	_, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{AccountID: ptr("acc2")})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
	got, err := svc.Get(context.Background(), "u1", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountID != "acc1" {
		t.Fatalf("AccountID = %q, want unchanged acc1", got.AccountID)
	}
}

func TestUpdateMoveToDisabledAccountRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "EUR")
	accounts.disabled["acc2"] = true
	categories.add("cat1")
	e := mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-01-01T00:00:00Z", ptr("cat1"))

	_, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{AccountID: ptr("acc2")})
	if !errors.Is(err, entry.ErrAccountDisabled) {
		t.Fatalf("err = %v, want ErrAccountDisabled", err)
	}
	got, err := svc.Get(context.Background(), "u1", e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccountID != "acc1" {
		t.Fatalf("AccountID = %q, want unchanged acc1", got.AccountID)
	}
}

func TestUpdateMovesBalanceAdjustmentLikeTransaction(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "EUR")
	e := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 10000, "2024-01-01T00:00:00Z", nil)

	got, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{AccountID: ptr("acc2")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.AccountID != "acc2" {
		t.Fatalf("AccountID = %q, want acc2", got.AccountID)
	}
}

func TestUpdateMoveBetweenDifferentCurrenciesLeavesAmountUnchanged(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "USD")
	categories.add("cat1")
	e := mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-01-01T00:00:00Z", ptr("cat1"))

	got, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{AccountID: ptr("acc2")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.AccountID != "acc2" {
		t.Fatalf("AccountID = %q, want acc2", got.AccountID)
	}
	if got.Amount != 100 {
		t.Fatalf("Amount = %d, want unchanged 100", got.Amount)
	}
}

func TestSoftDeleteExcludesFromGetAndListing(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	e, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(100)),
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

// --- balance adjustment delta recompute ---------------------------------
//
// See design.md's worked example: only the earliest non-deleted adjustment
// at or after a mutation's position, and the one immediately after it, can
// ever need their amount recomputed.

func TestCreateBalanceAdjustmentWithNoHistoryComputesDeltaEqualToReading(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	e := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 10000, "2024-01-01T00:00:00Z", nil)
	if e.Amount != 10000 {
		t.Fatalf("Amount = %d, want 10000 (delta from a 0 baseline)", e.Amount)
	}
	if e.Balance == nil || *e.Balance != 10000 {
		t.Fatalf("Balance = %v, want 10000", e.Balance)
	}
}

func TestTransactionInsertedBeforeAdjustmentShiftsItsDelta(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	adj := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 20000, "2024-02-01T00:00:00Z", nil)
	if adj.Amount != 20000 {
		t.Fatalf("initial Amount = %d, want 20000", adj.Amount)
	}

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 5000, "2024-01-01T00:00:00Z", ptr("cat1"))

	got, err := svc.Get(context.Background(), "u1", adj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount != 15000 {
		t.Fatalf("adjustment Amount after inserting an earlier transaction = %d, want 15000", got.Amount)
	}
	bal, err := svc.Balance(context.Background(), "u1", "acc1", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if bal != 20000 {
		t.Fatalf("balance = %d, want 20000 (the adjustment's reading, unaffected)", bal)
	}
}

func TestEditingTransactionBetweenTwoAdjustmentsOnlyRecomputesTheFollowingOne(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	adjA := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 10000, "2024-01-01T00:00:00Z", nil)
	txn := mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -2000, "2024-01-02T00:00:00Z", ptr("cat1"))
	adjB := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 20000, "2024-01-03T00:00:00Z", nil)

	if got, err := svc.Get(context.Background(), "u1", adjB.ID); err != nil || got.Amount != 12000 {
		t.Fatalf("AdjB.Amount = %d, err = %v, want 12000 (20000 - (10000-2000))", got.Amount, err)
	}

	newAmount := int64(-5000)
	if _, err := svc.Update(context.Background(), "u1", txn.ID, entry.Update{Amount: &newAmount}); err != nil {
		t.Fatal(err)
	}

	gotA, err := svc.Get(context.Background(), "u1", adjA.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotA.Amount != 10000 {
		t.Fatalf("AdjA.Amount = %d, want unchanged 10000 (nothing between it and its own predecessor changed)", gotA.Amount)
	}
	gotB, err := svc.Get(context.Background(), "u1", adjB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotB.Amount != 15000 {
		t.Fatalf("AdjB.Amount after editing the transaction between = %d, want 15000 (20000 - (10000-5000))", gotB.Amount)
	}
}

func TestDeletingAnAdjustmentShiftsTheNextOnesBaseline(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 10000, "2024-01-01T00:00:00Z", nil)
	adjB := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 15000, "2024-01-02T00:00:00Z", nil)
	adjC := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 20000, "2024-01-03T00:00:00Z", nil)

	if got, err := svc.Get(context.Background(), "u1", adjC.ID); err != nil || got.Amount != 5000 {
		t.Fatalf("AdjC.Amount before delete = %d, err = %v, want 5000", got.Amount, err)
	}

	if err := svc.Delete(context.Background(), "u1", adjB.ID); err != nil {
		t.Fatal(err)
	}

	gotC, err := svc.Get(context.Background(), "u1", adjC.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotC.Amount != 10000 {
		t.Fatalf("AdjC.Amount after deleting AdjB = %d, want 10000 (20000-10000, now against AdjA)", gotC.Amount)
	}
}

func TestMovingAnAdjustmentPastAnotherRecomputesBothNeighbors(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 10000, "2024-01-01T00:00:00Z", nil)
	adjB := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 15000, "2024-01-05T00:00:00Z", nil)
	adjC := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 30000, "2024-01-10T00:00:00Z", nil)

	// Move AdjB (reading unchanged) to after AdjC.
	newTS := at("2024-01-15T00:00:00Z")
	if _, err := svc.Update(context.Background(), "u1", adjB.ID, entry.Update{BookingTimestamp: &newTS}); err != nil {
		t.Fatal(err)
	}

	gotC, err := svc.Get(context.Background(), "u1", adjC.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotC.Amount != 20000 {
		t.Fatalf("AdjC.Amount after AdjB moved past it = %d, want 20000 (30000-10000, now against AdjA)", gotC.Amount)
	}
	gotB, err := svc.Get(context.Background(), "u1", adjB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotB.Amount != -15000 {
		t.Fatalf("AdjB.Amount after moving past AdjC = %d, want -15000 (15000-30000, now against AdjC)", gotB.Amount)
	}
}

func TestEditingAnAdjustmentsBalanceRecomputesItsOwnDelta(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	adj := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 10000, "2024-01-01T00:00:00Z", nil)

	newBalance := int64(25000)
	got, err := svc.Update(context.Background(), "u1", adj.ID, entry.Update{Balance: &newBalance})
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount != 25000 {
		t.Fatalf("Amount after editing Balance with no predecessor = %d, want 25000", got.Amount)
	}
	if got.Balance == nil || *got.Balance != 25000 {
		t.Fatalf("Balance = %v, want 25000", got.Balance)
	}
}

// --- amount/balance kind contract ---------------------------------------

func TestCreateTransactionWithBalanceRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	bal := int64(100)
	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Balance: &bal,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X", CategoryID: ptr("cat1"),
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateBalanceAdjustmentWithAmountRejected(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	amt := int64(100)
	_, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindBalanceAdjustment, Amount: &amt,
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "X",
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateSettingBalanceOnTransactionRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	e := mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-01-01T00:00:00Z", ptr("cat1"))

	bal := int64(500)
	_, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{Balance: &bal})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateSettingAmountOnBalanceAdjustmentRejected(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	e := mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 10000, "2024-01-01T00:00:00Z", nil)

	amt := int64(500)
	_, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{Amount: &amt})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

// --- summing -----------------------------------------------------------

func TestSumGroupsByCurrencyAndExcludesBalanceAdjustments(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "USD")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -100, "2024-01-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -50, "2024-01-02T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc2", entry.KindTransaction, -20, "2024-01-03T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 99999, "2024-01-01T00:00:00Z", nil)

	summary, err := svc.Sum(context.Background(), "u1", entry.Filter{CategoryID: ptr("cat1")})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Count != 3 {
		t.Fatalf("Count = %d, want 3 (balance adjustment excluded)", summary.Count)
	}
	byCurrency := map[string]int64{}
	for _, s := range summary.Sums {
		byCurrency[s.Currency] = s.Amount
	}
	if len(byCurrency) != 2 || byCurrency["EUR"] != -150 || byCurrency["USD"] != -20 {
		t.Fatalf("Sums = %+v, want EUR -150 and USD -20", summary.Sums)
	}
}

func TestSumWithNoMatchesReturnsEmpty(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	summary, err := svc.Sum(context.Background(), "u1", entry.Filter{CategoryID: ptr("cat1")})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Count != 0 || len(summary.Sums) != 0 {
		t.Fatalf("summary = %+v, want empty", summary)
	}
}

func TestSumIgnoresKindFilterAlwaysExcludingBalanceAdjustments(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -10, "2024-01-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 500, "2024-01-01T00:00:00Z", nil)

	bal := entry.KindBalanceAdjustment
	summary, err := svc.Sum(context.Background(), "u1", entry.Filter{Kind: &bal})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Count != 1 || len(summary.Sums) != 1 || summary.Sums[0].Amount != -10 {
		t.Fatalf("summary = %+v, want the single transaction only, "+
			"even though Kind asked for balance_adjustment", summary)
	}
}

func TestSumExactCategoryModeExcludesDescendants(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("parent")
	categories.add("child")
	categories.children["parent"] = []string{"child"}

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -10, "2024-01-01T00:00:00Z", ptr("parent"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -5, "2024-01-02T00:00:00Z", ptr("child"))

	summary, err := svc.Sum(context.Background(), "u1", entry.Filter{
		CategoryID: ptr("parent"), CategoryMode: entry.ModeExact,
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Count != 1 || len(summary.Sums) != 1 || summary.Sums[0].Amount != -10 {
		t.Fatalf("summary = %+v, want only the parent-category entry", summary)
	}
}

// mustCreate creates an entry of kind with amount as its Amount (for
// KindTransaction) or Balance reading (for KindBalanceAdjustment) — a
// single numeric parameter covers both, since callers already pick the
// right one via kind.
func mustCreate(t *testing.T, svc *entry.Service, owner, accountID string, kind entry.Kind, amount int64, ts string, categoryID *string) entry.Entry {
	t.Helper()
	in := entry.New{
		AccountID: accountID, Kind: kind,
		BookingTimestamp: at(ts), Title: "x", CategoryID: categoryID,
	}
	if kind == entry.KindTransaction {
		in.Amount = &amount
	} else {
		in.Balance = &amount
	}
	e, err := svc.Create(context.Background(), owner, in)
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

func TestListFiltersByCategoryExactExcludesDescendants(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("parent")
	categories.add("child")
	categories.children["parent"] = []string{"child"}

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 1, "2024-01-01T00:00:00Z", ptr("child"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 2, "2024-01-02T00:00:00Z", ptr("parent"))

	items, _, err := svc.List(context.Background(), "u1", entry.Filter{
		CategoryID: ptr("parent"), CategoryMode: entry.ModeExact,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].CategoryID == nil || *items[0].CategoryID != "parent" {
		t.Fatalf("items = %+v, want just the parent-category entry", items)
	}
}

func TestListInvalidCategoryModeRejected(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	_, _, err := svc.List(context.Background(), "u1", entry.Filter{
		CategoryID: ptr("cat1"), CategoryMode: entry.CategoryMode("bogus"),
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestListSearchMatchesTitleOrDescription(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")
	if _, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(1)),
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

// --- flow summary --------------------------------------------------------

func TestFlowSummaryMonthlyBucketsForAYear(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 1000, "2024-01-15T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -300, "2024-01-20T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 500, "2024-03-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -200, "2025-01-01T00:00:00Z", ptr("cat1")) // different year, excluded

	buckets, err := svc.FlowSummary(context.Background(), "u1", entry.FlowFilter{Unit: entry.FlowUnitMonth, Year: 2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 12 {
		t.Fatalf("len(buckets) = %d, want 12", len(buckets))
	}
	if buckets[0].Period != "2024-01-01" {
		t.Fatalf("buckets[0].Period = %q, want 2024-01-01", buckets[0].Period)
	}
	if len(buckets[0].Income) != 1 || buckets[0].Income[0] != (entry.CurrencySum{Currency: "EUR", Amount: 1000}) {
		t.Fatalf("January income = %+v, want [{EUR 1000}]", buckets[0].Income)
	}
	if len(buckets[0].Outcome) != 1 || buckets[0].Outcome[0] != (entry.CurrencySum{Currency: "EUR", Amount: 300}) {
		t.Fatalf("January outcome = %+v, want [{EUR 300}]", buckets[0].Outcome)
	}
	if buckets[2].Period != "2024-03-01" || len(buckets[2].Income) != 1 || buckets[2].Income[0].Amount != 500 {
		t.Fatalf("March bucket = %+v", buckets[2])
	}
	for i, b := range buckets {
		if i == 0 || i == 2 {
			continue
		}
		if len(b.Income) != 0 || len(b.Outcome) != 0 {
			t.Fatalf("bucket %d (%s) = %+v, want empty (no special-casing an empty month)", i, b.Period, b)
		}
	}
}

func TestFlowSummaryDailyBucketsForAMonth(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-02-05T00:00:00Z", ptr("cat1"))

	buckets, err := svc.FlowSummary(context.Background(), "u1", entry.FlowFilter{Unit: entry.FlowUnitDay, Year: 2024, Month: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets) != 29 { // 2024 is a leap year
		t.Fatalf("len(buckets) = %d, want 29 (Feb 2024, leap year)", len(buckets))
	}
	if buckets[4].Period != "2024-02-05" || len(buckets[4].Income) != 1 || buckets[4].Income[0].Amount != 100 {
		t.Fatalf("Feb 5 bucket = %+v", buckets[4])
	}
}

func TestFlowSummaryIncludesBalanceAdjustmentDelta(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, -500, "2024-01-01T00:00:00Z", nil)

	buckets, err := svc.FlowSummary(context.Background(), "u1", entry.FlowFilter{Unit: entry.FlowUnitMonth, Year: 2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets[0].Outcome) != 1 || buckets[0].Outcome[0].Amount != 500 {
		t.Fatalf("January outcome = %+v, want [{EUR 500}] (the adjustment's own negative delta)", buckets[0].Outcome)
	}
	if len(buckets[0].Income) != 0 {
		t.Fatalf("January income = %+v, want empty", buckets[0].Income)
	}
}

func TestFlowSummaryGroupsByCurrencyAcrossAccounts(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "USD")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-01-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc2", entry.KindTransaction, 50, "2024-01-02T00:00:00Z", ptr("cat1"))

	buckets, err := svc.FlowSummary(context.Background(), "u1", entry.FlowFilter{Unit: entry.FlowUnitMonth, Year: 2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets[0].Income) != 2 {
		t.Fatalf("January income = %+v, want 2 currencies", buckets[0].Income)
	}
	byCurrency := map[string]int64{}
	for _, s := range buckets[0].Income {
		byCurrency[s.Currency] = s.Amount
	}
	if byCurrency["EUR"] != 100 || byCurrency["USD"] != 50 {
		t.Fatalf("byCurrency = %v, want EUR:100 USD:50", byCurrency)
	}
}

func TestFlowSummaryDayUnitRequiresMonth(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	_, err := svc.FlowSummary(context.Background(), "u1", entry.FlowFilter{Unit: entry.FlowUnitDay, Year: 2024})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestFlowSummaryMonthUnitRejectsMonth(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	_, err := svc.FlowSummary(context.Background(), "u1", entry.FlowFilter{Unit: entry.FlowUnitMonth, Year: 2024, Month: 3})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestFlowSummaryInvalidUnitRejected(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	_, err := svc.FlowSummary(context.Background(), "u1", entry.FlowFilter{Unit: entry.FlowUnit("bogus"), Year: 2024})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestFlowSummaryUsesCallerTimezoneForBucketBoundaries(t *testing.T) {
	accounts := newStubAccounts()
	categories := newStubCategories()
	tags := newStubTags()
	tz := newStubTimezones()
	tz.tz["u1"] = "America/New_York"
	svc := entry.NewService(memory.NewEntryStore(), accounts, categories, tags, entry.WithTimezoneLookup(tz))
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	// 2024-02-01T02:00:00Z is 2024-01-31T21:00:00-05:00 in America/New_York
	// — January there, not February.
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2024-02-01T02:00:00Z", ptr("cat1"))

	buckets, err := svc.FlowSummary(context.Background(), "u1", entry.FlowFilter{Unit: entry.FlowUnitMonth, Year: 2024})
	if err != nil {
		t.Fatal(err)
	}
	if len(buckets[0].Income) != 1 || buckets[0].Income[0].Amount != 100 {
		t.Fatalf("January (America/New_York) bucket = %+v, want the entry counted there", buckets[0].Income)
	}
	if len(buckets[1].Income) != 0 {
		t.Fatalf("February (America/New_York) bucket = %+v, want empty — the entry belongs to January there", buckets[1].Income)
	}
}

// balanceAt returns the amount for a currency at one BalancePoint, or a
// fatal error if that currency is not listed on the point.
func balanceAt(t *testing.T, p entry.BalancePoint, currency string) int64 {
	t.Helper()
	for _, b := range p.Balances {
		if b.Currency == currency {
			return b.Amount
		}
	}
	t.Fatalf("point %s has no %s balance: %+v", p.Period, currency, p.Balances)
	return 0
}

func TestBalanceSeriesHasOnePointPerDayPlusAClosingPoint(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	points, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitDay, Year: 2026, Month: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 32 {
		t.Fatalf("len(points) = %d, want 32 (31 days of March + a closing point)", len(points))
	}
	if points[0].Period != "2026-03-01" {
		t.Fatalf("points[0].Period = %q, want 2026-03-01", points[0].Period)
	}
	if points[30].Period != "2026-03-31" {
		t.Fatalf("points[30].Period = %q, want 2026-03-31", points[30].Period)
	}
	if points[31].Period != "2026-04-01" {
		t.Fatalf("points[31].Period = %q, want 2026-04-01 (closing point)", points[31].Period)
	}
}

func TestBalanceSeriesFirstPointIsAllPriorHistory(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 12000, "2026-01-10T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -2000, "2026-02-20T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 500, "2026-03-05T12:00:00Z", ptr("cat1"))

	points, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitDay, Year: 2026, Month: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 12000 - 2000, and nothing booked on or after 2026-03-01T00:00:00.
	if got := balanceAt(t, points[0], "EUR"); got != 10000 {
		t.Fatalf("points[0] EUR = %d, want 10000 (prior history only)", got)
	}
}

func TestBalanceSeriesTransactionMovesTheFollowingDay(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100000, "2026-02-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -2500, "2026-03-10T12:00:00Z", ptr("cat1"))

	points, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitDay, Year: 2026, Month: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := balanceAt(t, points[9], "EUR"); got != 100000 { // 2026-03-10
		t.Fatalf("points[9] (10 Mar) EUR = %d, want 100000", got)
	}
	if got := balanceAt(t, points[10], "EUR"); got != 97500 { // 2026-03-11
		t.Fatalf("points[10] (11 Mar) EUR = %d, want 97500", got)
	}
}

func TestBalanceSeriesMidMonthAdjustmentReanchorsEveryLaterPoint(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 48000, "2026-02-15T00:00:00Z", ptr("cat1"))
	// Reading 50000 against a pre-adjustment balance of 48000 — a +2000 delta.
	mustCreate(t, svc, "u1", "acc1", entry.KindBalanceAdjustment, 50000, "2026-03-15T10:00:00Z", nil)
	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 1000, "2026-03-20T09:00:00Z", ptr("cat1"))

	points, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitDay, Year: 2026, Month: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := balanceAt(t, points[14], "EUR"); got != 48000 { // 15 Mar, before the adjustment's day
		t.Fatalf("points[14] (15 Mar) EUR = %d, want 48000", got)
	}
	if got := balanceAt(t, points[15], "EUR"); got != 50000 { // 16 Mar, adjustment now in effect
		t.Fatalf("points[15] (16 Mar) EUR = %d, want 50000 (re-anchored)", got)
	}
	if got := balanceAt(t, points[20], "EUR"); got != 51000 { // 21 Mar, + the 1000 txn
		t.Fatalf("points[20] (21 Mar) EUR = %d, want 51000", got)
	}
	if got := balanceAt(t, points[31], "EUR"); got != 51000 { // closing point
		t.Fatalf("closing point EUR = %d, want 51000", got)
	}
}

func TestBalanceSeriesUsesCallerTimezoneForDayBoundaries(t *testing.T) {
	build := func(tz string) []entry.BalancePoint {
		accounts := newStubAccounts()
		categories := newStubCategories()
		tags := newStubTags()
		tzs := newStubTimezones()
		tzs.tz["u1"] = tz
		svc := entry.NewService(memory.NewEntryStore(), accounts, categories, tags, entry.WithTimezoneLookup(tzs))
		accounts.add("acc1", "u1", "EUR")
		categories.add("cat1")
		// 23:30Z on 9 March is 00:30 on 10 March in Europe/Vienna (UTC+1).
		mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 500, "2026-03-09T23:30:00Z", ptr("cat1"))
		points, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
			Unit: entry.FlowUnitDay, Year: 2026, Month: 3,
		})
		if err != nil {
			t.Fatal(err)
		}
		return points
	}

	utc := build("UTC")
	if got := balanceAt(t, utc[9], "EUR"); got != 500 { // 10 Mar: the entry is on 9 Mar UTC
		t.Fatalf("UTC points[9] (10 Mar) EUR = %d, want 500", got)
	}

	vienna := build("Europe/Vienna")
	if got := balanceAt(t, vienna[9], "EUR"); got != 0 { // 10 Mar: the entry is 00:30 on 10 Mar local
		t.Fatalf("Vienna points[9] (10 Mar) EUR = %d, want 0", got)
	}
	if got := balanceAt(t, vienna[10], "EUR"); got != 500 { // 11 Mar: now in effect
		t.Fatalf("Vienna points[10] (11 Mar) EUR = %d, want 500", got)
	}
}

func TestBalanceSeriesGroupsByCurrencyAndAlwaysListsEveryCurrency(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "USD")
	accounts.add("acc3", "u1", "EUR")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2026-02-01T00:00:00Z", ptr("cat1"))
	mustCreate(t, svc, "u1", "acc3", entry.KindTransaction, 40, "2026-02-01T00:00:00Z", ptr("cat1"))
	// acc2 (USD) has no entries at all.

	points, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitDay, Year: 2026, Month: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range points {
		if len(p.Balances) != 2 {
			t.Fatalf("point %s has %d currencies, want 2 (EUR always listed, USD listed with 0)", p.Period, len(p.Balances))
		}
		if got := balanceAt(t, p, "EUR"); got != 140 { // acc1 + acc3
			t.Fatalf("point %s EUR = %d, want 140", p.Period, got)
		}
		if got := balanceAt(t, p, "USD"); got != 0 {
			t.Fatalf("point %s USD = %d, want 0 (present even though acc2 is empty)", p.Period, got)
		}
	}
}

func TestBalanceSeriesNoActivityIsAFlatLine(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 777, "2026-01-01T00:00:00Z", ptr("cat1"))

	points, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitDay, Year: 2026, Month: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range points {
		if got := balanceAt(t, p, "EUR"); got != 777 {
			t.Fatalf("point %s EUR = %d, want 777 (flat — no activity in or after March)", p.Period, got)
		}
	}
}

func TestBalanceSeriesIgnoresSoftDeletedEntries(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat1")

	mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, 100, "2026-02-01T00:00:00Z", ptr("cat1"))
	doomed := mustCreate(t, svc, "u1", "acc1", entry.KindTransaction, -30, "2026-03-05T12:00:00Z", ptr("cat1"))
	if err := svc.Delete(context.Background(), "u1", doomed.ID); err != nil {
		t.Fatal(err)
	}

	points, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitDay, Year: 2026, Month: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := balanceAt(t, points[31], "EUR"); got != 100 {
		t.Fatalf("closing point EUR = %d, want 100 (the -30 entry was deleted)", got)
	}
}

func TestBalanceSeriesRejectsNonDayUnit(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	_, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitMonth, Year: 2026, Month: 3,
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestBalanceSeriesRequiresMonth(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("acc1", "u1", "EUR")

	_, err := svc.BalanceSeries(context.Background(), "u1", entry.BalanceFilter{
		Unit: entry.FlowUnitDay, Year: 2026,
	})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}
