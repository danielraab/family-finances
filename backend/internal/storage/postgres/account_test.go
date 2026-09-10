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

	typ, err := store.CreateType(ctx, owner.ID, "Checking", "")
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

	got, err := store.Get(ctx, acc.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != acc.ID || got.Permission != account.PermissionOwner {
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

func TestPGAccountCrossOwnerHasNoPermission(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner2@example.com")
	other := mustUser(t, authStore, "other2@example.com")

	typ, _ := store.CreateType(ctx, owner.ID, "Checking2", "")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Store.Get itself never returns ErrNotFound just because the caller
	// lacks permission — it always returns the row, with Permission empty;
	// account.Service.Get is what translates that into ErrNotFound (see
	// design.md).
	got, err := store.Get(ctx, acc.ID, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Permission != "" {
		t.Fatalf("Permission = %q, want empty", got.Permission)
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

	typ, _ := store.CreateType(ctx, owner.ID, "Checking4", "")
	opening, _ := account.ParseDate("2024-01-01")
	closing, _ := account.ParseDate("2024-06-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening, ClosingDate: &closing,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.Update(ctx, acc.ID, account.Update{
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

	typ, _ := store.CreateType(ctx, owner.ID, "Checking5", "")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SoftDelete(ctx, acc.ID); err != nil {
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

	typ, _ := store.CreateType(ctx, owner.ID, "Checking6", "")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.SetDisabled(ctx, acc.ID, true)
	if err != nil || !got.Disabled {
		t.Fatalf("SetDisabled true: got=%+v err=%v", got, err)
	}
	access, err := store.Access(ctx, acc.ID, owner.ID)
	if err != nil || access.Currency != "EUR" || !access.Disabled || access.Permission != account.PermissionOwner {
		t.Fatalf("Access: %+v %v", access, err)
	}
}

func TestPGAccountTypeDeleteInUseConflict(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner7@example.com")

	typ, _ := store.CreateType(ctx, owner.ID, "Checking7", "")
	opening, _ := account.ParseDate("2024-01-01")
	if _, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	}); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteType(ctx, owner.ID, typ.ID); !errors.Is(err, account.ErrTypeInUse) {
		t.Fatalf("err = %v, want ErrTypeInUse", err)
	}
}

func TestPGAccountTypeSameTitleAllowedAcrossOwners(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner1 := mustUser(t, authStore, "owner8a@example.com")
	owner2 := mustUser(t, authStore, "owner8b@example.com")

	if _, err := store.CreateType(ctx, owner1.ID, "Savings-dup", ""); err != nil {
		t.Fatal(err)
	}
	// No instance-wide uniqueness anymore: a second owner (or the same one)
	// can use the same title.
	if _, err := store.CreateType(ctx, owner2.ID, "Savings-dup", ""); err != nil {
		t.Fatalf("second owner's CreateType with the same title: %v", err)
	}
}

func TestPGAccountTypeCrossOwnerNotFound(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner8c@example.com")
	other := mustUser(t, authStore, "owner8d@example.com")

	typ, err := store.CreateType(ctx, owner.ID, "Checking8c", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetType(ctx, other.ID, typ.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("cross-owner GetType err = %v, want ErrNotFound", err)
	}
	if _, err := store.UpdateType(ctx, other.ID, typ.ID, "Hijacked", ""); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("cross-owner UpdateType err = %v, want ErrNotFound", err)
	}

	types, err := store.ListTypes(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 {
		t.Fatalf("ListTypes(owner) = %+v, want exactly the one type owner created", types)
	}
}

func TestPGAccountTypeTitleDescriptionRoundTrip(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner9@example.com")

	typ, err := store.CreateType(ctx, owner.ID, "Checking8", "A day-to-day account")
	if err != nil {
		t.Fatal(err)
	}
	if typ.Title != "Checking8" || typ.Description != "A day-to-day account" || typ.Disabled {
		t.Fatalf("typ = %+v", typ)
	}

	got, err := store.GetType(ctx, owner.ID, typ.ID)
	if err != nil || got.Title != "Checking8" {
		t.Fatalf("GetType = %+v, err = %v", got, err)
	}

	updated, err := store.UpdateType(ctx, owner.ID, typ.ID, "Checking8 Renamed", "Updated description")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Checking8 Renamed" || updated.Description != "Updated description" {
		t.Fatalf("updated = %+v", updated)
	}
}

func TestPGAccountTypeDisableEnable(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner10@example.com")

	typ, err := store.CreateType(ctx, owner.ID, "Checking9", "")
	if err != nil {
		t.Fatal(err)
	}

	disabled, err := store.SetTypeDisabled(ctx, owner.ID, typ.ID, true)
	if err != nil || !disabled.Disabled {
		t.Fatalf("SetTypeDisabled true: got=%+v err=%v", disabled, err)
	}

	enabled, err := store.SetTypeDisabled(ctx, owner.ID, typ.ID, false)
	if err != nil || enabled.Disabled {
		t.Fatalf("SetTypeDisabled false: got=%+v err=%v", enabled, err)
	}
}

func TestPGAccountTypeGetUnknownIsNotFound(t *testing.T) {
	store, authStore := newAccountStore(t)
	owner := mustUser(t, authStore, "owner11@example.com")
	if _, err := store.GetType(context.Background(), owner.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGAccountTypeSeedDefaults(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner12@example.com")

	if err := store.SeedDefaultTypes(ctx, owner.ID); err != nil {
		t.Fatalf("SeedDefaultTypes: %v", err)
	}
	got, err := store.ListTypes(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(account.DefaultTypeTitles) {
		t.Fatalf("ListTypes after SeedDefaultTypes = %+v, want %d types", got, len(account.DefaultTypeTitles))
	}
}
