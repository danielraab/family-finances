package postgres

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/dashboard"
)

func newDashboardStore(t *testing.T) (*DashboardStore, *AuthStore) {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewDashboardStore(pool), NewAuthStore(pool)
}

func TestPGDashboardCreateGetList(t *testing.T) {
	store, authStore := newDashboardStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "dashowner1@example.com")

	accountID := "acc-1"
	created, err := store.Create(ctx, owner.ID, dashboard.New{
		Type:   dashboard.CardTypeAccountStat,
		Config: dashboard.Config{AccountID: &accountID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.SortOrder != 0 {
		t.Fatalf("sort_order = %d, want 0", created.SortOrder)
	}
	if created.Config.AccountID == nil || *created.Config.AccountID != accountID {
		t.Fatalf("config = %+v, want account_id round-tripped", created.Config)
	}

	got, err := store.Get(ctx, owner.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != dashboard.CardTypeAccountStat {
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

func TestPGDashboardCrossOwnerNotFound(t *testing.T) {
	store, authStore := newDashboardStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "dashowner2@example.com")
	other := mustUser(t, authStore, "dashowner3@example.com")

	created, err := store.Create(ctx, owner.ID, dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, other.ID, created.ID); !errors.Is(err, dashboard.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if _, err := store.Update(ctx, other.ID, created.ID, dashboard.Update{}); !errors.Is(err, dashboard.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if err := store.Delete(ctx, other.ID, created.ID); !errors.Is(err, dashboard.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGDashboardCreateAppendsSortOrder(t *testing.T) {
	store, authStore := newDashboardStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "dashowner4@example.com")

	a, err := store.Create(ctx, owner.ID, dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(ctx, owner.ID, dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	if a.SortOrder != 0 || b.SortOrder != 1 {
		t.Fatalf("sort orders = %d, %d, want 0, 1", a.SortOrder, b.SortOrder)
	}
}

func TestPGDashboardMoveUpSwapsWithPrevious(t *testing.T) {
	store, authStore := newDashboardStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "dashowner5@example.com")

	a, err := store.Create(ctx, owner.ID, dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.Create(ctx, owner.ID, dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.MoveUp(ctx, owner.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	list, err := store.List(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ID != b.ID || list[1].ID != a.ID {
		t.Fatalf("list = %+v, want [b, a]", list)
	}
}

func TestPGDashboardMoveUpFirstIsNoop(t *testing.T) {
	store, authStore := newDashboardStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "dashowner6@example.com")

	a, err := store.Create(ctx, owner.ID, dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.MoveUp(ctx, owner.ID, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.SortOrder != a.SortOrder {
		t.Fatalf("sort_order = %d, want unchanged %d", updated.SortOrder, a.SortOrder)
	}
}

func TestPGDashboardUpdateReplacesConfig(t *testing.T) {
	store, authStore := newDashboardStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "dashowner7@example.com")

	created, err := store.Create(ctx, owner.ID, dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	tagID := "tag-1"
	updated, err := store.Update(ctx, owner.ID, created.ID, dashboard.Update{Config: dashboard.Config{TagID: &tagID}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Config.TagID == nil || *updated.Config.TagID != tagID {
		t.Fatalf("config = %+v, want tag_id round-tripped", updated.Config)
	}
}

func TestPGDashboardDeleteRemovesRow(t *testing.T) {
	store, authStore := newDashboardStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "dashowner8@example.com")

	created, err := store.Create(ctx, owner.ID, dashboard.New{Type: dashboard.CardTypeQueryStat})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, owner.ID, created.ID); err != nil {
		t.Fatal(err)
	}
	list, err := store.List(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("list = %v, want empty", list)
	}
}
