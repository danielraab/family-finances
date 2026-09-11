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

func TestPGTagShareCreateListResolvesRealNames(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagshareowner@example.com")
	member := mustUser(t, authStore, "tagsharemember@example.com")

	tg, err := store.Create(ctx, owner.ID, "Groceries")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.CreateOrUpdateShare(ctx, tg.ID, member.ID, tag.PermissionAppend, owner.ID); err != nil {
		t.Fatalf("CreateOrUpdateShare: %v", err)
	}

	shares, err := store.ListShares(ctx, tg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 {
		t.Fatalf("shares = %+v, want 1", shares)
	}
	sh := shares[0]
	if sh.UserID != member.ID || sh.Email != "tagsharemember@example.com" || sh.Permission != tag.PermissionAppend {
		t.Fatalf("share = %+v", sh)
	}
	if sh.GrantedBy != owner.ID || sh.GrantedByName != "tagshareowner@example.com" {
		t.Fatalf("share granted_by = %+v", sh)
	}

	got, err := store.GetForCaller(ctx, member.ID, tg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Permission != tag.PermissionAppend || !got.Shared || got.OwnerName != "tagshareowner@example.com" {
		t.Fatalf("GetForCaller(member) = %+v", got)
	}
}

func TestPGTagShareUpsertUpdatesPermissionInPlace(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagshareowner2@example.com")
	member := mustUser(t, authStore, "tagsharemember2@example.com")

	tg, err := store.Create(ctx, owner.ID, "Groceries2")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.CreateOrUpdateShare(ctx, tg.ID, member.ID, tag.PermissionView, owner.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, tg.ID, member.ID, tag.PermissionAppend, owner.ID); err != nil {
		t.Fatal(err)
	}

	shares, err := store.ListShares(ctx, tg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 || shares[0].Permission != tag.PermissionAppend {
		t.Fatalf("shares = %+v, want exactly one at append", shares)
	}
}

func TestPGTagShareDeleteAndListVisibility(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagshareowner3@example.com")
	member := mustUser(t, authStore, "tagsharemember3@example.com")

	tg, err := store.Create(ctx, owner.ID, "Groceries3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, tg.ID, member.ID, tag.PermissionView, owner.ID); err != nil {
		t.Fatal(err)
	}

	list, err := store.List(ctx, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || !list[0].Shared {
		t.Fatalf("List(member) = %+v, want the shared tag", list)
	}

	if err := store.DeleteShare(ctx, tg.ID, member.ID); err != nil {
		t.Fatalf("DeleteShare: %v", err)
	}
	list2, err := store.List(ctx, member.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != 0 {
		t.Fatalf("List(member) after delete = %+v, want empty", list2)
	}

	if err := store.DeleteShare(ctx, tg.ID, member.ID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("DeleteShare (already gone): err = %v, want ErrNotFound", err)
	}
}

// TestPGTagShareEntryCountScopedPerViewer verifies that once a tag is
// shared at append tier and both the real owner and the recipient tag
// their own entries with it, each side's own GetForCaller/List reports
// entry_count for only their own entries — not a combined total. Mirrors
// category_sharing_test.go's equivalent.
func TestPGTagShareEntryCountScopedPerViewer(t *testing.T) {
	store, authStore := newTagStore(t)
	accStore := NewAccountStore(store.pool)
	entryStore := NewEntryStore(store.pool)
	ctx := context.Background()

	owner := mustUser(t, authStore, "tagshareowner4@example.com")
	member := mustUser(t, authStore, "tagsharemember4@example.com")

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

	tg, err := store.Create(ctx, owner.ID, "Groceries4")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, tg.ID, member.ID, tag.PermissionAppend, owner.ID); err != nil {
		t.Fatal(err)
	}

	catStore := NewCategoryStore(store.pool)
	cat, err := catStore.Create(ctx, owner.ID, category.New{Name: "Groceries4Cat"})
	if err != nil {
		t.Fatal(err)
	}

	bookedAt := at("2024-01-01T00:00:00Z")
	mkEntry := func(createdBy string) {
		if _, err := entryStore.Create(ctx, createdBy, entry.New{
			AccountID: acc.ID, Kind: entry.KindTransaction, Amount: ptrInt64(100),
			BookingTimestamp: bookedAt, Title: "x", CategoryID: &cat.ID, TagIDs: []string{tg.ID},
		}); err != nil {
			t.Fatalf("create entry: %v", err)
		}
	}
	mkEntry(owner.ID)
	mkEntry(owner.ID)
	mkEntry(member.ID)

	ownerView, err := store.GetForCaller(ctx, owner.ID, tg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ownerView.EntryCount != 2 {
		t.Fatalf("owner EntryCount = %d, want 2", ownerView.EntryCount)
	}

	memberView, err := store.GetForCaller(ctx, member.ID, tg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if memberView.EntryCount != 1 {
		t.Fatalf("member EntryCount = %d, want 1", memberView.EntryCount)
	}
}

func TestPGTagDeleteBlockedWhileShared(t *testing.T) {
	store, authStore := newTagStore(t)
	ctx := context.Background()
	owner := mustUser(t, authStore, "tagshareowner5@example.com")
	member := mustUser(t, authStore, "tagsharemember5@example.com")

	tg, err := store.Create(ctx, owner.ID, "Groceries5")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateOrUpdateShare(ctx, tg.ID, member.ID, tag.PermissionView, owner.ID); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(ctx, owner.ID, tg.ID); !errors.Is(err, tag.ErrInUse) {
		t.Fatalf("Delete while shared: err = %v, want ErrInUse", err)
	}

	if err := store.DeleteShare(ctx, tg.ID, member.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, owner.ID, tg.ID); err != nil {
		t.Fatalf("Delete after unshare: %v", err)
	}
}
