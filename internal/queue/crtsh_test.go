package queue

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/wire"
)

func TestCountingSeq(t *testing.T) {
	saw := false
	got := slices.Collect(countingSeq(slices.Values([]string{"a", "b"}), &saw))
	if !saw {
		t.Errorf("saw not set for a non-empty sequence")
	}
	if !slices.Equal(got, []string{"a", "b"}) {
		t.Errorf("passed-through names = %v, want [a b]", got)
	}

	empty := false
	if n := len(slices.Collect(countingSeq(slices.Values([]string{}), &empty))); n != 0 {
		t.Errorf("empty sequence yielded %d names", n)
	}
	if empty {
		t.Errorf("saw set for an empty sequence — a false-empty would be missed")
	}
}

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

	status, _, err = f.Fetch(context.Background(), srv.URL+"/404")
	if err != nil {
		t.Fatalf("Fetch(404) errored on the transport, want a returned 404: %v", err)
	}
	if status != http.StatusNotFound {
		t.Errorf("status = %d, want 404", status)
	}
}

func TestHTTPCTFetcherDoesNotFollowRedirect(t *testing.T) {
	var hopReached bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestCTFetchOutcome(t *testing.T) {
	// net/http wraps a ctx error in *url.Error, so a bare context.Canceled would not test the unwrap.
	wrappedCancel := fmt.Errorf("Get %q: %w", "https://crt.sh", context.Canceled)
	wrappedDeadline := fmt.Errorf("Get %q: %w", "https://crt.sh", context.DeadlineExceeded)

	if got := ctFetchOutcome(wrappedCancel, 0); got != (wire.CTContextCancelled{}) {
		t.Errorf("wrapped ctx-cancel: got %#v, want CTContextCancelled", got)
	}
	if got := ctFetchOutcome(wrappedDeadline, 0); got != (wire.CTContextCancelled{}) {
		t.Errorf("wrapped ctx-deadline: got %#v, want CTContextCancelled", got)
	}

	transport := errors.New("dial tcp: i/o timeout")
	got := ctFetchOutcome(transport, 0)
	te, ok := got.(wire.CTTransportError)
	if !ok || te.Text != transport.Error() {
		t.Errorf("transport error: got %#v, want CTTransportError{%q}", got, transport.Error())
	}

	got = ctFetchOutcome(nil, http.StatusBadGateway)
	h, ok := got.(wire.CTHTTP)
	if !ok || h.Status != http.StatusBadGateway {
		t.Errorf("non-200: got %#v, want CTHTTP{502}", got)
	}

	if got := ctFetchOutcome(nil, http.StatusOK); got != (wire.CTHTTP{Status: http.StatusOK}) {
		t.Errorf("200: got %#v, want CTHTTP{200}", got)
	}
}

func TestSleepUntil(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) }

	past := now().Add(-time.Minute)
	if err := sleepUntil(context.Background(), now, past); err != nil {
		t.Errorf("sleepUntil(past) = %v, want nil (no wait)", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	future := time.Now().Add(time.Hour)
	if err := sleepUntil(ctx, time.Now, future); err == nil {
		t.Errorf("sleepUntil(future, cancelled ctx) = nil, want ctx error")
	}
}
