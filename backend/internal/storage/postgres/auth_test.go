package postgres

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newAuthStore(t *testing.T) (*AuthStore, *pgxpool.Pool) {
	t.Helper()
	pool := newTestPool(t)
	if err := Migrate(context.Background(), pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return NewAuthStore(pool), pool
}

func hashOf(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

func emailIdentity(email string) auth.Identity {
	return auth.Identity{Kind: auth.IdentityEmail, Email: email, EmailVerified: true}
}

func TestPGCreateUserBootstrapAdmin(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()

	u1, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "first@example.com"}, emailIdentity("first@example.com"))
	if err != nil {
		t.Fatalf("create first: %v", err)
	}
	if !u1.IsAdmin {
		t.Fatal("first user should be admin")
	}

	u2, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "second@example.com"}, emailIdentity("second@example.com"))
	if err != nil {
		t.Fatalf("create second: %v", err)
	}
	if u2.IsAdmin {
		t.Fatal("second user should not be admin")
	}

	// Duplicate email identity -> conflict.
	if _, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "third@example.com"}, emailIdentity("second@example.com")); !errors.Is(err, auth.ErrIdentityConflict) {
		t.Fatalf("dup identity err = %v, want ErrIdentityConflict", err)
	}
}

func TestPGAddIdentityAndConflict(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()

	u, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "u@example.com"}, emailIdentity("u@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	oidcID := auth.Identity{UserID: u.ID, Kind: auth.IdentityOIDC, Provider: "https://idp", Subject: "sub-1", Email: "u@example.com", EmailVerified: true}
	if _, err := store.AddIdentity(ctx, oidcID); err != nil {
		t.Fatalf("AddIdentity: %v", err)
	}
	if _, err := store.AddIdentity(ctx, oidcID); !errors.Is(err, auth.ErrIdentityConflict) {
		t.Fatalf("dup oidc identity err = %v, want ErrIdentityConflict", err)
	}

	got, err := store.IdentityByProviderSubject(ctx, "https://idp", "sub-1")
	if err != nil || got.UserID != u.ID {
		t.Fatalf("lookup = %+v err %v", got, err)
	}

	// Unknown user -> ErrNotFound.
	if _, err := store.AddIdentity(ctx, auth.Identity{UserID: "00000000-0000-0000-0000-000000000000", Kind: auth.IdentityEmail, Email: "x@example.com"}); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("orphan identity err = %v, want ErrNotFound", err)
	}
}

func TestPGSessionLifecycle(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	u, _, _ := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "s@example.com"}, emailIdentity("s@example.com"))

	now := time.Now().UTC().Truncate(time.Second)
	hash := hashOf("session-token-1")
	sess, err := store.CreateSession(ctx, auth.Session{
		UserID: u.ID, Client: auth.ClientAPI, UserAgent: "test-agent", IP: "203.0.113.5",
		CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour),
	}, hash)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	got, err := store.SessionByTokenHash(ctx, hash)
	if err != nil {
		t.Fatalf("SessionByTokenHash: %v", err)
	}
	if got.UserID != u.ID || got.Client != auth.ClientAPI || got.IP != "203.0.113.5" {
		t.Fatalf("session round-trip mismatch: %+v", got)
	}

	if err := store.TouchSession(ctx, sess.ID, now.Add(time.Minute), now.Add(2*time.Hour)); err != nil {
		t.Fatalf("TouchSession: %v", err)
	}
	if err := store.DeleteSessionByTokenHash(ctx, hash); err != nil {
		t.Fatalf("DeleteSessionByTokenHash: %v", err)
	}
	if _, err := store.SessionByTokenHash(ctx, hash); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("post-delete lookup err = %v, want ErrNotFound", err)
	}
	if err := store.DeleteSessionByTokenHash(ctx, hash); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("re-delete err = %v, want ErrNotFound", err)
	}
}

