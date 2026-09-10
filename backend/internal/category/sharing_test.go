package category_test

import (
	"context"
	"errors"
	"testing"

	"at.draab/familyfinances/internal/category"
	"at.draab/familyfinances/internal/storage/memory"
)

// fakeUsers is a minimal category.UserLookup double — sharing tests don't
// depend on the real internal/auth package, mirroring internal/account's
// sharing_test.go.
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

// fakeMailer is a minimal category.Mailer double, recording every send.
type fakeMailer struct {
	sent []sentShare
}

type sentShare struct{ addr, categoryName, granterName, permission, link string }

func (f *fakeMailer) SendCategoryShare(_ context.Context, addr, categoryName, granterName, permission, link string) error {
	f.sent = append(f.sent, sentShare{addr, categoryName, granterName, permission, link})
	return nil
}

// sharingFixture wires a category.Service with a fake UserLookup/Mailer and
// one category owned by "owner", for the sharing tests below.
type sharingFixture struct {
	svc     *category.Service
	users   *fakeUsers
	mailer  *fakeMailer
	ownerID string
	catID   string
}

func newSharingFixture(t *testing.T) sharingFixture {
	t.Helper()
	store := memory.NewCategoryStore()
	users := newFakeUsers()
	mailer := &fakeMailer{}
	svc := category.NewService(store, category.WithMailer(mailer), category.WithBaseURL("https://app.example"))
	svc.SetUserLookup(users)

	ctx := context.Background()
	cat, err := svc.Create(ctx, "owner", category.New{Name: "Groceries"})
	if err != nil {
		t.Fatal(err)
	}

	return sharingFixture{svc: svc, users: users, mailer: mailer, ownerID: "owner", catID: cat.ID}
}

func TestCategoryInviteShareMatchedCreatesShareAndSendsEmail(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()

	result, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "member@example.com", category.PermissionAppend)
	if err != nil {
		t.Fatalf("InviteShare: %v", err)
	}
	if !result.Matched || result.Share == nil || result.Share.Permission != category.PermissionAppend {
		t.Fatalf("result = %+v", result)
	}
	if len(f.mailer.sent) != 1 || f.mailer.sent[0].addr != "member@example.com" || f.mailer.sent[0].granterName != "Owner" {
		t.Fatalf("mailer.sent = %+v", f.mailer.sent)
	}

	ok, err := f.svc.Usable(ctx, "u2", f.catID)
	if err != nil || !ok {
		t.Fatalf("Usable(u2) = %v, %v, want true", ok, err)
	}
}

func TestCategoryInviteShareUnmatchedReportsInviteAllowed(t *testing.T) {
	f := newSharingFixture(t)
	f.users.inviteEnabled = true
	ctx := context.Background()

	result, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "nobody@example.com", category.PermissionView)
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

func TestCategoryInviteShareRejectsSelfEmail(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("owner-self@example.com", f.ownerID, "Owner")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "owner-self@example.com", category.PermissionView); !errors.Is(err, category.ErrInvalidValue) {
		t.Fatalf("sharing with own email: err = %v, want ErrInvalidValue", err)
	}
}

func TestCategoryInviteShareRequiresRealOwner(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("appender@example.com", "u2", "Appender")
	f.users.add("target@example.com", "u3", "Target")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "appender@example.com", category.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.InviteShare(ctx, "u2", "Appender", f.catID, "target@example.com", category.PermissionView); !errors.Is(err, category.ErrForbidden) {
		t.Fatalf("append-tier InviteShare: err = %v, want ErrForbidden", err)
	}
}

func TestCategoryReSharingUpdatesExistingShareInPlace(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()

	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "member@example.com", category.PermissionView); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "member@example.com", category.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	shares, err := f.svc.ListShares(ctx, f.ownerID, f.catID)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != 1 || shares[0].Permission != category.PermissionAppend {
		t.Fatalf("shares = %+v, want exactly one at append", shares)
	}
}

func TestCategoryListSharesVisibleAtViewTierReadOnly(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("viewer@example.com", "u2", "Viewer")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "viewer@example.com", category.PermissionView); err != nil {
		t.Fatal(err)
	}

	shares, err := f.svc.ListShares(ctx, "u2", f.catID)
	if err != nil {
		t.Fatalf("view-tier ListShares: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("shares = %+v", shares)
	}
}

