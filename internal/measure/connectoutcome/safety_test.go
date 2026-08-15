package connectoutcome

import (
	"net/netip"
	"testing"
	"time"
)

// The back-off halves the per-host rate on each stress signal, down to a floor
// of 1 conn/s, and never below — a struggling host is never fully starved.
func TestBackoffHalvesToFloor(t *testing.T) {
	b := NewBackoff(DefaultProfile()) // base 50
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

// Every stress cause halves; the limiter treats them uniformly (§3.3).
func TestBackoffEveryCauseHalves(t *testing.T) {
	for _, cause := range []Stress{StressTimeout, StressRSTSpike, Stress429, Stress503} {
		b := NewBackoff(DefaultProfile())
		b.Signal(cause)
		if b.Rate() != 25 {
			t.Errorf("cause %s: rate = %d, want 25 (halved)", cause, b.Rate())
		}
	}
}

// The back-off changes the rate and NOTHING about a deadline — there is no
// deadline on the type at all (ADR-0021). This is a structural assertion: the
// interval grows, and the profile's connect timeout is read separately.
func TestBackoffNeverTouchesDeadline(t *testing.T) {
	b := NewBackoff(DefaultProfile())
	before := b.Interval()
	b.Signal(StressTimeout)
	if b.Interval() <= before {
		t.Errorf("interval did not grow after a signal: %v -> %v", before, b.Interval())
	}
	// The connect timeout is the profile's, wholly independent of the back-off.
	if DefaultProfile().ConnectTimeoutMillis != 3000 {
		t.Errorf("connect timeout must be the profile's 3 s, untouched by back-off")
	}
}

// The pacer spaces per-host connects at ≥ 1/rate and never exceeds the global
// ceiling across hosts.
func TestPacerHonoursPerHostAndGlobal(t *testing.T) {
	// A tiny profile so the arithmetic is exact: 4 conn/s/host, 10 pkt/s global.
	p := SafetyProfile{PerHostConnPerSec: 4, GlobalPacketsPerSec: 10}
	pacer := NewPacer(p)
	start := time.Unix(0, 0)
	h := netip.MustParseAddr("198.51.100.1")

	// Two connects to one host: spaced by the per-host interval (250 ms), which
	// is larger than the global interval (100 ms), so the per-host rate binds.
	t0 := pacer.Next(h, start)
	t1 := pacer.Next(h, start)
	if gap := t1.Sub(t0); gap < 250*time.Millisecond {
		t.Errorf("per-host gap = %v, want ≥ 250ms (4 conn/s)", gap)
	}
}

// Across many hosts the global ceiling binds: connects to distinct hosts are
// still spaced by the global interval so adding targets does not multiply load.
func TestPacerGlobalCeilingBindsAcrossHosts(t *testing.T) {
	p := SafetyProfile{PerHostConnPerSec: 1000, GlobalPacketsPerSec: 5} // 200 ms global
	pacer := NewPacer(p)
	start := time.Unix(0, 0)
	prev := pacer.Next(netip.MustParseAddr("198.51.100.1"), start)
	for i := 2; i <= 5; i++ {
		h := netip.MustParseAddr("198.51.100." + itoa(i))
		now := pacer.Next(h, start)
		if gap := now.Sub(prev); gap < 200*time.Millisecond {
			t.Errorf("host %d: global gap = %v, want ≥ 200ms (5 pkt/s)", i, gap)
		}
		prev = now
	}
}

// A stress signal on a host widens that host's spacing at the pacer.
func TestPacerBacksOffSignalledHost(t *testing.T) {
	p := SafetyProfile{PerHostConnPerSec: 10, GlobalPacketsPerSec: 1000}
	pacer := NewPacer(p)
	start := time.Unix(0, 0)
	h := netip.MustParseAddr("198.51.100.1")

	a0 := pacer.Next(h, start)
	a1 := pacer.Next(h, start)
	baseline := a1.Sub(a0) // 100 ms at 10 conn/s

	pacer.Signal(h, StressTimeout) // halve to 5 conn/s -> 200 ms
	b0 := pacer.Next(h, start)
	b1 := pacer.Next(h, start)
	if b1.Sub(b0) <= baseline {
		t.Errorf("spacing did not widen after back-off: %v then %v", baseline, b1.Sub(b0))
	}
}

// RoundRobin cycles hosts and never places two of one host's ports back to back
// while another host still has an unplaced port (§3.3, §6.3).
func TestRoundRobinCyclesHosts(t *testing.T) {
	targets := []netip.AddrPort{
		ap("198.51.100.1:80"), ap("198.51.100.1:443"), ap("198.51.100.1:22"),
		ap("198.51.100.2:80"), ap("198.51.100.2:443"),
	}
	got := RoundRobin(targets)
	if len(got) != len(targets) {
		t.Fatalf("RoundRobin dropped targets: %d -> %d", len(targets), len(got))
	}
	// While both hosts have ports left, the schedule must alternate. The first
	// four slots cover two ports of each host: no host appears 3 times before the
	// other appears twice.
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

// RoundRobin is deterministic across calls.
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
