package main

import (
	"net/http"
	"sync"
	"time"
)

// Per-process and best-effort: a flash is a courtesy, never a record a restart must survive.

type flashStore struct {
	mu sync.Mutex
	m  map[int64]toastVM
}

func newFlashStore() *flashStore {
	return &flashStore{m: map[int64]toastVM{}}
}

func (f *flashStore) set(accountID int64, t toastVM) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.m[accountID] = t
	f.mu.Unlock()
}

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

// A typed reason can be sensitive, and a URL reaches the access log and browser history.
// A session key separates two devices; two tabs of one browser still share one slot.

type formFlashStore struct {
	mu sync.Mutex
	m  map[int64]formFlashEntry
}

type formFlashEntry struct {
	value     any
	stashedAt time.Time
}

// With no expiry a stash whose landing never came fires on the session's next visit.

const formFlashTTL = time.Minute

const maxFormFlashes = 256

func newFormFlashStore() *formFlashStore {
	return &formFlashStore{m: map[int64]formFlashEntry{}}
}

func (f *formFlashStore) pending(now time.Time) bool {
	// Resolving a session id costs a query, so the common no-flash GET pays a map walk instead.
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

func stashFormFlash(s *server, r *http.Request, v any) bool {
	// Every caller is behind a login gate, so a false answer is the revoked-mid-request race.
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

func takeFormFlash[T any](s *server, r *http.Request) (T, bool) {
	return takeFormFlashIf[T](s, r, nil)
}

func takeFormFlashIf[T any](s *server, r *http.Request, accept func(T) bool) (T, bool) {
	var zero T
	if s == nil || s.formFlash == nil {
		return zero, false
	}
	// The lock is released between the two reads, so the age check below is not redundant.
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
