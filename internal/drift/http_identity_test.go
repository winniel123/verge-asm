package drift

import "testing"

// The drift machinery is facet-agnostic: a Span is one period a value was held,
// and that shape does not vary by facet (ADR-0007). These cases pin that the same
// Fold / Break / Transition / Closure the resolution and reachability tests
// exercise also serve the `http-identity` facet on an `Endpoint` subject — the
// (Name, Service) pair. The http-exchange leaf (#198) reuses this machinery
// unchanged, so the facet stays a parameter and never a hardcode.

var endpointKey = TimelineKey{
	SubjectKind: "endpoint",
	SubjectKey:  "api.example.com@198.51.100.1:443/tcp",
	Facet:       "http-identity",
	Source:      "prober",
}

// AC #198: re-running the hot Scan with a changed HTTP identity produces the
// correct Span transition — the old identity span closes and the new one opens.
func TestHTTPIdentityOpensAndClosesOnChange(t *testing.T) {
	v := vec("http-exchange", "http-exchange/v1")
	spans := Fold(endpointKey, []Reading{
		{Value: `{"status":200,"server":"nginx","body_sha256":"sha256:aaa","body_bytes":10,"body_truncated":false}`, Vector: v, ObservedAt: day(0)},
		{Value: `{"status":200,"server":"nginx","body_sha256":"sha256:aaa","body_bytes":10,"body_truncated":false}`, Vector: v, ObservedAt: day(1)},
		{Value: `{"status":200,"server":"caddy","body_sha256":"sha256:bbb","body_bytes":12,"body_truncated":false}`, Vector: v, ObservedAt: day(2)},
	})
	if len(spans) != 2 {
		t.Fatalf("an identity change = %d spans, want 2 (old closed, new open)", len(spans))
	}
	if spans[0].Open() || !spans[0].ClosedAt.Equal(day(2)) {
		t.Errorf("the first identity span must close at the change instant, got open=%v closed=%v", spans[0].Open(), spans[0].ClosedAt)
	}
	if !spans[1].Open() {
		t.Error("the new identity span is current and open")
	}
	if spans[0].Reason != "" {
		t.Error("an identity change is an ordinary value move — no closure reason")
	}
}

// An http-exchange Version bump Breaks the http-identity timeline exactly as a
// leaf bump Breaks resolution — the Break names the moved leaf.
func TestHTTPIdentityVersionBumpBreaks(t *testing.T) {
	before := vec("http-exchange", "http-exchange/v1")
	after := vec("http-exchange", "http-exchange/v2")
	spans := Fold(endpointKey, []Reading{
		{Value: `{"status":200,"body_sha256":"sha256:aaa","body_bytes":1,"body_truncated":false}`, Vector: before, ObservedAt: day(0)},
		{Value: `{"status":200,"body_sha256":"sha256:aaa","body_bytes":1,"body_truncated":false}`, Vector: after, ObservedAt: day(1)},
	})
	breaks := Breaks(spans)
	if len(breaks) != 1 || breaks[0].MovedLeaves[0] != "http-exchange" {
		t.Errorf("an http-exchange bump must Break naming the leaf, got %v", breaks)
	}
}

// AC #198: an Endpoint closes when its Service withdraws (the underlying Address
// is de-cited) — CloseWithdrawal carries the `uncited` ground onto the Endpoint's
// http-identity timeline, since an Endpoint closes when either leg withdraws.
func TestHTTPIdentityClosesUncitedWhenServiceWithdraws(t *testing.T) {
	open := []Span{
		{Key: endpointKey, Value: `{"status":200}`, OpenedAt: day(0)},
		{Key: TimelineKey{SubjectKind: "endpoint", SubjectKey: "@198.51.100.1:80/tcp", Facet: "http-identity", Source: "prober"}, OpenedAt: day(0)},
	}
	closed := CloseWithdrawal(open, day(3), ReasonUncited)
	for _, s := range closed {
		if s.Open() {
			t.Error("a withdrawn Service closes every Endpoint http-identity timeline it held")
		}
		if s.Reason != ReasonUncited {
			t.Errorf("the closure ground is uncited, got %q", s.Reason)
		}
	}
}
