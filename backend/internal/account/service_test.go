package account_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/storage/memory"
)

func ptr[T any](v T) *T { return &v }

func newService(t *testing.T) (*account.Service, *memory.AccountStore) {
	t.Helper()
	store := memory.NewAccountStore()
	return account.NewService(store), store
}

// newAccount is the common create fixture: a valid account owned by
// ownerID with a plain free-text type.
func newAccount(t *testing.T, svc *account.Service, ownerID, title, typ string) account.Account {
	t.Helper()
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), ownerID, account.New{
		Title: title, Type: typ, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return acc
}

func TestCreateAccount(t *testing.T) {
	svc, _ := newService(t)
	acc := newAccount(t, svc, "u1", "Main Checking", "Checking")
	if acc.OwnerID != "u1" || acc.Title != "Main Checking" || acc.Type != "Checking" || acc.Disabled {
		t.Fatalf("acc = %+v", acc)
	}
}

func TestCreateTrimsType(t *testing.T) {
	svc, _ := newService(t)
	acc := newAccount(t, svc, "u1", "X", "  Checking  ")
	if acc.Type != "Checking" {
		t.Fatalf("Type = %q, want it trimmed to %q", acc.Type, "Checking")
	}
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	svc, _ := newService(t)
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "  ", Type: "Checking", Currency: "EUR", OpeningDate: opening})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsBlankType(t *testing.T) {
	svc, _ := newService(t)
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "X", Type: "   ", Currency: "EUR", OpeningDate: opening})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsInvalidCurrency(t *testing.T) {
	svc, _ := newService(t)
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "X", Type: "Checking", Currency: "eur", OpeningDate: opening})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsClosingBeforeOpening(t *testing.T) {
	svc, _ := newService(t)
	opening, _ := account.ParseDate("2024-06-01")
	closing, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{
		Title: "X", Type: "Checking", Currency: "EUR", OpeningDate: opening, ClosingDate: &closing,
	})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestGetIsScopedToOwner(t *testing.T) {
	svc, _ := newService(t)
	acc := newAccount(t, svc, "u1", "X", "Checking")

	if _, err := svc.Get(context.Background(), "u2", acc.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("cross-owner Get err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(context.Background(), "u1", acc.ID); err != nil {
		t.Fatalf("owner Get: %v", err)
	}
}

func TestUpdateChangesType(t *testing.T) {
	svc, _ := newService(t)
	acc := newAccount(t, svc, "u1", "X", "Checking")

	got, err := svc.Update(context.Background(), "u1", acc.ID, account.Update{Type: ptr("  Savings  ")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.Type != "Savings" {
		t.Fatalf("Type = %q, want the trimmed %q", got.Type, "Savings")
	}
}

func TestUpdateRejectsBlankType(t *testing.T) {
	svc, _ := newService(t)
	acc := newAccount(t, svc, "u1", "X", "Checking")

	_, err := svc.Update(context.Background(), "u1", acc.ID, account.Update{Type: ptr("   ")})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateClosingDateCanBeCleared(t *testing.T) {
	svc, _ := newService(t)
	opening, _ := account.ParseDate("2024-01-01")
	closing, _ := account.ParseDate("2024-06-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{
		Title: "X", Type: "Checking", Currency: "EUR", OpeningDate: opening, ClosingDate: &closing,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Update(context.Background(), "u1", acc.ID, account.Update{
		ClosingDate: account.OptionalDate{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.ClosingDate != nil {
		t.Fatalf("ClosingDate = %v, want nil", got.ClosingDate)
	}
}

func TestDisableBlocksNothingButItself(t *testing.T) {
	svc, _ := newService(t)
	acc := newAccount(t, svc, "u1", "X", "Checking")

	got, err := svc.Disable(context.Background(), "u1", acc.ID)
	if err != nil || !got.Disabled {
		t.Fatalf("Disable: got=%+v err=%v", got, err)
	}
	if got, err := svc.Get(context.Background(), "u1", acc.ID); err != nil || !got.Disabled {
		t.Fatalf("Get after disable: %+v %v", got, err)
	}

	got, err = svc.Enable(context.Background(), "u1", acc.ID)
	if err != nil || got.Disabled {
		t.Fatalf("Enable: got=%+v err=%v", got, err)
	}
}

func TestSoftDeleteExcludesFromListingAndGet(t *testing.T) {
	svc, _ := newService(t)
	acc := newAccount(t, svc, "u1", "X", "Checking")

	if err := svc.Delete(context.Background(), "u1", acc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get(context.Background(), "u1", acc.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
	}
	list, err := svc.List(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("List after delete = %v, want empty", list)
	}
}

func TestAccessLookup(t *testing.T) {
	svc, _ := newService(t)
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{Title: "X", Type: "Checking", Currency: "USD", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}

	currency, disabled, permission, err := svc.Access(context.Background(), acc.ID, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if currency != "USD" || disabled || permission != string(account.PermissionOwner) {
		t.Fatalf("Access = %q %v %q", currency, disabled, permission)
	}

	// A user with no ownership or share has an empty permission, not an
	// error — Access never returns ErrNotFound just because the caller
	// lacks access; that translation happens in Service.Get.
	_, _, permission, err = svc.Access(context.Background(), acc.ID, "u2")
	if err != nil {
		t.Fatal(err)
	}
	if permission != "" {
		t.Fatalf("Access(u2) permission = %q, want empty", permission)
	}
}

func TestListInUseTypesIsDistinctSortedAndOwnerScoped(t *testing.T) {
	svc, _ := newService(t)
	newAccount(t, svc, "u1", "A", "Savings")
	newAccount(t, svc, "u1", "B", "checking")
	newAccount(t, svc, "u1", "C", "Checking")
	newAccount(t, svc, "u1", "D", "Checking") // exact-match duplicate collapses
	newAccount(t, svc, "u2", "E", "Brokerage")

	got, err := svc.ListInUseTypes(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	// Case-insensitive sort (ties broken by raw value, so "Checking"
	// precedes "checking"), exact-match dedupe, only u1's own accounts.
	want := []string{"Checking", "checking", "Savings"}
	if len(got) != len(want) {
		t.Fatalf("ListInUseTypes = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListInUseTypes = %v, want %v", got, want)
		}
	}
}

func TestListInUseTypesEmptyForUserWithNoAccounts(t *testing.T) {
	svc, _ := newService(t)
	got, err := svc.ListInUseTypes(context.Background(), "nobody")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("ListInUseTypes = %v, want empty", got)
	}
}

func TestListInUseTypesExcludesSoftDeletedAccounts(t *testing.T) {
	svc, _ := newService(t)
	keep := newAccount(t, svc, "u1", "keep", "Savings")
	gone := newAccount(t, svc, "u1", "gone", "Checking")
	_ = keep
	if err := svc.Delete(context.Background(), "u1", gone.ID); err != nil {
		t.Fatal(err)
	}

	got, err := svc.ListInUseTypes(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != "Savings" {
		t.Fatalf("ListInUseTypes = %v, want [Savings]", got)
	}
}

func TestDateJSONRoundTrip(t *testing.T) {
	d, err := account.ParseDate("2024-03-05")
	if err != nil {
		t.Fatal(err)
	}
	b, err := d.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"2024-03-05"` {
		t.Fatalf("MarshalJSON = %s", b)
	}
	var got account.Date
	if err := got.UnmarshalJSON(b); err != nil {
		t.Fatal(err)
	}
	if !got.Time.Equal(d.Time) {
		t.Fatalf("round-trip mismatch: got %v want %v", got.Time, d.Time)
	}
}
