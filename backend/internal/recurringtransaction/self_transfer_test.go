package recurringtransaction_test

import (
	"context"
	"errors"
	"testing"

	rt "at.draab/familyfinances/internal/recurringtransaction"
)

// baseSelfTransfer is baseNew's self_transfer counterpart: no category (it
// is optional for this kind), a to-account, and the same recurrence rule.
func baseSelfTransfer(accountID, toAccountID string) rt.New {
	in := rt.New{
		AccountID:     accountID,
		ToAccountID:   &toAccountID,
		Kind:          rt.KindSelfTransfer,
		Title:         "To savings",
		Amount:        -50000,
		IntervalUnit:  rt.UnitMonth,
		IntervalCount: 1,
		StartsOn:      mustDateT(2026, 1, 1),
	}
	return in
}

// selfTransferFixture builds a service with two same-currency accounts the
// owner holds outright, and returns both ids.
func selfTransferFixture(t *testing.T) (*rt.Service, *stubAccounts, string, string) {
	t.Helper()
	svc, accounts, _, _, _ := newService()
	accounts.add("checking", "u1", "EUR")
	accounts.add("savings", "u1", "EUR")
	return svc, accounts, "checking", "savings"
}

func TestCreateSelfTransferNeedsNoCategory(t *testing.T) {
	svc, _, from, to := selfTransferFixture(t)

	created, err := svc.Create(context.Background(), "u1", baseSelfTransfer(from, to))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Kind != rt.KindSelfTransfer {
		t.Fatalf("kind = %q, want self_transfer", created.Kind)
	}
	if created.ToAccountID == nil || *created.ToAccountID != to {
		t.Fatalf("to_account_id = %v, want %q", created.ToAccountID, to)
	}
	if created.CategoryID != nil {
		t.Fatalf("category_id = %v, want nil", created.CategoryID)
	}
	if !created.Native {
		t.Fatal("a created template should be native")
	}
}

