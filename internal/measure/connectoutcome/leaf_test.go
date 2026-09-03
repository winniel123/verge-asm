package connectoutcome

import (
	"context"
	"net/netip"
	"testing"
)

type scriptConnector struct {
	seq   map[netip.AddrPort][]ConnResult
	calls map[netip.AddrPort]int
}

func (s *scriptConnector) Connect(_ context.Context, t netip.AddrPort) ConnResult {
	if s.calls == nil {
		s.calls = map[netip.AddrPort]int{}
	}
	i := s.calls[t]
	s.calls[t]++
	res := s.seq[t]
	if i < len(res) {
		return res[i]
	}
	if len(res) == 0 {
		return ConnError
	}
	return res[len(res)-1]
}

func ap(s string) netip.AddrPort { return netip.MustParseAddrPort(s) }

func TestDecideVerdict(t *testing.T) {
	cases := map[ConnResult]Outcome{
		ConnOpen:     Reached,
		ConnRefused:  NotReached,
		ConnTimedOut: NotReached,
		ConnError:    NotReached,
	}
	for res, want := range cases {
		if got := Decide(res); got != want {
			t.Errorf("Decide(%s) = %s, want %s", res, got, want)
		}
	}
}

func TestProbeOpenDecidesImmediately(t *testing.T) {
	target := ap("198.51.100.1:443")
	c := &scriptConnector{seq: map[netip.AddrPort][]ConnResult{target: {ConnOpen}}}
	out, raw := Probe(context.Background(), c, DefaultProfile(), target)
	if out != Reached || raw != ConnOpen {
		t.Fatalf("got %s/%s, want reached/open", out, raw)
	}
	if c.calls[target] != 1 {
		t.Errorf("open should decide in one attempt, got %d", c.calls[target])
	}
}

func TestProbeRefusalNotRetried(t *testing.T) {
	target := ap("198.51.100.1:3306")
	c := &scriptConnector{seq: map[netip.AddrPort][]ConnResult{target: {ConnRefused}}}
	out, _ := Probe(context.Background(), c, DefaultProfile(), target)
	if out != NotReached {
		t.Fatalf("refused connect = %s, want not-reached", out)
	}
	if c.calls[target] != 1 {
		t.Errorf("a refusal must not be retried, got %d attempts", c.calls[target])
	}
}

func TestProbeRetriesTransientTimeout(t *testing.T) {
	target := ap("198.51.100.1:8080")
	c := &scriptConnector{seq: map[netip.AddrPort][]ConnResult{target: {ConnTimedOut, ConnOpen}}}
	out, _ := Probe(context.Background(), c, DefaultProfile(), target)
	if out != Reached {
		t.Fatalf("got %s, want reached after a retry", out)
	}
	if c.calls[target] != 2 {
		t.Errorf("want 2 attempts (1 timeout + 1 open), got %d", c.calls[target])
	}
}

func TestProbeExhaustsRetriesToNotReached(t *testing.T) {
	target := ap("198.51.100.1:9200")
	c := &scriptConnector{seq: map[netip.AddrPort][]ConnResult{target: {ConnTimedOut}}}
	out, raw := Probe(context.Background(), c, DefaultProfile(), target)
	if out != NotReached || raw != ConnTimedOut {
		t.Fatalf("got %s/%s, want not-reached/timed-out", out, raw)
	}
	if c.calls[target] != 3 {
		t.Errorf("want 3 attempts (1 + 2 retries), got %d", c.calls[target])
	}
}

func TestDefaultProfileMatchesTable(t *testing.T) {
	// The shipped profile is the v1 spec §3.3 safety table exactly.
	p := DefaultProfile()
	if p.Technique != "tcp-connect" {
		t.Errorf("technique = %q, want tcp-connect (never SYN)", p.Technique)
	}
	if p.HostDiscovery != "skipped" {
		t.Errorf("host discovery = %q, want skipped (-Pn)", p.HostDiscovery)
	}
	if p.PerHostConnPerSec != 50 {
		t.Errorf("per-host rate = %d, want 50 conn/s", p.PerHostConnPerSec)
	}
	if p.PerHostConcurrency != 20 {
		t.Errorf("per-host concurrency = %d, want 20", p.PerHostConcurrency)
	}
	if p.ConnectTimeoutMillis != 3000 {
		t.Errorf("timeout = %d ms, want 3000", p.ConnectTimeoutMillis)
	}
	if p.Retries != 2 {
		t.Errorf("retries = %d, want 2", p.Retries)
	}
	if p.PerVantagePacketsPerSec != 200 {
		t.Errorf("per-vantage ceiling = %d, want 200 pkt/s", p.PerVantagePacketsPerSec)
	}
	if !p.RoundRobinByHost {
		t.Errorf("round-robin by host must be set")
	}
	if p.AdaptiveBackoff.TouchesDeadline {
		t.Errorf("adaptive back-off must never touch the deadline (ADR-0021)")
	}
	if !p.AdaptiveBackoff.HalveOnTimeout || !p.AdaptiveBackoff.HalveOnRSTSpike ||
		!p.AdaptiveBackoff.HalveOn429 || !p.AdaptiveBackoff.HalveOn503 {
		t.Errorf("adaptive back-off must halve on timeout/RST-spike/429/503")
	}
}
