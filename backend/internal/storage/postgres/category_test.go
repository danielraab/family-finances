package postgres

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/category"
)

func newCategoryStore(t *testing.T) *CategoryStore {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewCategoryStore(pool)
}

func TestPGCategoryCreateGetList(t *testing.T) {
	store := newCategoryStore(t)
	ctx := context.Background()

	root, err := store.Create(ctx, category.New{Name: "Expenses"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := store.Create(ctx, category.New{ParentID: &root.ID, Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != root.ID {
		t.Fatalf("child = %+v", child)
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("List = %v", list)
	}
}

func TestPGCategoryDeleteWithChildrenConflict(t *testing.T) {
	store := newCategoryStore(t)
	ctx := context.Background()

	root, err := store.Create(ctx, category.New{Name: "Expenses2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, category.New{ParentID: &root.ID, Name: "Groceries2"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, root.ID); !errors.Is(err, category.ErrInUse) {
		t.Fatalf("err = %v, want ErrInUse", err)
	}
}

func TestPGCategoryDeleteLeafSucceeds(t *testing.T) {
	store := newCategoryStore(t)
	ctx := context.Background()

	c, err := store.Create(ctx, category.New{Name: "Leaf"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, c.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGCategorySubtreeRecursive(t *testing.T) {
	store := newCategoryStore(t)
	ctx := context.Background()

	a, err := store.Create(ctx, category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(ctx, category.New{ParentID: &a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := store.Create(ctx, category.New{ParentID: &b.ID, Name: "C"})
	if err != nil {
		t.Fatal(err)
	}

	ids, err := store.Subtree(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{a.ID: true, b.ID: true, c.ID: true}
	if len(ids) != 3 {
		t.Fatalf("Subtree = %v", ids)
	}
	for _, id := range ids {
		if !want[id] {
			t.Fatalf("unexpected id %s", id)
		}
	}
}

func TestPGCategoryUpdateToRootClearsParent(t *testing.T) {
	store := newCategoryStore(t)
	ctx := context.Background()

	a, err := store.Create(ctx, category.New{Name: "RootA"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(ctx, category.New{ParentID: &a.ID, Name: "ChildB"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Update(ctx, b.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ParentID != nil {
		t.Fatalf("ParentID = %v, want nil", got.ParentID)
	}
}
