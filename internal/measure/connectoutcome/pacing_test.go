package connectoutcome

import (
	"context"
	"io"
	"net/netip"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/measure/blanketdiscrim"
)

type scriptedConnector struct {
	result ConnResult
	calls  int
}

func (s *scriptedConnector) Connect(context.Context, netip.AddrPort) ConnResult {
	s.calls++
	return s.result
}

type scriptedHandshaker struct {
	calls int
}

func (s *scriptedHandshaker) Handshake(context.Context, netip.AddrPort, string) HandshakeResult {
	s.calls++
	return HandshakeResult{Outcome: NoTLS}
}

type fakeClock struct {
	now    time.Time
	slepts []time.Duration
}

func (f *fakeClock) Now() time.Time { return f.now }

func (f *fakeClock) Sleep(_ context.Context, d time.Duration) {
	f.slepts = append(f.slepts, d)
	f.now = f.now.Add(d)
}

func TestPacedPairSpendsAnIntervalOnEveryHandshake(t *testing.T) {
	profile := DefaultProfile()
	scope := Scope{
		Vantage:   "v",
		Addresses: []string{"203.0.113.10"},
		TCPPorts:  []uint16{443, 8443},
		Profile:   profile,
	}
	clock := &fakeClock{now: time.Unix(0, 0).UTC()}
	conn := &scriptedConnector{result: ConnOpen}
	hs := &scriptedHandshaker{}

	c, h := pacedPair(profile, conn, hs, clock.Now, clock.Sleep)
	if err := RunExchange(context.Background(), c, h, blanketdiscrim.FixedPorts{}, "batch", scope, io.Discard); err != nil {
		t.Fatalf("RunExchange: %v", err)
	}

	if conn.calls != 2 || hs.calls != 2 {
		t.Fatalf("drove %d connects and %d handshakes, want 2 and 2", conn.calls, hs.calls)
	}
	// Interval is 1s / PerHostConnPerSec, and the per-host arm always beats the aggregate one.
	interval := time.Second / time.Duration(profile.PerHostConnPerSec)
	want := []time.Duration{interval, interval, interval}
	if len(clock.slepts) != len(want) {
		t.Fatalf("paced %d waits %v, want %d %v", len(clock.slepts), clock.slepts, len(want), want)
	}
	for i, w := range want {
		if clock.slepts[i] != w {
			t.Fatalf("wait %d was %v, want %v", i, clock.slepts[i], w)
		}
	}
}

func TestPacedPairSharesOnePacerAcrossBothPaths(t *testing.T) {
	profile := DefaultProfile()
	target := netip.MustParseAddrPort("203.0.113.10:443")
	clock := &fakeClock{now: time.Unix(0, 0).UTC()}
	c, h := pacedPair(profile, &scriptedConnector{result: ConnOpen}, &scriptedHandshaker{}, clock.Now, clock.Sleep)

	c.Connect(context.Background(), target)
	h.Handshake(context.Background(), target, "")
	c.Connect(context.Background(), target)

	interval := time.Second / time.Duration(profile.PerHostConnPerSec)
	if got := clock.now.Sub(time.Unix(0, 0).UTC()); got != 2*interval {
		t.Fatalf("three attempts spanned %v, want %v", got, 2*interval)
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
