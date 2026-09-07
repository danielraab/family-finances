package account_test

import (
	"context"
	"errors"
	"fmt"
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

var typeNameSeq int

// mustType creates a fresh account type owned by ownerID with a unique
// title and returns its id — callers within the same test that need more
// than one live type just call it again.
func mustType(t *testing.T, svc *account.Service, ownerID string) string {
	t.Helper()
	typeNameSeq++
	typ, err := svc.CreateType(context.Background(), ownerID, fmt.Sprintf("Checking-%d", typeNameSeq), "")
	if err != nil {
		t.Fatalf("CreateType: %v", err)
	}
	return typ.ID
}

func TestCreateAccount(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	typeID := mustType(t, svc, "u1")

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
	typeID := mustType(t, svc, "u1")
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "  ", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsInvalidCurrency(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u1")
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
	typeID := mustType(t, svc, "u1")
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
	typeID := mustType(t, svc, "u1")
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
	typeID := mustType(t, svc, "u1")
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
	typeID := mustType(t, svc, "u1")
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
	typeID := mustType(t, svc, "u1")
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
	typeID := mustType(t, svc, "u1")
	opening, _ := account.ParseDate("2024-01-01")
	if _, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteType(context.Background(), "u1", typeID); !errors.Is(err, account.ErrTypeInUse) {
		t.Fatalf("err = %v, want ErrTypeInUse", err)
	}
}

func TestOwnerLookup(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u1")
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

func TestCreateType(t *testing.T) {
	svc, _ := newService(t)
	typ, err := svc.CreateType(context.Background(), "u1", "Checking", "A day-to-day account")
	if err != nil {
		t.Fatalf("CreateType: %v", err)
	}
	if typ.Title != "Checking" || typ.Description != "A day-to-day account" || typ.Disabled {
		t.Fatalf("typ = %+v", typ)
	}
}

func TestListTypesIsScopedToOwner(t *testing.T) {
	svc, _ := newService(t)
	mustType(t, svc, "u1")
	mustType(t, svc, "u2")

	got, err := svc.ListTypes(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("ListTypes(u1) = %+v, want exactly u1's own type", got)
	}
}

func TestCreateRejectsAnotherOwnersType(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u2")
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateTypeCrossOwnerIsNotFound(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u1")
	if _, err := svc.UpdateType(context.Background(), "u2", typeID, "Renamed", ""); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestSeedDefaultsInsertsStarterSet(t *testing.T) {
	svc, _ := newService(t)
	if err := svc.SeedDefaults(context.Background(), "u1"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	got, err := svc.ListTypes(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(account.DefaultTypeTitles) {
		t.Fatalf("ListTypes after SeedDefaults = %+v, want %d types", got, len(account.DefaultTypeTitles))
	}
	titles := map[string]bool{}
	for _, ty := range got {
		titles[ty.Title] = true
	}
	for _, want := range account.DefaultTypeTitles {
		if !titles[want] {
			t.Fatalf("missing seeded type %q, got %+v", want, got)
		}
	}
}

func TestCreateRejectsDisabledType(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u1")
	if _, err := svc.DisableType(context.Background(), "u1", typeID); err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	_, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if !errors.Is(err, account.ErrTypeDisabled) {
		t.Fatalf("err = %v, want ErrTypeDisabled", err)
	}
}

func TestUpdateRejectsWhenCurrentTypeIsDisabled(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u1")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DisableType(context.Background(), "u1", typeID); err != nil {
		t.Fatal(err)
	}

	// An update that doesn't touch type_id at all is still rejected, because
	// the account's current (disabled) type is the effective one.
	_, err = svc.Update(context.Background(), "u1", acc.ID, account.Update{
		FinancialInstitute: ptr("Some Bank"),
	})
	if !errors.Is(err, account.ErrTypeDisabled) {
		t.Fatalf("err = %v, want ErrTypeDisabled", err)
	}

	// Supplying a live type in the same update succeeds and un-sticks the
	// account.
	liveTypeID := mustType(t, svc, "u1")
	got, err := svc.Update(context.Background(), "u1", acc.ID, account.Update{
		FinancialInstitute: ptr("Some Bank"),
		TypeID:             &liveTypeID,
	})
	if err != nil {
		t.Fatalf("Update with live type: %v", err)
	}
	if got.TypeID != liveTypeID || got.FinancialInstitute != "Some Bank" {
		t.Fatalf("got = %+v", got)
	}

	// A later edit no longer needs to resupply type_id.
	if _, err := svc.Update(context.Background(), "u1", acc.ID, account.Update{
		FinancialInstitute: ptr("Another Bank"),
	}); err != nil {
		t.Fatalf("follow-up Update: %v", err)
	}
}

func TestUpdateRejectsAssigningADisabledType(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u1")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}
	otherTypeID := mustType(t, svc, "u1")
	if _, err := svc.DisableType(context.Background(), "u1", otherTypeID); err != nil {
		t.Fatal(err)
	}

	_, err = svc.Update(context.Background(), "u1", acc.ID, account.Update{TypeID: &otherTypeID})
	if !errors.Is(err, account.ErrTypeDisabled) {
		t.Fatalf("err = %v, want ErrTypeDisabled", err)
	}
}

func TestDisableTypeDoesNotAffectExistingAccounts(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u1")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.DisableType(context.Background(), "u1", typeID); err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(context.Background(), "u1", acc.ID)
	if err != nil || got.TypeID != typeID {
		t.Fatalf("Get after type disabled: %+v %v", got, err)
	}

	typ, err := svc.EnableType(context.Background(), "u1", typeID)
	if err != nil || typ.Disabled {
		t.Fatalf("EnableType: %+v %v", typ, err)
	}
}

func TestDeleteTypeInUseRejectedRegardlessOfDisabled(t *testing.T) {
	svc, _ := newService(t)
	typeID := mustType(t, svc, "u1")
	opening, _ := account.ParseDate("2024-01-01")
	if _, err := svc.Create(context.Background(), "u1", account.New{Title: "X", TypeID: typeID, Currency: "EUR", OpeningDate: opening}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DisableType(context.Background(), "u1", typeID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteType(context.Background(), "u1", typeID); !errors.Is(err, account.ErrTypeInUse) {
		t.Fatalf("err = %v, want ErrTypeInUse", err)
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
