package queue

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The production fetcher sends a distinctive User-Agent (the operator asked for
// one) and returns the status and body without erroring on a non-200 — a 404 or
// 5xx is the caller's transient failure to classify, never mapped to empty here
// (ADR-0027 §7).
func TestHTTPCTFetcher(t *testing.T) {
	var gotUA string
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name_value":"a.example.com"}]`))
	})
	mux.HandleFunc("/404", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	f := NewHTTPCTFetcher("9.9.9")

	status, body, err := f.Fetch(context.Background(), srv.URL+"/ok")
	if err != nil {
		t.Fatalf("Fetch(ok): %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(string(body), "a.example.com") {
		t.Errorf("body not returned: %s", body)
	}
	if !strings.HasPrefix(gotUA, "verge-asm/9.9.9") {
		t.Errorf("User-Agent = %q, want a verge-asm/<version> identifier", gotUA)
	}

	// A 404 is returned as status 404 with no error — the caller decides it is
	// transient. Mapping it to an error here would be fine too, but mapping it to
	// an empty admission is the failure the runner must not make.
	status, _, err = f.Fetch(context.Background(), srv.URL+"/404")
	if err != nil {
		t.Fatalf("Fetch(404) errored on the transport, want a returned 404: %v", err)
	}
	if status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}
}

// A 3xx from crt.sh is NOT followed: the fetcher returns the redirect's own
// status and never issues the next hop, so a compromised or MITM'd crt.sh cannot
// bounce the fetch to an internal host (blind SSRF). The redirect target is a
// second httptest server whose handler flips a flag if it is ever reached; the
// fetch must leave that flag unset and surface the 302 as a non-200 the caller
// classifies as transient (ADR-0027 §7).
func TestHTTPCTFetcherDoesNotFollowRedirect(t *testing.T) {
	var hopReached bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A link-local IMDS or RFC-1918 host in production; a loopback stand-in
		// here. Reaching this at all is the SSRF the fix prevents.
		hopReached = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"name_value":"pwned.example.com"}]`))
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/latest/meta-data/", http.StatusFound)
	}))
	defer redirector.Close()

	f := NewHTTPCTFetcher("9.9.9")

	status, body, err := f.Fetch(context.Background(), redirector.URL+"/redirect")
	if err != nil {
		t.Fatalf("Fetch(redirect) errored on the transport, want the unfollowed 302: %v", err)
	}
	if hopReached {
		t.Fatal("fetcher followed the redirect and reached the next hop: blind SSRF")
	}
	if status != http.StatusFound {
		t.Errorf("status = %d, want 302 (the unfollowed redirect surfaced as-is)", status)
	}
	if strings.Contains(string(body), "pwned.example.com") {
		t.Errorf("body carries the redirect target's response, meaning the hop was followed: %s", body)
	}
}

// sleepUntil returns immediately when the reserved slot is already in the past
// (the common case once the throttle has spaced requests out), and honours ctx
// cancellation when the slot is in the future.
func TestSleepUntil(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) }

	// Slot in the past: no wait.
	past := now().Add(-time.Minute)
	if err := sleepUntil(context.Background(), now, past); err != nil {
		t.Errorf("sleepUntil(past) = %v, want nil (no wait)", err)
	}

	// Slot in the future with a cancelled ctx: returns promptly with the ctx error
	// rather than blocking for the full interval.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	future := time.Now().Add(time.Hour)
	if err := sleepUntil(ctx, time.Now, future); err == nil {
		t.Errorf("sleepUntil(future, cancelled ctx) = nil, want ctx error")
	}
}