func TestPGMagicLinkConsume(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	ok := hashOf("mlt-ok")
	if err := store.CreateMagicLinkToken(ctx, ok, "a@example.com", now.Add(15*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if email, err := store.ConsumeMagicLinkToken(ctx, ok, now); err != nil || email != "a@example.com" {
		t.Fatalf("first consume: email %q err %v", email, err)
	}
	if _, err := store.ConsumeMagicLinkToken(ctx, ok, now); !errors.Is(err, auth.ErrTokenConsumed) {
		t.Fatalf("second consume err = %v, want ErrTokenConsumed", err)
	}

	exp := hashOf("mlt-exp")
	if err := store.CreateMagicLinkToken(ctx, exp, "b@example.com", now.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConsumeMagicLinkToken(ctx, exp, now); !errors.Is(err, auth.ErrTokenExpired) {
		t.Fatalf("expired consume err = %v, want ErrTokenExpired", err)
	}

	if _, err := store.ConsumeMagicLinkToken(ctx, hashOf("missing"), now); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("missing consume err = %v, want ErrTokenInvalid", err)
	}
}

func TestPGInviteFlow(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	admin, _, _ := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: "admin@example.com"}, emailIdentity("admin@example.com"))

	tok := hashOf("invite-1")
	inv, err := store.CreateInvite(ctx, auth.Invite{
		Email: "guest@example.com", InvitedBy: admin.ID, CreatedAt: now, ExpiresAt: now.Add(48 * time.Hour),
	}, tok)
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	active, err := store.ActiveInviteForEmail(ctx, "guest@example.com", now)
	if err != nil || active.ID != inv.ID {
		t.Fatalf("ActiveInviteForEmail = %+v err %v", active, err)
	}

	consumed, err := store.ConsumeInvite(ctx, tok, now)
	if err != nil || consumed.Email != "guest@example.com" {
		t.Fatalf("ConsumeInvite = %+v err %v", consumed, err)
	}
	if err := store.MarkInviteAcceptedBy(ctx, consumed.ID, admin.ID, now); err != nil {
		t.Fatalf("MarkInviteAcceptedBy: %v", err)
	}
	if _, err := store.ConsumeInvite(ctx, tok, now); !errors.Is(err, auth.ErrTokenConsumed) {
		t.Fatalf("re-consume err = %v, want ErrTokenConsumed", err)
	}
	if _, err := store.ActiveInviteForEmail(ctx, "guest@example.com", now); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("accepted invite still active: %v", err)
	}
}

func TestPGOIDCState(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()
	now := time.Now().UTC()

	st := auth.OIDCState{State: "state-1", Nonce: "n", PKCEVerifier: "v", Provider: "https://idp", ReturnTo: "/home", ExpiresAt: now.Add(10 * time.Minute)}
	if err := store.CreateOIDCState(ctx, st); err != nil {
		t.Fatal(err)
	}
	got, err := store.ConsumeOIDCState(ctx, "state-1", now)
	if err != nil || got.Nonce != "n" || got.PKCEVerifier != "v" || got.ReturnTo != "/home" {
		t.Fatalf("ConsumeOIDCState = %+v err %v", got, err)
	}
	if _, err := store.ConsumeOIDCState(ctx, "state-1", now); !errors.Is(err, auth.ErrTokenInvalid) {
		t.Fatalf("re-consume err = %v, want ErrTokenInvalid", err)
	}

	exp := auth.OIDCState{State: "state-2", Nonce: "n", PKCEVerifier: "v", Provider: "p", ExpiresAt: now.Add(-time.Minute)}
	if err := store.CreateOIDCState(ctx, exp); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConsumeOIDCState(ctx, "state-2", now); !errors.Is(err, auth.ErrTokenExpired) {
		t.Fatalf("expired consume err = %v, want ErrTokenExpired", err)
	}
}

func TestPGBootstrapAdminRace(t *testing.T) {
	store, _ := newAuthStore(t)
	ctx := context.Background()

	const n = 8
	var wg sync.WaitGroup
	admins := make([]bool, n)
	errs := make([]error, n)
	start := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			email := fmt.Sprintf("racer%d@example.com", i)
			u, _, err := store.CreateUserWithIdentity(ctx, auth.NewUser{Email: email}, emailIdentity(email))
			errs[i] = err
			admins[i] = err == nil && u.IsAdmin
		}(i)
	}
	close(start)
	wg.Wait()

	adminCount := 0
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("racer %d: %v", i, errs[i])
		}
		if admins[i] {
			adminCount++
		}
	}
	if adminCount != 1 {
		t.Fatalf("admin count = %d, want exactly 1 (bootstrap race not serialized)", adminCount)
	}
}
