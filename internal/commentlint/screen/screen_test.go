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

func TestRoundOneFailuresNowScreen(t *testing.T) {
	// Each payload is the reason clause of a block round 1 read as load-bearing (#1136).
	cases := []struct {
		name    string
		payload string
		want    string
	}{
		{"a bare so", "it is exported so SendSigned can be handed the same resolver", SignalWhyMarker},
		{"a rejected alternative", "it reports ok=false rather than fabricate a figure", SignalWhyMarker},
		{"an alternative named instead", "the mark is derived instead of stored", SignalWhyMarker},
		{"a never", "the board is a census, never a sampled or ranked view", SignalWhyMarker},
		{"an absent thing", "there is no network_position field and no wizard step", SignalWhyMarker},
		{"absent things", "there are no mail credentials on a self-hosted host", SignalWhyMarker},
		{"a section citation", "a database restore does not rotate it (v1 spec §4.3)", SignalCitation},
		{"a bare section sign", "the cap the rubric sets (§4.8)", SignalCitation},
		{"a CONTEXT reference", "the row declares the vantage class (CONTEXT.md)", SignalCitation},
		{"a design-fixture reference", "one per-file zone-upload refusal (DF-F2)", SignalCitation},
		{"a gosec waiver", "the password is #nosec G101: only ever a dev database", SignalToolMarker},
		// A bare "spec" is no citation: it rides in a URL, which already withholds the block (#1136).
		{"a spec URL keeps its own reason", "see https://go.dev/ref/spec", SignalBareURL},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Signal(c.payload); got != c.want {
				t.Errorf("Signal(%q) is %q, want %q", c.payload, got, c.want)
			}
		})
	}
}

func TestTheWideningNarrowedNothing(t *testing.T) {
	// §3.2 lets a revision widen the screen and never narrow it (#1136).
	why := []string{
		"because", "otherwise", "so that", "avoid", "avoids", "workaround",
		"work around", "race", "races", "panic", "panics", "deliberate",
		"deliberately", "intentional", "intentionally", "on purpose", "hazard",
		"gotcha", "beware", "deadlock", "must not", "cannot", "prevent",
		"prevents", "load-bearing", "unsafe", "breaks", "corrupts",
	}
	for _, word := range why {
		if !HasWhyMarker("the guard " + word + " here") {
			t.Errorf("the widened WHY list dropped %q", word)
		}
	}
	for _, payload := range []string{"see ADR-0127", "the shape #1133 fixed"} {
		if !HasCitation(payload) {
			t.Errorf("the widened citation test dropped %q", payload)
		}
	}
	if Signal("F reports the estate.\n\nDeprecated: use G instead.") != SignalToolMarker {
		t.Error("the widened tool-marker test dropped the Deprecated paragraph")
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
