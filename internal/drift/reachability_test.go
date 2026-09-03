package drift

import "testing"

// The facet stays a parameter and never a hardcode, so the wave-4 facets reuse this (#195).

var svcKey = TimelineKey{
	SubjectKind: "service",
	SubjectKey:  "198.51.100.1:443/tcp",
	Facet:       "reachability",
	Source:      "prober",
}

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

func TestReachabilityServiceClosesUncitedOnDeCitation(t *testing.T) {
	// A resolution that stops citing an Address withdraws every Service beneath it (#195).
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
