package screen

import "testing"

func TestSignal(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    string
	}{
		{"a clean label", "ports census", ""},
		{"an ADR citation", "see ADR-0127 for the ruling", SignalCitation},
		{"an issue citation", "the shape #1133 fixed", SignalCitation},
		{"an RFC", "RFC3339 UTC", SignalExternalSpec},
		{"a bare RFC word", "the RFC says so", SignalExternalSpec},
		{"an ISO number", "ISO 8601 ordering", SignalExternalSpec},
		{"a reason word", "the read sits here because the guard runs first", SignalWhyMarker},
		{"a hazard word", "this panics on a nil estate", SignalWhyMarker},
		{"a bare URL", "see https://go.dev/ref/spec", SignalBareURL},
		{"a citation outranks a URL", "https://go.dev and ADR-0001", SignalCitation},
		{"a Deprecated paragraph", "F reports the estate.\n\nDeprecated: use G instead.", SignalToolMarker},
		{"the word deprecated in prose is not a tool marker", "the deprecated shim stays", ""},
		{"history is not a screen signal", "no longer used by the worker", ""},
		{"loose narration is not a screen signal", "now the queue drains", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Signal(c.payload); got != c.want {
				t.Errorf("Signal(%q) is %q, want %q", c.payload, got, c.want)
			}
		})
	}
}

func TestHistoryAndLooseNarrationSplit(t *testing.T) {
	if !HasHistoryMarker("previously the worker owned it") {
		t.Error("the strict set missed a history marker")
	}
	if HasHistoryMarker("now the queue drains") {
		t.Error("the strict set matched loose narration")
	}
	if !HasLooseNarration("now the queue drains") {
		t.Error("the loose set missed its own trap word")
	}
}
