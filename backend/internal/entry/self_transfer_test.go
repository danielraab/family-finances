package entry_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/entry"
)

func newSelfTransfer(accountID, toAccountID string, amount int64) entry.New {
	return entry.New{
		AccountID: accountID, ToAccountID: &toAccountID, Kind: entry.KindSelfTransfer,
		Amount: ptr(amount), BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Move to savings",
	}
}

func TestSelfTransferCreateSucceedsWithAppendOnBothSameCurrency(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")

	e, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.Kind != entry.KindSelfTransfer || e.ToAccountID == nil || *e.ToAccountID != "b" {
		t.Fatalf("e = %+v", e)
	}

	balA, err := svc.Balance(context.Background(), "u1", "a", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balA != -1000 {
		t.Fatalf("balance A = %d, want -1000", balA)
	}
	balB, err := svc.Balance(context.Background(), "u1", "b", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balB != 1000 {
		t.Fatalf("balance B = %d, want 1000", balB)
	}
}

func TestSelfTransferCreateRejectedWithoutPermissionOnToAccount(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u2", "EUR") // u1 has no access at all to b

	_, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestSelfTransferCreateRejectedWithOnlyViewOnToAccount(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u2", "EUR")
	accounts.share("b", "u1", "view")

	_, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestSelfTransferCreateRejectedOnCurrencyMismatch(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "USD")

	_, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestSelfTransferCreateRejectedForSameAccount(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")

	_, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "a", -1000))
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestSelfTransferCreateRejectedAgainstDisabledToAccount(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	accounts.disabled["b"] = true

	_, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if !errors.Is(err, entry.ErrAccountDisabled) {
		t.Fatalf("err = %v, want ErrAccountDisabled", err)
	}
}

func TestSelfTransferCreateAllowsEmptyCategory(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")

	e, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.CategoryID != nil {
		t.Fatalf("CategoryID = %v, want nil", e.CategoryID)
	}
}

func TestSelfTransferCreateRejectsCounterparty(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")

	in := newSelfTransfer("a", "b", -1000)
	in.Counterparty = "Someone"
	_, err := svc.Create(context.Background(), "u1", in)
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestSelfTransferVisibleViaEitherAccount(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u2", "EUR")
	accounts.share("a", "u2", "view")
	accounts.share("b", "u1", "append")

	e, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// u2 has no permission on "a" at all, but view+ on "b" — still visible.
	if _, err := svc.Get(context.Background(), "u2", e.ID); err != nil {
		t.Fatalf("Get by u2 (view on b): %v", err)
	}
}

func TestSelfTransferListedOncePerAccountInScope(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	if _, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000)); err != nil {
		t.Fatal(err)
	}

	// Only "a" in scope: listed once, amount as stored.
	items, _, err := svc.List(context.Background(), "u1", entry.Filter{AccountIDs: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Amount != -1000 || items[0].AccountID != "a" {
		t.Fatalf("items (a only) = %+v", items)
	}

	// Only "b" in scope: listed once, amount flipped, account_id swapped.
	items, _, err = svc.List(context.Background(), "u1", entry.Filter{AccountIDs: []string{"b"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Amount != 1000 || items[0].AccountID != "b" || items[0].ToAccountID == nil || *items[0].ToAccountID != "a" {
		t.Fatalf("items (b only) = %+v", items)
	}

	// Both in scope (unfiltered, since u1 owns both): listed twice.
	items, _, err = svc.List(context.Background(), "u1", entry.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items (both) = %+v, want 2", items)
	}
}

func TestSelfTransferIncludedInSumAndNetsToZero(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	if _, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000)); err != nil {
		t.Fatal(err)
	}

	sum, err := svc.Sum(context.Background(), "u1", entry.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sum.Sums) != 1 || sum.Sums[0].Currency != "EUR" || sum.Sums[0].Amount != 0 {
		t.Fatalf("Sum = %+v, want a single EUR entry netting to 0", sum)
	}
	if sum.Count != 2 {
		t.Fatalf("Count = %d, want 2 (once per account)", sum.Count)
	}
}

func TestSelfTransferUpdateRequiresAppendOnBothAccounts(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u2", "EUR")
	accounts.share("b", "u1", "append")
	e, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if err != nil {
		t.Fatal(err)
	}

	// u1 holds append+ on both (owner of a, shared append on b): succeeds.
	if _, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{Title: ptr("Renamed")}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestSelfTransferUpdateRejectedAfterLosingAccessToOneSide(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u2", "EUR")
	accounts.share("b", "u1", "append")
	e, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if err != nil {
		t.Fatal(err)
	}

	// Revoke u1's share on "b" — even a change unrelated to amount/accounts
	// is now rejected, since u1 no longer holds append+ on both sides.
	delete(accounts.shares["b"], "u1")
	if _, err := svc.Update(context.Background(), "u1", e.ID, entry.Update{Title: ptr("Renamed")}); !errors.Is(err, entry.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestSelfTransferUpdateRejectsAccountIDChange(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	accounts.add("c", "u1", "EUR")
	e, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.Update(context.Background(), "u1", e.ID, entry.Update{AccountID: ptr("c")})
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestSelfTransferDeleteSucceedsViaEitherAccessibleSide(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u2", "EUR")
	accounts.share("b", "u1", "append")
	e, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if err != nil {
		t.Fatal(err)
	}

	// Revoke u1's access to "b" entirely — u1 still owns "a", so delete
	// (unlike edit) still succeeds via that side alone.
	delete(accounts.shares["b"], "u1")
	if err := svc.Delete(context.Background(), "u1", e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), "u1", e.ID); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}
}

func TestSelfTransferRecomputesBalanceAdjustmentOnReceivingAccount(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")

	adj, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "b", Kind: entry.KindBalanceAdjustment, Balance: ptr(int64(5000)),
		BookingTimestamp: at("2024-02-01T00:00:00Z"), Title: "b opening",
	})
	if err != nil {
		t.Fatal(err)
	}

	in := newSelfTransfer("a", "b", -1000)
	in.BookingTimestamp = at("2024-01-15T00:00:00Z")
	if _, err := svc.Create(context.Background(), "u1", in); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(context.Background(), "u1", adj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Amount != 4000 {
		t.Fatalf("adj.Amount = %d, want 4000 (5000 - 1000 received before it)", got.Amount)
	}
}

func TestSelfTransferDeleteStillRespectsAppendCreatedByRule(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	accounts.share("a", "u2", "append")
	accounts.share("b", "u2", "append")
	e, err := svc.Create(context.Background(), "u1", newSelfTransfer("a", "b", -1000))
	if err != nil {
		t.Fatal(err)
	}

	// u2 holds only append (not entry_admin/owner) on either account, and
	// did not create the entry.
	if err := svc.Delete(context.Background(), "u2", e.ID); !errors.Is(err, entry.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}
