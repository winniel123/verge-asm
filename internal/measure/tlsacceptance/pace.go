package tlsacceptance

import (
	"context"
	"net/netip"
	"time"
)

type Pacer struct {
	interval time.Duration
	lastHost map[string]time.Time
}

func NewPacer(set CandidateSet) *Pacer {
	rate := set.MaxHandshakesPerSecPerHost
	if rate < 1 {
		rate = 1
	}
	return &Pacer{
		interval: time.Second / time.Duration(rate),
		lastHost: map[string]time.Time{},
	}
}

func (p *Pacer) Next(host string, now time.Time) time.Time {
	earliest := now
	if last, ok := p.lastHost[host]; ok {
		if t := last.Add(p.interval); t.After(earliest) {
			earliest = t
		}
	}
	p.lastHost[host] = earliest
	return earliest
}

// The recorded ceiling is the commitment the runtime honours, not a label (ADR-0025, #1659).

type pacedEnumerator struct {
	inner Enumerator
	pacer *Pacer
	now   func() time.Time
	sleep func(context.Context, time.Duration)
}

func (p *pacedEnumerator) Handshake(ctx context.Context, target netip.AddrPort, version string, offeredCiphers []string) Attempt {
	// Pacing moves only when a handshake starts, never its deadline or its value (ADR-0021).
	due := p.pacer.Next(target.Addr().String(), p.now())
	if wait := due.Sub(p.now()); wait > 0 {
		p.sleep(ctx, wait)
	}
	return p.inner.Handshake(ctx, target, version, offeredCiphers)
}

func sleepCtx(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
