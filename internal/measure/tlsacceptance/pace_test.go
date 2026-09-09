package tlsacceptance

import (
	"context"
	"io"
	"net/netip"
	"testing"
	"time"
)

type fakeClock struct {
	now    time.Time
	sleeps []time.Duration
}

func (f *fakeClock) Now() time.Time { return f.now }

func (f *fakeClock) Sleep(_ context.Context, d time.Duration) {
	f.sleeps = append(f.sleeps, d)
	f.now = f.now.Add(d)
}

func TestPacerSpacesPerHostHandshakes(t *testing.T) {
	p := NewPacer(CandidateSet{MaxHandshakesPerSecPerHost: 5})
	base := time.Unix(0, 0)
	first := p.Next("198.51.100.1", base)
	second := p.Next("198.51.100.1", base)
	if got := second.Sub(first); got != 200*time.Millisecond {
		t.Errorf("per-host spacing = %v, want 200ms at 5 handshakes/s", got)
	}
	if other := p.Next("198.51.100.2", base); !other.Equal(base) {
		t.Errorf("a fresh host should start at now, got %v", other)
	}
}

func TestPacedEnumeratorSpendsAnIntervalOnEveryHandshakeToOneHost(t *testing.T) {
	set := DefaultCandidateSet()
	scope := Scope{
		Vantage: "v",
		// Two Services on one host: the ceiling is per host, not per Service (spec §1.5).
		Services:   []ServiceTarget{{Address: "203.0.113.10", Port: 443}, {Address: "203.0.113.10", Port: 8443}},
		Candidates: set,
	}
	clock := &fakeClock{now: time.Unix(0, 0).UTC()}
	inner := newScript(true, map[string][]string{})
	paced := &pacedEnumerator{inner: inner, pacer: NewPacer(set), now: clock.Now, sleep: clock.Sleep}

	if err := RunWithEnumerator(context.Background(), paced, "batch", scope, io.Discard); err != nil {
		t.Fatalf("RunWithEnumerator: %v", err)
	}
	// A refusing listener costs one handshake per version: four per Service, eight in all.
	if inner.calls != 8 {
		t.Fatalf("drove %d handshakes, want 8", inner.calls)
	}
	interval := time.Second / time.Duration(set.MaxHandshakesPerSecPerHost)
	if len(clock.sleeps) != 7 {
		t.Fatalf("paced %d waits %v, want 7 of %v (the first handshake is free)", len(clock.sleeps), clock.sleeps, interval)
	}
	for i, w := range clock.sleeps {
		if w != interval {
			t.Fatalf("wait %d was %v, want %v", i, w, interval)
		}
	}
}

func TestPacedEnumeratorDoesNotPaceAcrossHosts(t *testing.T) {
	set := DefaultCandidateSet()
	clock := &fakeClock{now: time.Unix(0, 0).UTC()}
	paced := &pacedEnumerator{inner: newScript(true, nil), pacer: NewPacer(set), now: clock.Now, sleep: clock.Sleep}

	paced.Handshake(context.Background(), netip.MustParseAddrPort("203.0.113.10:443"), TLS12, set.Ciphers)
	paced.Handshake(context.Background(), netip.MustParseAddrPort("203.0.113.11:443"), TLS12, set.Ciphers)
	if len(clock.sleeps) != 0 {
		t.Fatalf("a second host waited %v behind the first", clock.sleeps)
	}
}

func TestSleepCtxReturnsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	sleepCtx(ctx, time.Hour)
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("a cancelled wait took %v", elapsed)
	}
}
