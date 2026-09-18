package postgres

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/entry"
	rt "at.draab/familyfinances/internal/recurringtransaction"
)

func TestPGConvertToSelfTransferSenderKeepsRecurringLinkAndAmount(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()
	recurring := NewRecurringTransactionStore(f.pool)

	starts, _ := account.ParseDate("2024-01-01")
	created, err := recurring.Create(ctx, f.owner, rt.New{
		AccountID: f.accID, Title: "Netflix", CategoryID: &f.catID,
		Amount: -1500, IntervalUnit: rt.UnitMonth, IntervalCount: 1,
		StartsOn: rt.NewDate(starts.Time),
	})
	if err != nil {
		t.Fatal(err)
	}

	original, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-1000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Groceries", CategoryID: &f.catID,
		Counterparty: "Rewe", Location: "Berlin", RecurringTransactionID: &created.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	converted, err := f.entries.ConvertToSelfTransfer(ctx, original.ID, acc2, entry.RoleSender, f.owner)
	if err != nil {
		t.Fatalf("ConvertToSelfTransfer: %v", err)
	}
	if converted.ID == original.ID {
		t.Fatalf("converted.ID = %s, want a different id", converted.ID)
	}
	if converted.Kind != entry.KindSelfTransfer || converted.AccountID != f.accID || converted.ToAccountID == nil || *converted.ToAccountID != acc2 {
		t.Fatalf("converted = %+v", converted)
	}
	if converted.Amount != -1000 {
		t.Fatalf("Amount = %d, want -1000 (unchanged)", converted.Amount)
	}
	if converted.CategoryID == nil || *converted.CategoryID != f.catID {
		t.Fatalf("CategoryID = %v, want %s", converted.CategoryID, f.catID)
	}
	if converted.RecurringTransactionID == nil || *converted.RecurringTransactionID != created.ID {
		t.Fatalf("RecurringTransactionID = %v, want %s (kept on the sender path)", converted.RecurringTransactionID, created.ID)
	}
	if converted.Counterparty != "" || converted.Location != "" {
		t.Fatalf("converted = %+v, want empty counterparty/location", converted)
	}

	if _, err := f.entries.Get(ctx, original.ID); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("Get(original) err = %v, want ErrNotFound", err)
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

func TestPGConvertToSelfTransferReceiverSwapsAccountsAndDropsRecurringLink(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()
	recurring := NewRecurringTransactionStore(f.pool)

	starts, _ := account.ParseDate("2024-01-01")
	created, err := recurring.Create(ctx, f.owner, rt.New{
		AccountID: f.accID, Title: "Netflix", CategoryID: &f.catID,
		Amount: -1500, IntervalUnit: rt.UnitMonth, IntervalCount: 1,
		StartsOn: rt.NewDate(starts.Time),
	})
	if err != nil {
		t.Fatal(err)
	}

	original, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-1000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Groceries", CategoryID: &f.catID,
		RecurringTransactionID: &created.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	converted, err := f.entries.ConvertToSelfTransfer(ctx, original.ID, acc2, entry.RoleReceiver, f.owner)
	if err != nil {
		t.Fatalf("ConvertToSelfTransfer: %v", err)
	}
	if converted.AccountID != acc2 || converted.ToAccountID == nil || *converted.ToAccountID != f.accID {
		t.Fatalf("converted = %+v, want account_id=%s to_account_id=%s", converted, acc2, f.accID)
	}
	if converted.Amount != 1000 {
		t.Fatalf("Amount = %d, want 1000 (negated)", converted.Amount)
	}
	if converted.RecurringTransactionID != nil {
		t.Fatalf("RecurringTransactionID = %v, want nil (dropped on the receiver path)", converted.RecurringTransactionID)
	}

	asOf := at("2024-06-01T00:00:00Z")
	bal1, err := f.entries.Balance(ctx, f.accID, asOf)
	if err != nil {
		t.Fatal(err)
	}
	if bal1 != -1000 {
		t.Fatalf("account 1 (original account) balance = %d, want -1000 (unchanged economic effect)", bal1)
	}
	bal2, err := f.entries.Balance(ctx, acc2, asOf)
	if err != nil {
		t.Fatal(err)
	}
	if bal2 != 1000 {
		t.Fatalf("account 2 balance = %d, want 1000", bal2)
	}
}

func TestPGConvertToSelfTransferRecomputesBalanceAdjustmentsOnBothSides(t *testing.T) {
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

	original, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindTransaction, Amount: ptrInt64(-1000),
		BookingTimestamp: at("2024-01-15T00:00:00Z"), Title: "Early groceries", CategoryID: &f.catID,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.entries.ConvertToSelfTransfer(ctx, original.ID, acc2, entry.RoleSender, f.owner); err != nil {
		t.Fatalf("ConvertToSelfTransfer: %v", err)
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

func TestPGConvertToSelfTransferRejectedForNonTransactionKind(t *testing.T) {
	f, acc2 := newSelfTransferFixture(t)
	ctx := context.Background()

	adj, err := f.entries.Create(ctx, f.owner, entry.New{
		AccountID: f.accID, Kind: entry.KindBalanceAdjustment, Balance: ptrInt64(5000),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Opening",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.entries.ConvertToSelfTransfer(ctx, adj.ID, acc2, entry.RoleSender, f.owner); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
