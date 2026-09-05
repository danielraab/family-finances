package postgres

import (
	"context"
	"errors"
	"testing"

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
