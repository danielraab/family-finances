package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
)

// CleanupContract checks the cleanup deletes: what can never be used again
// goes, everything else stays.
func CleanupContract(t *testing.T, newStore func(t *testing.T) auth.Store) {
	ctx := context.Background()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	maxAge := 90 * 24 * time.Hour

	hash := func(s string) []byte { return []byte(s + "-hash-padding-to-32-bytes......") }

	t.Run("sessions: expired and over-age go, active stays", func(t *testing.T) {
		s := newStore(t)
		u, _, err := s.CreateUserWithIdentity(ctx, auth.NewUser{Email: "a@example.com"},
			auth.Identity{Kind: auth.IdentityEmail, Email: "a@example.com", EmailVerified: true})
		if err != nil {
			t.Fatal(err)
		}
		mk := func(token string, created, expires time.Time) {
			if _, err := s.CreateSession(ctx, auth.Session{UserID: u.ID, Client: auth.ClientWeb,
				CreatedAt: created, LastSeenAt: created, ExpiresAt: expires}, hash(token)); err != nil {
				t.Fatal(err)
			}
		}
		mk("expired", now.Add(-48*time.Hour), now.Add(-time.Minute))
		mk("overage", now.Add(-maxAge-time.Hour), now.Add(time.Hour)) // sliding expiry still ahead
		mk("active", now.Add(-time.Hour), now.Add(time.Hour))

		if err := s.DeleteExpiredSessions(ctx, now, now.Add(-maxAge)); err != nil {
			t.Fatalf("DeleteExpiredSessions: %v", err)
		}
		for token, want := range map[string]bool{"expired": false, "overage": false, "active": true} {
			_, err := s.SessionByTokenHash(ctx, hash(token))
			if got := err == nil; got != want {
				t.Errorf("session %s present = %v, want %v (err %v)", token, got, want, err)
			}
		}
		if _, err := s.IdentityByEmail(ctx, "a@example.com"); err != nil {
			t.Errorf("identity removed: %v", err)
		}
		if _, err := s.UserByID(ctx, u.ID); err != nil {
			t.Errorf("user removed: %v", err)
		}
	})

	t.Run("magic-link tokens: consumed and expired go, live stays", func(t *testing.T) {
		s := newStore(t)
		for token, expires := range map[string]time.Time{
			"consumed": now.Add(time.Hour), "expired": now.Add(-time.Minute), "live": now.Add(time.Hour),
		} {
			if err := s.CreateMagicLinkToken(ctx, hash(token), "a@example.com", expires); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := s.ConsumeMagicLinkToken(ctx, hash("consumed"), now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := s.DeleteStaleMagicLinkTokens(ctx, now); err != nil {
			t.Fatalf("DeleteStaleMagicLinkTokens: %v", err)
		}
		// A deleted token reads as unknown (invalid), not as consumed/expired.
		for _, token := range []string{"consumed", "expired"} {
			if _, err := s.ConsumeMagicLinkToken(ctx, hash(token), now); !errors.Is(err, auth.ErrTokenInvalid) {
				t.Errorf("token %s: err = %v, want ErrTokenInvalid (deleted)", token, err)
			}
		}
		if _, err := s.ConsumeMagicLinkToken(ctx, hash("live"), now); err != nil {
			t.Errorf("live token removed: %v", err)
		}
	})

	t.Run("oidc state: expired goes, live stays", func(t *testing.T) {
		s := newStore(t)
		for state, expires := range map[string]time.Time{"old": now.Add(-time.Minute), "live": now.Add(time.Minute)} {
			if err := s.CreateOIDCState(ctx, auth.OIDCState{State: state, Nonce: "n", PKCEVerifier: "v",
				Provider: "p", ExpiresAt: expires}); err != nil {
				t.Fatal(err)
			}
		}
		if err := s.DeleteExpiredOIDCState(ctx, now); err != nil {
			t.Fatalf("DeleteExpiredOIDCState: %v", err)
		}
		if _, err := s.ConsumeOIDCState(ctx, "old", now); !errors.Is(err, auth.ErrTokenInvalid) {
			t.Errorf("old state: err = %v, want ErrTokenInvalid (deleted)", err)
		}
		if _, err := s.ConsumeOIDCState(ctx, "live", now); err != nil {
			t.Errorf("live state removed: %v", err)
		}
	})

	t.Run("invites are never touched", func(t *testing.T) {
		s := newStore(t)
		u, _, _ := s.CreateUserWithIdentity(ctx, auth.NewUser{Email: "a@example.com"},
			auth.Identity{Kind: auth.IdentityEmail, Email: "a@example.com", EmailVerified: true})
		if _, err := s.CreateInvite(ctx, auth.Invite{Email: "b@example.com", InvitedBy: u.ID,
			CreatedAt: now.Add(-30 * 24 * time.Hour), ExpiresAt: now.Add(-20 * 24 * time.Hour)}, hash("invite")); err != nil {
			t.Fatal(err)
		}
		_ = s.DeleteExpiredSessions(ctx, now, now.Add(-maxAge))
		_ = s.DeleteStaleMagicLinkTokens(ctx, now)
		_ = s.DeleteExpiredOIDCState(ctx, now)
		_ = s.DeleteExpiredWebAuthnChallenges(ctx, now)
		list, err := s.ListInvites(ctx)
		if err != nil || len(list) != 1 {
			t.Fatalf("invites after cleanup = %d (%v), want the expired one kept", len(list), err)
		}
	})
}
