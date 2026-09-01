package main

import (
	"net/http"
	"sync"
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
// It is keyed by SESSION id, not by account id like the toast flashStore above. Two
// tabs signed in as one account are two sessions, and a rejected form belongs to the
// tab that submitted it — an account key would let one tab consume the other's error
// and leave the operator staring at an unexplained page. currentSessionID (auth.go)
// resolves the key.
//
// The value is `any` so one store serves every surface's own form struct: this ticket
// lands signalsForms, and tickets #974-#978 bring the settings, scope and remaining
// handlers onto the same carrier without a new field per surface. takeFormFlash is
// typed, so a stash of one shape is never read as another.
//
// Like the toast store it is per-process and best-effort by design. A restart, a
// second tab racing the read, or an operator who abandons the page simply drops the
// flash, and the operator sees the page with no callout rather than a wrong one.
type formFlashStore struct {
	mu sync.Mutex
	m  map[int64]any
}

// maxFormFlashes bounds the store. Sessions are unbounded over an instance's uptime —
// every login mints one — so an entry stashed by a session that never lands would
// otherwise sit in the map until the process restarts. On overflow the whole map is
// dropped rather than one entry evicted: there is no arrival order to evict by, and a
// flash is a courtesy, so the cost of the reset is at most a lost callout on a page an
// operator has not loaded yet. A console with this many sessions each holding an
// unconsumed rejected form at one instant is not a shape a real deployment reaches.
const maxFormFlashes = 256

func newFormFlashStore() *formFlashStore {
	return &formFlashStore{m: map[int64]any{}}
}

// pending reports whether ANY session has a form flash waiting. It is the cheap gate
// takeFormFlash checks first: resolving a session id costs a session-registry read,
// and the ordinary case on every console GET is that no form anywhere was rejected, so
// the common path pays a mutex and a len() rather than a query.
func (f *formFlashStore) pending() bool {
	if f == nil {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.m) > 0
}

// set stashes one rejected form for a session, replacing any unconsumed one. The
// latest submission wins: an operator never needs to see the errors of a form they
// have already re-submitted.
func (f *formFlashStore) set(sessionID int64, v any) {
	if f == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, held := f.m[sessionID]; !held && len(f.m) >= maxFormFlashes {
		f.m = map[int64]any{}
	}
	f.m[sessionID] = v
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
	s.formFlash.set(sessionID, v)
	return true
}

// takeFormFlash reads and deletes the pending rejected form of shape T for the session
// making r. The delete is what makes the flash single-consume: a second load of the
// same URL, whether the operator reloads it or an auto-refresh does, finds nothing and
// shows no stale callout.
//
// A flash of a DIFFERENT shape is left in place, not consumed. It belongs to another
// surface the same session submitted, and that surface's own GET is still entitled to
// render it. The next stash for the session replaces it either way, so nothing
// accumulates.
func takeFormFlash[T any](s *server, r *http.Request) (T, bool) {
	var zero T
	if s == nil || s.formFlash == nil || !s.formFlash.pending() {
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
	if !ok {
		return zero, false
	}
	typed, ok := held.(T)
	if !ok {
		return zero, false
	}
	delete(f.m, sessionID)
	return typed, true
}
