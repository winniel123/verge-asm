package httpexchange

import "time"

// Pacer computes when each `GET /` may start, honouring the per-host request
// ceiling (§3.3). It holds no timers: Next is pure arithmetic over a
// caller-supplied clock, so a test drives it with a fixed `now` and asserts the
// spacing rather than sleeping; a real run sleeps until the returned instant. Like
// the connect-outcome limiter it lives OUTSIDE the identity fold and can neither
// manufacture nor suppress an http-identity value — it changes only timing.
type Pacer struct {
	interval time.Duration
	lastHost map[string]time.Time
}

// NewPacer builds a Pacer over the profile's per-host request rate, flooring at
// one request per second so a zero or negative rate cannot divide by zero.
func NewPacer(p Params) *Pacer {
	rate := p.PerHostReqPerSec
	if rate < 1 {
		rate = 1
	}
	return &Pacer{
		interval: time.Second / time.Duration(rate),
		lastHost: map[string]time.Time{},
	}
}

// Next returns the instant the next exchange to host may start, given the current
// clock: the per-host interval since that host's last emit, or now where the host
// is fresh. Calling Next records the emit at the returned instant, so a sequence of
// calls to one host produces a correctly-spaced schedule.
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
