package connectoutcome

import (
	"net/netip"
	"testing"
	"time"
)

func TestBackoffHalvesToFloor(t *testing.T) {
	// The floor is one connection per second, so a struggling host is slowed and never silenced.
	b := NewBackoff(DefaultProfile())
	if b.Rate() != 50 {
		t.Fatalf("initial rate = %d, want 50", b.Rate())
	}
	want := []int{25, 12, 6, 3, 1, 1, 1}
	for i, w := range want {
		b.Signal(StressTimeout)
		if b.Rate() != w {
			t.Errorf("after %d signals rate = %d, want %d", i+1, b.Rate(), w)
		}
	}
}

func TestBackoffEveryCauseHalves(t *testing.T) {
	for _, cause := range []Stress{StressTimeout, StressRSTSpike, Stress429, Stress503} {
		b := NewBackoff(DefaultProfile())
		b.Signal(cause)
		if b.Rate() != 25 {
			t.Errorf("cause %s: rate = %d, want 25 (halved)", cause, b.Rate())
		}
	}
}

func TestBackoffHonoursDeclaredPolicy(t *testing.T) {
	p := DefaultProfile()
	p.AdaptiveBackoff = BackoffPolicy{HalveOnTimeout: true}
	b := NewBackoff(p)
	b.Signal(Stress429)
	if b.Rate() != b.base {
		t.Errorf("a disabled cause halved the rate: got %d, want %d", b.Rate(), b.base)
	}
	b.Signal(StressTimeout)
	if b.Rate() != b.base/2 {
		t.Errorf("the enabled cause did not halve: got %d, want %d", b.Rate(), b.base/2)
	}
}

func TestBackoffNeverTouchesDeadline(t *testing.T) {
	b := NewBackoff(DefaultProfile())
	before := b.Interval()
	b.Signal(StressTimeout)
	if b.Interval() <= before {
		t.Errorf("interval did not grow after a signal: %v -> %v", before, b.Interval())
	}
	if DefaultProfile().ConnectTimeoutMillis != 3000 {
		t.Errorf("connect timeout must be the profile's 3 s, untouched by back-off")
	}
}

func TestPacerHonoursPerHostAndAggregate(t *testing.T) {
	p := SafetyProfile{PerHostConnPerSec: 4, PerVantagePacketsPerSec: 10}
	pacer := NewPacer(p)
	start := time.Unix(0, 0)
	h := netip.MustParseAddr("198.51.100.1")

	t0 := pacer.Next(h, start)
	t1 := pacer.Next(h, start)
	if gap := t1.Sub(t0); gap < 250*time.Millisecond {
		t.Errorf("per-host gap = %v, want ≥ 250ms (4 conn/s)", gap)
	}
}

func TestPacerAggregateCeilingBindsAcrossHosts(t *testing.T) {
	p := SafetyProfile{PerHostConnPerSec: 1000, PerVantagePacketsPerSec: 5}
	pacer := NewPacer(p)
	start := time.Unix(0, 0)
	prev := pacer.Next(netip.MustParseAddr("198.51.100.1"), start)
	for i := 2; i <= 5; i++ {
		h := netip.MustParseAddr("198.51.100." + itoa(i))
		now := pacer.Next(h, start)
		if gap := now.Sub(prev); gap < 200*time.Millisecond {
			t.Errorf("host %d: aggregate gap = %v, want ≥ 200ms (5 pkt/s)", i, gap)
		}
		prev = now
	}
}

func TestPacerBacksOffSignalledHost(t *testing.T) {
	p := SafetyProfile{
		PerHostConnPerSec:       10,
		PerVantagePacketsPerSec: 1000,
		AdaptiveBackoff:         BackoffPolicy{HalveOnTimeout: true},
	}
	pacer := NewPacer(p)
	start := time.Unix(0, 0)
	h := netip.MustParseAddr("198.51.100.1")

	a0 := pacer.Next(h, start)
	a1 := pacer.Next(h, start)
	baseline := a1.Sub(a0)

	pacer.Signal(h, StressTimeout)
	b0 := pacer.Next(h, start)
	b1 := pacer.Next(h, start)
	if b1.Sub(b0) <= baseline {
		t.Errorf("spacing did not widen after back-off: %v then %v", baseline, b1.Sub(b0))
	}
}

func TestRoundRobinCyclesHosts(t *testing.T) {
	targets := []netip.AddrPort{
		ap("198.51.100.1:80"), ap("198.51.100.1:443"), ap("198.51.100.1:22"),
		ap("198.51.100.2:80"), ap("198.51.100.2:443"),
	}
	got := RoundRobin(targets)
	if len(got) != len(targets) {
		t.Fatalf("RoundRobin dropped targets: %d -> %d", len(targets), len(got))
	}
	count := map[netip.Addr]int{}
	for i := 0; i < 4; i++ {
		count[got[i].Addr()]++
	}
	for h, n := range count {
		if n > 2 {
			t.Errorf("host %s bursts %d times in the first 4 slots; round-robin must interleave", h, n)
		}
	}
}

func TestRoundRobinDeterministic(t *testing.T) {
	targets := []netip.AddrPort{
		ap("203.0.113.5:8080"), ap("198.51.100.1:80"), ap("198.51.100.1:443"),
	}
	a := RoundRobin(targets)
	b := RoundRobin(targets)
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("RoundRobin not deterministic at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}
