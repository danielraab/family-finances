package tag_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/storage/memory"
	"at.draab/familyfinances/internal/tag"
)

// fakeUsers is a minimal tag.UserLookup double — sharing tests don't
// depend on the real internal/auth package, mirroring
// internal/category's sharing_test.go.
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

// fakeMailer is a minimal tag.Mailer double, recording every send.
type fakeMailer struct {
	sent []sentShare
}

type sentShare struct{ addr, tagName, granterName, permission, link string }

func (f *fakeMailer) SendTagShare(_ context.Context, addr, tagName, granterName, permission, link string) error {
	f.sent = append(f.sent, sentShare{addr, tagName, granterName, permission, link})
	return nil
}

// sharingFixture wires a tag.Service with a fake UserLookup/Mailer and one
// tag owned by "owner", for the sharing tests below.
type sharingFixture struct {
	svc     *tag.Service
	users   *fakeUsers
	mailer  *fakeMailer
	ownerID string
	tagID   string
}

func newSharingFixture(t *testing.T) sharingFixture {
	t.Helper()
	store := memory.NewTagStore()
	users := newFakeUsers()
	mailer := &fakeMailer{}
	svc := tag.NewService(store, tag.WithMailer(mailer), tag.WithBaseURL("https://app.example"))
	svc.SetUserLookup(users)

	ctx := context.Background()
	tg, err := svc.Create(ctx, "owner", "groceries")
	if err != nil {
		t.Fatal(err)
	}

	return sharingFixture{svc: svc, users: users, mailer: mailer, ownerID: "owner", tagID: tg.ID}
}

func TestTagInviteShareMatchedCreatesShareAndSendsEmail(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()

	result, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionAppend)
	if err != nil {
		t.Fatalf("InviteShare: %v", err)
	}
	if !result.Matched || result.Share == nil || result.Share.Permission != tag.PermissionAppend {
		t.Fatalf("result = %+v", result)
	}
	if len(f.mailer.sent) != 1 || f.mailer.sent[0].addr != "member@example.com" || f.mailer.sent[0].granterName != "Owner" {
		t.Fatalf("mailer.sent = %+v", f.mailer.sent)
	}

	ok, err := f.svc.Usable(ctx, "u2", []string{f.tagID})
	if err != nil || !ok {
		t.Fatalf("Usable(u2) = %v, %v, want true", ok, err)
	}
}

func TestTagInviteShareUnmatchedReportsInviteAllowed(t *testing.T) {
	f := newSharingFixture(t)
	f.users.inviteEnabled = true
	ctx := context.Background()

	result, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "nobody@example.com", tag.PermissionView)
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

func TestTagInviteShareRejectsSelfEmail(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("owner-self@example.com", f.ownerID, "Owner")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "owner-self@example.com", tag.PermissionView); !errors.Is(err, tag.ErrInvalidValue) {
		t.Fatalf("sharing with own email: err = %v, want ErrInvalidValue", err)
	}
}

func TestTagInviteShareRequiresRealOwner(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("appender@example.com", "u2", "Appender")
	f.users.add("target@example.com", "u3", "Target")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "appender@example.com", tag.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.InviteShare(ctx, "u2", "Appender", f.tagID, "target@example.com", tag.PermissionView); !errors.Is(err, tag.ErrForbidden) {
		t.Fatalf("append-tier InviteShare: err = %v, want ErrForbidden", err)
	}
}

