package httpexchange

import "time"

type Pacer struct {
	interval time.Duration
	lastHost map[string]time.Time
}

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
