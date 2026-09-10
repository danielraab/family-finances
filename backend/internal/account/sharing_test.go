package account_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/account"
	"at.draab/familyfinances/internal/storage/memory"
)

// fakeUsers is a minimal account.UserLookup double — sharing tests don't
// depend on the real internal/auth package, matching the package-boundaries
// decision in design.md.
type fakeUsers struct {
	byEmail       map[string]fakeUser
	inviteEnabled bool
}

type fakeUser struct{ id, name string }

func newFakeUsers() *fakeUsers { return &fakeUsers{byEmail: map[string]fakeUser{}} }

func (f *fakeUsers) add(email, id, name string) { f.byEmail[email] = fakeUser{id: id, name: name} }

func (f *fakeUsers) ByEmail(_ context.Context, email string) (string, string, bool, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return "", "", false, nil
	}
	return u.id, u.name, true, nil
}

func (f *fakeUsers) InviteEnabled() bool { return f.inviteEnabled }

// fakeMailer is a minimal account.Mailer double, recording every send.
type fakeMailer struct {
	sent []sentShare
}

type sentShare struct{ addr, accountTitle, granterName, permission, link string }

func (f *fakeMailer) SendAccountShare(_ context.Context, addr, accountTitle, granterName, permission, link string) error {
	f.sent = append(f.sent, sentShare{addr, accountTitle, granterName, permission, link})
	return nil
}

// sharingFixture wires an account.Service with a fake UserLookup/Mailer and
// one account owned by "owner", for the sharing tests below.
type sharingFixture struct {
	svc     *account.Service
	users   *fakeUsers
	mailer  *fakeMailer
	ownerID string
	accID   string
}

func newSharingFixture(t *testing.T) sharingFixture {
	t.Helper()
	store := memory.NewAccountStore()
	users := newFakeUsers()
	mailer := &fakeMailer{}
	svc := account.NewService(store, account.WithMailer(mailer), account.WithBaseURL("https://app.example"))
	svc.SetUserLookup(users)

	ctx := context.Background()
	typ, err := svc.CreateType(ctx, "owner", "Checking", "")
	if err != nil {
		t.Fatal(err)
	}
	opening, _ := account.ParseDate("2024-01-01")
	acc, err := svc.Create(ctx, "owner", account.New{Title: "Joint", TypeID: typ.ID, Currency: "EUR", OpeningDate: opening})
	if err != nil {
		t.Fatal(err)
	}

	return sharingFixture{svc: svc, users: users, mailer: mailer, ownerID: "owner", accID: acc.ID}
}

func TestInviteShareMatchedCreatesShareAndSendsEmail(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()

	result, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "member@example.com", account.PermissionAppend)
	if err != nil {
		t.Fatalf("InviteShare: %v", err)
	}
	if !result.Matched || result.Share == nil || result.Share.Permission != account.PermissionAppend {
		t.Fatalf("result = %+v", result)
	}
	if len(f.mailer.sent) != 1 || f.mailer.sent[0].addr != "member@example.com" || f.mailer.sent[0].granterName != "Owner" {
		t.Fatalf("mailer.sent = %+v", f.mailer.sent)
	}

	currency, disabled, perm, err := f.svc.Access(ctx, f.accID, "u2")
	if err != nil || currency != "EUR" || disabled || perm != string(account.PermissionAppend) {
		t.Fatalf("Access(u2) = %q %v %q %v", currency, disabled, perm, err)
	}
}

func TestInviteShareUnmatchedReportsInviteAllowed(t *testing.T) {
	f := newSharingFixture(t)
	f.users.inviteEnabled = true
	ctx := context.Background()

	result, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "nobody@example.com", account.PermissionView)
	if err != nil {
		t.Fatalf("InviteShare: %v", err)
	}
	if result.Matched || !result.InviteAllowed {
		t.Fatalf("result = %+v, want unmatched with invite_allowed", result)
	}
	if len(f.mailer.sent) != 0 {
		t.Fatalf("mailer.sent = %+v, want no email sent", f.mailer.sent)
	}
}

func TestInviteShareUnmatchedInviteDisabled(t *testing.T) {
	f := newSharingFixture(t)
	f.users.inviteEnabled = false
	ctx := context.Background()

	result, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "nobody@example.com", account.PermissionView)
	if err != nil {
		t.Fatalf("InviteShare: %v", err)
	}
	if result.Matched || result.InviteAllowed {
		t.Fatalf("result = %+v, want invite_allowed=false", result)
	}
}

func TestInviteShareRejectsSelfAndRealOwnerEmail(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("owner-self@example.com", f.ownerID, "Owner")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "owner-self@example.com", account.PermissionView); !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("sharing with own email: err = %v, want ErrInvalidValue", err)
	}
}

func TestInviteShareRequiresOwnerTier(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("appender@example.com", "u2", "Appender")
	f.users.add("target@example.com", "u3", "Target")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "appender@example.com", account.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.InviteShare(ctx, "u2", "Appender", f.accID, "target@example.com", account.PermissionView); !errors.Is(err, account.ErrForbidden) {
		t.Fatalf("append-tier InviteShare: err = %v, want ErrForbidden", err)
	}
}

func TestReSharingUpdatesExistingShareInPlace(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "member@example.com", account.PermissionView); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "member@example.com", account.PermissionEntryAdmin); err != nil {
		t.Fatal(err)
	}

	shares, err := f.svc.ListShares(ctx, f.ownerID, f.accID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 || shares[0].Permission != account.PermissionEntryAdmin {
		t.Fatalf("shares = %+v, want exactly one at entry_admin", shares)
	}
}

