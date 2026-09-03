package connectoutcome

import (
	"net/netip"
	"sort"
	"time"
)

// Pacing sits outside the verdict: it can neither manufacture nor suppress a value (ADR-0021).

// No per-signal record is kept: the commitment recorded is the declared policy (ADR-0025).

type Stress string

const (
	StressTimeout  Stress = "timeout"
	StressRSTSpike Stress = "rst-spike" // a burst of refusals: a middlebox shedding load
	Stress429      Stress = "429"
	Stress503      Stress = "503"
)

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

// The rate is the limiter's, the deadline the connect's: halving never moves one (ADR-0021).

type Backoff struct {
	base     int
	halvings int
	policy   BackoffPolicy
}

func NewBackoff(profile SafetyProfile) *Backoff {
	base := profile.PerHostConnPerSec
	if base < 1 {
		base = 1
	}
	return &Backoff{base: base, policy: profile.AdaptiveBackoff}
}

func (b *Backoff) Signal(cause Stress) {
	// A cause the offers did not declare is a no-op, so the runtime honours the commitment (ADR-0025).
	if !cause.enabledIn(b.policy) {
		return
	}
	if b.Rate() > 1 {
		b.halvings++
	}
}

func (b *Backoff) Rate() int {
	r := b.base >> b.halvings
	if r < 1 {
		return 1
	}
	return r
}

func (b *Backoff) Interval() time.Duration {
	return time.Second / time.Duration(b.Rate())
}

// The limiter lives in one prober process, so the guide forbids --scale worker=N (ADR-0137).

type Pacer struct {
	aggregateInterval time.Duration
	lastAggregate     time.Time
	lastHost          map[netip.Addr]time.Time
	backoff           map[netip.Addr]*Backoff
	profile           SafetyProfile
}

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

func (p *Pacer) Signal(host netip.Addr, cause Stress) { p.backoffFor(host).Signal(cause) }

func (p *Pacer) Next(host netip.Addr, now time.Time) time.Time {
	earliest := now
	// 50 conn/s is a 20 ms interval and 200 pkt/s a 5 ms one, so the per-host arm always wins (#1092).
	if !p.lastAggregate.IsZero() {
		if t := p.lastAggregate.Add(p.aggregateInterval); t.After(earliest) {
			earliest = t
		}
	}
	if last, ok := p.lastHost[host]; ok {
		if t := last.Add(p.backoffFor(host).Interval()); t.After(earliest) {
			earliest = t
		}
	}
	p.lastAggregate = earliest
	p.lastHost[host] = earliest
	return earliest
}

func RoundRobin(targets []netip.AddrPort) []netip.AddrPort {
	// Bursting one host's ports is the canonical port-scan signature and fills middleboxes (§3.3).
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
