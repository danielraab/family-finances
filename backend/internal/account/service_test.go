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

func mustType(t *testing.T, svc *account.Service) string {
	t.Helper()
	typ, err := svc.CreateType(context.Background(), "Checking")
	if err != nil {
		t.Fatalf("CreateType: %v", err)
	}
	return typ.ID
}

func TestCreateAccount(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	typeID := mustType(t, svc)

	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(ctx, "u1", account.New{
		Title:       "Main Checking",
		TypeID:      typeID,
		Currency:    "EUR",
		OpeningDate: opening,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if acc.OwnerID != "u1" || acc.Title != "Main Checking" || acc.Disabled {
		t.Fatalf("acc = %+v", acc)
	}
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "  ", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsInvalidCurrency(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "eur", OpeningDate: opening})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsUnknownType(t *testing.T) {
	svc, _ := newService(t)
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: "nope", Currency: "EUR", OpeningDate: opening})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsClosingBeforeOpening(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-06-01")
	closing, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{
		Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening, ClosingDate: &closing,
	})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestGetIsScopedToOwner(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Get(context.Background(), "u2", acc.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("cross-owner Get err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(context.Background(), "u1", acc.ID); err != nil {
		t.Fatalf("owner Get: %v", err)
	}
}

func TestUpdateClosingDateCanBeCleared(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-01-01")
	closing, _ := account.ParseDate("2024-06-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{
		Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening, ClosingDate: &closing,
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
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Disable(context.Background(), "u1", acc.ID)
	if err != nil || !got.Disabled {
		t.Fatalf("Disable: got=%+v err=%v", got, err)
	}
	// still visible/gettable
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
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}

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

func TestDeleteTypeInUseRejected(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-01-01")
	if _, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteType(context.Background(), typeID); !errors.Is(err, account.ErrTypeInUse) {
		t.Fatalf("err = %v, want ErrTypeInUse", err)
	}
}

func TestOwnerLookup(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc)
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "USD", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}

	ownerID, currency, disabled, err := svc.Owner(context.Background(), acc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ownerID != "u1" || currency != "USD" || disabled {
		t.Fatalf("Owner = %q %q %v", ownerID, currency, disabled)
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
