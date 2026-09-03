package main

import (
	"strings"
	"sync"
	"time"
)

// loginLimiter is the dependency-free, in-process failed-attempt throttle that
// guards the two credential endpoints (#322). A 6-digit TOTP is otherwise
// brute-forceable — VerifyTOTP accepts 3 of 10^6 codes per instant with no cap on
// attempts — and an online password guess has the same unbounded budget. The
// limiter keys failures both per-account and per-IP: locking the account stops a
// distributed guess at one credential, and locking the source IP stops one host
// hammering many accounts. It holds no DB migration and no new dependency — just a
// mutex-guarded map keyed to a fake-able clock — so it is unit-testable and adds no
// schema.
//
// Reset is on two axes: a successful auth clears the key at once, and a key whose
// last failure is older than window is treated as fresh (a slow trickle never
// accretes into a lockout). Lockout is exponential past the threshold, so a
// persistent attacker faces geometrically longer waits, capped so a locked-out key
// always recovers.
type loginLimiter struct {
	now func() time.Time

	maxFailures int
	window      time.Duration
	baseLockout time.Duration
	maxLockout  time.Duration

	// acctPrefix marks the victim-scoped keys (the named account) as opposed to the
	// attacker-scoped per-IP keys, and acctLockCeiling bounds how long such a key may
	// stay locked, measured from its FIRST lock (#738). Past that ceiling locked()
	// stops honouring the account lock, so an unauthenticated attacker who keeps a
	// known username failing can deny the real operator for at most acctLockCeiling
	// rather than indefinitely — while the per-IP key, scoped to the guessing host,
	// keeps throttling. A zero ceiling disables the cap (every key locks fully).
	acctPrefix      string
	acctLockCeiling time.Duration

	mu      sync.Mutex
	entries map[string]*limiterEntry
}

type limiterEntry struct {
	failures    int
	lastFailure time.Time
	lockedUntil time.Time
	// firstLockedAt is when this key first entered a locked state (zero until then),
	// anchoring the per-account lock ceiling (#738). It is cleared on a reset or an
	// idle-window reset so a later lockout starts a fresh ceiling.
	firstLockedAt time.Time
}

func newLoginLimiter(now func() time.Time) *loginLimiter {
	return &loginLimiter{
		now:         now,
		maxFailures: 5,
		window:      5 * time.Minute,
		baseLockout: 5 * time.Minute,
		maxLockout:  time.Hour,
		// A named account (the victim-scoped key) can be held locked for at most
		// fifteen minutes from its first lock before locked() releases it (#738), so a
		// pre-auth attacker cannot deny a known username indefinitely; the per-IP key
		// keeps its full escalating lock against the guessing host.
		acctPrefix:      "acct:",
		acctLockCeiling: 15 * time.Minute,
		entries:         map[string]*limiterEntry{},
	}
}

// locked reports whether any of the given keys is currently within its lockout
// window. A credential attempt is refused when this is true, before any password
// or TOTP work runs, so a locked key costs nothing to reject.
func (l *loginLimiter) locked(keys ...string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for _, k := range keys {
		e := l.entries[k]
		if e == nil || !now.Before(e.lockedUntil) {
			continue
		}
		// #738: an account key's lock is honoured only within acctLockCeiling of its
		// first lock, so an unauthenticated attacker cannot hold a known username
		// locked out indefinitely — the per-IP key, scoped to the guessing host, keeps
		// throttling. Every other key (the per-IP key) locks for its full span.
		if l.acctLockCeiling > 0 && l.acctPrefix != "" && strings.HasPrefix(k, l.acctPrefix) &&
			!e.firstLockedAt.IsZero() && !now.Before(e.firstLockedAt.Add(l.acctLockCeiling)) {
			continue
		}
		return true
	}
	return false
}

// fail records one failed attempt against every given key and returns whether any
// of them is now locked. The caller uses that to clear a half-finished flow (e.g.
// the pending-TOTP cookie) once a key trips its threshold, so a locked-out attempt
// cannot keep re-presenting against the same pending grant.
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
		// A key idle longer than the window starts a fresh count — a slow trickle
		// never accretes into a lockout — unless it is still inside an active lock.
		if !now.Before(e.lockedUntil) && !e.lastFailure.IsZero() && now.Sub(e.lastFailure) > l.window {
			e.failures = 0
			e.firstLockedAt = time.Time{}
		}
		e.failures++
		e.lastFailure = now
		if e.failures >= l.maxFailures {
			// Anchor the per-account lock ceiling to the FIRST lock so repeated
			// re-locks cannot slide the window forward and deny the account forever (#738).
			if e.firstLockedAt.IsZero() {
				e.firstLockedAt = now
			}
			e.lockedUntil = now.Add(l.lockoutFor(e.failures))
			nowLocked = true
		}
	}
	return nowLocked
}

// reset clears every given key on a successful auth, so a legitimate operator who
// mistyped a few times before signing in starts clean and is never carried toward
// a lockout by past misses.
func (l *loginLimiter) reset(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, k := range keys {
		delete(l.entries, k)
	}
}

// lockoutFor is the lockout span for a key that has just reached failures: the base
// span doubled once per failure past the threshold, capped at maxLockout so a key
// always recovers rather than locking forever.
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