func TestCreateTransactionStillRequiresCategory(t *testing.T) {
	svc, accounts, _, _, _ := newService()
	accounts.add("a1", "u1", "EUR")

	in := baseNew("a1", "c1")
	in.CategoryID = nil
	if _, err := svc.Create(context.Background(), "u1", in); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateSelfTransferRejectsMissingOrEqualToAccount(t *testing.T) {
	svc, _, from, _ := selfTransferFixture(t)

	missing := baseSelfTransfer(from, "")
	missing.ToAccountID = nil
	if _, err := svc.Create(context.Background(), "u1", missing); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("missing to_account_id: err = %v, want ErrInvalidValue", err)
	}

	same := baseSelfTransfer(from, from)
	if _, err := svc.Create(context.Background(), "u1", same); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("to_account_id == account_id: err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateTransactionRejectsToAccount(t *testing.T) {
	svc, accounts, categories, _, _ := newService()
	accounts.add("a1", "u1", "EUR")
	categories.add("c1")

	to := "a2"
	in := baseNew("a1", "c1")
	in.ToAccountID = &to
	if _, err := svc.Create(context.Background(), "u1", in); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateSelfTransferRejectsCounterpartyAndLocation(t *testing.T) {
	svc, _, from, to := selfTransferFixture(t)

	withCounterparty := baseSelfTransfer(from, to)
	withCounterparty.Counterparty = "Bank"
	if _, err := svc.Create(context.Background(), "u1", withCounterparty); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("counterparty: err = %v, want ErrInvalidValue", err)
	}

	withLocation := baseSelfTransfer(from, to)
	withLocation.Location = "Vienna"
	if _, err := svc.Create(context.Background(), "u1", withLocation); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("location: err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateSelfTransferRejectsCurrencyMismatch(t *testing.T) {
	svc, accounts, _, _, _ := newService()
	accounts.add("checking", "u1", "EUR")
	accounts.add("usd", "u1", "USD")

	if _, err := svc.Create(context.Background(), "u1", baseSelfTransfer("checking", "usd")); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCreateSelfTransferRequiresAppendOnBothAccounts(t *testing.T) {
	svc, accounts, _, _, _ := newService()
	accounts.add("mine", "u1", "EUR")
	accounts.add("theirs", "u2", "EUR")
	accounts.share("theirs", "u1", "view")

	if _, err := svc.Create(context.Background(), "u1", baseSelfTransfer("mine", "theirs")); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("view on the far account: err = %v, want ErrInvalidValue", err)
	}

	accounts.share("theirs", "u1", "append")
	if _, err := svc.Create(context.Background(), "u1", baseSelfTransfer("mine", "theirs")); err != nil {
		t.Fatalf("append on the far account: %v", err)
	}
}

func TestCreateSelfTransferRejectsDisabledFarAccount(t *testing.T) {
	svc, accounts, from, to := selfTransferFixture(t)
	accounts.disabled[to] = true

	if _, err := svc.Create(context.Background(), "u1", baseSelfTransfer(from, to)); !errors.Is(err, rt.ErrAccountDisabled) {
		t.Fatalf("err = %v, want ErrAccountDisabled", err)
	}
}

func TestUpdateSelfTransferRequiresBothAccountsStill(t *testing.T) {
	svc, accounts, _, _, _ := newService()
	accounts.add("mine", "u1", "EUR")
	accounts.add("shared", "u2", "EUR")
	accounts.share("shared", "u1", "append")

	created, err := svc.Create(context.Background(), "u1", baseSelfTransfer("mine", "shared"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	title := "Renamed"
	if _, err := svc.Update(context.Background(), "u1", created.ID, rt.Update{Title: &title}); err != nil {
		t.Fatalf("edit while both are held: %v", err)
	}

	// The share on the far account is revoked; the template becomes
	// read-only, even for a change unrelated to either account.
	accounts.share("shared", "u1", "view")
	again := "Renamed again"
	if _, err := svc.Update(context.Background(), "u1", created.ID, rt.Update{Title: &again}); !errors.Is(err, rt.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestDeleteSelfTransferNeedsOnlyTheOwnAccount(t *testing.T) {
	svc, accounts, _, _, _ := newService()
	accounts.add("mine", "u1", "EUR")
	accounts.add("shared", "u2", "EUR")
	accounts.share("shared", "u1", "append")

	created, err := svc.Create(context.Background(), "u1", baseSelfTransfer("mine", "shared"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Editing would now be forbidden; deleting is not — see the
	// edit-needs-both/delete-needs-one asymmetry in design.md.
	accounts.share("shared", "u1", "view")
	if err := svc.Delete(context.Background(), "u1", created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestUpdateCannotMoveASelfTransfer(t *testing.T) {
	svc, accounts, from, to := selfTransferFixture(t)
	accounts.add("third", "u1", "EUR")

	created, err := svc.Create(context.Background(), "u1", baseSelfTransfer(from, to))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	third := "third"
	if _, err := svc.Update(context.Background(), "u1", created.ID, rt.Update{AccountID: &third}); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestUpdateSelfTransferRejectsCounterparty(t *testing.T) {
	svc, _, from, to := selfTransferFixture(t)
	created, err := svc.Create(context.Background(), "u1", baseSelfTransfer(from, to))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	counterparty := "Bank"
	if _, err := svc.Update(context.Background(), "u1", created.ID, rt.Update{Counterparty: &counterparty}); !errors.Is(err, rt.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestGetSelfTransferFromTheReceivingAccountAlone(t *testing.T) {
	svc, accounts, _, _, _ := newService()
	accounts.add("mine", "u1", "EUR")
	accounts.add("theirs", "u2", "EUR")
	accounts.share("theirs", "u1", "append")

	created, err := svc.Create(context.Background(), "u1", baseSelfTransfer("mine", "theirs"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// u2 owns only the receiving account and has no permission at all on
	// the sending one — either parent account is enough to read it.
	got, err := svc.Get(context.Background(), "u2", created.ID)
	if err != nil {
		t.Fatalf("get from the receiving side: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("id = %q, want %q", got.ID, created.ID)
	}
}

func TestListSelfTransferModes(t *testing.T) {
	svc, _, from, to := selfTransferFixture(t)
	created, err := svc.Create(context.Background(), "u1", baseSelfTransfer(from, to))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	for _, tc := range []struct {
		name      string
		mode      rt.SelfTransferMode
		wantRows  int
		wantFirst int64
	}{
		{"excluded by default", rt.SelfTransferExclude, 0, 0},
		{"native only", rt.SelfTransferNative, 1, -50000},
		{"both legs", rt.SelfTransferBothLegs, 2, -50000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items, err := svc.List(context.Background(), "u1", rt.Filter{SelfTransfers: tc.mode})
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			if len(items) != tc.wantRows {
				t.Fatalf("got %d rows, want %d", len(items), tc.wantRows)
			}
			if tc.wantRows == 0 {
				return
			}
			if items[0].Amount != tc.wantFirst {
				t.Fatalf("first amount = %d, want %d", items[0].Amount, tc.wantFirst)
			}
			if !items[0].Native {
				t.Fatal("the first row should be the native leg")
			}
			if tc.wantRows < 2 {
				return
			}
			if items[1].ID != created.ID {
				t.Fatalf("second row id = %q, want the same template", items[1].ID)
			}
			if items[1].Amount != -items[0].Amount {
				t.Fatalf("second amount = %d, want %d", items[1].Amount, -items[0].Amount)
			}
			if items[1].AccountID != to || items[1].ToAccountID == nil || *items[1].ToAccountID != from {
				t.Fatalf("second row accounts = %q -> %v, want %q -> %q", items[1].AccountID, items[1].ToAccountID, to, from)
			}
			if items[1].Native {
				t.Fatal("the second row should be the flipped leg")
			}
		})
	}
}

func TestListOneLegHidesATransferOnlyVisibleFromTheReceivingSide(t *testing.T) {
	svc, accounts, _, _, _ := newService()
	accounts.add("mine", "u1", "EUR")
	accounts.add("theirs", "u2", "EUR")
	accounts.share("theirs", "u1", "append")
	if _, err := svc.Create(context.Background(), "u1", baseSelfTransfer("mine", "theirs")); err != nil {
		t.Fatalf("create: %v", err)
	}

	// u2 sees only the receiving account, which is not the stored
	// account_id — so the native-only mode has nothing to show them.
	native, err := svc.List(context.Background(), "u2", rt.Filter{SelfTransfers: rt.SelfTransferNative})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(native) != 0 {
		t.Fatalf("got %d rows, want 0", len(native))
	}

	both, err := svc.List(context.Background(), "u2", rt.Filter{SelfTransfers: rt.SelfTransferBothLegs})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(both) != 1 {
		t.Fatalf("got %d rows, want 1", len(both))
	}
	if both[0].Amount != 50000 {
		t.Fatalf("amount = %d, want 50000 (money arriving)", both[0].Amount)
	}
}

func TestSummaryBothLegsCancel(t *testing.T) {
	svc, _, from, to := selfTransferFixture(t)
	if _, err := svc.Create(context.Background(), "u1", baseSelfTransfer(from, to)); err != nil {
		t.Fatalf("create: %v", err)
	}

	excluded, err := svc.Summary(context.Background(), "u1", rt.Filter{SelfTransfers: rt.SelfTransferExclude})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if excluded.Count != 0 || len(excluded.Sums) != 0 {
		t.Fatalf("excluded summary = %+v, want empty", excluded)
	}

	native, err := svc.Summary(context.Background(), "u1", rt.Filter{SelfTransfers: rt.SelfTransferNative})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if native.Count != 1 || len(native.Sums) != 1 || native.Sums[0].Amount != -600000 {
		t.Fatalf("native summary = %+v, want one EUR row of -600000", native)
	}

	both, err := svc.Summary(context.Background(), "u1", rt.Filter{SelfTransfers: rt.SelfTransferBothLegs})
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if both.Count != 2 {
		t.Fatalf("count = %d, want 2", both.Count)
	}
	if len(both.Sums) != 1 || both.Sums[0].Amount != 0 {
		t.Fatalf("summary = %+v, want the two legs to cancel to 0", both)
	}
}

func TestPreviewAlwaysProjectsBothLegs(t *testing.T) {
	svc, _, from, to := selfTransferFixture(t)
	if _, err := svc.Create(context.Background(), "u1", baseSelfTransfer(from, to)); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Preview takes no self-transfer mode at all: both legs, always.
	items, err := svc.Preview(context.Background(), "u1", rt.PreviewFilter{
		To: mustDateT(2026, 1, 31).Time,
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2 (one per account)", len(items))
	}
	if items[0].Kind != rt.KindSelfTransfer {
		t.Fatalf("kind = %q, want self_transfer", items[0].Kind)
	}
	if items[0].Amount != -items[1].Amount {
		t.Fatalf("amounts %d and %d should be opposite", items[0].Amount, items[1].Amount)
	}
}

func TestHasSelfTransferAccountCoversBothSides(t *testing.T) {
	svc, accounts, from, to := selfTransferFixture(t)
	accounts.add("untouched", "u1", "EUR")
	if _, err := svc.Create(context.Background(), "u1", baseSelfTransfer(from, to)); err != nil {
		t.Fatalf("create: %v", err)
	}

	for _, tc := range []struct {
		account string
		want    bool
	}{{from, true}, {to, true}, {"untouched", false}} {
		got, err := svc.HasSelfTransferAccount(context.Background(), tc.account)
		if err != nil {
			t.Fatalf("lookup %q: %v", tc.account, err)
		}
		if got != tc.want {
			t.Fatalf("%q = %v, want %v", tc.account, got, tc.want)
		}
	}
}
