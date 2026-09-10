package postgres

import (
	"context"
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

// mustAccount creates an account owned by ownerID with a plain free-text type.
func mustAccount(t *testing.T, store *AccountStore, ownerID, title, typ string) account.Account {
	t.Helper()
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(context.Background(), ownerID, account.New{
		Title: title, Type: typ, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return acc
}

func TestPGAccountCreateGetList(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner@example.com")

	// The store persists Type verbatim; trimming is the service's job.
	acc := mustAccount(t, store, owner.ID, "Main", "Checking")
	if acc.OwnerID != owner.ID || acc.Currency != "EUR" {
		t.Fatalf("acc = %+v", acc)
	}

	got, err := store.Get(ctx, acc.ID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != acc.ID || got.Permission != account.PermissionOwner || got.Type != "Checking" {
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

	acc := mustAccount(t, store, owner.ID, "Main", "Checking")

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

func TestPGAccountUpdateChangesType(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner-type@example.com")

	acc := mustAccount(t, store, owner.ID, "Main", "Checking")
	got, err := store.Update(ctx, acc.ID, account.Update{Type: strptr("Savings")})
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != "Savings" {
		t.Fatalf("Type = %q, want Savings", got.Type)
	}
}

func TestPGAccountUpdateClosingDateClear(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "owner4@example.com")

	opening, _ := account.ParseDate("2024-01-01")
	closing, _ := account.ParseDate("2024-06-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", Type: "Checking", Currency: "EUR", OpeningDate: opening, ClosingDate: &closing,
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

	acc := mustAccount(t, store, owner.ID, "Main", "Checking")
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

	acc := mustAccount(t, store, owner.ID, "Main", "Checking")

	got, err := store.SetDisabled(ctx, acc.ID, true)
	if err != nil || !got.Disabled {
		t.Fatalf("SetDisabled true: got=%+v err=%v", got, err)
	}
	access, err := store.Access(ctx, acc.ID, owner.ID)
	if err != nil || access.Currency != "EUR" || !access.Disabled || access.Permission != account.PermissionOwner {
		t.Fatalf("Access: %+v %v", access, err)
	}
}

func TestPGListInUseTypes(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "types-owner@example.com")
	other := mustUser(t, authStore, "types-other@example.com")

	mustAccount(t, store, owner.ID, "A", "Savings")
	mustAccount(t, store, owner.ID, "B", "Checking")
	mustAccount(t, store, owner.ID, "C", "Checking") // exact duplicate collapses
	gone := mustAccount(t, store, owner.ID, "D", "Loan")
	if err := store.SoftDelete(ctx, gone.ID); err != nil {
		t.Fatal(err)
	}
	mustAccount(t, store, other.ID, "E", "Brokerage") // another owner's — excluded

	got, err := store.ListInUseTypes(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Checking", "Savings"} // sorted, deduped, no deleted, no other owner
	if len(got) != len(want) {
		t.Fatalf("ListInUseTypes = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ListInUseTypes = %v, want %v", got, want)
		}
	}

	empty, err := store.ListInUseTypes(ctx, "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatal(err)
	}
	if len(empty) != 0 {
		t.Fatalf("ListInUseTypes for a user with no accounts = %v, want empty", empty)
	}
}

func strptr(s string) *string { return &s }
