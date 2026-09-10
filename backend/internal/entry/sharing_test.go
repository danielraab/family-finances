package entry_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/entry"
)

func TestAppendTierCreatesAndEditsOnlyOwnEntries(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "owner", "EUR")
	accounts.share("acc1", "appender", "append")
	categories.addOwnedBy("cat-appender", "appender")

	e, err := svc.Create(context.Background(), "appender", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(-500)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Groceries", CategoryID: ptr("cat-appender"),
	})
	if err != nil {
		t.Fatalf("append-tier Create: %v", err)
	}
	if e.CreatedBy != "appender" {
		t.Fatalf("CreatedBy = %q, want appender (not the account's real owner)", e.CreatedBy)
	}

	// append can edit their own entry.
	newTitle := "Groceries (edited)"
	if _, err := svc.Update(context.Background(), "appender", e.ID, entry.Update{Title: &newTitle}); err != nil {
		t.Fatalf("append editing own entry: %v", err)
	}

	// A different append-tier user (or the owner themselves, also just
	// append-equivalent for this check since they didn't create it) cannot
	// edit or delete it.
	categories.addOwnedBy("cat-owner", "owner")
	ownerEntry, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(-100)),
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "Owner's own", CategoryID: ptr("cat-owner"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Update(context.Background(), "appender", ownerEntry.ID, entry.Update{Title: &newTitle}); !errors.Is(err, entry.ErrForbidden) {
		t.Fatalf("append editing another user's entry: err = %v, want ErrForbidden", err)
	}
	if err := svc.Delete(context.Background(), "appender", ownerEntry.ID); !errors.Is(err, entry.ErrForbidden) {
		t.Fatalf("append deleting another user's entry: err = %v, want ErrForbidden", err)
	}
}

func TestEntryAdminEditsAnyEntryOnTheAccount(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "owner", "EUR")
	accounts.share("acc1", "admin", "entry_admin")
	categories.addOwnedBy("cat-owner", "owner")

	e, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(-100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Owner's entry", CategoryID: ptr("cat-owner"),
	})
	if err != nil {
		t.Fatal(err)
	}

	newTitle := "Edited by admin"
	if _, err := svc.Update(context.Background(), "admin", e.ID, entry.Update{Title: &newTitle}); err != nil {
		t.Fatalf("entry_admin editing owner's entry: %v", err)
	}
	if err := svc.Delete(context.Background(), "admin", e.ID); err != nil {
		t.Fatalf("entry_admin deleting owner's entry: %v", err)
	}
}

func TestViewTierCanReadButNotWrite(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "owner", "EUR")
	accounts.share("acc1", "viewer", "view")
	categories.addOwnedBy("cat-owner", "owner")

	e, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(-100)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "x", CategoryID: ptr("cat-owner"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Get(context.Background(), "viewer", e.ID); err != nil {
		t.Fatalf("view-tier Get: %v", err)
	}
	items, _, err := svc.List(context.Background(), "viewer", entry.Filter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("view-tier List: items=%v err=%v", items, err)
	}

	if _, err := svc.Create(context.Background(), "viewer", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-02T00:00:00Z"), Title: "y", CategoryID: ptr("cat-owner"),
	}); !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("view-tier Create: err = %v, want ErrInvalidValue", err)
	}
	newTitle := "hijacked"
	if _, err := svc.Update(context.Background(), "viewer", e.ID, entry.Update{Title: &newTitle}); !errors.Is(err, entry.ErrForbidden) {
		t.Fatalf("view-tier Update: err = %v, want ErrForbidden", err)
	}
	if err := svc.Delete(context.Background(), "viewer", e.ID); !errors.Is(err, entry.ErrForbidden) {
		t.Fatalf("view-tier Delete: err = %v, want ErrForbidden", err)
	}
}

func TestRevokedUserLosesAllAccessIncludingOwnEntries(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "owner", "EUR")
	accounts.share("acc1", "member", "append")
	categories.addOwnedBy("cat-member", "member")

	e, err := svc.Create(context.Background(), "member", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(-50)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Snacks", CategoryID: ptr("cat-member"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Revoke the share entirely.
	delete(accounts.shares["acc1"], "member")

	if _, err := svc.Get(context.Background(), "member", e.ID); !errors.Is(err, entry.ErrNotFound) {
		t.Fatalf("Get after revoke: err = %v, want ErrNotFound", err)
	}
	// The owner (unaffected) still sees it.
	if _, err := svc.Get(context.Background(), "owner", e.ID); err != nil {
		t.Fatalf("owner Get after member's revoke: %v", err)
	}
}

func TestNoPermissionAtAllCannotCreateOnAccount(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc1", "owner", "EUR")
	categories.addOwnedBy("cat-stranger", "stranger")

	if _, err := svc.Create(context.Background(), "stranger", entry.New{
		AccountID: "acc1", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "x", CategoryID: ptr("cat-stranger"),
	}); !errors.Is(err, entry.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}
