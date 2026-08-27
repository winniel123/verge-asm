package main

import "sync"

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
