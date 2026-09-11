package entry_test

import (
	"context"
	"testing"

	"at.draab/familyfinances/internal/entry"
)

// TestCategoryShareSurfacesEntriesAcrossAccountsForRecipient covers task
// 3.1: a category shared (view or append — the stub doesn't distinguish,
// see stubCategories.share) with a user who has no account-level access at
// all to its owner's account still surfaces that category's entries when
// the recipient filters by it.
func TestCategoryShareSurfacesEntriesAcrossAccountsForRecipient(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc-owner", "owner", "EUR")
	categories.addOwnedBy("cat1", "owner")
	categories.share("cat1", "recipient")

	if _, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc-owner", Kind: entry.KindTransaction, Amount: ptr(int64(-1200)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Groceries", CategoryID: ptr("cat1"),
	}); err != nil {
		t.Fatal(err)
	}

	// Sanity: recipient has zero account access, so an unfiltered list
	// (or Get) sees nothing at all.
	if visible, err := accounts.VisibleIDs(context.Background(), "recipient"); err != nil || len(visible) != 0 {
		t.Fatalf("fixture sanity: recipient should have no visible accounts, got %v (err %v)", visible, err)
	}

	items, _, err := svc.List(context.Background(), "recipient", entry.Filter{CategoryID: ptr("cat1")})
	if err != nil {
		t.Fatalf("recipient List filtered by shared category: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Groceries" {
		t.Fatalf("items = %+v, want the owner's single entry", items)
	}
}

// TestCategoryOwnerSeesEntriesOnRecipientsAccount covers task 3.2: the
// symmetric direction — a category's real owner filtering by their own
// category sees an entry an append-tier share recipient created against an
// account the owner has no access to.
func TestCategoryOwnerSeesEntriesOnRecipientsAccount(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc-recipient", "recipient", "EUR")
	categories.addOwnedBy("cat1", "owner")
	categories.share("cat1", "recipient")

	if _, err := svc.Create(context.Background(), "recipient", entry.New{
		AccountID: "acc-recipient", Kind: entry.KindTransaction, Amount: ptr(int64(-500)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Coffee", CategoryID: ptr("cat1"),
	}); err != nil {
		t.Fatal(err)
	}

	items, _, err := svc.List(context.Background(), "owner", entry.Filter{CategoryID: ptr("cat1")})
	if err != nil {
		t.Fatalf("owner List filtered by own category: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Coffee" {
		t.Fatalf("items = %+v, want the recipient's single entry", items)
	}
}

// TestCategoryShareUnfilteredListingStaysAccountScoped covers task 3.3:
// neither side of a category share sees the other's entries without an
// explicit category filter naming it.
func TestCategoryShareUnfilteredListingStaysAccountScoped(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc-owner", "owner", "EUR")
	accounts.add("acc-recipient", "recipient", "EUR")
	categories.addOwnedBy("cat1", "owner")
	categories.share("cat1", "recipient")

	if _, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc-owner", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "owner's", CategoryID: ptr("cat1"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "recipient", entry.New{
		AccountID: "acc-recipient", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "recipient's", CategoryID: ptr("cat1"),
	}); err != nil {
		t.Fatal(err)
	}

	recipientItems, _, err := svc.List(context.Background(), "recipient", entry.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(recipientItems) != 1 || recipientItems[0].Title != "recipient's" {
		t.Fatalf("recipient's unfiltered list = %+v, want only their own entry", recipientItems)
	}

	ownerItems, _, err := svc.List(context.Background(), "owner", entry.Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerItems) != 1 || ownerItems[0].Title != "owner's" {
		t.Fatalf("owner's unfiltered list = %+v, want only their own entry", ownerItems)
	}
}

// TestCategoryFilterWithExplicitAccountStaysAccountScoped covers task 3.4:
// naming a specific account alongside the shared category filter excludes
// an entry under that category on a different, inaccessible account.
func TestCategoryFilterWithExplicitAccountStaysAccountScoped(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc-owner", "owner", "EUR")
	accounts.add("acc-recipient", "recipient", "EUR")
	categories.addOwnedBy("cat1", "owner")
	categories.share("cat1", "recipient")

	if _, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc-owner", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "owner's", CategoryID: ptr("cat1"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "recipient", entry.New{
		AccountID: "acc-recipient", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "recipient's", CategoryID: ptr("cat1"),
	}); err != nil {
		t.Fatal(err)
	}

	items, _, err := svc.List(context.Background(), "recipient", entry.Filter{
		CategoryID: ptr("cat1"), AccountIDs: []string{"acc-recipient"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "recipient's" {
		t.Fatalf("items = %+v, want only the entry on the explicitly named account", items)
	}
}

// TestCategoryWithNoPermissionNeverWidens covers task 3.5: a category_id
// filter for a category the caller has no ownership or share on never
// surfaces another user's entries, in either CategoryMode.
func TestCategoryWithNoPermissionNeverWidens(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc-stranger", "stranger", "EUR")
	accounts.add("acc-caller", "caller", "EUR")
	categories.addOwnedBy("cat-stranger", "stranger")
	categories.addOwnedBy("cat-caller", "caller")
	// deliberately no categories.share("cat-stranger", "caller")

	if _, err := svc.Create(context.Background(), "stranger", entry.New{
		AccountID: "acc-stranger", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "stranger's", CategoryID: ptr("cat-stranger"),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "caller", entry.New{
		AccountID: "acc-caller", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "caller's", CategoryID: ptr("cat-caller"),
	}); err != nil {
		t.Fatal(err)
	}

	for _, mode := range []entry.CategoryMode{entry.ModeSubtree, entry.ModeExact} {
		items, _, err := svc.List(context.Background(), "caller", entry.Filter{
			CategoryID: ptr("cat-stranger"), CategoryMode: mode,
		})
		if err != nil {
			t.Fatalf("mode %q: %v", mode, err)
		}
		for _, it := range items {
			if it.Title == "stranger's" {
				t.Fatalf("mode %q: stranger's entry leaked to an unpermitted caller: %+v", mode, items)
			}
		}
	}
}

// TestCategoryShareWidensReportTotals covers task 3.6: Sum's per-currency
// report totals include a widened cross-account entry — a regression guard
// on AccountLookup.Access's already-permission-blind currency lookup (see
// design.md), not new production behavior.
func TestCategoryShareWidensReportTotals(t *testing.T) {
	svc, accounts, categories, _ := newFixture()
	accounts.add("acc-owner", "owner", "USD")
	categories.addOwnedBy("cat1", "owner")
	categories.share("cat1", "recipient")

	if _, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc-owner", Kind: entry.KindTransaction, Amount: ptr(int64(-2500)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Groceries", CategoryID: ptr("cat1"),
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := svc.Sum(context.Background(), "recipient", entry.Filter{CategoryID: ptr("cat1")})
	if err != nil {
		t.Fatalf("recipient Sum filtered by shared category: %v", err)
	}
	if summary.Count != 1 {
		t.Fatalf("Count = %d, want 1", summary.Count)
	}
	if len(summary.Sums) != 1 || summary.Sums[0].Currency != "USD" || summary.Sums[0].Amount != -2500 {
		t.Fatalf("Sums = %+v, want a single -2500 USD total", summary.Sums)
	}
}
