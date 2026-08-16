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
