package postgres

import (
	"context"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
)

func TestPGListUsersExcludesSoftDeleted(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()

	u1, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "one@example.com"}, emailIdentity("one@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	u2, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "two@example.com"}, emailIdentity("two@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.SoftDeleteUser(ctx, u2.ID, time.Now()); err != nil {
		t.Fatalf("SoftDeleteUser: %v", err)
	}

	users, err := store.ListUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].ID != u1.ID {
		t.Fatalf("ListUsers = %+v, want only %q", users, u1.ID)
	}
}

func TestPGSetUserDisabledAndDeleteSessions(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()

	u, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "u@example.com"}, emailIdentity("u@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if _, err := store.CreateSession(ctx, auth.Session{
		UserID: u.ID, Client: auth.ClientAPI, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour),
	}, hashOf("tok-1")); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := store.SetUserDisabled(ctx, u.ID, true); err != nil {
		t.Fatalf("SetUserDisabled: %v", err)
	}
	got, err := store.UserByID(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Disabled {
		t.Fatal("Disabled not set")
	}

	if err := store.DeleteSessionsByUserID(ctx, u.ID); err != nil {
		t.Fatalf("DeleteSessionsByUserID: %v", err)
	}
	if _, err := store.SessionByTokenHash(ctx, hashOf("tok-1")); err != auth.ErrNotFound {
		t.Fatalf("session should be gone, err = %v", err)
	}
}

func TestPGSetUserDisabledUnknownUser(t *testing.T) {
	store, _ := newAuthStore(t)
	if err := store.SetUserDisabled(context.Background(), "00000000-0000-0000-0000-000000000000", true); err != auth.ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestPGListInvitesIncludesInviter(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()

	inviter, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "inviter@example.com"}, emailIdentity("inviter@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if _, err := store.CreateInvite(ctx, auth.Invite{
		Email: "invitee@example.com", InvitedBy: inviter.ID, CreatedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}, hashOf("invite-1")); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	invites, err := store.ListInvites(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(invites) != 1 {
		t.Fatalf("len(invites) = %d, want 1", len(invites))
	}
	if invites[0].InvitedBy.Email != inviter.Email || invites[0].InvitedBy.ID != inviter.ID {
		t.Fatalf("InvitedBy = %+v, want id=%q email=%q", invites[0].InvitedBy, inviter.ID, inviter.Email)
	}
	if invites[0].AcceptedAt != nil {
		t.Fatalf("AcceptedAt = %v, want nil", invites[0].AcceptedAt)
	}
}
