package postgres

import (
	"context"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/category"
)

func TestPGAccountIconColorRoundTripAndClear(t *testing.T) {
	store, authStore := newAccountStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "iconacc@example.com")

	typ, err := store.CreateType(ctx, owner.ID, "Checking", "")
	if err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := store.Create(ctx, owner.ID, account.New{
		Title: "Main", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening,
		Icon: "wallet", Color: "blue",
	})
	if err != nil {
		t.Fatal(err)
	}
	if acc.Icon != "wallet" || acc.Color != "blue" {
		t.Fatalf("after Create: icon/color = %q/%q, want wallet/blue", acc.Icon, acc.Color)
	}

	got, err := store.Get(ctx, owner.ID, acc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Icon != "wallet" || got.Color != "blue" {
		t.Fatalf("after Get: icon/color = %q/%q, want wallet/blue", got.Icon, got.Color)
	}

	// Clear the icon, leave the color untouched (nil pointer).
	empty := ""
	upd, err := store.Update(ctx, owner.ID, acc.ID, account.Update{Icon: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Icon != "" {
		t.Fatalf("after clearing icon: icon = %q, want empty", upd.Icon)
	}
	if upd.Color != "blue" {
		t.Fatalf("after clearing icon: color = %q, want blue (untouched)", upd.Color)
	}

	// Set a new icon value.
	newIcon := "piggy-bank"
	upd2, err := store.Update(ctx, owner.ID, acc.ID, account.Update{Icon: &newIcon})
	if err != nil {
		t.Fatal(err)
	}
	if upd2.Icon != "piggy-bank" {
		t.Fatalf("after setting icon: icon = %q, want piggy-bank", upd2.Icon)
	}
}

func TestPGCategoryIconColorRoundTripAndClear(t *testing.T) {
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	store := NewCategoryStore(pool)
	authStore := NewAuthStore(pool)
	ctx := context.Background()
	owner := mustUser(t, authStore, "iconcat@example.com")

	c, err := store.Create(ctx, owner.ID, category.New{Name: "Groceries", Icon: "shopping-cart", Color: "orange"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Icon != "shopping-cart" || c.Color != "orange" {
		t.Fatalf("after Create: icon/color = %q/%q", c.Icon, c.Color)
	}

	empty := ""
	upd, err := store.Update(ctx, owner.ID, c.ID, category.Update{Color: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Color != "" {
		t.Fatalf("after clearing color: color = %q, want empty", upd.Color)
	}
	if upd.Icon != "shopping-cart" {
		t.Fatalf("after clearing color: icon = %q, want shopping-cart (untouched)", upd.Icon)
	}
}

func TestPGCategorySeedDefaultsCarryIconAndColor(t *testing.T) {
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	store := NewCategoryStore(pool)
	authStore := NewAuthStore(pool)
	ctx := context.Background()
	owner := mustUser(t, authStore, "seedicon@example.com")

	if err := store.SeedDefaults(ctx, owner.ID); err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	got, err := store.List(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(category.DefaultCategories) {
		t.Fatalf("List after SeedDefaults = %d categories, want %d", len(got), len(category.DefaultCategories))
	}
	for i, c := range got {
		want := category.DefaultCategories[i]
		if c.Name != want.Name || c.Icon != want.Icon || c.Color != want.Color {
			t.Fatalf("seeded category %d = {%q %q %q}, want {%q %q %q}",
				i, c.Name, c.Icon, c.Color, want.Name, want.Icon, want.Color)
		}
	}
}
