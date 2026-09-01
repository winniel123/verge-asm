package main

import (
	"net/http"
	"sync"
	"time"
)

// flashStore is a tiny in-process, single-consume toast store keyed by account id.
//
// The scan trigger (#252) and stop/terminate (DF-F4, #633) surfaces auto-refresh while
// a scan is in flight — the head emits <meta http-equiv="refresh"> and the browser
// reloads the same URL every few seconds. A toast carried in the redirect URL's `toast`
// query would re-fire on every one of those reloads: the "Scan started" toast spam the
// dogfood reported (WORK-ORDER-DOGFOOD-R1 item 1). Instead these acts stash one toast
// here and redirect to a clean URL; injectChrome consumes it on the FIRST chrome render
// (read-and-delete), so the auto-refresh that reloads the same clean URL finds nothing
// to show. One flash per dispatch, shown exactly once.
//
// It is per-process and best-effort by design: a flash is a courtesy, not a record. A
// restart or a second tab racing the read simply drops it. Never used for anything that
// must persist.
type flashStore struct {
	mu sync.Mutex
	m  map[int64]toastVM
}

func newFlashStore() *flashStore {
	return &flashStore{m: map[int64]toastVM{}}
}

// set stashes one toast for an account, replacing any unconsumed one (the latest act
// wins — an operator never needs to see a superseded receipt).
func (f *flashStore) set(accountID int64, t toastVM) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.m[accountID] = t
	f.mu.Unlock()
}

// take reads and deletes an account's pending toast, returning ok=false when none is
// waiting. The delete is what makes a flash single-consume, so an in-flight
// auto-refresh does not re-show it.
func (f *flashStore) take(accountID int64) (toastVM, bool) {
	if f == nil {
		return toastVM{}, false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.m[accountID]
	if ok {
		delete(f.m, accountID)
	}
	return t, ok
}

// formFlashStore is the session-keyed, single-consume carrier for a REJECTED FORM
// (ADR-0130 §1, map #969 ticket #972). A validation failure no longer re-renders in
// place at the POST URL. The handler stashes the field error and the operator's typed
// values here, answers 303 to the URL the form was submitted from (backurl.go), and
// the GET handler for that URL reads the stash once and renders the same callouts.
//
// That turns the error path into an ordinary post-redirect-get, indistinguishable
// from a success: the load is a normal navigation to the same URL, so the scroll key
// the shell stashed on submit (shell.tmpl, ticket #970) hits and the operator keeps
// their place. This is failure class A.
//
// The payload never enters the URL. A typed reason is bulky, it can be sensitive, and
// a URL is written to the access log and kept in browser history. The `?toast=` query
// carrier (shell.go toastRedirect) stays for toasts only, which are short controlled
// strings the server itself chose.
//
// It is keyed by SESSION id rather than by account id like the toast flashStore above.
// Be precise about what that buys, because tickets #974-#978 build on it. A session is
// one SIGN-IN, so the key separates an operator's laptop from their phone, and either
// from a private window — two sign-ins as one account, each with its own refusals. It
// does NOT separate two tabs of one browser: tabs share the session cookie, resolve to
// one session row, and so share one slot. That residue is real. A tab that loads a
// console page in the window between another tab's refused POST and that tab's own
// landing GET consumes the flash, and the submitting tab lands with no callout. An
// account key would have made that the behaviour ACROSS DEVICES too, which is the
// wider and more confusing failure, so the session key is the right one. It is simply
// not a per-tab guarantee, and no server-side store keyed by a cookie can be.
//
// The value is `any` so one store serves every surface's own form struct: this ticket
// lands signalsForms, and tickets #974-#978 bring the settings, scope and remaining
// handlers onto the same carrier without a new field per surface. takeFormFlash is
// typed, so a stash of one shape is never read as another.
//
// Like the toast store it is per-process and best-effort by design. A restart, the tab
// race above, or an operator who abandons the page simply drops the flash, and the
// operator sees the page with no callout rather than a wrong one.
type formFlashStore struct {
	mu sync.Mutex
	m  map[int64]formFlashEntry
}

// formFlashEntry is one stashed rejected form and the instant it was stashed. The
// instant is what bounds a flash that is never collected (see formFlashTTL).
type formFlashEntry struct {
	value     any
	stashedAt time.Time
}

// formFlashTTL bounds how long a stashed rejected form stays collectable. The landing
// GET is the browser's very next request after the 303, so a minute is orders of
// magnitude more time than the redirect needs.
//
// The bound exists because that GET is not guaranteed to arrive. An operator who
// closes the tab as they submit, a dropped connection, or a client that does not
// follow redirects each leave the entry stashed. With no expiry it would sit there and
// fire on that session's NEXT visit to the surface — an unexplained callout, and a
// pre-filled reason if that later URL happened to re-open the same drawer. An expired
// entry is dropped rather than shown: a refusal whose landing never came is stale, and
// a stale callout is worse than none.
const formFlashTTL = time.Minute

// maxFormFlashes bounds the store against the one shape the TTL does not: a burst of
// refusals inside a single TTL window. On overflow the whole map is dropped rather
// than one entry evicted, since a flash is a courtesy and the cost of the reset is at
// most a lost callout on a page an operator has not loaded yet. A console with this
// many sessions each holding an unconsumed rejected form at one instant is not a shape
// a real deployment reaches.
const maxFormFlashes = 256

func newFormFlashStore() *formFlashStore {
	return &formFlashStore{m: map[int64]formFlashEntry{}}
}

// pending drops every entry older than formFlashTTL and reports whether any live one
// is left, for ANY session. It is the cheap gate takeFormFlash checks first: resolving
// a session id costs a session-registry read, and the ordinary case on a console GET
// is that no form anywhere was rejected, so the common path pays a mutex and a map
// walk rather than a query.
//
// The prune is what keeps the gate honest. A stranded entry nothing ever collects
// would otherwise hold the gate open for the life of the process, and every signed-in
// operator's every GET would then pay the registry read the gate exists to avoid. With
// the prune that widening is bounded by the TTL, and the walk it costs runs only while
// an entry is actually waiting.
func (f *formFlashStore) pending(now time.Time) bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for id, e := range f.m {
		if now.Sub(e.stashedAt) > formFlashTTL {
			delete(f.m, id)
		}
	}
	return len(f.m) > 0
}

