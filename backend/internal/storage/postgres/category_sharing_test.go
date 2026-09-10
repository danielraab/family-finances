package postgres

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/entry"
)

func TestPGCategoryShareCreateListResolvesRealNames(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "catshareowner@example.com")
	member := mustUser(t, authStore, "catsharemember@example.com")

	cat, err := store.Create(ctx, owner.ID, category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.CreateOrUpdateShare(ctx, cat.ID, member.ID, category.PermissionAppend, owner.ID); err != nil {
		t.Fatalf("CreateOrUpdateShare: %v", err)
	}

	shares, err := store.ListShares(ctx, cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 {
		t.Fatalf("shares = %+v, want 1", shares)
	}
	sh := shares[0]
	if sh.UserID != member.ID || sh.Email != "catsharemember@example.com" || sh.Permission != category.PermissionAppend {
		t.Fatalf("share = %+v", sh)
	}
	if sh.GrantedBy != owner.ID || sh.GrantedByName != "catshareowner@example.com" {
		t.Fatalf("share granted_by = %+v", sh)
	}

	got, err := store.GetForCaller(ctx, member.ID, cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Permission != category.PermissionAppend || !got.Shared || got.OwnerName != "catshareowner@example.com" {
		t.Fatalf("GetForCaller(member) = %+v", got)
	}
}

func TestPGCategoryShareUpsertUpdatesPermissionInPlace(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "catshareowner2@example.com")
	member := mustUser(t, authStore, "catsharemember2@example.com")

	cat, err := store.Create(ctx, owner.ID, category.New{Name: "Groceries2"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.CreateOrUpdateShare(ctx, cat.ID, member.ID, category.PermissionView, owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, cat.ID, member.ID, category.PermissionAppend, owner.ID); err != nil {
		t.Fatal(err)
	}

	shares, err := store.ListShares(ctx, cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 || shares[0].Permission != category.PermissionAppend {
		t.Fatalf("shares = %+v, want exactly one at append", shares)
	}
}

func TestPGCategoryShareDeleteAndListVisibility(t *testing.T) {
	store, authStore := newCategoryStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "catshareowner3@example.com")
	member := mustUser(t, authStore, "catsharemember3@example.com")

	cat, err := store.Create(ctx, owner.ID, category.New{Name: "Groceries3"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, cat.ID, member.ID, category.PermissionView, owner.ID); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(ctx, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !list[0].Shared {
		t.Fatalf("List(member) = %+v, want the shared category", list)
	}

	if err := store.DeleteShare(ctx, cat.ID, member.ID); err != nil {
		t.Fatalf("DeleteShare: %v", err)
	}
	list2, err := store.List(ctx, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != 0 {
		t.Fatalf("List(member) after delete = %+v, want empty", list2)
	}

	if err := store.DeleteShare(ctx, cat.ID, member.ID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("DeleteShare (already gone): err = %v, want ErrNotFound", err)
	}
}

// TestPGCategoryShareEntryCountScopedPerViewer verifies that once a
// category is shared at append tier and both the real owner and the
// recipient categorize their own entries under it, each side's own
// GetForCaller/List reports entry_count for only their own entries — not
// a combined total. This is the behavior that only matters once sharing
// lets a non-owner categorize entries under someone else's category.
func TestPGCategoryShareEntryCountScopedPerViewer(t *testing.T) {
	store, authStore := newCategoryStore(t)
	accStore := NewAccountStore(store.pool)
	entryStore := NewEntryStore(store.pool)
	ctx := context.Background()

	owner := mustUser(t, authStore, "catshareowner4@example.com")
	member := mustUser(t, authStore, "catsharemember4@example.com")

	opening, _ := account.ParseDate("2024-01-01")
	acc, err := accStore.Create(ctx, owner.ID, account.New{
		Title: "Shared", Type: "Checking", Currency: "EUR", OpeningDate: opening,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := accStore.CreateOrUpdateShare(ctx, acc.ID, member.ID, account.PermissionAppend, owner.ID); err != nil {
		t.Fatal(err)
	}

	cat, err := store.Create(ctx, owner.ID, category.New{Name: "Groceries4"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, cat.ID, member.ID, category.PermissionAppend, owner.ID); err != nil {
		t.Fatal(err)
	}

	bookedAt := at("2024-01-01T00:00:00Z")
	mkEntry := func(createdBy string) {
		if _, err := entryStore.Create(ctx, createdBy, entry.New{
			AccountID: acc.ID, Kind: entry.KindTransaction, Amount: ptrInt64(100),
			BookingTimestamp: bookedAt, Title: "x", CategoryID: &cat.ID,
		}); err != nil {
			t.Fatalf("create entry: %v", err)
		}
	}
	mkEntry(owner.ID)
	mkEntry(owner.ID)
	mkEntry(member.ID)

	ownerView, err := store.GetForCaller(ctx, owner.ID, cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ownerView.EntryCount != 2 {
		t.Fatalf("owner EntryCount = %d, want 2", ownerView.EntryCount)
	}

	memberView, err := store.GetForCaller(ctx, member.ID, cat.ID)
	if err != nil {
		t.Fatal(err)
	}
	if memberView.EntryCount != 1 {
		t.Fatalf("member EntryCount = %d, want 1", memberView.EntryCount)
	}
}
