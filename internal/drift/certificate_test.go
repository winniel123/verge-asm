package drift

import "testing"

// The facet is a parameter and never a hardcode, so #197 needed no core change (ADR-0007).
// A certificate cannot itself change; what drifts is which chain an Endpoint presents (CONTEXT.md).

var epKey = TimelineKey{
	SubjectKind: "endpoint",
	SubjectKey:  "api.example.com@198.51.100.1:443/tcp",
	Facet:       "certificate",
	Source:      "prober",
}

func TestCertificateChainRotationOpensNewSpan(t *testing.T) {
	v := vec("tls-handshake", "tls-handshake/v1")
	spans := Fold(epKey, []Reading{
		{Value: `{"outcome":"presented","chain":["sha256:leafA","sha256:ca"]}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"outcome":"presented","chain":["sha256:leafA","sha256:ca"]}`, Vector: v, ObservedAt: day(1)},
		{Value: `{"outcome":"presented","chain":["sha256:leafB","sha256:ca"]}`, Vector: v, ObservedAt: day(2)},
	})
	if len(spans) != 2 {
		t.Fatalf("a chain rotation = %d spans, want 2 (old chain closed, new open)", len(spans))
	}
	if spans[0].Open() || !spans[0].ClosedAt.Equal(day(2)) {
		t.Errorf("the old chain span must close at the rotation instant, got open=%v closed=%v", spans[0].Open(), spans[0].ClosedAt)
	}
	if !spans[1].Open() || spans[1].Value != `{"outcome":"presented","chain":["sha256:leafB","sha256:ca"]}` {
		t.Error("the new chain span is current and open")
	}
	if spans[0].Reason != "" {
		t.Error("a rotation is an ordinary value move — no closure reason")
	}
}

func TestCertificatePresentedToNegativeIsAValueMove(t *testing.T) {
	v := vec("tls-handshake", "tls-handshake/v1")
	spans := Fold(epKey, []Reading{
		{Value: `{"outcome":"presented","chain":["sha256:leafA"]}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"outcome":"no-tls"}`, Vector: v, ObservedAt: day(1)},
	})
	if len(spans) != 2 || spans[0].Open() || !spans[1].Open() {
		t.Fatalf("presented -> no-tls = %d spans (want 2, first closed, second open)", len(spans))
	}
	for _, s := range spans {
		if s.IsGap {
			t.Error("a measured TLS negative is a value, never a Gap")
		}
	}
}

func TestCertificateVersionBumpBreaks(t *testing.T) {
	before := vec("tls-handshake", "tls-handshake/v1")
	after := vec("tls-handshake", "tls-handshake/v2")
	spans := Fold(epKey, []Reading{
		{Value: `{"outcome":"presented","chain":["sha256:leafA"]}`, Vector: before, ObservedAt: day(0)},
		{Value: `{"outcome":"presented","chain":["sha256:leafA"]}`, Vector: after, ObservedAt: day(1)},
	})
	breaks := Breaks(spans)
	if len(breaks) != 1 || breaks[0].MovedLeaves[0] != "tls-handshake" {
		t.Errorf("a tls-handshake bump must Break naming the leaf, got %v", breaks)
	}
}

func TestCertificateEndpointClosesUncitedWhenServiceWithdraws(t *testing.T) {
	// An Endpoint withdraws when either leg does (CONTEXT.md Endpoint).
	open := []Span{
		{Key: epKey, Value: `{"outcome":"presented","chain":["sha256:leafA"]}`, OpenedAt: day(0)},
	}
	closed := CloseWithdrawal(open, day(3), ReasonUncited)
	for _, s := range closed {
		if s.Open() {
			t.Error("a withdrawn Endpoint closes its certificate timeline")
		}
		if s.Reason != ReasonUncited {
			t.Errorf("the closure ground is uncited, got %q", s.Reason)
		}
	}
}
