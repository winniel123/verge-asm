package main

import (
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

	// maxFailures is how many failures a key may accrue before it locks.
	maxFailures int
	// window is the idle span after which a key's failure count is considered
	// stale and reset — a reset that costs no goroutine, applied lazily on the
	// next touch.
	window time.Duration
	// baseLockout is the first lockout span once the threshold is crossed; each
	// further failure past the threshold doubles it, up to maxLockout.
	baseLockout time.Duration
	maxLockout  time.Duration

	mu      sync.Mutex
	entries map[string]*limiterEntry
}

// limiterEntry is one key's running state: how many failures it has accrued, when
// the last one landed (for the idle reset), and the instant it is locked until.
type limiterEntry struct {
	failures    int
	lastFailure time.Time
	lockedUntil time.Time
}

// newLoginLimiter builds the limiter over an injectable clock with sane defaults:
// five failures locks a key, a five-minute idle window resets a stale count, and
// lockout runs from five minutes doubling to a one-hour ceiling. Tests inject a
// fixed or steppable clock to assert lock and reset boundaries without sleeping.
func newLoginLimiter(now func() time.Time) *loginLimiter {
	return &loginLimiter{
		now:         now,
		maxFailures: 5,
		window:      5 * time.Minute,
		baseLockout: 5 * time.Minute,
		maxLockout:  time.Hour,
		entries:     map[string]*limiterEntry{},
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
		if e != nil && now.Before(e.lockedUntil) {
			return true
		}
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
		}
		e.failures++
		e.lastFailure = now
		if e.failures >= l.maxFailures {
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
