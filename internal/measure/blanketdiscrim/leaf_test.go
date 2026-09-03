package blanketdiscrim

import "testing"

func TestDecideVerdicts(t *testing.T) {
	all := func(r ControlResult, n int) []ControlResult {
		out := make([]ControlResult, n)
		for i := range out {
			out[i] = r
		}
		return out
	}
	cases := []struct {
		name string
		in   []ControlResult
		want Verdict
	}{
		{"whole set answers is blanket", all(ControlAnswered, 8), VerdictBlanket},
		{"one refusal clears blanket", []ControlResult{ControlAnswered, ControlAnswered, ControlClosed}, VerdictNotBlanket},
		{"a refusal anywhere wins over silence", []ControlResult{ControlAnswered, ControlIncomplete, ControlClosed}, VerdictNotBlanket},
		{"all silent is a gap", all(ControlIncomplete, 8), VerdictGap},
		{"answered plus silent, no refusal, is a gap", []ControlResult{ControlAnswered, ControlIncomplete}, VerdictGap},
		{"empty set does not gap (opt-out)", nil, VerdictNotBlanket},
	}
	for _, c := range cases {
		if got := Decide(c.in); got != c.want {
			t.Errorf("%s: Decide(%v) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestVerdictGapsAndReasons(t *testing.T) {
	if !VerdictBlanket.Gaps() || !VerdictGap.Gaps() {
		t.Error("a blanket responder and an incomplete probe both gap the reach")
	}
	if VerdictNotBlanket.Gaps() {
		t.Error("a not-blanket verdict must not gap the reach")
	}
	if ReasonFor(VerdictBlanket) != ReasonBlanket {
		t.Errorf("blanket reason = %q", ReasonFor(VerdictBlanket))
	}
	if ReasonFor(VerdictGap) != ReasonIncomplete {
		t.Errorf("incomplete reason = %q", ReasonFor(VerdictGap))
	}
	if ReasonFor(VerdictNotBlanket) != "" {
		t.Error("a not-blanket verdict has no gap reason")
	}
}

func TestCryptoPortsShape(t *testing.T) {
	p := CryptoPorts{}.Ports()
	if len(p) != ControlPortCount {
		t.Fatalf("drew %d ports, want %d", len(p), ControlPortCount)
	}
	seen := map[uint16]struct{}{}
	for i, port := range p {
		if port < portBandLow || port > portBandHigh {
			t.Errorf("port %d outside the dynamic range [%d,%d]", port, portBandLow, portBandHigh)
		}
		if _, dup := seen[port]; dup {
			t.Errorf("duplicate control port %d", port)
		}
		seen[port] = struct{}{}
		if i > 0 && p[i-1] > port {
			t.Errorf("ports not sorted ascending at %d", i)
		}
	}
}

func TestFixedPortsDeterministic(t *testing.T) {
	got := FixedPorts{P: []uint16{60000, 50000, 60000, 55000}}.Ports()
	want := []uint16{50000, 55000, 60000}
	if len(got) != len(want) {
		t.Fatalf("FixedPorts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("FixedPorts = %v, want %v", got, want)
		}
	}
}

func TestParamsDigestStable(t *testing.T) {
	if DefaultParams().Digest() != DefaultParams().Digest() {
		t.Error("params digest is not stable")
	}
}
