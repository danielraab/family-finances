package postgres

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/account"
)

func TestPGAccountShareCreateListResolvesRealNames(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "shareowner@example.com")
	member := mustUser(t, authStore, "sharemember@example.com")

	typ, err := store.CreateType(ctx, owner.ID, "Checking-share", "")
	if err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Joint", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.CreateOrUpdateShare(ctx, acc.ID, member.ID, account.PermissionAppend, owner.ID); err != nil {
		t.Fatalf("CreateOrUpdateShare: %v", err)
	}

	shares, err := store.ListShares(ctx, acc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 {
		t.Fatalf("shares = %+v, want 1", shares)
	}
	sh := shares[0]
	if sh.UserID != member.ID || sh.Email != "sharemember@example.com" || sh.Permission != account.PermissionAppend {
		t.Fatalf("share = %+v", sh)
	}
	if sh.GrantedBy != owner.ID || sh.GrantedByName != "shareowner@example.com" {
		// No display_name set at creation, so the resolved name falls back
		// to email — mirrors accounts.owner_name's own COALESCE.
		t.Fatalf("share granted_by = %+v", sh)
	}

	// Get/Access reflect the share for the member, and the account carries
	// the real owner's resolved name for them.
	got, err := store.Get(ctx, acc.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Permission != account.PermissionAppend || !got.Shared || got.OwnerName != "shareowner@example.com" {
		t.Fatalf("Get(member) = %+v", got)
	}

	access, err := store.Access(ctx, acc.ID, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if access.Permission != account.PermissionAppend {
		t.Fatalf("Access(member) = %+v", access)
	}
}

func TestPGAccountShareUpsertUpdatesPermissionInPlace(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "shareowner2@example.com")
	member := mustUser(t, authStore, "sharemember2@example.com")

	typ, _ := store.CreateType(ctx, owner.ID, "Checking-share2", "")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Joint2", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.CreateOrUpdateShare(ctx, acc.ID, member.ID, account.PermissionView, owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, acc.ID, member.ID, account.PermissionOwner, owner.ID); err != nil {
		t.Fatal(err)
	}

	shares, err := store.ListShares(ctx, acc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 || shares[0].Permission != account.PermissionOwner {
		t.Fatalf("shares = %+v, want exactly one at owner tier", shares)
	}
}

func TestPGAccountShareDeleteAndListVisibility(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "shareowner3@example.com")
	member := mustUser(t, authStore, "sharemember3@example.com")

	typ, _ := store.CreateType(ctx, owner.ID, "Checking-share3", "")
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Joint3", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, acc.ID, member.ID, account.PermissionView, owner.ID); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(ctx, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("List(member) = %+v, want the shared account", list)
	}
	ids, err := store.VisibleIDs(ctx, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != acc.ID {
		t.Fatalf("VisibleIDs(member) = %v", ids)
	}

	if err := store.DeleteShare(ctx, acc.ID, member.ID); err != nil {
		t.Fatalf("DeleteShare: %v", err)
	}
	list2, err := store.List(ctx, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != 0 {
		t.Fatalf("List(member) after delete = %+v, want empty", list2)
	}

	if err := store.DeleteShare(ctx, acc.ID, member.ID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("DeleteShare (already gone): err = %v, want ErrNotFound", err)
	}
}
