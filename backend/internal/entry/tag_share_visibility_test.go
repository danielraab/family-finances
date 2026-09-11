package entry_test

import (
	"context"
	"testing"

	"at.draab/familyfinances/internal/entry"
)

// TestTagShareSurfacesEntriesAcrossAccountsForRecipient mirrors
// TestCategoryShareSurfacesEntriesAcrossAccountsForRecipient: a tag shared
// (view or append — the stub doesn't distinguish, see stubTags.share) with a
// user who has no account-level access at all to its owner's account still
// surfaces that tag's entries when the recipient filters by it.
func TestTagShareSurfacesEntriesAcrossAccountsForRecipient(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc-owner", "owner", "EUR")
	categories.addOwnedBy("cat1", "owner")
	tags.add("tag1", "owner")
	tags.share("tag1", "recipient")

	if _, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc-owner", Kind: entry.KindTransaction, Amount: ptr(int64(-1200)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Groceries", CategoryID: ptr("cat1"), TagIDs: []string{"tag1"},
	}); err != nil {
		t.Fatal(err)
	}

	// Sanity: recipient has zero account access, so an unfiltered list
	// (or Get) sees nothing at all.
	if visible, err := accounts.VisibleIDs(context.Background(), "recipient"); err != nil || len(visible) != 0 {
		t.Fatalf("fixture sanity: recipient should have no visible accounts, got %v (err %v)", visible, err)
	}

	items, _, err := svc.List(context.Background(), "recipient", entry.Filter{TagID: ptr("tag1")})
	if err != nil {
		t.Fatalf("recipient List filtered by shared tag: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Groceries" {
		t.Fatalf("items = %+v, want the owner's single entry", items)
	}
}

// TestTagOwnerSeesEntriesOnRecipientsAccount mirrors
// TestCategoryOwnerSeesEntriesOnRecipientsAccount: the symmetric direction —
// a tag's real owner filtering by their own tag sees an entry an
// append-tier share recipient created against an account the owner has no
// access to.
func TestTagOwnerSeesEntriesOnRecipientsAccount(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc-recipient", "recipient", "EUR")
	categories.addOwnedBy("cat1", "recipient")
	tags.add("tag1", "owner")
	tags.share("tag1", "recipient")

	if _, err := svc.Create(context.Background(), "recipient", entry.New{
		AccountID: "acc-recipient", Kind: entry.KindTransaction, Amount: ptr(int64(-500)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Coffee", CategoryID: ptr("cat1"), TagIDs: []string{"tag1"},
	}); err != nil {
		t.Fatal(err)
	}

	items, _, err := svc.List(context.Background(), "owner", entry.Filter{TagID: ptr("tag1")})
	if err != nil {
		t.Fatalf("owner List filtered by own tag: %v", err)
	}
	if len(items) != 1 || items[0].Title != "Coffee" {
		t.Fatalf("items = %+v, want the recipient's single entry", items)
	}
}

// TestTagShareUnfilteredListingStaysAccountScoped mirrors
// TestCategoryShareUnfilteredListingStaysAccountScoped: neither side of a
// tag share sees the other's entries without an explicit tag filter naming
// it.
func TestTagShareUnfilteredListingStaysAccountScoped(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc-owner", "owner", "EUR")
	accounts.add("acc-recipient", "recipient", "EUR")
	categories.addOwnedBy("cat-owner", "owner")
	categories.addOwnedBy("cat-recipient", "recipient")
	tags.add("tag1", "owner")
	tags.share("tag1", "recipient")

	if _, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc-owner", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "owner's", CategoryID: ptr("cat-owner"), TagIDs: []string{"tag1"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "recipient", entry.New{
		AccountID: "acc-recipient", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "recipient's", CategoryID: ptr("cat-recipient"),
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

// TestTagFilterWithExplicitAccountStaysAccountScoped mirrors
// TestCategoryFilterWithExplicitAccountStaysAccountScoped: naming a
// specific account alongside the shared tag filter excludes an entry
// carrying that tag on a different, inaccessible account.
func TestTagFilterWithExplicitAccountStaysAccountScoped(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc-owner", "owner", "EUR")
	accounts.add("acc-recipient", "recipient", "EUR")
	categories.addOwnedBy("cat-owner", "owner")
	categories.addOwnedBy("cat-recipient", "recipient")
	tags.add("tag1", "owner")
	tags.share("tag1", "recipient")

	if _, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc-owner", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "owner's", CategoryID: ptr("cat-owner"), TagIDs: []string{"tag1"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "recipient", entry.New{
		AccountID: "acc-recipient", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "recipient's", CategoryID: ptr("cat-recipient"), TagIDs: []string{"tag1"},
	}); err != nil {
		t.Fatal(err)
	}

	items, _, err := svc.List(context.Background(), "recipient", entry.Filter{
		TagID: ptr("tag1"), AccountIDs: []string{"acc-recipient"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "recipient's" {
		t.Fatalf("items = %+v, want only the entry on the explicitly named account", items)
	}
}

// TestTagWithNoPermissionNeverWidens mirrors
// TestCategoryWithNoPermissionNeverWidens: a tag_id filter for a tag the
// caller has no ownership or share on never surfaces another user's
// entries.
func TestTagWithNoPermissionNeverWidens(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc-stranger", "stranger", "EUR")
	accounts.add("acc-caller", "caller", "EUR")
	categories.addOwnedBy("cat-stranger", "stranger")
	categories.addOwnedBy("cat-caller", "caller")
	tags.add("tag-stranger", "stranger")
	tags.add("tag-caller", "caller")
	// deliberately no tags.share("tag-stranger", "caller")

	if _, err := svc.Create(context.Background(), "stranger", entry.New{
		AccountID: "acc-stranger", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "stranger's", CategoryID: ptr("cat-stranger"), TagIDs: []string{"tag-stranger"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), "caller", entry.New{
		AccountID: "acc-caller", Kind: entry.KindTransaction, Amount: ptr(int64(-1)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "caller's", CategoryID: ptr("cat-caller"), TagIDs: []string{"tag-caller"},
	}); err != nil {
		t.Fatal(err)
	}

	items, _, err := svc.List(context.Background(), "caller", entry.Filter{TagID: ptr("tag-stranger")})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.Title == "stranger's" {
			t.Fatalf("stranger's entry leaked to an unpermitted caller: %+v", items)
		}
	}
}

// TestTagShareWidensReportTotals mirrors TestCategoryShareWidensReportTotals:
// Sum's per-currency report totals include a widened cross-account entry
// when filtering by a shared tag.
func TestTagShareWidensReportTotals(t *testing.T) {
	svc, accounts, categories, tags := newFixture()
	accounts.add("acc-owner", "owner", "USD")
	categories.addOwnedBy("cat1", "owner")
	tags.add("tag1", "owner")
	tags.share("tag1", "recipient")

	if _, err := svc.Create(context.Background(), "owner", entry.New{
		AccountID: "acc-owner", Kind: entry.KindTransaction, Amount: ptr(int64(-2500)),
		BookingTimestamp: at("2024-01-01T00:00:00Z"), Title: "Groceries", CategoryID: ptr("cat1"), TagIDs: []string{"tag1"},
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := svc.Sum(context.Background(), "recipient", entry.Filter{TagID: ptr("tag1")})
	if err != nil {
		t.Fatalf("recipient Sum filtered by shared tag: %v", err)
	}
	if summary.Count != 1 {
		t.Fatalf("Count = %d, want 1", summary.Count)
	}
	if len(summary.Sums) != 1 || summary.Sums[0].Currency != "USD" || summary.Sums[0].Amount != -2500 {
		t.Fatalf("Sums = %+v, want a single -2500 USD total", summary.Sums)
	}
}
