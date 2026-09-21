package postgres

import (
	"context"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/entry"
)

// newSelfTransferFixture extends newEntryFixture with a second account of
// the same owner/currency, for exercising a self_transfer's two-account
// behavior against a real Postgres database (the entry_legs view, the
// dual-account Balance CASE, and recompute-on-both-sides).
func newSelfTransferFixture(t *testing.T) (entryFixture, string) {
	t.Helper()
	f := newEntryFixture(t)
	opening, _ := account.ParseDate("2024-01-01")
	acc2, err := f.accounts.Create(context.Background(), f.owner, account.New{
		Title: "Savings", Type: "Savings", Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	return f, acc2.ID
}

func TestPGSelfTransferBalancesBothAccounts(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()

	e, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, ToAccountID: &acc2, Kind: entry.KindSelfTransfer,
		Amount: ptrInt64(-1000), BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "To savings",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if e.ToAccountID == nil || *e.ToAccountID != acc2 {
		t.Fatalf("ToAccountID = %v, want %s", e.ToAccountID, acc2)
	}

	asOf := at("2024-06-01T00:00:00Z")
	bal1, err := f.entries.Balance(ctx, f.accID, asOf)
	if err != nil {
		t.Fatal(err)
	}
	if bal1 != -1000 {
		t.Fatalf("account 1 balance = %d, want -1000", bal1)
	}
	bal2, err := f.entries.Balance(ctx, acc2, asOf)
	if err != nil {
		t.Fatal(err)
	}
	if bal2 != 1000 {
		t.Fatalf("account 2 balance = %d, want 1000", bal2)
	}
}

func TestPGSelfTransferRecomputesBalanceAdjustmentsOnBothSides(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()

	adj1, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(10000),
		BookingTimestamp: at("2024-02-01T00:00:00Z"), Title: "acc1 opening",
	})
	if err != nil {
		t.Fatal(err)
	}
	adj2, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: acc2, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(5000),
		BookingTimestamp: at("2024-02-01T00:00:00Z"), Title: "acc2 opening",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Self-transfer lands between the two accounts' opening adjustments and
	// "now" — each adjustment's own stored amount never needs to move
	// (nothing precedes them), but a transfer placed *before* an adjustment
	// must shift it. Use a timestamp before both adjustments instead.
	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, ToAccountID: &acc2, Kind: entry.KindSelfTransfer,
		Amount: ptrInt64(-1000), BookingTimestamp: at("2024-01-15T00:00:00Z"), Title: "Early transfer",
	}); err != nil {
		t.Fatal(err)
	}

	gotAdj1, err := f.entries.Get(ctx, adj1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAdj1.Amount != 11000 {
		t.Fatalf("adj1.Amount = %d, want 11000 (10000 - (-1000))", gotAdj1.Amount)
	}
	gotAdj2, err := f.entries.Get(ctx, adj2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAdj2.Amount != 4000 {
		t.Fatalf("adj2.Amount = %d, want 4000 (5000 - 1000)", gotAdj2.Amount)
	}
}

func TestPGSelfTransferListedOncePerAccountInScope(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, ToAccountID: &acc2, Kind: entry.KindSelfTransfer,
		Amount: ptrInt64(-1000), BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "To savings",
	}); err != nil {
		t.Fatal(err)
	}

	items, _, err := f.entries.List(ctx, entry.Filter{AccountIDs: []string{f.accID}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Amount != -1000 || items[0].AccountID != f.accID {
		t.Fatalf("items (account 1 only) = %+v", items)
	}

	items, _, err = f.entries.List(ctx, entry.Filter{AccountIDs: []string{acc2}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Amount != 1000 || items[0].AccountID != acc2 {
		t.Fatalf("items (account 2 only) = %+v", items)
	}

	items, _, err = f.entries.List(ctx, entry.Filter{AccountIDs: []string{f.accID, acc2}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items (both accounts) = %+v, want 2", items)
	}
}

func TestPGSelfTransferSumNetsToZeroAcrossBothAccounts(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, ToAccountID: &acc2, Kind: entry.KindSelfTransfer,
		Amount: ptrInt64(-1000), BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "To savings",
	}); err != nil {
		t.Fatal(err)
	}

	perAccount, count, err := f.entries.Sum(ctx, entry.Filter{AccountIDs: []string{f.accID, acc2}})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
	if perAccount[f.accID].Amount != -1000 || perAccount[acc2].Amount != 1000 {
		t.Fatalf("perAccount = %v", perAccount)
	}
	// Each leg still contributes to its own side's income/outcome even
	// though the net (above) cancels out across the two accounts.
	if perAccount[f.accID].Outcome != 1000 || perAccount[f.accID].Income != 0 {
		t.Fatalf("perAccount[accID] income/outcome = %+v, want income=0 outcome=1000", perAccount[f.accID])
	}
	if perAccount[acc2].Income != 1000 || perAccount[acc2].Outcome != 0 {
		t.Fatalf("perAccount[acc2] income/outcome = %+v, want income=1000 outcome=0", perAccount[acc2])
	}
}

func TestPGHasEntriesForAccount(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()

	has, err := f.entries.HasEntriesForAccount(ctx, acc2)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatalf("acc2 should have no entries yet")
	}

	if _, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, ToAccountID: &acc2, Kind: entry.KindSelfTransfer,
		Amount: ptrInt64(-1000), BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "To savings",
	}); err != nil {
		t.Fatal(err)
	}

	// acc2 only ever appears as to_account_id, never account_id — must
	// still report true.
	has, err = f.entries.HasEntriesForAccount(ctx, acc2)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatalf("acc2 should now have an entry (as to_account_id)")
	}
}

func TestPGSelfTransferSoftDeleteRecomputesBothAccounts(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()

	adj, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: acc2, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(5000),
		BookingTimestamp: at("2024-02-01T00:00:00Z"), Title: "acc2 opening",
	})
	if err != nil {
		t.Fatal(err)
	}
	transfer, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, ToAccountID: &acc2, Kind: entry.KindSelfTransfer,
		Amount: ptrInt64(-1000), BookingTimestamp: at("2024-01-15T00:00:00Z"), Title: "Early transfer",
	})
	if err != nil {
		t.Fatal(err)
	}
	gotAdj, err := f.entries.Get(ctx, adj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAdj.Amount != 4000 {
		t.Fatalf("adj.Amount before delete = %d, want 4000", gotAdj.Amount)
	}

	if err := f.entries.SoftDelete(ctx, transfer.ID); err != nil {
		t.Fatal(err)
	}
	gotAdj, err = f.entries.Get(ctx, adj.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotAdj.Amount != 5000 {
		t.Fatalf("adj.Amount after delete = %d, want 5000 (transfer's contribution removed)", gotAdj.Amount)
	}
}
