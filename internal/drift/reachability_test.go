package drift

import "testing"

// The drift machinery is facet-agnostic: a Span is one period a value was held,
// and that shape does not vary by facet (ADR-0007). These cases pin that the same
// Fold / Break / Transition / Closure the resolution tests exercise on a `Name`
// also serve the `reachability` facet on a `Service` subject — the wave-4 facet
// tickets (#197 certificate, #198 http-identity) reuse this unchanged, so the
// facet must remain a parameter and never a hardcode (AC #195).

var svcKey = TimelineKey{
	SubjectKind: "service",
	SubjectKey:  "198.51.100.1:443/tcp",
	Facet:       "reachability",
	Source:      "prober",
}

// AC #195: re-running the hot Scan with a Service opening produces the correct
// Span transition — a not-reached span closes and a reached span opens.
func TestReachabilityServiceOpensAndCloses(t *testing.T) {
	v := vec("connect-outcome", "connect-outcome/v1")
	spans := Fold(svcKey, []Reading{
		{Value: `{"outcome":"not-reached","result":"refused"}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"outcome":"not-reached","result":"refused"}`, Vector: v, ObservedAt: day(1)},
		{Value: `{"outcome":"reached","result":"open"}`, Vector: v, ObservedAt: day(2)},
	})
	if len(spans) != 2 {
		t.Fatalf("a Service opening = %d spans, want 2 (not-reached closed, reached open)", len(spans))
	}
	if spans[0].Open() || !spans[0].ClosedAt.Equal(day(2)) {
		t.Errorf("the not-reached span must close at the opening instant, got open=%v closed=%v", spans[0].Open(), spans[0].ClosedAt)
	}
	if !spans[1].Open() || spans[1].Value != `{"outcome":"reached","result":"open"}` {
		t.Error("the reached span is current and open")
	}
	if spans[0].Reason != "" {
		t.Error("a Service opening is an ordinary value move — no closure reason")
	}
}

// A Service closing (reached -> not-reached) is the mirror transition.
func TestReachabilityServiceClosesPort(t *testing.T) {
	v := vec("connect-outcome", "connect-outcome/v1")
	spans := Fold(svcKey, []Reading{
		{Value: `{"outcome":"reached","result":"open"}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"outcome":"not-reached","result":"timed-out"}`, Vector: v, ObservedAt: day(1)},
	})
	if len(spans) != 2 || spans[0].Open() || !spans[1].Open() {
		t.Fatalf("a closing port = %d spans (want 2, first closed, second open)", len(spans))
	}
}

// A connect-outcome Version bump Breaks the reachability timeline exactly as a
// leaf bump Breaks resolution — the Break names the moved leaf.
func TestReachabilityVersionBumpBreaks(t *testing.T) {
	before := vec("connect-outcome", "connect-outcome/v1")
	after := vec("connect-outcome", "connect-outcome/v2")
	spans := Fold(svcKey, []Reading{
		{Value: `{"outcome":"reached","result":"open"}`, Vector: before, ObservedAt: day(0)},
		{Value: `{"outcome":"reached","result":"open"}`, Vector: after, ObservedAt: day(1)},
	})
	breaks := Breaks(spans)
	if len(breaks) != 1 || breaks[0].MovedLeaves[0] != "connect-outcome" {
		t.Errorf("a connect-outcome bump must Break naming the leaf, got %v", breaks)
	}
}

// AC #195: an Address's Services close under the `uncited` ground when a
// resolution stops citing the Address — CloseWithdrawal carries that ground onto
// every reachability timeline the Service held.
func TestReachabilityServiceClosesUncitedOnDeCitation(t *testing.T) {
	open := []Span{
		{Key: svcKey, Value: `{"outcome":"reached","result":"open"}`, OpenedAt: day(0)},
		{Key: TimelineKey{SubjectKind: "service", SubjectKey: "198.51.100.1:80/tcp", Facet: "reachability", Source: "prober"}, OpenedAt: day(0)},
	}
	closed := CloseWithdrawal(open, day(3), ReasonUncited)
	for _, s := range closed {
		if s.Open() {
			t.Error("a de-cited Address withdraws every Service reachability timeline it held")
		}
		if s.Reason != ReasonUncited {
			t.Errorf("the closure ground is uncited, got %q", s.Reason)
		}
	}
}
