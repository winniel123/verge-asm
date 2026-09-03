package drift

import "testing"

// The http-exchange leaf reuses this machinery unchanged, so the facet stays a parameter (#198).

var endpointKey = TimelineKey{
	SubjectKind: "endpoint",
	SubjectKey:  "api.example.com@198.51.100.1:443/tcp",
	Facet:       "http-identity",
	Source:      "prober",
}

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
