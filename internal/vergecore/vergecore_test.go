package vergecore

import "testing"

func TestShippedComposition(t *testing.T) {
	c := Default().Count()
	if c.Frequency != 123 {
		t.Errorf("frequency half = %d, want 123 (§3.5)", c.Frequency)
	}
	if c.Sensitive != 38 {
		t.Errorf("sensitive half = %d pairs, want 38 (§3.5)", c.Sensitive)
	}
	if c.Union != 136 {
		t.Errorf("union = %d, want 136 pairs (§3.5)", c.Union)
	}
	if c.TCP != 131 {
		t.Errorf("union TCP = %d, want 131 (§3.5)", c.TCP)
	}
	if c.UDP != 5 {
		t.Errorf("union UDP = %d, want 5 (§3.5)", c.UDP)
	}
}

func TestFrequencyIsTCPOnly(t *testing.T) {
	// Every UDP pair sits on the sensitive half, so the operator can never edit one.
	for _, p := range Default().FrequencyPairs() {
		if p.Transport != TCP {
			t.Errorf("frequency pair %s is not TCP; the frequency half is TCP-only", p)
		}
	}
	udp := 0
	for _, p := range Default().SensitivePairs() {
		if p.Transport == UDP {
			udp++
		}
	}
	if udp != 5 {
		t.Errorf("sensitive half holds %d UDP pairs, want all 5", udp)
	}
}

func TestTCPProbedAndUDPRecorded(t *testing.T) {
	l := Default()
	if got := len(l.TCPProbed()); got != 131 {
		t.Errorf("TCPProbed = %d, want 131", got)
	}
	if got := len(l.UDPRecorded()); got != 5 {
		t.Errorf("UDPRecorded = %d, want 5", got)
	}
	for _, p := range l.TCPProbed() {
		if p.Transport != TCP {
			t.Errorf("TCPProbed returned non-TCP pair %s", p)
		}
	}
	for _, p := range l.UDPRecorded() {
		if p.Transport != UDP {
			t.Errorf("UDPRecorded returned non-UDP pair %s", p)
		}
	}
}

func TestUnionDeduplicates(t *testing.T) {
	c := Default().Count()
	// 123 frequency plus 33 sensitive TCP would be 156 if disjoint; the union folds to 131.
	if c.Frequency+33 == c.TCP {
		t.Errorf("union did not deduplicate the frequency∩sensitive overlap")
	}
}

func TestFrequencyEditAdd(t *testing.T) {
	before := Default().Count()
	edited := Default().WithFrequencyEdits([]FrequencyEdit{{Port: 12345, Action: ActionAdd}})
	after := edited.Count()
	if after.Frequency != before.Frequency+1 {
		t.Errorf("add: frequency %d -> %d, want +1", before.Frequency, after.Frequency)
	}
	if !edited.IsFrequency(Pair{Port: 12345, Transport: TCP}) {
		t.Errorf("added port 12345/tcp is not on the frequency half")
	}
	if after.TCP != before.TCP+1 {
		t.Errorf("add: probed TCP %d -> %d, want +1", before.TCP, after.TCP)
	}
}

func TestFrequencyEditRemoveFrequencyOnly(t *testing.T) {
	p := Pair{Port: 22, Transport: TCP}
	l := Default()
	if l.IsSensitive(p) {
		t.Fatalf("test assumes 22/tcp is frequency-only")
	}
	edited := l.WithFrequencyEdits([]FrequencyEdit{{Port: 22, Action: ActionRemove}})
	if edited.IsFrequency(p) {
		t.Errorf("22/tcp still on the frequency half after removal")
	}
	if edited.Count().TCP != l.Count().TCP-1 {
		t.Errorf("removing a frequency-only port did not shrink the probed union")
	}
}

func TestFrequencyEditCannotMoveSensitive(t *testing.T) {
	// The §3.5 invariant: an operator edit can never move the sensitive half.
	p := Pair{Port: 445, Transport: TCP}
	l := Default()
	if !l.IsSensitive(p) || !l.IsFrequency(p) {
		t.Fatalf("test assumes 445/tcp is on both halves")
	}
	edited := l.WithFrequencyEdits([]FrequencyEdit{{Port: 445, Action: ActionRemove}})
	if edited.IsFrequency(p) {
		t.Errorf("445/tcp should be off the frequency half after removal")
	}
	if !edited.IsSensitive(p) {
		t.Errorf("445/tcp left the sensitive half — the sensitive half must be immutable")
	}
	in := false
	for _, u := range edited.Union() {
		if u == p {
			in = true
		}
	}
	if !in {
		t.Errorf("445/tcp left the probed union; a frequency removal must not move a sensitive pair")
	}
}

func TestFrequencyEditIgnoresNonTCP(t *testing.T) {
	before := Default().Count()
	after := Default().WithFrequencyEdits([]FrequencyEdit{{Port: 0, Action: ActionAdd}}).Count()
	if after != before {
		t.Errorf("a zero-port edit changed the set: %+v -> %+v", before, after)
	}
}
