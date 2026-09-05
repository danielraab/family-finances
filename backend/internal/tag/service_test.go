package tag_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/storage/memory"
	"at.draab/familyfinances/internal/tag"
)

func newService() *tag.Service {
	return tag.NewService(memory.NewTagStore())
}

func TestCreateAndGet(t *testing.T) {
	svc := newService()
	created, err := svc.Create(context.Background(), "u1", "groceries")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Get(context.Background(), "u1", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "groceries" {
		t.Fatalf("got = %+v", got)
	}
}

func TestCrossOwnerGetNotFound(t *testing.T) {
	svc := newService()
	created, err := svc.Create(context.Background(), "u1", "groceries")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "u2", created.ID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestDuplicateNameRejected(t *testing.T) {
	svc := newService()
	if _, err := svc.Create(context.Background(), "u1", "groceries"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "u1", "groceries"); !errors.Is(err, tag.ErrDuplicateName) {
		t.Fatalf("err = %v, want ErrDuplicateName", err)
	}
}

func TestSameNameAcrossOwnersAllowed(t *testing.T) {
	svc := newService()
	if _, err := svc.Create(context.Background(), "u1", "groceries"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "u2", "groceries"); err != nil {
		t.Fatalf("second owner: %v", err)
	}
}

func TestGetOrCreateReusesExisting(t *testing.T) {
	svc := newService()
	first, err := svc.Create(context.Background(), "u1", "groceries")
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetOrCreate(context.Background(), "u1", "groceries")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != first.ID {
		t.Fatalf("GetOrCreate created a duplicate: %+v vs %+v", got, first)
	}
}

func TestGetOrCreateCreatesWhenMissing(t *testing.T) {
	svc := newService()
	got, err := svc.GetOrCreate(context.Background(), "u1", "new-tag")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "new-tag" {
		t.Fatalf("got = %+v", got)
	}
}

func TestOwnedByChecksOwnerAndExistence(t *testing.T) {
	svc := newService()
	mine, err := svc.Create(context.Background(), "u1", "a")
	if err != nil {
		t.Fatal(err)
	}
	theirs, err := svc.Create(context.Background(), "u2", "b")
	if err != nil {
		t.Fatal(err)
	}

	ok, err := svc.OwnedBy(context.Background(), "u1", []string{mine.ID})
	if err != nil || !ok {
		t.Fatalf("OwnedBy own tag = %v, %v", ok, err)
	}
	ok, err = svc.OwnedBy(context.Background(), "u1", []string{mine.ID, theirs.ID})
	if err != nil || ok {
		t.Fatalf("OwnedBy with foreign tag = %v, %v, want false", ok, err)
	}
}

func TestDeleteRemovesTag(t *testing.T) {
	svc := newService()
	created, err := svc.Create(context.Background(), "u1", "groceries")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), "u1", created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "u1", created.ID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