func TestListSharesVisibleAtViewTierReadOnly(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("viewer@example.com", "u2", "Viewer")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "viewer@example.com", account.PermissionView); err != nil {
		t.Fatal(err)
	}

	shares, err := f.svc.ListShares(ctx, "u2", f.accID)
	if err != nil {
		t.Fatalf("view-tier ListShares: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("shares = %+v", shares)
	}
}

func TestListSharesNotFoundForNoPermission(t *testing.T) {
	f := newSharingFixture(t)
	if _, err := f.svc.ListShares(context.Background(), "stranger", f.accID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestUpdateSharePermissionRejectsRealOwnerTarget(t *testing.T) {
	f := newSharingFixture(t)
	ctx := context.Background()
	if _, err := f.svc.UpdateSharePermission(ctx, f.ownerID, f.accID, f.ownerID, account.PermissionView); !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestRevokeShareRemovesAllAccess(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "member@example.com", account.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.RevokeShare(ctx, f.ownerID, f.accID, "u2"); err != nil {
		t.Fatalf("RevokeShare: %v", err)
	}

	if _, err := f.svc.Get(ctx, "u2", f.accID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("Get after revoke: err = %v, want ErrNotFound", err)
	}
}

func TestSelfLeaveWorksForAnyNonOwnerTier(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "member@example.com", account.PermissionView); err != nil {
		t.Fatal(err)
	}

	// Self-leave requires no owner-tier permission of its own.
	if err := f.svc.RevokeShare(ctx, "u2", f.accID, "u2"); err != nil {
		t.Fatalf("self-leave: %v", err)
	}
	if _, err := f.svc.Get(ctx, "u2", f.accID); !errors.Is(err, account.ErrNotFound) {
		t.Fatalf("Get after self-leave: err = %v, want ErrNotFound", err)
	}
}

func TestRealOwnerCannotSelfLeave(t *testing.T) {
	f := newSharingFixture(t)
	if err := f.svc.RevokeShare(context.Background(), f.ownerID, f.accID, f.ownerID); !errors.Is(err, account.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestSharedOwnerHasFullAccountRightsExceptType(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("coowner@example.com", "u2", "Co-owner")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "coowner@example.com", account.PermissionOwner); err != nil {
		t.Fatal(err)
	}

	// Full metadata edit (no type_id).
	newTitle := "Renamed by co-owner"
	if _, err := f.svc.Update(ctx, "u2", f.accID, account.Update{Title: &newTitle}); err != nil {
		t.Fatalf("shared-owner Update: %v", err)
	}
	// Disable/Enable.
	if _, err := f.svc.Disable(ctx, "u2", f.accID); err != nil {
		t.Fatalf("shared-owner Disable: %v", err)
	}
	if _, err := f.svc.Enable(ctx, "u2", f.accID); err != nil {
		t.Fatalf("shared-owner Enable: %v", err)
	}
	// Managing shares.
	f.users.add("third@example.com", "u3", "Third")
	if _, err := f.svc.InviteShare(ctx, "u2", "Co-owner", f.accID, "third@example.com", account.PermissionView); err != nil {
		t.Fatalf("shared-owner InviteShare: %v", err)
	}

	// type_id reassignment is real-owner-only.
	otherType, err := f.svc.CreateType(ctx, f.ownerID, "Savings", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Update(ctx, "u2", f.accID, account.Update{TypeID: &otherType.ID}); !errors.Is(err, account.ErrForbidden) {
		t.Fatalf("shared-owner reassigning type_id: err = %v, want ErrForbidden", err)
	}

	// Real owner can still delete the account.
	if err := f.svc.Delete(ctx, f.ownerID, f.accID); err != nil {
		t.Fatalf("real owner Delete: %v", err)
	}
}

func TestViewTierCannotEditOrManageShares(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("viewer@example.com", "u2", "Viewer")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "viewer@example.com", account.PermissionView); err != nil {
		t.Fatal(err)
	}

	newTitle := "Hijacked"
	if _, err := f.svc.Update(ctx, "u2", f.accID, account.Update{Title: &newTitle}); !errors.Is(err, account.ErrForbidden) {
		t.Fatalf("view-tier Update: err = %v, want ErrForbidden", err)
	}
	if _, err := f.svc.Disable(ctx, "u2", f.accID); !errors.Is(err, account.ErrForbidden) {
		t.Fatalf("view-tier Disable: err = %v, want ErrForbidden", err)
	}
	if err := f.svc.Delete(ctx, "u2", f.accID); !errors.Is(err, account.ErrForbidden) {
		t.Fatalf("view-tier Delete: err = %v, want ErrForbidden", err)
	}
	f.users.add("target@example.com", "u3", "Target")
	if _, err := f.svc.InviteShare(ctx, "u2", "Viewer", f.accID, "target@example.com", account.PermissionView); !errors.Is(err, account.ErrForbidden) {
		t.Fatalf("view-tier InviteShare: err = %v, want ErrForbidden", err)
	}

	// But reading still works.
	if _, err := f.svc.Get(ctx, "u2", f.accID); err != nil {
		t.Fatalf("view-tier Get: %v", err)
	}
}

func TestListIncludesOwnedAndSharedAccounts(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.accID, "member@example.com", account.PermissionView); err != nil {
		t.Fatal(err)
	}

	// u2 owns no accounts of their own, but sees the shared one.
	list, err := f.svc.List(ctx, "u2")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Permission != account.PermissionView || !list[0].Shared {
		t.Fatalf("List(u2) = %+v", list)
	}

	// The real owner's own listing never shows Shared, even though they've
	// shared the account with someone else.
	ownerList, err := f.svc.List(ctx, f.ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerList) != 1 || ownerList[0].Shared {
		t.Fatalf("List(owner) = %+v, want Shared=false", ownerList)
	}
}
