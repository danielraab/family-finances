// Package storetest holds store-contract tests shared by the in-memory and
// PostgreSQL auth.Store implementations, so the memory store's emulation of
// database behaviour (cascades, uniqueness) is held to the same expectations
// as the real thing. It is imported only from _test.go files.
package storetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
)

// PasskeyContract runs the passkey and ceremony-challenge contract against
// stores built by newStore (each subtest gets a fresh, empty store).
func PasskeyContract(t *testing.T, newStore func(t *testing.T) auth.Store) {
	ctx := context.Background()
	base := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

	newUser := func(t *testing.T, s auth.Store, email string) auth.User {
		t.Helper()
		u, _, err := s.CreateUserWithIdentity(ctx, auth.NewUser{Email: email},
			auth.Identity{Kind: auth.IdentityEmail, Email: email, EmailVerified: true})
		if err != nil {
			t.Fatalf("create user %s: %v", email, err)
		}
		return u
	}
	newPasskey := func(t *testing.T, s auth.Store, userID, credID string, at time.Time) auth.Passkey {
		t.Helper()
		p, err := s.CreatePasskey(ctx, auth.Passkey{
			UserID:         userID,
			CredentialID:   []byte(credID),
			PublicKey:      []byte("pk-" + credID),
			SignCount:      3,
			Transports:     []string{"internal", "hybrid"},
			AAGUID:         []byte("0123456789abcdef"),
			BackupEligible: true,
			BackupState:    true,
			Name:           "Key " + credID,
			CreatedAt:      at,
		})
		if err != nil {
			t.Fatalf("create passkey %s: %v", credID, err)
		}
		return p
	}
	newSession := func(t *testing.T, s auth.Store, userID, passkeyID, token string) auth.Session {
		t.Helper()
		sess, err := s.CreateSession(ctx, auth.Session{
			UserID: userID, Client: auth.ClientWeb,
			CreatedAt: base, LastSeenAt: base, ExpiresAt: base.Add(time.Hour),
			PasskeyCredentialID: passkeyID,
		}, []byte(token+"-hash-padding-to-32-bytes......"))
		if err != nil {
			t.Fatalf("create session: %v", err)
		}
		return sess
	}
	sessionExists := func(t *testing.T, s auth.Store, token string) bool {
		t.Helper()
		_, err := s.SessionByTokenHash(ctx, []byte(token+"-hash-padding-to-32-bytes......"))
		if err != nil && !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("session lookup: %v", err)
		}
		return err == nil
	}

	t.Run("create and look up round-trips every field", func(t *testing.T) {
		s := newStore(t)
		u := newUser(t, s, "a@example.com")
		created := newPasskey(t, s, u.ID, "cred-1", base)
		if created.ID == "" {
			t.Fatal("created passkey has no id")
		}
		got, err := s.PasskeyByCredentialID(ctx, []byte("cred-1"))
		if err != nil {
			t.Fatalf("PasskeyByCredentialID: %v", err)
		}
		if got.ID != created.ID || got.UserID != u.ID || string(got.PublicKey) != "pk-cred-1" ||
			got.SignCount != 3 || len(got.Transports) != 2 || string(got.AAGUID) != "0123456789abcdef" ||
			!got.BackupEligible || !got.BackupState || got.Name != "Key cred-1" ||
			!got.CreatedAt.Equal(base) || got.LastUsedAt != nil {
			t.Fatalf("round-trip mismatch: %+v", got)
		}
		if _, err := s.PasskeyByCredentialID(ctx, []byte("nope")); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("unknown credential err = %v, want ErrNotFound", err)
		}
	})

	t.Run("duplicate credential id is rejected", func(t *testing.T) {
		s := newStore(t)
		a := newUser(t, s, "a@example.com")
		b := newUser(t, s, "b@example.com")
		newPasskey(t, s, a.ID, "cred-1", base)
		_, err := s.CreatePasskey(ctx, auth.Passkey{UserID: b.ID, CredentialID: []byte("cred-1"),
			PublicKey: []byte("x"), Name: "dup", CreatedAt: base})
		if !errors.Is(err, auth.ErrPasskeyConflict) {
			t.Fatalf("dup err = %v, want ErrPasskeyConflict", err)
		}
		got, _ := s.PasskeyByCredentialID(ctx, []byte("cred-1"))
		if got.UserID != a.ID {
			t.Fatal("existing passkey was changed by the rejected duplicate")
		}
	})

	t.Run("list is per user and oldest first", func(t *testing.T) {
		s := newStore(t)
		a := newUser(t, s, "a@example.com")
		b := newUser(t, s, "b@example.com")
		second := newPasskey(t, s, a.ID, "cred-2", base.Add(time.Minute))
		first := newPasskey(t, s, a.ID, "cred-1", base)
		newPasskey(t, s, b.ID, "cred-3", base)
		got, err := s.ListPasskeysByUser(ctx, a.ID)
		if err != nil {
			t.Fatalf("ListPasskeysByUser: %v", err)
		}
		if len(got) != 2 || got[0].ID != first.ID || got[1].ID != second.ID {
			t.Fatalf("list = %+v, want [%s %s]", got, first.ID, second.ID)
		}
	})

	t.Run("usage update records counter, backup state and time", func(t *testing.T) {
		s := newStore(t)
		u := newUser(t, s, "a@example.com")
		p := newPasskey(t, s, u.ID, "cred-1", base)
		used := base.Add(time.Hour)
		if err := s.UpdatePasskeyUsage(ctx, p.ID, 9, false, used); err != nil {
			t.Fatalf("UpdatePasskeyUsage: %v", err)
		}
		got, _ := s.PasskeyByCredentialID(ctx, []byte("cred-1"))
		if got.SignCount != 9 || got.BackupState || got.LastUsedAt == nil || !got.LastUsedAt.Equal(used) {
			t.Fatalf("after usage update: %+v", got)
		}
	})

	t.Run("deleting a passkey ends only its sessions", func(t *testing.T) {
		s := newStore(t)
		u := newUser(t, s, "a@example.com")
		pa := newPasskey(t, s, u.ID, "cred-a", base)
		pb := newPasskey(t, s, u.ID, "cred-b", base)
		newSession(t, s, u.ID, pa.ID, "s1")
		newSession(t, s, u.ID, pb.ID, "s2")
		newSession(t, s, u.ID, "", "s3")

		if err := s.DeletePasskey(ctx, u.ID, pa.ID); err != nil {
			t.Fatalf("DeletePasskey: %v", err)
		}
		if sessionExists(t, s, "s1") {
			t.Error("session created by the deleted passkey survived")
		}
		if !sessionExists(t, s, "s2") || !sessionExists(t, s, "s3") {
			t.Error("an unrelated session was revoked")
		}
		if _, err := s.PasskeyByCredentialID(ctx, []byte("cred-a")); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("deleted passkey still found: %v", err)
		}
	})

	t.Run("session records its passkey", func(t *testing.T) {
		s := newStore(t)
		u := newUser(t, s, "a@example.com")
		p := newPasskey(t, s, u.ID, "cred-a", base)
		newSession(t, s, u.ID, p.ID, "s1")
		newSession(t, s, u.ID, "", "s2")
		got, _ := s.SessionByTokenHash(ctx, []byte("s1-hash-padding-to-32-bytes......"))
		if got.PasskeyCredentialID != p.ID {
			t.Fatalf("PasskeyCredentialID = %q, want %q", got.PasskeyCredentialID, p.ID)
		}
		got, _ = s.SessionByTokenHash(ctx, []byte("s2-hash-padding-to-32-bytes......"))
		if got.PasskeyCredentialID != "" {
			t.Fatalf("non-passkey session PasskeyCredentialID = %q, want empty", got.PasskeyCredentialID)
		}
	})

	t.Run("cannot delete another user's or an unknown passkey", func(t *testing.T) {
		s := newStore(t)
		a := newUser(t, s, "a@example.com")
		b := newUser(t, s, "b@example.com")
		p := newPasskey(t, s, a.ID, "cred-a", base)
		if err := s.DeletePasskey(ctx, b.ID, p.ID); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("foreign delete err = %v, want ErrNotFound", err)
		}
		if err := s.DeletePasskey(ctx, a.ID, "not-an-id"); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("unknown delete err = %v, want ErrNotFound", err)
		}
		if _, err := s.PasskeyByCredentialID(ctx, []byte("cred-a")); err != nil {
			t.Fatalf("passkey gone after a refused delete: %v", err)
		}
	})

	t.Run("challenge consume is single-use and kind-checked", func(t *testing.T) {
		s := newStore(t)
		ch, err := s.CreateWebAuthnChallenge(ctx, auth.WebAuthnChallenge{
			Kind: auth.ChallengeLogin, Data: []byte(`{"challenge":"abc"}`), ExpiresAt: base.Add(5 * time.Minute),
		}, base)
		if err != nil {
			t.Fatalf("CreateWebAuthnChallenge: %v", err)
		}
		if _, err := s.ConsumeWebAuthnChallenge(ctx, ch.ID, auth.ChallengeRegistration); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("wrong-kind consume err = %v, want ErrNotFound", err)
		}
		got, err := s.ConsumeWebAuthnChallenge(ctx, ch.ID, auth.ChallengeLogin)
		if err != nil {
			t.Fatalf("consume: %v", err)
		}
		if got.SessionID != "" || !got.ExpiresAt.Equal(ch.ExpiresAt) || !jsonEqual(got.Data, []byte(`{"challenge":"abc"}`)) {
			t.Fatalf("consumed challenge = %+v", got)
		}
		if _, err := s.ConsumeWebAuthnChallenge(ctx, ch.ID, auth.ChallengeLogin); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("second consume err = %v, want ErrNotFound", err)
		}
		if _, err := s.ConsumeWebAuthnChallenge(ctx, "not-an-id", auth.ChallengeLogin); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("unknown consume err = %v, want ErrNotFound", err)
		}
	})

	t.Run("creating a challenge purges expired ones", func(t *testing.T) {
		s := newStore(t)
		old, _ := s.CreateWebAuthnChallenge(ctx, auth.WebAuthnChallenge{
			Kind: auth.ChallengeLogin, Data: []byte(`{}`), ExpiresAt: base.Add(time.Minute),
		}, base)
		live, _ := s.CreateWebAuthnChallenge(ctx, auth.WebAuthnChallenge{
			Kind: auth.ChallengeLogin, Data: []byte(`{}`), ExpiresAt: base.Add(10 * time.Minute),
		}, base)
		if _, err := s.CreateWebAuthnChallenge(ctx, auth.WebAuthnChallenge{
			Kind: auth.ChallengeLogin, Data: []byte(`{}`), ExpiresAt: base.Add(15 * time.Minute),
		}, base.Add(5*time.Minute)); err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := s.ConsumeWebAuthnChallenge(ctx, old.ID, auth.ChallengeLogin); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("expired challenge survived: %v", err)
		}
		if _, err := s.ConsumeWebAuthnChallenge(ctx, live.ID, auth.ChallengeLogin); err != nil {
			t.Fatalf("unexpired challenge was purged: %v", err)
		}
	})

	t.Run("delete expired challenges keeps live ones", func(t *testing.T) {
		s := newStore(t)
		old, _ := s.CreateWebAuthnChallenge(ctx, auth.WebAuthnChallenge{
			Kind: auth.ChallengeLogin, Data: []byte(`{}`), ExpiresAt: base.Add(time.Minute),
		}, base)
		live, _ := s.CreateWebAuthnChallenge(ctx, auth.WebAuthnChallenge{
			Kind: auth.ChallengeLogin, Data: []byte(`{}`), ExpiresAt: base.Add(10 * time.Minute),
		}, base)
		if err := s.DeleteExpiredWebAuthnChallenges(ctx, base.Add(5*time.Minute)); err != nil {
			t.Fatalf("DeleteExpiredWebAuthnChallenges: %v", err)
		}
		if _, err := s.ConsumeWebAuthnChallenge(ctx, old.ID, auth.ChallengeLogin); !errors.Is(err, auth.ErrNotFound) {
			t.Fatal("expired challenge survived")
		}
		if _, err := s.ConsumeWebAuthnChallenge(ctx, live.ID, auth.ChallengeLogin); err != nil {
			t.Fatalf("unexpired challenge removed: %v", err)
		}
	})

	t.Run("deleting a session discards its registration challenge", func(t *testing.T) {
		s := newStore(t)
		u := newUser(t, s, "a@example.com")
		sess := newSession(t, s, u.ID, "", "s1")
		ch, err := s.CreateWebAuthnChallenge(ctx, auth.WebAuthnChallenge{
			Kind: auth.ChallengeRegistration, SessionID: sess.ID, Data: []byte(`{}`), ExpiresAt: base.Add(time.Hour),
		}, base)
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if err := s.DeleteSessionByTokenHash(ctx, []byte("s1-hash-padding-to-32-bytes......")); err != nil {
			t.Fatalf("delete session: %v", err)
		}
		if _, err := s.ConsumeWebAuthnChallenge(ctx, ch.ID, auth.ChallengeRegistration); !errors.Is(err, auth.ErrNotFound) {
			t.Fatalf("registration challenge outlived its session: %v", err)
		}
	})
}

// jsonEqual compares two JSON documents ignoring whitespace differences that
// a jsonb round-trip introduces.
func jsonEqual(a, b []byte) bool {
	strip := func(in []byte) string {
		out := make([]byte, 0, len(in))
		for _, c := range in {
			if c != ' ' && c != '\n' && c != '\t' {
				out = append(out, c)
			}
		}
		return string(out)
	}
	return strip(a) == strip(b)
}
