package postgres

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/auth"
)

func newAccountStore(t *testing.T) (*AccountStore, *AuthStore) {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewAccountStore(pool), NewAuthStore(pool)
}

func mustUser(t *testing.T, authStore *AuthStore, email string) auth.User {
	t.Helper()
	u, _, err := authStore.CreateUserWithIdentity(context.Background(), auth.NewUser{Email: email}, emailIdentity(email))
	if err != nil {
		t.Fatalf("CreateUserWithIdentity: %v", err)
	}
	return u
}

func TestPGAccountCreateGetList(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner@example.com")

	typ, err := store.CreateType(ctx, "Checking", "")
	if err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	if acc.OwnerID != owner.ID || acc.Currency != "EUR" {
		t.Fatalf("acc = %+v", acc)
	}

	got, err := store.Get(ctx, owner.ID, acc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != acc.ID {
		t.Fatalf("Get = %+v", got)
	}

	list, err := store.List(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("List = %v", list)
	}
}

func TestPGAccountCrossOwnerNotFound(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner2@example.com")
	other := mustUser(t, authStore, "other2@example.com")

	typ, _ := store.CreateType(ctx, "Checking2", "")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Get(ctx, other.ID, acc.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGAccountCreateWithUnknownTypeIsInvalid(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner3@example.com")

	opening, _ := account.ParseDate("2024-01-01")
	_, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: "00000000-0000-0000-0000-000000000000", Currency: "EUR", OpeningDate: opening,
	})
	if !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestPGAccountUpdateClosingDateClear(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner4@example.com")

	typ, _ := store.CreateType(ctx, "Checking4", "")
	opening, _ := account.ParseDate("2024-01-01")
	closing, _ := account.ParseDate("2024-06-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening, ClosingDate: &closing,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.Update(ctx, owner.ID, acc.ID, account.Update{
		ClosingDate: account.OptionalDate{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ClosingDate != nil {
		t.Fatalf("ClosingDate = %v, want nil", got.ClosingDate)
	}
}

func TestPGAccountSoftDeleteExcludesFromListing(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner5@example.com")

	typ, _ := store.CreateType(ctx, "Checking5", "")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SoftDelete(ctx, owner.ID, acc.ID); err != nil {
		t.Fatal(err)
	}
	list, err := store.List(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list after delete = %v", list)
	}
}

func TestPGAccountDisableEnable(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner6@example.com")

	typ, _ := store.CreateType(ctx, "Checking6", "")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.SetDisabled(ctx, owner.ID, acc.ID, true)
	if err != nil || !got.Disabled {
		t.Fatalf("SetDisabled true: got=%+v err=%v", got, err)
	}
	ownerID, currency, disabled, err := store.Owner(ctx, acc.ID)
	if err != nil || ownerID != owner.ID || currency != "EUR" || !disabled {
		t.Fatalf("Owner: %q %q %v %v", ownerID, currency, disabled, err)
	}
}

func TestPGAccountTypeDeleteInUseConflict(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner7@example.com")

	typ, _ := store.CreateType(ctx, "Checking7", "")
	opening, _ := account.ParseDate("2024-01-01")
	if _, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteType(ctx, typ.ID); !errors.Is(err, account.ErrTypeInUse) {
		t.Fatalf("err = %v, want ErrTypeInUse", err)
	}
}

func TestPGAccountTypeDuplicateNameRejected(t *testing.T) {
	store, _ := newAccountStore(t)
	ctx := context.Background()

	if _, err := store.CreateType(ctx, "Savings-dup", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateType(ctx, "Savings-dup", ""); !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestPGAccountTypeTitleDescriptionRoundTrip(t *testing.T) {
	store, _ := newAccountStore(t)
	ctx := context.Background()

	typ, err := store.CreateType(ctx, "Checking8", "A day-to-day account")
	if err != nil {
		t.Fatal(err)
	}
	if typ.Title != "Checking8" || typ.Description != "A day-to-day account" || typ.Disabled {
		t.Fatalf("typ = %+v", typ)
	}

	got, err := store.GetType(ctx, typ.ID)
	if err != nil || got.Title != "Checking8" {
		t.Fatalf("GetType = %+v, err = %v", got, err)
	}

	updated, err := store.UpdateType(ctx, typ.ID, "Checking8 Renamed", "Updated description")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Checking8 Renamed" || updated.Description != "Updated description" {
		t.Fatalf("updated = %+v", updated)
	}
}

func TestPGAccountTypeDisableEnable(t *testing.T) {
	store, _ := newAccountStore(t)
	ctx := context.Background()

	typ, err := store.CreateType(ctx, "Checking9", "")
	if err != nil {
		t.Fatal(err)
	}

	disabled, err := store.SetTypeDisabled(ctx, typ.ID, true)
	if err != nil || !disabled.Disabled {
		t.Fatalf("SetTypeDisabled true: got=%+v err=%v", disabled, err)
	}

	enabled, err := store.SetTypeDisabled(ctx, typ.ID, false)
	if err != nil || enabled.Disabled {
		t.Fatalf("SetTypeDisabled false: got=%+v err=%v", enabled, err)
	}
}

func TestPGAccountTypeGetUnknownIsNotFound(t *testing.T) {
	store, _ := newAccountStore(t)
	if _, err := store.GetType(context.Background(), "00000000-0000-0000-0000-000000000000"); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
