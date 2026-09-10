package postgres

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/entry"
)

func newCategoryStore(t *testing.T) (*CategoryStore, *AuthStore) {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewCategoryStore(pool), NewAuthStore(pool)
}

func TestPGCategoryCreateGetList(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner@example.com").ID

	root, err := store.Create(ctx, owner, category.New{Name: "Expenses"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := store.Create(ctx, owner, category.New{ParentID: &root.ID, Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != root.ID {
		t.Fatalf("child = %+v", child)
	}

	list, err := store.List(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("List = %v", list)
	}
}

func TestPGCategoryCrossOwnerNotFound(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner2@example.com").ID
	other := mustUser(t, authStore, "cat-other2@example.com").ID

	c, err := store.Create(ctx, owner, category.New{Name: "Mine"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, other, c.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGCategoryDeleteWithChildrenConflict(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner3@example.com").ID

	root, err := store.Create(ctx, owner, category.New{Name: "Expenses2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(ctx, owner, category.New{ParentID: &root.ID, Name: "Groceries2"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, owner, root.ID); !errors.Is(err, category.ErrInUse) {
		t.Fatalf("err = %v, want ErrInUse", err)
	}
}

func TestPGCategoryDeleteLeafSucceeds(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner4@example.com").ID

	c, err := store.Create(ctx, owner, category.New{Name: "Leaf"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, owner, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, owner, c.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	list, err := store.List(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("List after soft delete = %v, want empty", list)
	}
}

func TestPGCategorySetDisabledDoesNotAffectVisibility(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner5@example.com").ID

	c, err := store.Create(ctx, owner, category.New{Name: "ToDisable"})
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := store.SetDisabled(ctx, owner, c.ID, true)
	if err != nil || !disabled.Disabled {
		t.Fatalf("SetDisabled: got=%+v err=%v", disabled, err)
	}
	list, err := store.List(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !list[0].Disabled {
		t.Fatalf("List after disable = %v, want 1 disabled category", list)
	}
}

func TestPGCategoryMoveUpMoveDown(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner6@example.com").ID

	a, err := store.Create(ctx, owner, category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(ctx, owner, category.New{Name: "B"})
	if err != nil {
		t.Fatal(err)
	}

	moved, err := store.MoveUp(ctx, owner, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.SortOrder != a.SortOrder {
		t.Fatalf("moved.SortOrder = %d, want %d", moved.SortOrder, a.SortOrder)
	}

	noop, err := store.MoveUp(ctx, owner, moved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if noop.SortOrder != moved.SortOrder {
		t.Fatalf("no-op move-up changed SortOrder: got %d, want %d", noop.SortOrder, moved.SortOrder)
	}
}

func TestPGCategorySubtreeRecursive(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner7@example.com").ID

	a, err := store.Create(ctx, owner, category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(ctx, owner, category.New{ParentID: &a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := store.Create(ctx, owner, category.New{ParentID: &b.ID, Name: "C"})
	if err != nil {
		t.Fatal(err)
	}

	ids, err := store.Subtree(ctx, owner, a.ID)
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
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner8@example.com").ID

	a, err := store.Create(ctx, owner, category.New{Name: "RootA"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(ctx, owner, category.New{ParentID: &a.ID, Name: "ChildB"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Update(ctx, owner, b.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ParentID != nil {
		t.Fatalf("ParentID = %v, want nil", got.ParentID)
	}
}

func TestPGCategoryReparentAppendsToNewParentsSiblingOrder(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-owner9@example.com").ID

	oldParent, err := store.Create(ctx, owner, category.New{Name: "Old"})
	if err != nil {
		t.Fatal(err)
	}
	newParent, err := store.Create(ctx, owner, category.New{Name: "New"})
	if err != nil {
		t.Fatal(err)
	}
	existingChild, err := store.Create(ctx, owner, category.New{ParentID: &newParent.ID, Name: "Existing"})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := store.Create(ctx, owner, category.New{ParentID: &oldParent.ID, Name: "Moved"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := store.Update(ctx, owner, moved.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: &newParent.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.SortOrder <= existingChild.SortOrder {
		t.Fatalf("SortOrder = %d, want > existing child's %d", got.SortOrder, existingChild.SortOrder)
	}
}

func TestPGCategorySeedDefaults(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "cat-seed@example.com").ID

	if err := store.SeedDefaults(ctx, owner); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	got, err := store.List(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(category.DefaultNames) {
		t.Fatalf("List after SeedDefaults = %+v, want %d categories", got, len(category.DefaultNames))
	}
	for i, c := range got {
		if c.Name != category.DefaultNames[i] {
			t.Fatalf("category %d name = %q, want %q (in order)", i, c.Name, category.DefaultNames[i])
		}
	}
}

// TestPGCategoryEntryCount verifies entry_count counts only the caller's
// own non-deleted entries that directly reference the category — a
// descendant category's entries never roll up to an ancestor.
func TestPGCategoryEntryCount(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	authStore := NewAuthStore(pool)
	accStore := NewAccountStore(pool)
	catStore := NewCategoryStore(pool)
	entryStore := NewEntryStore(pool)

	owner := mustUser(t, authStore, "cat-entrycount@example.com")
	other := mustUser(t, authStore, "cat-entrycount-other@example.com")

	opening, _ := account.ParseDate("2024-01-01")
	acc, err := accStore.Create(ctx, owner.ID, account.New{
		Title: "Main", Type: "Checking", Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}

	parent, err := catStore.Create(ctx, owner.ID, category.New{Name: "Parent"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := catStore.Create(ctx, owner.ID, category.New{ParentID: &parent.ID, Name: "Child"})
	if err != nil {
		t.Fatal(err)
	}

	if parent.EntryCount != 0 {
		t.Fatalf("fresh parent EntryCount = %d, want 0", parent.EntryCount)
	}

	bookedAt := at("2024-01-01T00:00:00Z")
	mkEntry := func(title string, catID string) entry.Entry {
		e, err := entryStore.Create(ctx, owner.ID, entry.New{
			AccountID: acc.ID, Kind: entry.KindTransaction, Amount: ptrInt64(100),
			BookingTimestamp: bookedAt, Title: title, CategoryID: &catID,
		})
		if err != nil {
			t.Fatalf("create entry %q: %v", title, err)
		}
		return e
	}

	p1 := mkEntry("p1", parent.ID)
	mkEntry("p2", parent.ID)
	mkEntry("c1", child.ID)
	mkEntry("c2", child.ID)
	mkEntry("c3", child.ID)

	assertCount := func(id string, want int) {
		t.Helper()
		got, err := catStore.Get(ctx, owner.ID, id)
		if err != nil {
			t.Fatal(err)
		}
		if got.EntryCount != want {
			t.Fatalf("category %s EntryCount = %d, want %d", got.Name, got.EntryCount, want)
		}
	}

	assertCount(parent.ID, 2)
	assertCount(child.ID, 3)

	if err := entryStore.SoftDelete(ctx, p1.ID); err != nil {
		t.Fatal(err)
	}
	assertCount(parent.ID, 1)
	assertCount(child.ID, 3)

	// List reports the same counts as Get.
	list, err := catStore.List(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]int{}
	for _, c := range list {
		byID[c.ID] = c.EntryCount
	}
	if byID[parent.ID] != 1 || byID[child.ID] != 3 {
		t.Fatalf("List counts = parent %d, child %d; want 1, 3", byID[parent.ID], byID[child.ID])
	}

	// Another user's identically-named category is independent and empty.
	otherCat, err := catStore.Create(ctx, other.ID, category.New{Name: "Parent"})
	if err != nil {
		t.Fatal(err)
	}
	assertCountOther := func() {
		got, err := catStore.Get(ctx, other.ID, otherCat.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.EntryCount != 0 {
			t.Fatalf("other user's category EntryCount = %d, want 0", got.EntryCount)
		}
	}
	assertCountOther()
}
