package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeCleaner struct {
	mu    sync.Mutex
	calls int
	fail  bool
}

func (f *fakeCleaner) Cleanup(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.fail {
		return errors.New("database is down")
	}
	return nil
}

func (f *fakeCleaner) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func TestRunCleanupRunsAtStartThenPeriodicallyAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	c := &fakeCleaner{fail: true} // failures must not stop the loop
	done := make(chan struct{})
	go func() {
		runCleanup(ctx, c, 10*time.Millisecond)
		close(done)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for c.count() < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if c.count() < 3 {
		t.Fatalf("ran %d passes, want at least 3 (startup + ticks despite failures)", c.count())
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runCleanup did not stop on cancel")
	}
	after := c.count()
	time.Sleep(50 * time.Millisecond)
	if c.count() != after {
		t.Fatal("a pass ran after cancellation")
	}
}

func TestRunCleanupFirstPassIsImmediate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &fakeCleaner{}
	go runCleanup(ctx, c, time.Hour)
	deadline := time.Now().Add(time.Second)
	for c.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if c.count() != 1 {
		t.Fatalf("passes = %d, want the startup pass before the first hourly tick", c.count())
	}
}
