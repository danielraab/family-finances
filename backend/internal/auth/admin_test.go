package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"at.draab/familyfinances/internal/auth"
)

func TestDisableUserRevokesSessionImmediately(t *testing.T) {
	svc, _, mailer, _ := newSvc(t, baseParams())
	user, tok := signInEmail(t, svc, mailer, "person@example.com")

	got, err := svc.DisableUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("DisableUser: %v", err)
	}
	if !got.Disabled {
		t.Fatal("DisableUser did not set Disabled")
	}

	if _, err := svc.Authenticate(context.Background(), tok); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("Authenticate after disable = %v, want ErrNotFound (session revoked)", err)
	}
}

// TestAuthenticateRejectsDisabledEvenWithSurvivingSession exercises the
// middleware's belt-and-suspenders check directly against the store, without
// going through DisableUser's own session revocation — simulating a session
// row that outlives the disable for whatever reason.
func TestAuthenticateRejectsDisabledEvenWithSurvivingSession(t *testing.T) {
	svc, store, mailer, _ := newSvc(t, baseParams())
	user, tok := signInEmail(t, svc, mailer, "person@example.com")

	if err := store.SetUserDisabled(context.Background(), user.ID, true); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Authenticate(context.Background(), tok); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("Authenticate = %v, want ErrNotFound", err)
	}
}

func TestEnableUserDoesNotRestoreSession(t *testing.T) {
	svc, _, mailer, _ := newSvc(t, baseParams())
	user, tok := signInEmail(t, svc, mailer, "person@example.com")

	if _, err := svc.DisableUser(context.Background(), user.ID); err != nil {
		t.Fatalf("DisableUser: %v", err)
	}
	got, err := svc.EnableUser(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("EnableUser: %v", err)
	}
	if got.Disabled {
		t.Fatal("EnableUser did not clear Disabled")
	}
	if _, err := svc.Authenticate(context.Background(), tok); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("old token should still be rejected after enable, got %v", err)
	}
}

func TestDisabledUserCannotSignInAgain(t *testing.T) {
	p := baseParams()
	oidc := &stubOIDC{claims: auth.OIDCClaims{Issuer: "https://idp.example", Subject: "sub-1", Email: "person@example.com", EmailVerified: true}}
	svc, _, mailer, _ := newSvc(t, p, withOIDC(oidc))

	user, _ := signInEmail(t, svc, mailer, "person@example.com")
	if _, err := svc.DisableUser(context.Background(), user.ID); err != nil {
		t.Fatalf("DisableUser: %v", err)
	}

	// Magic link: no new email for a disabled address.
	before := len(mailer.sent)
	if err := svc.StartEmailLogin(context.Background(), "person@example.com"); err != nil {
		t.Fatalf("StartEmailLogin: %v", err)
	}
	if len(mailer.sent) != before {
		t.Fatal("StartEmailLogin sent a mail for a disabled account")
	}

	// OIDC: the callback itself rejects, since email/start never even fires.
	redirect, err := svc.StartOIDC(context.Background(), "")
	if err != nil {
		t.Fatalf("StartOIDC: %v", err)
	}
	state := redirect[strings.Index(redirect, "state=")+len("state="):]
	if _, _, _, err := svc.CompleteOIDC(context.Background(), state, "code", "", auth.SessionContext{}); !errors.Is(err, auth.ErrAccountDisabled) {
		t.Fatalf("CompleteOIDC = %v, want ErrAccountDisabled", err)
	}
}

func TestSoftDeletedUserExcludedFromListingAndCannotSignIn(t *testing.T) {
	svc, _, mailer, _ := newSvc(t, baseParams())
	user, _ := signInEmail(t, svc, mailer, "person@example.com")

	if err := svc.SoftDeleteUser(context.Background(), user.ID); err != nil {
		t.Fatalf("SoftDeleteUser: %v", err)
	}

	users, err := svc.ListUsers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range users {
		if u.ID == user.ID {
			t.Fatal("soft-deleted user still listed")
		}
	}

	before := len(mailer.sent)
	if err := svc.StartEmailLogin(context.Background(), "person@example.com"); err != nil {
		t.Fatalf("StartEmailLogin: %v", err)
	}
	if len(mailer.sent) != before {
		t.Fatal("StartEmailLogin sent a mail for a soft-deleted account")
	}
}

func TestAdminMayDisableOrDeleteThemselvesAsLastAdmin(t *testing.T) {
	svc, _, mailer, _ := newSvc(t, baseParams())
	// First (and only) account is the bootstrap admin.
	admin, _ := signInEmail(t, svc, mailer, "admin@example.com")
	if !admin.IsAdmin {
		t.Fatal("bootstrap account should be admin")
	}

	if _, err := svc.DisableUser(context.Background(), admin.ID); err != nil {
		t.Fatalf("admin disabling themselves should succeed, got %v", err)
	}
	if err := svc.SoftDeleteUser(context.Background(), admin.ID); err != nil {
		t.Fatalf("admin deleting themselves should succeed, got %v", err)
	}
}

func TestListInvitesIncludesInviter(t *testing.T) {
	svc, _, mailer, _ := newSvc(t, baseParams())
	inviter, _ := signInEmail(t, svc, mailer, "inviter@example.com")

	if _, err := svc.CreateInvite(context.Background(), inviter.ID, "invitee@example.com"); err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	invites, err := svc.ListInvites(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(invites) != 1 {
		t.Fatalf("len(invites) = %d, want 1", len(invites))
	}
	if invites[0].InvitedBy.Email != inviter.Email {
		t.Fatalf("InvitedBy.Email = %q, want %q", invites[0].InvitedBy.Email, inviter.Email)
	}
	if invites[0].Email != "invitee@example.com" {
		t.Fatalf("Email = %q, want invitee@example.com", invites[0].Email)
	}
}
