package main

import (
	"strings"
	"sync"
	"time"
)

// A 6-digit TOTP is brute-forceable: three of a million codes are live each instant (#322).

type loginLimiter struct {
	now func() time.Time

	maxFailures int
	window      time.Duration
	baseLockout time.Duration
	maxLockout  time.Duration

	acctPrefix      string
	acctLockCeiling time.Duration

	mu      sync.Mutex
	entries map[string]*limiterEntry
}

type limiterEntry struct {
	failures      int
	lastFailure   time.Time
	lockedUntil   time.Time
	firstLockedAt time.Time
}

func newLoginLimiter(now func() time.Time) *loginLimiter {
	return &loginLimiter{
		now:             now,
		maxFailures:     5,
		window:          5 * time.Minute,
		baseLockout:     5 * time.Minute,
		maxLockout:      time.Hour,
		acctPrefix:      "acct:",
		acctLockCeiling: 15 * time.Minute,
		entries:         map[string]*limiterEntry{},
	}
}

func (l *loginLimiter) locked(keys ...string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for _, k := range keys {
		e := l.entries[k]
		if e == nil || !now.Before(e.lockedUntil) {
			continue
		}
		// A capped account lock stops a pre-auth attacker locking a known username out for good.
		if l.acctLockCeiling > 0 && l.acctPrefix != "" && strings.HasPrefix(k, l.acctPrefix) &&
			!e.firstLockedAt.IsZero() && !now.Before(e.firstLockedAt.Add(l.acctLockCeiling)) {
			continue
		}
		return true
	}
	return false
}

func (l *loginLimiter) fail(keys ...string) (nowLocked bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for _, k := range keys {
		e := l.entries[k]
		if e == nil {
			e = &limiterEntry{}
			l.entries[k] = e
		}
		if !now.Before(e.lockedUntil) && !e.lastFailure.IsZero() && now.Sub(e.lastFailure) > l.window {
			e.failures = 0
			e.firstLockedAt = time.Time{}
		}
		e.failures++
		e.lastFailure = now
		if e.failures >= l.maxFailures {
			// Anchoring to the first lock stops repeated re-locks sliding the ceiling forward.
			if e.firstLockedAt.IsZero() {
				e.firstLockedAt = now
			}
			e.lockedUntil = now.Add(l.lockoutFor(e.failures))
			nowLocked = true
		}
	}
	return nowLocked
}

func (l *loginLimiter) reset(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		delete(l.entries, k)
	}
}

func (l *loginLimiter) lockoutFor(failures int) time.Duration {
	d := l.baseLockout
	for i := l.maxFailures; i < failures; i++ {
		d *= 2
		if d >= l.maxLockout {
			return l.maxLockout
		}
	}
	if d > l.maxLockout {
		return l.maxLockout
	}
	return d
}
