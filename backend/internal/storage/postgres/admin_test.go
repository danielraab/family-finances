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

func TestPGSetUserDisplayName(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()

	u, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "name@example.com"}, emailIdentity("name@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	name := "Jane O'Brien-Doe"
	got, err := store.SetUserDisplayName(ctx, u.ID, &name)
	if err != nil {
		t.Fatalf("SetUserDisplayName: %v", err)
	}
	if got.DisplayName != name {
		t.Fatalf("DisplayName = %q, want %q", got.DisplayName, name)
	}

	// Clearing with nil returns an empty name and persists NULL.
	got, err = store.SetUserDisplayName(ctx, u.ID, nil)
	if err != nil {
		t.Fatalf("SetUserDisplayName(nil): %v", err)
	}
	if got.DisplayName != "" {
		t.Fatalf("DisplayName after clear = %q, want empty", got.DisplayName)
	}
	if reread, err := store.UserByID(ctx, u.ID); err != nil || reread.DisplayName != "" {
		t.Fatalf("UserByID after clear = %+v, %v", reread, err)
	}

	// An empty string clears too.
	empty := ""
	if got, err := store.SetUserDisplayName(ctx, u.ID, &empty); err != nil || got.DisplayName != "" {
		t.Fatalf("SetUserDisplayName(\"\") = %+v, %v", got, err)
	}
}

func TestPGSetUserDisplayNameMissing(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	name := "Nobody"

	if _, err := store.SetUserDisplayName(ctx, "00000000-0000-0000-0000-000000000000", &name); err != auth.ErrNotFound {
		t.Fatalf("unknown id err = %v, want ErrNotFound", err)
	}

	u, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "gone@example.com"}, emailIdentity("gone@example.com"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SoftDeleteUser(ctx, u.ID, time.Now()); err != nil {
		t.Fatalf("SoftDeleteUser: %v", err)
	}
	if _, err := store.SetUserDisplayName(ctx, u.ID, &name); err != auth.ErrNotFound {
		t.Fatalf("soft-deleted err = %v, want ErrNotFound", err)
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
