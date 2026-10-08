package ratelimit_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"at.draab/familyfinances/internal/ratelimit"
)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newClock() *clock { return &clock{t: time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)} }

func TestAllowUpToLimitThenRefuse(t *testing.T) {
	c := newClock()
	l := ratelimit.New(3, time.Minute, c.now)
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("k"); !ok {
			t.Fatalf("attempt %d refused", i+1)
		}
		c.advance(10 * time.Second)
	}
	ok, retry := l.Allow("k")
	if ok {
		t.Fatal("attempt 4 allowed")
	}
	// The first attempt was 30s ago; it leaves the 1m window in 30s.
	if retry != 30*time.Second {
		t.Fatalf("retryAfter = %s, want 30s", retry)
	}
}

func TestRecoversAfterWindowWithoutBoundaryBurst(t *testing.T) {
	c := newClock()
	l := ratelimit.New(2, time.Minute, c.now)
	l.Allow("k")
	c.advance(50 * time.Second)
	l.Allow("k")
	c.advance(11 * time.Second) // first attempt has left the window, second has not
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("refused after the oldest attempt expired")
	}
	if ok, _ := l.Allow("k"); ok {
		t.Fatal("allowed a third attempt inside one window")
	}
}

func TestRefusedAttemptsAreNotRecorded(t *testing.T) {
	c := newClock()
	l := ratelimit.New(1, time.Minute, c.now)
	l.Allow("k")
	for i := 0; i < 10; i++ {
		c.advance(5 * time.Second)
		l.Allow("k") // refused, must not extend the lockout
	}
	c.advance(11 * time.Second) // 61s after the only recorded attempt
	if ok, _ := l.Allow("k"); !ok {
		t.Fatal("refused attempts extended the window")
	}
}

func TestKeysAreIndependent(t *testing.T) {
	l := ratelimit.New(1, time.Minute, newClock().now)
	l.Allow("a")
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("key b throttled by key a")
	}
}

func TestEvictRemovesOnlyIdleKeys(t *testing.T) {
	c := newClock()
	l := ratelimit.New(5, time.Minute, c.now)
	l.Allow("idle")
	c.advance(50 * time.Second)
	l.Allow("active")
	c.advance(20 * time.Second) // idle: 70s ago; active: 20s ago
	l.Evict()
	if l.Len() != 1 {
		t.Fatalf("Len = %d, want 1", l.Len())
	}
	for i := 0; i < 4; i++ {
		l.Allow("active")
	}
	if ok, _ := l.Allow("active"); ok {
		t.Fatal("eviction dropped the active key's count")
	}
}

func TestConcurrentUse(t *testing.T) {
	l := ratelimit.New(100, time.Minute, nil)
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if ok, _ := l.Allow("shared"); ok {
					mu.Lock()
					allowed++
					mu.Unlock()
				}
				l.Allow(fmt.Sprintf("own-%d", g))
				if i%10 == 0 {
					l.Evict()
				}
			}
		}(g)
	}
	wg.Wait()
	if allowed != 100 {
		t.Fatalf("allowed %d attempts on the shared key, want exactly 100", allowed)
	}
}
