package main

import (
	"testing"
	"time"
)

type steppableClock struct{ t time.Time }

func (c *steppableClock) now() time.Time      { return c.t }
func (c *steppableClock) add(d time.Duration) { c.t = c.t.Add(d) }

func newTestLimiter(c *steppableClock) *loginLimiter {
	l := newLoginLimiter(c.now)
	return l
}

func TestLimiterLocksAtThreshold(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newTestLimiter(c)
	key := "acct:alice"

	for i := 0; i < l.maxFailures-1; i++ {
		if locked := l.fail(key); locked {
			t.Fatalf("locked after %d failures, want lock only at %d", i+1, l.maxFailures)
		}
		if l.locked(key) {
			t.Fatalf("key reported locked after %d failures, want unlocked", i+1)
		}
	}
	if locked := l.fail(key); !locked {
		t.Fatalf("fail #%d did not report a lock", l.maxFailures)
	}
	if !l.locked(key) {
		t.Fatal("key not locked after reaching the failure threshold")
	}
}

func TestLimiterResetOnSuccess(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newTestLimiter(c)
	key := "acct:bob"

	for i := 0; i < l.maxFailures-1; i++ {
		l.fail(key)
	}
	l.reset(key)
	if locked := l.fail(key); locked {
		t.Fatal("a single failure after reset locked the key; reset did not clear the count")
	}
	if l.locked(key) {
		t.Fatal("key locked after reset+1 failure")
	}
}

func TestLimiterIdleWindowResets(t *testing.T) {
	c := &steppableClock{t: time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)}
	l := newTestLimiter(c)
	key := "ip:203.0.113.7"

	for i := 0; i < l.maxFailures*3; i++ {
		l.fail(key)
		if l.locked(key) {
			t.Fatalf("key locked despite failures spaced beyond the window (iter %d)", i)
		}
		c.add(l.window + time.Second)
	}
}

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
	c.add(l.baseLockout + time.Second)
	if l.locked(key) {
		t.Fatal("key still locked after the lockout span elapsed")
	}
}
