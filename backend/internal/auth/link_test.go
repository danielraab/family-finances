package auth_test

import (
	"testing"

	"at.draab/familyfinances/internal/auth"
)

func TestResolveLink(t *testing.T) {
	cases := []struct {
		name       string
		in         auth.LinkInput
		wantAction auth.LinkAction
		wantUserID string
	}{
		{
			name:       "magic link, unknown address -> create",
			in:         auth.LinkInput{Kind: auth.IdentityEmail, EmailVerified: true},
			wantAction: auth.ActionCreate,
		},
		{
			name:       "magic link, known address -> attach to that user",
			in:         auth.LinkInput{Kind: auth.IdentityEmail, EmailVerified: true, EmailMatchUserID: "u1"},
			wantAction: auth.ActionAttach,
			wantUserID: "u1",
		},
		{
			name:       "existing identity -> sign in as its owner",
			in:         auth.LinkInput{Kind: auth.IdentityOIDC, ExistingIdentityUserID: "u7", EmailMatchUserID: "u9"},
			wantAction: auth.ActionSignIn,
			wantUserID: "u7",
		},
		{
			name:       "oidc verified email matches a user -> attach",
			in:         auth.LinkInput{Kind: auth.IdentityOIDC, EmailVerified: true, EmailMatchUserID: "u2"},
			wantAction: auth.ActionAttach,
			wantUserID: "u2",
		},
		{
			name:       "oidc unverified email, no session -> create separate account",
			in:         auth.LinkInput{Kind: auth.IdentityOIDC, EmailVerified: false, EmailMatchUserID: "u2"},
			wantAction: auth.ActionCreate,
		},
		{
			name:       "authenticated, unattached identity -> attach to current user",
			in:         auth.LinkInput{Kind: auth.IdentityOIDC, EmailVerified: false, EmailMatchUserID: "u2", CurrentUserID: "u5"},
			wantAction: auth.ActionAttach,
			wantUserID: "u5",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action, userID := auth.ResolveLink(tc.in)
			if action != tc.wantAction {
				t.Fatalf("action = %d, want %d", action, tc.wantAction)
			}
			if userID != tc.wantUserID {
				t.Fatalf("userID = %q, want %q", userID, tc.wantUserID)
			}
		})
	}
}

func TestNormalizeAndValidateEmail(t *testing.T) {
	if got := auth.NormalizeEmail("  User@Example.COM "); got != "user@example.com" {
		t.Fatalf("NormalizeEmail = %q", got)
	}
	if err := auth.ValidateEmail("user@example.com"); err != nil {
		t.Fatalf("ValidateEmail(valid) = %v", err)
	}
	for _, bad := range []string{"", "no-at", "a@", "@b", "x y@z.com"} {
		if err := auth.ValidateEmail(bad); err == nil {
			t.Errorf("ValidateEmail(%q) = nil, want error", bad)
		}
	}
}

func TestDomainAllowed(t *testing.T) {
	if !auth.DomainAllowed("a@x.com", nil) {
		t.Error("empty list should allow any domain")
	}
	if !auth.DomainAllowed("a@X.com", []string{"x.com"}) {
		t.Error("case-insensitive match expected")
	}
	if auth.DomainAllowed("a@y.com", []string{"x.com"}) {
		t.Error("y.com should be rejected")
	}
}
