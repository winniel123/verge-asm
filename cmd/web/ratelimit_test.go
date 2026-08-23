package main

import (
	"testing"
	"time"
)

// steppableClock is a hand-advanced clock for the limiter tests: the limiter reads
// now() on every touch, so a test moves time forward deliberately to cross the
// window and lockout boundaries without sleeping.
type steppableClock struct{ t time.Time }

func (c *steppableClock) now() time.Time      { return c.t }
func (c *steppableClock) add(d time.Duration) { c.t = c.t.Add(d) }

func newTestLimiter(c *steppableClock) *loginLimiter {
	l := newLoginLimiter(c.now)
	return l
}

// TestLimiterLocksAtThreshold is the core #322 guarantee: the (maxFailures)th miss
// locks the key, and every attempt while locked is refused regardless of anything
// the caller then presents.
func TestLimiterLocksAtThreshold(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newTestLimiter(c)
	key := "acct:alice"

	// Up to the threshold the key is not yet locked.
	for i := 0; i < l.maxFailures-1; i++ {
		if locked := l.fail(key); locked {
			t.Fatalf("locked after %d failures, want lock only at %d", i+1, l.maxFailures)
		}
		if l.locked(key) {
			t.Fatalf("key reported locked after %d failures, want unlocked", i+1)
		}
	}
	// The threshold failure locks it.
	if locked := l.fail(key); !locked {
		t.Fatalf("fail #%d did not report a lock", l.maxFailures)
	}
	if !l.locked(key) {
		t.Fatal("key not locked after reaching the failure threshold")
	}
}

// TestLimiterResetOnSuccess covers the success path: a legitimate operator who
// mistyped a few times, then authenticated, starts clean.
func TestLimiterResetOnSuccess(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newTestLimiter(c)
	key := "acct:bob"

	for i := 0; i < l.maxFailures-1; i++ {
		l.fail(key)
	}
	l.reset(key)
	// After a reset the count is clear: one more failure must not lock (it would
	// have been the threshold failure without the reset).
	if locked := l.fail(key); locked {
		t.Fatal("a single failure after reset locked the key; reset did not clear the count")
	}
	if l.locked(key) {
		t.Fatal("key locked after reset+1 failure")
	}
}

// TestLimiterIdleWindowResets covers the time-based reset: a slow trickle of misses
// spaced beyond the window never accretes into a lockout.
func TestLimiterIdleWindowResets(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newTestLimiter(c)
	key := "ip:203.0.113.7"

	for i := 0; i < l.maxFailures*3; i++ {
		l.fail(key)
		if l.locked(key) {
			t.Fatalf("key locked despite failures spaced beyond the window (iter %d)", i)
		}
		// Advance past the idle window so the next failure starts a fresh count.
		c.add(l.window + time.Second)
	}
}

// TestLimiterUnlocksAfterLockout covers recovery: once the lockout span elapses the
// key is usable again, so a locked-out operator is never locked out forever.
func TestLimiterUnlocksAfterLockout(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newTestLimiter(c)
	key := "acct:carol"

	for i := 0; i < l.maxFailures; i++ {
		l.fail(key)
	}
	if !l.locked(key) {
		t.Fatal("key not locked after threshold")
	}
	// Move just past the base lockout: the key clears.
	c.add(l.baseLockout + time.Second)
	if l.locked(key) {
		t.Fatal("key still locked after the lockout span elapsed")
	}
}
