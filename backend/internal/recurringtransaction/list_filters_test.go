package recurringtransaction_test

import (
	"context"
	"errors"
	"testing"

	rt "at.draab/familyfinances/internal/recurringtransaction"
)

// The listing and the summary resolve their category/tag filters through
// the same Service.resolveFilter the preview endpoint uses, so these
// mirror TestPreviewFiltersByAccountCategoryAndTag one endpoint down.

// listFixture builds three templates on one visible account:
//
//	Rent      — cat_parent, no tags
//	Internet  — cat_child (a descendant of cat_parent), tagged "tag1"
//	Gym       — cat_other, tagged "tag1"
//
// Every one carries a category, since a non-self_transfer template must
// (see validateNew) — the uncategorized case is its own test below.
func listFixture(t *testing.T) *rt.Service {
	t.Helper()
	svc, accounts, categories, tags, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	categories.add("cat_parent")
	categories.addChild("cat_parent", "cat_child")
	categories.add("cat_other")
	tags.add("tag1", "u1")

	create := func(title, categoryID string, tagIDs ...string) {
		t.Helper()
		in := baseNew("acc1", categoryID)
		in.Title = title
		in.TagIDs = tagIDs
		if _, err := svc.Create(context.Background(), "u1", in); err != nil {
			t.Fatal(err)
		}
	}
	create("Rent", "cat_parent")
	create("Internet", "cat_child", "tag1")
	create("Gym", "cat_other", "tag1")

	return svc
}

// titles is the set of titles in a listing, for order-independent asserts.
func titles(items []rt.RecurringTransaction) map[string]bool {
	out := map[string]bool{}
	for _, item := range items {
		out[item.Title] = true
	}
	return out
}

func wantTitles(t *testing.T, items []rt.RecurringTransaction, want ...string) {
	t.Helper()
	got := titles(items)
	if len(got) != len(want) {
		t.Fatalf("listed %v, want exactly %v", got, want)
	}
	for _, title := range want {
		if !got[title] {
			t.Fatalf("listed %v, want exactly %v", got, want)
		}
	}
}

func TestListFilterByCategoryIncludesDescendants(t *testing.T) {
	svc := listFixture(t)
	parent := "cat_parent"

	items, err := svc.List(context.Background(), "u1", rt.Filter{CategoryID: &parent})
	if err != nil {
		t.Fatal(err)
	}
	wantTitles(t, items, "Rent", "Internet")
}

func TestListFilterByCategoryExactExcludesDescendants(t *testing.T) {
	svc := listFixture(t)
	parent := "cat_parent"

	items, err := svc.List(context.Background(), "u1", rt.Filter{
		CategoryID:   &parent,
		CategoryMode: rt.CategoryModeExact,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantTitles(t, items, "Rent")
}

// A self_transfer template is the one kind that may carry no category at
// all (see validateNew), so it is what a "never matches a category filter"
// assertion has to be written against.
func TestListCategoryFilterNeverMatchesAnUncategorizedTemplate(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u1", "EUR")
	categories.add("cat1")

	toAccount := "acc2"
	transfer := rt.New{
		AccountID:     "acc1",
		ToAccountID:   &toAccount,
		Kind:          rt.KindSelfTransfer,
		Title:         "To savings",
		Amount:        -50000,
		IntervalUnit:  rt.UnitMonth,
		IntervalCount: 1,
		StartsOn:      mustDateT(2026, 1, 1),
	}
	if _, err := svc.Create(context.Background(), "u1", transfer); err != nil {
		t.Fatal(err)
	}

	// Without a category filter the transfer is listed…
	items, err := svc.List(context.Background(), "u1", rt.Filter{SelfTransfers: rt.SelfTransferNative})
	if err != nil {
		t.Fatal(err)
	}
	wantTitles(t, items, "To savings")

	// …and with one it is not, having no category to match.
	cat1 := "cat1"
	items, err = svc.List(context.Background(), "u1", rt.Filter{
		SelfTransfers: rt.SelfTransferNative,
		CategoryID:    &cat1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("listed %v, want nothing — the template has no category", titles(items))
	}
}

// A category the caller holds no permission on resolves to an empty
// subtree. That has to match nothing; the bug it guards against is the
// empty set reading as "no filter" and returning the caller's whole list.
func TestListCategoryFilterOnAnInvisibleCategoryMatchesNothing(t *testing.T) {
	svc := listFixture(t)
	stranger := "cat_someone_elses"

	items, err := svc.List(context.Background(), "u1", rt.Filter{CategoryID: &stranger})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("listed %v, want nothing — the caller cannot see that category", titles(items))
	}
}

func TestListFilterByTag(t *testing.T) {
	svc := listFixture(t)
	tag := "tag1"

	items, err := svc.List(context.Background(), "u1", rt.Filter{TagID: &tag})
	if err != nil {
		t.Fatal(err)
	}
	wantTitles(t, items, "Internet", "Gym")
}

func TestListCategoryAndTagFiltersCombine(t *testing.T) {
	svc := listFixture(t)
	parent, tag := "cat_parent", "tag1"

	items, err := svc.List(context.Background(), "u1", rt.Filter{CategoryID: &parent, TagID: &tag})
	if err != nil {
		t.Fatal(err)
	}
	wantTitles(t, items, "Internet")
}

func TestListAndSummaryRejectAnInvalidCategoryMode(t *testing.T) {
	svc := listFixture(t)
	parent := "cat_parent"
	f := rt.Filter{CategoryID: &parent, CategoryMode: rt.CategoryMode("ancestors")}

	if _, err := svc.List(context.Background(), "u1", f); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("List error = %v, want ErrInvalidValue", err)
	}
	if _, err := svc.Summary(context.Background(), "u1", f); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("Summary error = %v, want ErrInvalidValue", err)
	}
}

// The summary's contract is that it totals exactly the rows the listing
// returns under the same filter — so assert it against those rows, not
// against a hardcoded figure.
func TestSummaryTotalsExactlyTheFilteredRows(t *testing.T) {
	svc := listFixture(t)
	parent := "cat_parent"
	f := rt.Filter{CategoryID: &parent}

	items, err := svc.List(context.Background(), "u1", f)
	if err != nil {
		t.Fatal(err)
	}
	var want int64
	for _, item := range items {
		want += item.PerYearAmount
	}

	sum, err := svc.Summary(context.Background(), "u1", f)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Count != len(items) {
		t.Fatalf("summary count = %d, want %d (the listed rows)", sum.Count, len(items))
	}
	if len(sum.Sums) != 1 || sum.Sums[0].Currency != "EUR" {
		t.Fatalf("summary sums = %+v, want one EUR total", sum.Sums)
	}
	if sum.Sums[0].Amount != want {
		t.Fatalf("summary total = %d, want %d (the listed rows' per-year amounts)", sum.Sums[0].Amount, want)
	}
}

// Unlike internal/entry, a category the caller owns never widens the
// account scope here — see Filter's doc comment.
func TestListCategoryFilterDoesNotReachBeyondVisibleAccounts(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("acc1", "u1", "EUR")
	accounts.add("acc2", "u2", "EUR")
	categories.add("cat1")

	theirs := baseNew("acc2", "cat1")
	theirs.Title = "On someone else's account"
	if _, err := svc.Create(context.Background(), "u2", theirs); err != nil {
		t.Fatal(err)
	}

	cat1 := "cat1"
	items, err := svc.List(context.Background(), "u1", rt.Filter{CategoryID: &cat1})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("listed %v, want nothing — u1 has no permission on acc2", titles(items))
	}
}