// set stashes one rejected form for a session, replacing any unconsumed one. The
// latest submission wins: an operator never needs to see the errors of a form they
// have already re-submitted.
func (f *formFlashStore) set(sessionID int64, v any, now time.Time) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, held := f.m[sessionID]; !held && len(f.m) >= maxFormFlashes {
		f.m = map[int64]formFlashEntry{}
	}
	f.m[sessionID] = formFlashEntry{value: v, stashedAt: now}
}

// stashFormFlash records v as the pending rejected form for the session making r, and
// reports whether it was stored. It resolves the session the same way currentAccount
// does, so a request that reached an authenticated handler resolves one.
//
// A false answer means no session resolved and the error message is dropped. Every
// caller is behind requireLogin or requireAdmin, which already proved a live session
// row, so this is the revoked-between-two-reads race and nothing else. The redirect
// still happens: an operator then lands on a page with no callout, which is worse than
// the callout but is not a wrong answer.
func stashFormFlash(s *server, r *http.Request, v any) bool {
	if s == nil || s.formFlash == nil {
		return false
	}
	sessionID, ok := s.currentSessionID(r)
	if !ok {
		return false
	}
	s.formFlash.set(sessionID, v, s.now())
	return true
}

// takeFormFlash reads and deletes the pending rejected form of shape T for the session
// making r. The delete is what makes the flash single-consume: a second load of the
// same URL, whether the operator reloads it or an auto-refresh does, finds nothing and
// shows no stale callout.
//
// A flash of a DIFFERENT shape is left in place, not consumed. It belongs to another
// surface the same session submitted, and that surface's own GET is still entitled to
// render it. The next stash for the session replaces it, and the TTL retires it if no
// landing ever comes, so nothing accumulates.
func takeFormFlash[T any](s *server, r *http.Request) (T, bool) {
	return takeFormFlashIf[T](s, r, nil)
}

// takeFormFlashIf is takeFormFlash with a claim check: it consumes the pending form
// only when accept says this reader is the landing the stash was written for, and
// LEAVES IT IN PLACE otherwise. A nil accept takes whatever is there.
//
// One shape can have several landing GETs, and they are not interchangeable. The
// settings surface is the case that forced this: /settings renders whichever tab its
// query names, and /scans renders the Scans section on a URL of its own, so both read
// a settingsForms flash — but a callout stashed by a refused CHANNEL act renders
// nothing on the Scans tab. Without the check that GET would delete it and show
// nothing, and the operator's own landing would arrive to a page with no error and no
// echo of what they typed. That is not a rare race: a Scans view with a scan in flight
// re-requests itself every six seconds (fillScansSection's Refresh), so it would eat
// the session's every refusal for as long as the scan ran.
//
// A declined flash keeps its stashedAt, so the TTL still retires it if its own landing
// never comes. Nothing accumulates.
func takeFormFlashIf[T any](s *server, r *http.Request, accept func(T) bool) (T, bool) {
	var zero T
	if s == nil || s.formFlash == nil {
		return zero, false
	}
	// pending prunes as it answers, so an expired entry is already gone by the time the
	// lookup below runs. The age check stays anyway: the two calls take the lock
	// separately, and the session read between them is a database round trip.
	now := s.now()
	if !s.formFlash.pending(now) {
		return zero, false
	}
	sessionID, ok := s.currentSessionID(r)
	if !ok {
		return zero, false
	}
	f := s.formFlash
	f.mu.Lock()
	defer f.mu.Unlock()
	held, ok := f.m[sessionID]
	if !ok || now.Sub(held.stashedAt) > formFlashTTL {
		return zero, false
	}
	typed, ok := held.value.(T)
	if !ok {
		return zero, false
	}
	if accept != nil && !accept(typed) {
		return zero, false
	}
	delete(f.m, sessionID)
	return typed, true
}
