package connectoutcome

import (
	"net/netip"
	"sort"
	"time"
)

// This file is the §3.3 safety limiter, kept deliberately OUTSIDE the verdict
// (ADR-0021): it paces the connects — per-host rate, per-host concurrency, a
// global ceiling round-robin by host, and an adaptive back-off that halves the
// rate on a stress signal — and can neither manufacture nor suppress a
// reachability value. Every part is pure and clock-injectable, so the pacing is
// tested without a real sleep and without the network.

// Stress is a signal that the limiter halves the rate on (§3.3). Each value is a
// distinct cause; the limiter halves once for any cause the declared
// BackoffPolicy enables, treating the enabled causes uniformly. It keeps no
// per-signal record of its own — the recorded commitment is the declared
// BackoffPolicy on the Batch (ADR-0025), not a log of which signals fired. This
// closed set of causes is the value space those four policy flags range over.
type Stress string

const (
	StressTimeout  Stress = "timeout"   // a connect timed out
	StressRSTSpike Stress = "rst-spike" // a burst of refusals — a middlebox shedding load
	Stress429      Stress = "429"       // an HTTP layer above returned Too Many Requests
	Stress503      Stress = "503"       // an HTTP layer above returned Service Unavailable
)

// enabledIn reports whether the declared back-off policy halves on this cause.
// It is the one place the runtime Stress value space and the recorded
// BackoffPolicy are tied together, so a cause the operator's offers did not
// enable does not silently halve, and every Stress value maps to exactly one
// declared flag.
func (s Stress) enabledIn(p BackoffPolicy) bool {
	switch s {
	case StressTimeout:
		return p.HalveOnTimeout
	case StressRSTSpike:
		return p.HalveOnRSTSpike
	case Stress429:
		return p.HalveOn429
	case Stress503:
		return p.HalveOn503
	default:
		return false
	}
}

// Backoff is the per-host adaptive back-off state. It halves the effective rate
// on each stress signal, down to a floor of one connection per second, and
// never touches the deadline — the deadline belongs to the connect attempt and
// the rate belongs here (ADR-0021). It starts at the profile's per-host rate.
type Backoff struct {
	base     int           // the profile's per-host conn/s ceiling
	halvings int           // how many times the rate has been halved
	policy   BackoffPolicy // which stress causes the declared offers halve on
}

// NewBackoff starts a back-off at the profile's per-host rate, honouring the
// profile's declared adaptive-back-off policy.
func NewBackoff(profile SafetyProfile) *Backoff {
	base := profile.PerHostConnPerSec
	if base < 1 {
		base = 1
	}
	return &Backoff{base: base, policy: profile.AdaptiveBackoff}
}

// Signal halves the rate in response to a stress cause the declared policy
// enables — a cause the offers did not enable is a no-op, so the runtime never
// halves on a cause the recorded commitment did not declare. It is idempotent
// per call — one enabled signal, one halving — and saturates at the floor, so a
// host that keeps shedding load is not driven below one connection per second.
// It never reads or writes any deadline.
func (b *Backoff) Signal(cause Stress) {
	if !cause.enabledIn(b.policy) {
		return
	}
	if b.Rate() > 1 {
		b.halvings++
	}
}

// Rate is the current effective per-host connection rate, in conn/s: the base
// halved once per signal, floored at 1.
func (b *Backoff) Rate() int {
	r := b.base >> b.halvings
	if r < 1 {
		return 1
	}
	return r
}

// Interval is the minimum spacing between two connects to the host at the
// current rate. It is what a Pacer adds to a host's last-emit instant.
func (b *Backoff) Interval() time.Duration {
	return time.Second / time.Duration(b.Rate())
}

// Pacer computes when each connect may start, honouring both a per-host minimum
// spacing (from that host's Backoff) and a single aggregate minimum spacing (the
// 200 pkt/s per-vantage ceiling). It holds no timers: Next is pure arithmetic
// over a caller-supplied clock, so a test drives it with a fixed `now` and
// asserts the spacing rather than sleeping. A real run sleeps until the returned
// instant.
type Pacer struct {
	aggregateInterval time.Duration
	lastEmit          time.Time
	lastHost          map[netip.Addr]time.Time
	backoff           map[netip.Addr]*Backoff
	profile           SafetyProfile
}

// NewPacer builds a Pacer over the profile's aggregate ceiling and per-host rate.
func NewPacer(profile SafetyProfile) *Pacer {
	pps := profile.PerVantagePacketsPerSec
	if pps < 1 {
		pps = 1
	}
	return &Pacer{
		aggregateInterval: time.Second / time.Duration(pps),
		lastHost:          map[netip.Addr]time.Time{},
		backoff:           map[netip.Addr]*Backoff{},
		profile:           profile,
	}
}

func (p *Pacer) backoffFor(host netip.Addr) *Backoff {
	b, ok := p.backoff[host]
	if !ok {
		b = NewBackoff(p.profile)
		p.backoff[host] = b
	}
	return b
}

// Signal halves the given host's rate on a stress cause, so the next Next for
// that host is spaced further out. It never moves the aggregate interval and
// never touches a deadline.
func (p *Pacer) Signal(host netip.Addr, cause Stress) { p.backoffFor(host).Signal(cause) }

// Next returns the instant the next connect to host may start, given the current
// clock. It is the later of: the per-host interval since that host's last emit,
// and the aggregate interval since any host's last emit — so the aggregate
// ceiling binds across hosts while each host also respects its own (possibly
// backed-off) rate. Calling Next records the emit at the returned instant, so a
// sequence of calls produces a correctly-spaced schedule.
func (p *Pacer) Next(host netip.Addr, now time.Time) time.Time {
	earliest := now
	if !p.lastEmit.IsZero() {
		if t := p.lastEmit.Add(p.aggregateInterval); t.After(earliest) {
			earliest = t
		}
	}
	if last, ok := p.lastHost[host]; ok {
		if t := last.Add(p.backoffFor(host).Interval()); t.After(earliest) {
			earliest = t
		}
	}
	p.lastEmit = earliest
	p.lastHost[host] = earliest
	return earliest
}

// RoundRobin orders a set of `(Address, port)` targets so the schedule cycles
// hosts and never bursts one host's ports back to back (§3.3, §6.3): iterating
// ports within a host is the canonical port-scan signature and the canonical way
// to fill a middlebox state table. Targets are grouped by host, each host's
// ports sorted, and the groups are then interleaved one port at a time. The
// output is deterministic: hosts are visited in address order on every round.
func RoundRobin(targets []netip.AddrPort) []netip.AddrPort {
	byHost := map[netip.Addr][]netip.AddrPort{}
	var hosts []netip.Addr
	for _, t := range targets {
		h := t.Addr()
		if _, ok := byHost[h]; !ok {
			hosts = append(hosts, h)
		}
		byHost[h] = append(byHost[h], t)
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].Less(hosts[j]) })
	for _, h := range hosts {
		ports := byHost[h]
		sort.Slice(ports, func(i, j int) bool { return ports[i].Port() < ports[j].Port() })
		byHost[h] = ports
	}

	out := make([]netip.AddrPort, 0, len(targets))
	for round := 0; len(out) < len(targets); round++ {
		for _, h := range hosts {
			ports := byHost[h]
			if round < len(ports) {
				out = append(out, ports[round])
			}
		}
	}
	return out
}
