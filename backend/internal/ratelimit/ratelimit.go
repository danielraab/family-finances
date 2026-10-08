// Package ratelimit is an in-process, per-key sliding-window rate limiter:
// at most Limit attempts per key in any Window. It keeps a small log of
// attempt times per key, so the limit is exact — no burst at window
// boundaries — and costs at most Limit timestamps per active key. State is
// per process: it resets on restart and is not shared between replicas.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter limits attempts per key. It is safe for concurrent use.
type Limiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu      sync.Mutex
	entries map[string][]time.Time // oldest first, at most limit entries
}

// New returns a Limiter allowing limit attempts per key per window. now is
// the clock (time.Now when nil).
func New(limit int, window time.Duration, now func() time.Time) *Limiter {
	if limit < 1 {
		limit = 1
	}
	if now == nil {
		now = time.Now
	}
	return &Limiter{limit: limit, window: window, now: now, entries: map[string][]time.Time{}}
}

// Allow records an attempt for key and reports whether it is within the
// limit. A refused attempt is not recorded, and retryAfter says how long
// until the oldest attempt in the window expires.
func (l *Limiter) Allow(key string) (ok bool, retryAfter time.Duration) {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	log := prune(l.entries[key], now.Add(-l.window))
	if len(log) >= l.limit {
		l.entries[key] = log
		return false, log[0].Add(l.window).Sub(now)
	}
	l.entries[key] = append(log, now)
	return true, 0
}

// Evict drops every key with no attempt inside the window, keeping memory
// proportional to recently active keys. Evicting never loosens a limit: an
// evicted key had nothing left to count.
func (l *Limiter) Evict() {
	cutoff := l.now().Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, log := range l.entries {
		if len(log) == 0 || !log[len(log)-1].After(cutoff) {
			delete(l.entries, key)
		}
	}
}

// Len is the number of keys currently tracked.
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// prune drops attempts at or before cutoff, reusing log's backing array.
func prune(log []time.Time, cutoff time.Time) []time.Time {
	i := 0
	for i < len(log) && !log[i].After(cutoff) {
		i++
	}
	if i == 0 {
		return log
	}
	return append(log[:0], log[i:]...)
}
