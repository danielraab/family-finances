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

func TestCreateRootCategory(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if c.ParentID != nil || c.Name != "Groceries" {
		t.Fatalf("c = %+v", c)
	}
}

func TestCreateChildCategory(t *testing.T) {
	svc := newService()
	parent, err := svc.Create(context.Background(), category.New{Name: "Expenses"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := svc.Create(context.Background(), category.New{ParentID: &parent.ID, Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatalf("child = %+v", child)
	}
}

func TestCreateRejectsUnknownParent(t *testing.T) {
	svc := newService()
	_, err := svc.Create(context.Background(), category.New{ParentID: ptr("nope"), Name: "X"})
	if !errors.Is(err, category.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	svc := newService()
	_, err := svc.Create(context.Background(), category.New{Name: "  "})
	if !errors.Is(err, category.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestDeleteWithChildrenRejected(t *testing.T) {
	svc := newService()
	parent, err := svc.Create(context.Background(), category.New{Name: "Expenses"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), category.New{ParentID: &parent.ID, Name: "Groceries"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), parent.ID); !errors.Is(err, category.ErrInUse) {
		t.Fatalf("err = %v, want ErrInUse", err)
	}
}

func TestDeleteLeafSucceeds(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), c.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("Get after delete err = %v, want ErrNotFound", err)
	}
}

func TestUpdateSelfParentRejected(t *testing.T) {
	svc := newService()
	c, err := svc.Create(context.Background(), category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Update(context.Background(), c.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: &c.ID},
	})
	if !errors.Is(err, category.ErrCycle) {
		t.Fatalf("err = %v, want ErrCycle", err)
	}
}

func TestUpdateCycleThroughDescendantRejected(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), category.New{ParentID: &a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Update(context.Background(), a.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: &b.ID},
	})
	if !errors.Is(err, category.ErrCycle) {
		t.Fatalf("err = %v, want ErrCycle", err)
	}
}

func TestUpdateToRootClearsParent(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), category.New{ParentID: &a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.Update(context.Background(), b.ID, category.Update{
		ParentID: category.OptionalID{Set: true, Value: nil},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ParentID != nil {
		t.Fatalf("ParentID = %v, want nil", got.ParentID)
	}
}

func TestSubtreeIncludesSelfAndDescendants(t *testing.T) {
	svc := newService()
	a, err := svc.Create(context.Background(), category.New{Name: "A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Create(context.Background(), category.New{ParentID: &a.ID, Name: "B"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := svc.Create(context.Background(), category.New{ParentID: &b.ID, Name: "C"})
	if err != nil {
		t.Fatal(err)
	}
	ids, err := svc.Subtree(context.Background(), a.ID)
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