func TestCategoryListSharesNotFoundForNoPermission(t *testing.T) {
	f := newSharingFixture(t)
	if _, err := f.svc.ListShares(context.Background(), "stranger", f.catID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCategoryUpdateSharePermissionRejectsRealOwnerTarget(t *testing.T) {
	f := newSharingFixture(t)
	ctx := context.Background()
	if _, err := f.svc.UpdateSharePermission(ctx, f.ownerID, f.catID, f.ownerID, category.PermissionView); !errors.Is(err, category.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCategoryUpdateSharePermissionRequiresRealOwner(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("appender@example.com", "u2", "Appender")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "appender@example.com", category.PermissionAppend); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.UpdateSharePermission(ctx, "u2", f.catID, "u2", category.PermissionView); !errors.Is(err, category.ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestCategoryRevokeShareStopsNewSelectionOnly(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "member@example.com", category.PermissionAppend); err != nil {
		t.Fatal(err)
	}

	if err := f.svc.RevokeShare(ctx, f.ownerID, f.catID, "u2"); err != nil {
		t.Fatalf("RevokeShare: %v", err)
	}

	ok, err := f.svc.Usable(ctx, "u2", f.catID)
	if err != nil || ok {
		t.Fatalf("Usable(u2) after revoke = %v, %v, want false", ok, err)
	}
	if _, err := f.svc.Get(ctx, "u2", f.catID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("Get after revoke: err = %v, want ErrNotFound", err)
	}
}

func TestCategorySelfLeaveWorksForAnyTier(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "member@example.com", category.PermissionView); err != nil {
		t.Fatal(err)
	}

	// Self-leave requires no owner permission of its own.
	if err := f.svc.RevokeShare(ctx, "u2", f.catID, "u2"); err != nil {
		t.Fatalf("self-leave: %v", err)
	}
	if _, err := f.svc.Get(ctx, "u2", f.catID); !errors.Is(err, category.ErrNotFound) {
		t.Fatalf("Get after self-leave: err = %v, want ErrNotFound", err)
	}
}

func TestCategoryRealOwnerCannotSelfLeave(t *testing.T) {
	f := newSharingFixture(t)
	if err := f.svc.RevokeShare(context.Background(), f.ownerID, f.catID, f.ownerID); !errors.Is(err, category.ErrInvalidValue) {
		t.Fatalf("err = %v, want ErrInvalidValue", err)
	}
}

func TestCategoryViewTierCannotSelectOrManageShares(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("viewer@example.com", "u2", "Viewer")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "viewer@example.com", category.PermissionView); err != nil {
		t.Fatal(err)
	}

	ok, err := f.svc.Usable(ctx, "u2", f.catID)
	if err != nil || ok {
		t.Fatalf("view-tier Usable = %v, %v, want false", ok, err)
	}
	f.users.add("target@example.com", "u3", "Target")
	if _, err := f.svc.InviteShare(ctx, "u2", "Viewer", f.catID, "target@example.com", category.PermissionView); !errors.Is(err, category.ErrForbidden) {
		t.Fatalf("view-tier InviteShare: err = %v, want ErrForbidden", err)
	}

	// But reading (List/Get) still works.
	if _, err := f.svc.Get(ctx, "u2", f.catID); err != nil {
		t.Fatalf("view-tier Get: %v", err)
	}
}

func TestCategoryDisabledSharedCategoryNotUsable(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "member@example.com", category.PermissionAppend); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Disable(ctx, f.ownerID, f.catID); err != nil {
		t.Fatal(err)
	}

	ok, err := f.svc.Usable(ctx, "u2", f.catID)
	if err != nil || ok {
		t.Fatalf("Usable(u2) on disabled shared category = %v, %v, want false", ok, err)
	}
}

func TestCategoryListIncludesOwnedAndSharedFlat(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "member@example.com", category.PermissionView); err != nil {
		t.Fatal(err)
	}

	// u2 owns no categories of their own, but sees the shared one.
	list, err := f.svc.List(ctx, "u2")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Permission != category.PermissionView || !list[0].Shared {
		t.Fatalf("List(u2) = %+v", list)
	}

	// The real owner's own listing never shows Shared.
	ownerList, err := f.svc.List(ctx, f.ownerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerList) != 1 || ownerList[0].Shared || ownerList[0].Permission != category.PermissionOwner {
		t.Fatalf("List(owner) = %+v", ownerList)
	}
}

func TestCategorySubtreeForSharedCategoryIsItselfAlone(t *testing.T) {
	f := newSharingFixture(t)
	f.users.add("member@example.com", "u2", "Member")
	ctx := context.Background()

	// Give the owner's category a child — not shared itself.
	child, err := f.svc.Create(ctx, f.ownerID, category.New{Name: "Snacks", ParentID: &f.catID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.InviteShare(ctx, f.ownerID, "Owner", f.catID, "member@example.com", category.PermissionView); err != nil {
		t.Fatal(err)
	}

	// The real owner's own filter resolution still walks the full subtree.
	ownerSubtree, err := f.svc.Subtree(ctx, f.ownerID, f.catID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerSubtree) != 2 {
		t.Fatalf("owner Subtree = %v, want id + child", ownerSubtree)
	}

	// The recipient's filter resolution never cascades to the unshared child.
	sharedSubtree, err := f.svc.Subtree(ctx, "u2", f.catID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sharedSubtree) != 1 || sharedSubtree[0] != f.catID {
		t.Fatalf("shared Subtree = %v, want [%s]", sharedSubtree, f.catID)
	}

	// A stranger with no access at all resolves to nothing.
	noneSubtree, err := f.svc.Subtree(ctx, "stranger", f.catID)
	if err != nil {
		t.Fatal(err)
	}
	if len(noneSubtree) != 0 {
		t.Fatalf("stranger Subtree = %v, want empty", noneSubtree)
	}

	_ = child
}
