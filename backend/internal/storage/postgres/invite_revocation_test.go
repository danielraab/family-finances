package postgres

import (
	"context"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
)

func createTestInvite(t *testing.T, store *AuthStore, inviterID, email string) auth.Invite {
	t.Helper()
	now := time.Now()
	inv, err := store.CreateInvite(context.Background(), auth.Invite{
		Email: email, InvitedBy: inviterID, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}, hashOf(email+"-token"))
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	return inv
}

func TestPGRevokeInviteIsIdempotent(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	inviter, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "inviter@example.com"}, emailIdentity("inviter@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	inv := createTestInvite(t, store, inviter.ID, "guest@example.com")

	first, err := store.RevokeInvite(ctx, inv.ID, time.Now())
	if err != nil {
		t.Fatalf("RevokeInvite: %v", err)
	}
	if first.RevokedAt == nil {
		t.Fatal("revoked_at not set")
	}

	second, err := store.RevokeInvite(ctx, inv.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("second RevokeInvite: %v", err)
	}
	if second.RevokedAt == nil || !second.RevokedAt.Equal(*first.RevokedAt) {
		t.Fatalf("revoked_at changed on repeat call: %v -> %v", first.RevokedAt, second.RevokedAt)
	}
}

func TestPGRevokeInviteUnknownID(t *testing.T) {
	store, _ := newAuthStore(t)
	if _, err := store.RevokeInvite(context.Background(), "00000000-0000-0000-0000-000000000000", time.Now()); err != auth.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGSoftDeleteInviteExcludesFromListings(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	inviter, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "inviter@example.com"}, emailIdentity("inviter@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	inv := createTestInvite(t, store, inviter.ID, "guest@example.com")

	if _, err := store.RevokeInvite(ctx, inv.ID, time.Now()); err != nil {
		t.Fatalf("RevokeInvite: %v", err)
	}
	if err := store.SoftDeleteInvite(ctx, inv.ID, time.Now()); err != nil {
		t.Fatalf("SoftDeleteInvite: %v", err)
	}

	if _, err := store.InviteByID(ctx, inv.ID); err != auth.ErrNotFound {
		t.Fatalf("InviteByID after delete = %v, want ErrNotFound", err)
	}

	all, err := store.ListInvites(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range all {
		if i.ID == inv.ID {
			t.Fatal("soft-deleted invite still in ListInvites")
		}
	}

	mine, err := store.ListInvitesByInviter(ctx, inviter.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range mine {
		if i.ID == inv.ID {
			t.Fatal("soft-deleted invite still in ListInvitesByInviter")
		}
	}

	if err := store.SoftDeleteInvite(ctx, inv.ID, time.Now()); err != auth.ErrNotFound {
		t.Fatalf("re-delete err = %v, want ErrNotFound", err)
	}
}

func TestPGListInvitesByInviterScopesToOwnInvites(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	a, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "a@example.com"}, emailIdentity("a@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "b@example.com"}, emailIdentity("b@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	createTestInvite(t, store, a.ID, "a-guest@example.com")
	createTestInvite(t, store, b.ID, "b-guest@example.com")

	mine, err := store.ListInvitesByInviter(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 1 || mine[0].Email != "a-guest@example.com" {
		t.Fatalf("ListInvitesByInviter(a) = %+v, want just a's own invite", mine)
	}
}

func TestPGConsumeInviteRejectsRevoked(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	inviter, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "inviter@example.com"}, emailIdentity("inviter@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	inv, err := store.CreateInvite(ctx, auth.Invite{
		Email: "guest@example.com", InvitedBy: inviter.ID, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}, hashOf("revoked-invite-token"))
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if _, err := store.RevokeInvite(ctx, inv.ID, now); err != nil {
		t.Fatalf("RevokeInvite: %v", err)
	}

	if _, err := store.ConsumeInvite(ctx, hashOf("revoked-invite-token"), now); err != auth.ErrInviteInvalid {
		t.Fatalf("ConsumeInvite err = %v, want ErrInviteInvalid", err)
	}
}
