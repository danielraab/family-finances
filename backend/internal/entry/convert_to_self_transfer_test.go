package entry_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/entry"
)

func newConvertibleTransaction(accountID string, amount int64) entry.New {
	return entry.New{
		AccountID: accountID, Kind: entry.KindTransaction, Amount: ptr(amount), CategoryID: ptr("cat1"),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Groceries",
	}
}

func TestConvertToSelfTransferSenderKeepsAccountAndAmountAndRecurringLink(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	categories.add("cat1")
	recurring := newStubRecurringTransactions()
	recurring.add("rt1", "a")
	svc.SetRecurringTransactionLookup(recurring)

	in := newConvertibleTransaction("a", -1000)
	in.RecurringTransactionID = ptr("rt1")
	original, err := svc.Create(context.Background(), "u1", in)
	if err != nil {
		t.Fatal(err)
	}

	converted, err := svc.ConvertToSelfTransfer(context.Background(), "u1", original.ID, "b", entry.RoleSender)
	if err != nil {
		t.Fatalf("ConvertToSelfTransfer: %v", err)
	}

	if converted.ID == original.ID {
		t.Fatalf("converted.ID = %s, want a different id from the original", converted.ID)
	}
	if converted.Kind != entry.KindSelfTransfer {
		t.Fatalf("Kind = %s, want self_transfer", converted.Kind)
	}
	if converted.AccountID != "a" || converted.ToAccountID == nil || *converted.ToAccountID != "b" {
		t.Fatalf("converted = %+v, want account_id=a to_account_id=b", converted)
	}
	if converted.Amount != -1000 {
		t.Fatalf("Amount = %d, want -1000 (unchanged)", converted.Amount)
	}
	if converted.CategoryID == nil || *converted.CategoryID != "cat1" {
		t.Fatalf("CategoryID = %v, want cat1", converted.CategoryID)
	}
	if converted.RecurringTransactionID == nil || *converted.RecurringTransactionID != "rt1" {
		t.Fatalf("RecurringTransactionID = %v, want rt1 (kept on the sender path)", converted.RecurringTransactionID)
	}

	// The original entry is soft-deleted.
	if _, err := svc.Get(context.Background(), "u1", original.ID); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("Get(original) err = %v, want ErrNotFound", err)
	}

	balA, err := svc.Balance(context.Background(), "u1", "a", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balA != -1000 {
		t.Fatalf("balance a = %d, want -1000", balA)
	}
	balB, err := svc.Balance(context.Background(), "u1", "b", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balB != 1000 {
		t.Fatalf("balance b = %d, want 1000", balB)
	}
}

func TestConvertToSelfTransferReceiverSwapsAccountsAndNegatesAmountAndDropsRecurringLink(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	categories.add("cat1")
	recurring := newStubRecurringTransactions()
	recurring.add("rt1", "a")
	svc.SetRecurringTransactionLookup(recurring)

	in := newConvertibleTransaction("a", -1000)
	in.RecurringTransactionID = ptr("rt1")
	original, err := svc.Create(context.Background(), "u1", in)
	if err != nil {
		t.Fatal(err)
	}

	converted, err := svc.ConvertToSelfTransfer(context.Background(), "u1", original.ID, "b", entry.RoleReceiver)
	if err != nil {
		t.Fatalf("ConvertToSelfTransfer: %v", err)
	}

	if converted.AccountID != "b" || converted.ToAccountID == nil || *converted.ToAccountID != "a" {
		t.Fatalf("converted = %+v, want account_id=b to_account_id=a", converted)
	}
	if converted.Amount != 1000 {
		t.Fatalf("Amount = %d, want 1000 (negated, since account a's own economic effect stays -1000)", converted.Amount)
	}
	if converted.RecurringTransactionID != nil {
		t.Fatalf("RecurringTransactionID = %v, want nil (dropped on the receiver path)", converted.RecurringTransactionID)
	}

	// Account a's real economic effect is unchanged: it still loses 1000.
	balA, err := svc.Balance(context.Background(), "u1", "a", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balA != -1000 {
		t.Fatalf("balance a = %d, want -1000 (unchanged)", balA)
	}
	balB, err := svc.Balance(context.Background(), "u1", "b", at("2024-06-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if balB != 1000 {
		t.Fatalf("balance b = %d, want 1000", balB)
	}
}

func TestConvertToSelfTransferDropsCounterpartyAndLocation(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	categories.add("cat1")

	in := newConvertibleTransaction("a", -1000)
	in.Counterparty = "Rewe"
	in.Location = "Berlin"
	original, err := svc.Create(context.Background(), "u1", in)
	if err != nil {
		t.Fatal(err)
	}

	converted, err := svc.ConvertToSelfTransfer(context.Background(), "u1", original.ID, "b", entry.RoleSender)
	if err != nil {
		t.Fatalf("ConvertToSelfTransfer: %v", err)
	}
	if converted.Counterparty != "" || converted.Location != "" {
		t.Fatalf("converted = %+v, want empty counterparty/location", converted)
	}
}

func TestConvertToSelfTransferRejectedWithoutPermissionOnToAccount(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u2", "EUR") // u1 has no access at all to b
	categories.add("cat1")

	original, err := svc.Create(context.Background(), "u1", newConvertibleTransaction("a", -1000))
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ConvertToSelfTransfer(context.Background(), "u1", original.ID, "b", entry.RoleSender)
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestConvertToSelfTransferRejectedAgainstDisabledToAccount(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	accounts.disabled["b"] = true
	categories.add("cat1")

	original, err := svc.Create(context.Background(), "u1", newConvertibleTransaction("a", -1000))
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ConvertToSelfTransfer(context.Background(), "u1", original.ID, "b", entry.RoleSender)
	if !errors.Is(err, entry.ErrAccountDisabled) {
		t.Fatalf("err = %v, want ErrAccountDisabled", err)
	}
}

func TestConvertToSelfTransferRejectedOnCurrencyMismatch(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "USD")
	categories.add("cat1")

	original, err := svc.Create(context.Background(), "u1", newConvertibleTransaction("a", -1000))
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ConvertToSelfTransfer(context.Background(), "u1", original.ID, "b", entry.RoleSender)
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestConvertToSelfTransferRejectedForSameAccount(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	categories.add("cat1")

	original, err := svc.Create(context.Background(), "u1", newConvertibleTransaction("a", -1000))
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ConvertToSelfTransfer(context.Background(), "u1", original.ID, "a", entry.RoleSender)
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestConvertToSelfTransferRejectedWhenCallerHasNoVisibility(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u2", "EUR")
	categories.add("cat1")

	original, err := svc.Create(context.Background(), "u1", newConvertibleTransaction("a", -1000))
	if err != nil {
		t.Fatal(err)
	}

	// u2 has no permission at all on "a".
	_, err = svc.ConvertToSelfTransfer(context.Background(), "u2", original.ID, "b", entry.RoleSender)
	if !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestConvertToSelfTransferRejectedForNonTransactionKind(t *testing.T) {
	svc, accounts, _, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")

	adj, err := svc.Create(context.Background(), "u1", entry.New{
		AccountID: "a", Kind: entry.KindBalanceAdjustment, Balance: ptr(int64(5000)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Opening",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ConvertToSelfTransfer(context.Background(), "u1", adj.ID, "b", entry.RoleSender)
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestConvertToSelfTransferRejectedForInvalidRole(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("a", "u1", "EUR")
	accounts.add("b", "u1", "EUR")
	categories.add("cat1")

	original, err := svc.Create(context.Background(), "u1", newConvertibleTransaction("a", -1000))
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.ConvertToSelfTransfer(context.Background(), "u1", original.ID, "b", entry.OriginalAccountRole("sideways"))
	if !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}
