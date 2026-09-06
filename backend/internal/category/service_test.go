package category_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/storage/memory"
)

func ptr[T any](v T) *T { return &v }

func newService() *category.Service {
	return category.NewService(memory.NewCategoryStore())
}

func TestSeedDefaultsInsertsStarterSetInOrder(t *testing.T) {
	svc := newService()
	if err := svc.SeedDefaults(context.Background(), "u1"); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	got, err := svc.List(context.Background(), "u1")
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
		if c.ParentID != nil {
			t.Fatalf("category %d = %+v, want a root category", i, c)
		}
	}
}

func TestCreateRootCategory(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if c.ParentID != nil || c.Name != "Groceries" || c.Disabled {
		t.Fatalf("c = %+v", c)
	}
}

func TestCreateChildCategory(t *testing.T) {
	svc := newService()
	parent, err := svc.Create(context.Background(), "u1", category.New{Name: "Expenses"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := svc.Create(context.Background(), "u1", category.New{ParentID: &parent.ID, Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatalf("child = %+v", child)
	}
}

func TestCreateAppendsToEndOfSiblingGroup(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), "u1", category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), "u1", category.New{Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	if b.SortOrder <= a.SortOrder {
		t.Fatalf("b.SortOrder = %d, want > a.SortOrder = %d", b.SortOrder, a.SortOrder)
	}
}

func TestCreateRejectsUnknownParent(t *testing.T) {
	svc := newService()
	_, err := svc.Create(context.Background(), "u1", category.New{ParentID: ptr("nope"), Name: "X"})
	if !errors.Is(err, category.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsAnotherOwnersCategoryAsParent(t *testing.T) {
	svc := newService()
	foreign, err := svc.Create(context.Background(), "u2", category.New{Name: "Foreign"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Create(context.Background(), "u1", category.New{ParentID: &foreign.ID, Name: "X"})
	if !errors.Is(err, category.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	svc := newService()
	_, err := svc.Create(context.Background(), "u1", category.New{Name: "  "})
	if !errors.Is(err, category.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCrossOwnerAccessNotFound(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "u2", c.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("Get err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Update(context.Background(), "u2", c.ID, category.Update{Name: ptr("X")}); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("Update err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(context.Background(), "u2", c.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("Delete err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Disable(context.Background(), "u2", c.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("Disable err = %v, want ErrNotFound", err)
	}
}

func TestDeleteWithChildrenRejected(t *testing.T) {
	svc := newService()
	parent, err := svc.Create(context.Background(), "u1", category.New{Name: "Expenses"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "u1", category.New{ParentID: &parent.ID, Name: "Groceries"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), "u1", parent.ID); !errors.Is(err, category.ErrInUse) {
		t.Fatalf("err = %v, want ErrInUse", err)
	}
}

func TestDeleteLeafSucceeds(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), "u1", c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), "u1", c.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
	}
}

func TestDisabledCategoryWithNoRelationsStillDeletable(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Disable(context.Background(), "u1", c.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), "u1", c.ID); err != nil {
		t.Fatalf("Delete of disabled, unreferenced category: %v", err)
	}
}

func TestDisableEnableRoundTrip(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	disabled, err := svc.Disable(context.Background(), "u1", c.ID)
	if err != nil || !disabled.Disabled {
		t.Fatalf("Disable: got=%+v err=%v", disabled, err)
	}
	got, err := svc.Get(context.Background(), "u1", c.ID)
	if err != nil || !got.Disabled {
		t.Fatalf("Get after disable: got=%+v err=%v", got, err)
	}
	enabled, err := svc.Enable(context.Background(), "u1", c.ID)
	if err != nil || enabled.Disabled {
		t.Fatalf("Enable: got=%+v err=%v", enabled, err)
	}
}

func TestUpdateSelfParentRejected(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Update(context.Background(), "u1", c.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: &c.ID},
	})
	if !errors.Is(err, category.ErrCycle) {
		t.Fatalf("err = %v, want ErrCycle", err)
	}
}

func TestUpdateCycleThroughDescendantRejected(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), "u1", category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), "u1", category.New{ParentID: &a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Update(context.Background(), "u1", a.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: &b.ID},
	})
	if !errors.Is(err, category.ErrCycle) {
		t.Fatalf("err = %v, want ErrCycle", err)
	}
}

func TestUpdateToRootClearsParent(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), "u1", category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), "u1", category.New{ParentID: &a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Update(context.Background(), "u1", b.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ParentID != nil {
		t.Fatalf("ParentID = %v, want nil", got.ParentID)
	}
}

func TestReparentAppendsToNewParentsSiblingOrder(t *testing.T) {
	svc := newService()
	oldParent, err := svc.Create(context.Background(), "u1", category.New{Name: "Old"})
	if err != nil {
		t.Fatal(err)
	}
	newParent, err := svc.Create(context.Background(), "u1", category.New{Name: "New"})
	if err != nil {
		t.Fatal(err)
	}
	existingChild, err := svc.Create(context.Background(), "u1", category.New{ParentID: &newParent.ID, Name: "Existing"})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := svc.Create(context.Background(), "u1", category.New{ParentID: &oldParent.ID, Name: "Moved"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Update(context.Background(), "u1", moved.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: &newParent.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.SortOrder <= existingChild.SortOrder {
		t.Fatalf("SortOrder = %d, want > existing child's %d", got.SortOrder, existingChild.SortOrder)
	}
}

func TestMoveUpSwapsWithPreviousSibling(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), "u1", category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), "u1", category.New{Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	aOrder, bOrder := a.SortOrder, b.SortOrder

	moved, err := svc.MoveUp(context.Background(), "u1", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.SortOrder != aOrder {
		t.Fatalf("moved.SortOrder = %d, want %d", moved.SortOrder, aOrder)
	}
	gotA, err := svc.Get(context.Background(), "u1", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotA.SortOrder != bOrder {
		t.Fatalf("a.SortOrder after swap = %d, want %d", gotA.SortOrder, bOrder)
	}
}

func TestMoveUpOnFirstSiblingIsNoOp(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), "u1", category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "u1", category.New{Name: "B"}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.MoveUp(context.Background(), "u1", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SortOrder != a.SortOrder {
		t.Fatalf("SortOrder changed on a no-op move-up: got %d, want %d", got.SortOrder, a.SortOrder)
	}
}

func TestMoveDownOnLastSiblingIsNoOp(t *testing.T) {
	svc := newService()
	if _, err := svc.Create(context.Background(), "u1", category.New{Name: "A"}); err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), "u1", category.New{Name: "B"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.MoveDown(context.Background(), "u1", b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SortOrder != b.SortOrder {
		t.Fatalf("SortOrder changed on a no-op move-down: got %d, want %d", got.SortOrder, b.SortOrder)
	}
}

func TestSubtreeIncludesSelfAndDescendants(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), "u1", category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), "u1", category.New{ParentID: &a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := svc.Create(context.Background(), "u1", category.New{ParentID: &b.ID, Name: "C"})
	if err != nil {
		t.Fatal(err)
	}
	ids, err := svc.Subtree(context.Background(), "u1", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{a.ID: true, b.ID: true, c.ID: true}
	if len(ids) != len(want) {
		t.Fatalf("Subtree = %v, want 3 ids", ids)
	}
	for _, id := range ids {
		if !want[id] {
			t.Fatalf("unexpected id %s in subtree", id)
		}
	}
}

func TestUsableRejectsDisabledAndForeignCategories(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), "u1", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := svc.Usable(context.Background(), "u1", c.ID); err != nil || !ok {
		t.Fatalf("Usable(owner, live) = %v, %v, want true, nil", ok, err)
	}
	if ok, err := svc.Usable(context.Background(), "u2", c.ID); err != nil || ok {
		t.Fatalf("Usable(other owner) = %v, %v, want false, nil", ok, err)
	}
	if _, err := svc.Disable(context.Background(), "u1", c.ID); err != nil {
		t.Fatal(err)
	}
	if ok, err := svc.Usable(context.Background(), "u1", c.ID); err != nil || ok {
		t.Fatalf("Usable(disabled) = %v, %v, want false, nil", ok, err)
	}
}
