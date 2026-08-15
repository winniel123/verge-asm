package drift

import "testing"

// The drift machinery is facet-agnostic: a Span is one period a value was held,
// and that shape does not vary by facet (ADR-0007). These cases pin that the same
// Fold / Break / Transition / Closure that serve `resolution` on a `Name` and
// `reachability` on a `Service` also serve the `certificate` facet on an
// `Endpoint` subject (#197) — the facet is a parameter, never a hardcode, so
// generalising to it needs no core change (only the facet CHECK widens).
//
// A certificate cannot itself change (CONTEXT.md `Certificate`); what drifts is
// WHICH certificate an Endpoint presents — the ordered chain of fingerprints —
// and the two TLS negatives (`tls-refused`, `no-tls`) are values that drift like
// any other, so a move between any two of the closed union opens a new Span.

var epKey = TimelineKey{
	SubjectKind: "endpoint",
	SubjectKey:  "api.example.com@198.51.100.1:443/tcp",
	Facet:       "certificate",
	Source:      "prober",
}

// A rotation — the Endpoint presents a new leaf — closes the old chain's span and
// opens the new one, an ordinary value move with no closure reason.
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

// A move from a presented chain to a measured negative (the listener stops
// speaking TLS) is a value move like any other — both are values in the closed
// union, so it opens a span rather than a Gap.
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

// A tls-handshake Version bump Breaks the certificate timeline exactly as a leaf
// bump Breaks resolution — the Break names the moved leaf.
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

// An Endpoint withdraws when either leg does (CONTEXT.md `Endpoint`): when the
// Service beneath it closes, CloseWithdrawal carries the `uncited` ground onto the
// certificate timeline the Endpoint held.
func TestCertificateEndpointClosesUncitedWhenServiceWithdraws(t *testing.T) {
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
