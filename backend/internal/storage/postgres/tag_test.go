package postgres

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/entry"
	"at.draab/familyfinances/internal/tag"
)

func newTagStore(t *testing.T) (*TagStore, *AuthStore) {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewTagStore(pool), NewAuthStore(pool)
}

func TestPGTagCreateGetList(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagowner1@example.com")

	created, err := store.Create(ctx, owner.ID, "groceries")
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ctx, owner.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "groceries" {
		t.Fatalf("got = %+v", got)
	}
	list, err := store.List(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %v", list)
	}
}

func TestPGTagCrossOwnerNotFound(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagowner2@example.com")
	other := mustUser(t, authStore, "tagowner3@example.com")

	created, err := store.Create(ctx, owner.ID, "groceries2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, other.ID, created.ID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGTagDuplicateNameSameOwnerRejected(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagowner4@example.com")

	if _, err := store.Create(ctx, owner.ID, "dup"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, owner.ID, "dup"); !errors.Is(err, tag.ErrDuplicateName) {
		t.Fatalf("err = %v, want ErrDuplicateName", err)
	}
}

func TestPGTagOwnedBy(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagowner5@example.com")
	other := mustUser(t, authStore, "tagowner6@example.com")

	mine, err := store.Create(ctx, owner.ID, "mine")
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := store.Create(ctx, other.ID, "theirs")
	if err != nil {
		t.Fatal(err)
	}

	ok, err := store.OwnedBy(ctx, owner.ID, []string{mine.ID})
	if err != nil || !ok {
		t.Fatalf("OwnedBy own = %v %v", ok, err)
	}
	ok, err = store.OwnedBy(ctx, owner.ID, []string{mine.ID, theirs.ID})
	if err != nil || ok {
		t.Fatalf("OwnedBy foreign = %v %v, want false", ok, err)
	}
}

func TestPGTagDeleteAlwaysAllowed(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagowner7@example.com")

	created, err := store.Create(ctx, owner.ID, "deleteme")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, owner.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, owner.ID, created.ID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGTagSetDisabled(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagowner8@example.com")

	created, err := store.Create(ctx, owner.ID, "flag-me")
	if err != nil {
		t.Fatal(err)
	}
	if created.Disabled {
		t.Fatalf("newly created Disabled = true, want false")
	}

	disabled, err := store.SetDisabled(ctx, owner.ID, created.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !disabled.Disabled {
		t.Fatalf("SetDisabled(true) did not set Disabled")
	}

	enabled, err := store.SetDisabled(ctx, owner.ID, created.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if enabled.Disabled {
		t.Fatalf("SetDisabled(false) did not clear Disabled")
	}
}

func TestPGTagUsable(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagowner9@example.com")
	other := mustUser(t, authStore, "tagowner10@example.com")

	live, err := store.Create(ctx, owner.ID, "usable-live")
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := store.Create(ctx, owner.ID, "usable-disabled")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetDisabled(ctx, owner.ID, disabled.ID, true); err != nil {
		t.Fatal(err)
	}
	theirs, err := store.Create(ctx, other.ID, "usable-theirs")
	if err != nil {
		t.Fatal(err)
	}

	if ok, err := store.Usable(ctx, owner.ID, []string{live.ID}); err != nil || !ok {
		t.Fatalf("Usable(live) = %v, %v, want true, nil", ok, err)
	}
	if ok, err := store.Usable(ctx, owner.ID, []string{disabled.ID}); err != nil || ok {
		t.Fatalf("Usable(disabled) = %v, %v, want false, nil", ok, err)
	}
	if ok, err := store.Usable(ctx, owner.ID, []string{theirs.ID}); err != nil || ok {
		t.Fatalf("Usable(foreign) = %v, %v, want false, nil", ok, err)
	}
	if ok, err := store.Usable(ctx, owner.ID, []string{live.ID, disabled.ID}); err != nil || ok {
		t.Fatalf("Usable(mixed) = %v, %v, want false, nil", ok, err)
	}
}

func TestPGTagEntryCountReflectsAttachedAndDeletedEntries(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	authStore := NewAuthStore(pool)
	accStore := NewAccountStore(pool)
	catStore := NewCategoryStore(pool)
	entryStore := NewEntryStore(pool)
	tagStore := NewTagStore(pool)

	owner := mustUser(t, authStore, "tagowner11@example.com")
	typ, err := accStore.CreateType(ctx, owner.ID, "Checking-tagcount", "")
	if err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := accStore.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := catStore.Create(ctx, owner.ID, category.New{Name: "Groceries-tagcount"})
	if err != nil {
		t.Fatal(err)
	}
	tg, err := tagStore.Create(ctx, owner.ID, "counted")
	if err != nil {
		t.Fatal(err)
	}
	if tg.EntryCount != 0 {
		t.Fatalf("EntryCount = %d, want 0", tg.EntryCount)
	}

	bookedAt := at("2024-01-01T00:00:00Z")
	e1, err := entryStore.Create(ctx, owner.ID, entry.New{
		AccountID: acc.ID, Kind: entry.KindTransaction, Amount: ptrInt64(100),
		BookingTimestamp: bookedAt, Title: "First", CategoryID: &cat.ID,
		TagIDs: []string{tg.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entryStore.Create(ctx, owner.ID, entry.New{
		AccountID: acc.ID, Kind: entry.KindTransaction, Amount: ptrInt64(200),
		BookingTimestamp: bookedAt, Title: "Second", CategoryID: &cat.ID,
		TagIDs: []string{tg.ID},
	}); err != nil {
		t.Fatal(err)
	}

	got, err := tagStore.Get(ctx, owner.ID, tg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.EntryCount != 2 {
		t.Fatalf("EntryCount after two attachments = %d, want 2", got.EntryCount)
	}

	if err := entryStore.SoftDelete(ctx, e1.ID); err != nil {
		t.Fatal(err)
	}
	got, err = tagStore.Get(ctx, owner.ID, tg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.EntryCount != 1 {
		t.Fatalf("EntryCount after one soft-delete = %d, want 1", got.EntryCount)
	}
}
