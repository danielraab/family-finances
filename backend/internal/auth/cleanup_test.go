package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"testing"
	"time"

	"at.draab/familyfinances/internal/auth"
	"at.draab/familyfinances/internal/storage/memory"
)

// cleanupStore records which cleanup deletes ran and fails the magic-link
// one; everything else is the in-memory store.
type cleanupStore struct {
	*memory.AuthStore
	calls         []string
	sessionsArgs  [2]time.Time
	failMagicLink bool
}

func (c *cleanupStore) DeleteExpiredSessions(ctx context.Context, now, createdBefore time.Time) error {
	c.calls = append(c.calls, "sessions")
	c.sessionsArgs = [2]time.Time{now, createdBefore}
	return c.AuthStore.DeleteExpiredSessions(ctx, now, createdBefore)
}

func (c *cleanupStore) DeleteStaleMagicLinkTokens(ctx context.Context, now time.Time) error {
	c.calls = append(c.calls, "magic")
	if c.failMagicLink {
		return errors.New("database is down")
	}
	return c.AuthStore.DeleteStaleMagicLinkTokens(ctx, now)
}

func (c *cleanupStore) DeleteExpiredOIDCState(ctx context.Context, now time.Time) error {
	c.calls = append(c.calls, "oidc")
	return c.AuthStore.DeleteExpiredOIDCState(ctx, now)
}

func (c *cleanupStore) DeleteExpiredWebAuthnChallenges(ctx context.Context, now time.Time) error {
	c.calls = append(c.calls, "challenges")
	return c.AuthStore.DeleteExpiredWebAuthnChallenges(ctx, now)
}

type countingEvicter struct{ n int }

func (e *countingEvicter) Evict() { e.n++ }

func TestCleanupRunsEveryStepDespiteAFailure(t *testing.T) {
	store := &cleanupStore{AuthStore: memory.NewAuthStore(), failMagicLink: true}
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	ev := &countingEvicter{}
	p := baseParams()
	svc := auth.NewService(store, &stubMailer{}, nil, p,
		auth.WithClock(func() time.Time { return now }), auth.WithEvicters(ev))

	err := svc.Cleanup(context.Background())
	if err == nil || !strings.Contains(err.Error(), "magic-link tokens") || !strings.Contains(err.Error(), "database is down") {
		t.Fatalf("Cleanup err = %v, want the magic-link failure reported", err)
	}
	if got := strings.Join(store.calls, ","); got != "sessions,magic,oidc,challenges" {
		t.Fatalf("steps run = %s, want all four", got)
	}
	if ev.n != 1 {
		t.Fatalf("evicter ran %d times, want 1", ev.n)
	}
	if !store.sessionsArgs[0].Equal(now) || !store.sessionsArgs[1].Equal(now.Add(-p.SessionMaxTTL)) {
		t.Fatalf("DeleteExpiredSessions(%v, %v), want now and now-SessionMaxTTL", store.sessionsArgs[0], store.sessionsArgs[1])
	}
}

func TestCleanupSucceedsAndKeepsLiveSessions(t *testing.T) {
	hr := newHarness(t)
	tok := hr.signIn(t, "alice@example.com", false)
	if err := hr.svc.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if !hr.authenticates(tok) {
		t.Fatal("cleanup removed a live session")
	}
	hr.clock.advance(31 * 24 * time.Hour) // past the 720h sliding TTL
	if err := hr.svc.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if _, err := hr.store.SessionByTokenHash(context.Background(), hashForTest(tok)); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("expired session still stored: %v", err)
	}
}

// hashForTest mirrors how the service stores a session token: its SHA-256.
func hashForTest(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
