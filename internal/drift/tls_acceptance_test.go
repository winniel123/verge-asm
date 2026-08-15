package drift

import "testing"

// The drift machinery is facet-agnostic: a Span is one period a value was held, and
// that shape does not vary by facet (ADR-0007). These cases pin that the same Fold /
// Break / Transition / Closure the resolution, reachability, certificate and
// http-identity tests exercise ALSO serve the `tls-acceptance` facet on a `Service`
// subject (AC #199: "Drift engine covers tls-acceptance"). The tls-acceptance leaf
// reuses this machinery unchanged, so the facet stays a parameter and never a
// hardcode.

var serviceTLSKey = TimelineKey{
	SubjectKind: "service",
	SubjectKey:  "198.51.100.1:443/tcp",
	Facet:       "tls-acceptance",
	Source:      "prober",
}

// A Service that stops accepting TLS 1.0 produces the correct Span transition — the
// old acceptance span closes and the new one opens. This is the drift that carries
// a `tls-1.0-accepted` clearing.
func TestTLSAcceptanceOpensAndClosesOnChange(t *testing.T) {
	v := vec("tls-acceptance", "tls-acceptance/v1")
	spans := Fold(serviceTLSKey, []Reading{
		{Value: `{"outcome":"enumerated","versions":[{"version":"1.0","ciphers":["TLS_RSA_WITH_AES_128_CBC_SHA"]},{"version":"1.2","ciphers":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]}]}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"outcome":"enumerated","versions":[{"version":"1.0","ciphers":["TLS_RSA_WITH_AES_128_CBC_SHA"]},{"version":"1.2","ciphers":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]}]}`, Vector: v, ObservedAt: day(7)},
		{Value: `{"outcome":"enumerated","versions":[{"version":"1.2","ciphers":["TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"]}]}`, Vector: v, ObservedAt: day(14)},
	})
	if len(spans) != 2 {
		t.Fatalf("an acceptance change = %d spans, want 2 (old closed, new open)", len(spans))
	}
	if spans[0].Open() || !spans[0].ClosedAt.Equal(day(14)) {
		t.Errorf("the first acceptance span must close at the change instant, got open=%v closed=%v", spans[0].Open(), spans[0].ClosedAt)
	}
	if !spans[1].Open() {
		t.Error("the new acceptance span is current and open")
	}
	if spans[0].Reason != "" {
		t.Error("an acceptance change is an ordinary value move — no closure reason")
	}
}

// The two TLS negatives are distinct values: a Service moving from `tls-refused` to
// `no-tls` is a value move that opens a new span, never a collapse into one bucket.
func TestTLSAcceptanceNegativesAreDistinctValues(t *testing.T) {
	v := vec("tls-acceptance", "tls-acceptance/v1")
	spans := Fold(serviceTLSKey, []Reading{
		{Value: `{"outcome":"tls-refused"}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"outcome":"no-tls"}`, Vector: v, ObservedAt: day(7)},
	})
	if len(spans) != 2 {
		t.Fatalf("tls-refused -> no-tls = %d spans, want 2 — the negatives are distinct values", len(spans))
	}
}

// A tls-acceptance Version bump Breaks the timeline exactly as a leaf bump Breaks
// resolution — the Break names the moved leaf. Widening the candidate set is such a
// bump (CONTEXT.md `tls-acceptance`).
func TestTLSAcceptanceVersionBumpBreaks(t *testing.T) {
	before := vec("tls-acceptance", "tls-acceptance/v1")
	after := vec("tls-acceptance", "tls-acceptance/v2")
	spans := Fold(serviceTLSKey, []Reading{
		{Value: `{"outcome":"enumerated","versions":[{"version":"1.3"}]}`, Vector: before, ObservedAt: day(0)},
		{Value: `{"outcome":"enumerated","versions":[{"version":"1.3"}]}`, Vector: after, ObservedAt: day(7)},
	})
	breaks := Breaks(spans)
	if len(breaks) != 1 || breaks[0].MovedLeaves[0] != "tls-acceptance" {
		t.Errorf("a tls-acceptance bump must Break naming the leaf, got %v", breaks)
	}
}

// A tls-acceptance timeline closes when its Service withdraws — CloseWithdrawal
// carries the `uncited` ground onto the Service's tls-acceptance timeline, since the
// facet rides the Service and closes with it.
func TestTLSAcceptanceClosesUncitedWhenServiceWithdraws(t *testing.T) {
	open := []Span{
		{Key: serviceTLSKey, Value: `{"outcome":"enumerated","versions":[{"version":"1.2"}]}`, OpenedAt: day(0)},
	}
	closed := CloseWithdrawal(open, day(21), ReasonUncited)
	for _, s := range closed {
		if s.Open() {
			t.Error("a withdrawn Service closes its tls-acceptance timeline")
		}
		if s.Reason != ReasonUncited {
			t.Errorf("the closure ground is uncited, got %q", s.Reason)
		}
	}
}