func TestTagReSharingUpdatesExistingShareInPlace(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	shares, err := f.svc.ListShares(ctx, f.ownerID, f.tagID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 || shares[0].Permission != tag.PermissionAppend {
		t.Fatalf("shares = %+v, want exactly one at append", shares)
	}
}

func TestTagListSharesVisibleAtViewTierReadOnly(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("viewer@example.com", "u2", "Viewer")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "viewer@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}

	shares, err := f.svc.ListShares(ctx, "u2", f.tagID)
	if err != nil {
		t.Fatalf("view-tier ListShares: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("shares = %+v", shares)
	}
}

func TestTagListSharesNotFoundForNoPermission(t *testing.T) {
	f := newSharingFixture(t)
	if _, err := f.svc.ListShares(context.Background(), "stranger", f.tagID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestTagUpdateSharePermissionRejectsRealOwnerTarget(t *testing.T) {
	f := newSharingFixture(t)
	ctx := context.Background()
	if _, err := f.svc.UpdateSharePermission(ctx, f.ownerID, f.tagID, f.ownerID, tag.PermissionView); !errors.Is(err, tag.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestTagUpdateSharePermissionRequiresRealOwner(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("appender@example.com", "u2", "Appender")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "appender@example.com", tag.PermissionAppend); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.UpdateSharePermission(ctx, "u2", f.tagID, "u2", tag.PermissionView); !errors.Is(err, tag.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestTagRevokeShareStopsNewSelectionOnly(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.RevokeShare(ctx, f.ownerID, f.tagID, "u2"); err != nil {
		t.Fatalf("RevokeShare: %v", err)
	}

	ok, err := f.svc.Usable(ctx, "u2", []string{f.tagID})
	if err != nil || ok {
		t.Fatalf("Usable(u2) after revoke = %v, %v, want false", ok, err)
	}
	if _, err := f.svc.Get(ctx, "u2", f.tagID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("Get after revoke: err = %v, want ErrNotFound", err)
	}
}

func TestTagSelfLeaveWorksForAnyTier(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}

	// Self-leave requires no owner permission of its own.
	if err := f.svc.RevokeShare(ctx, "u2", f.tagID, "u2"); err != nil {
		t.Fatalf("self-leave: %v", err)
	}
	if _, err := f.svc.Get(ctx, "u2", f.tagID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("Get after self-leave: err = %v, want ErrNotFound", err)
	}
}

func TestTagRealOwnerCannotSelfLeave(t *testing.T) {
	f := newSharingFixture(t)
	if err := f.svc.RevokeShare(context.Background(), f.ownerID, f.tagID, f.ownerID); !errors.Is(err, tag.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestTagViewTierCannotSelectOrManageShares(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("viewer@example.com", "u2", "Viewer")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "viewer@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}

	ok, err := f.svc.Usable(ctx, "u2", []string{f.tagID})
	if err != nil || ok {
		t.Fatalf("view-tier Usable = %v, %v, want false", ok, err)
	}
	f.users.add("target@example.com", "u3", "Target")
	if _, err := f.svc.InviteShare(ctx, "u2", "Viewer", f.tagID, "target@example.com", tag.PermissionView); !errors.Is(err, tag.ErrForbidden) {
		t.Fatalf("view-tier InviteShare: err = %v, want ErrForbidden", err)
	}

	// But reading (List/Get) still works.
	if _, err := f.svc.Get(ctx, "u2", f.tagID); err != nil {
		t.Fatalf("view-tier Get: %v", err)
	}
}

func TestTagDisabledSharedTagNotUsable(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionAppend); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Disable(ctx, f.ownerID, f.tagID); err != nil {
		t.Fatal(err)
	}

	ok, err := f.svc.Usable(ctx, "u2", []string{f.tagID})
	if err != nil || ok {
		t.Fatalf("Usable(u2) on disabled shared tag = %v, %v, want false", ok, err)
	}
}

func TestTagListIncludesOwnedAndSharedFlat(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}

	// u2 owns no tags of their own, but sees the shared one.
	list, err := f.svc.List(ctx, "u2")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Permission != tag.PermissionView || !list[0].Shared {
		t.Fatalf("List(u2) = %+v", list)
	}

	// The real owner's own listing never shows Shared.
	ownerList, err := f.svc.List(ctx, f.ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerList) != 1 || ownerList[0].Shared || ownerList[0].Permission != tag.PermissionOwner {
		t.Fatalf("List(owner) = %+v", ownerList)
	}
}

// --- delete-while-shared guard -----------------------------------------

func TestTagDeleteBlockedWhileShared(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.Delete(ctx, f.ownerID, f.tagID); !errors.Is(err, tag.ErrInUse) {
		t.Fatalf("Delete on shared tag: err = %v, want ErrInUse", err)
	}
}

func TestTagDeleteSucceedsAfterEveryShareIsGone(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.tagID, "member@example.com", tag.PermissionView); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.RevokeShare(ctx, f.ownerID, f.tagID, "u2"); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.Delete(ctx, f.ownerID, f.tagID); err != nil {
		t.Fatalf("Delete after unshare: %v", err)
	}
	if _, err := f.svc.Get(ctx, f.ownerID, f.tagID); !errors.Is(err, tag.ErrNotFound) {
		t.Fatalf("Get after delete: err = %v, want ErrNotFound", err)
	}
}
