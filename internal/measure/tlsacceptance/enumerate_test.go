package tlsacceptance

import (
	"context"
	"net/netip"
	"testing"
)

// scriptEnumerator answers each handshake from a fixed model of one listener: which
// versions it accepts, and — per TLS 1.0–1.2 version — the suites it accepts in its
// own selection-preference order. It counts calls so a test can assert the
// enumeration cost (measurement-offers §1.5: accepted + 1 handshakes per version).
type scriptEnumerator struct {
	spoke   bool
	accepts map[string][]string // version -> accepted suites in listener preference order (nil-but-present for 1.3)
	present map[string]bool
	calls   int
}

func newScript(spoke bool, accepts map[string][]string) *scriptEnumerator {
	present := map[string]bool{}
	for v := range accepts {
		present[v] = true
	}
	return &scriptEnumerator{spoke: spoke, accepts: accepts, present: present}
}

func (s *scriptEnumerator) Handshake(_ context.Context, _ netip.AddrPort, version string, offered []string) Attempt {
	s.calls++
	if !s.spoke {
		return Attempt{}
	}
	if !s.present[version] {
		return Attempt{Spoke: true}
	}
	if version == TLS13 {
		return Attempt{Spoke: true, Accepted: true}
	}
	// The listener selects the first suite in its preference order that is offered.
	offeredSet := map[string]bool{}
	for _, c := range offered {
		offeredSet[c] = true
	}
	for _, pref := range s.accepts[version] {
		if offeredSet[pref] {
			return Attempt{Spoke: true, Accepted: true, SelectedCipher: pref}
		}
	}
	return Attempt{Spoke: true} // all its suites already peeled off — the refusing round
}

func target() netip.AddrPort { return netip.MustParseAddrPort("198.51.100.1:443") }

// A modern listener: TLS 1.2 with two GCM suites and TLS 1.3. The value enumerates
// both versions, 1.2 carries its accepted suites in selection order, 1.3 carries
// none (its suites are the library's choice — measurement-offers §1.3).
func TestEnumerateModern(t *testing.T) {
	e := newScript(true, map[string][]string{
		TLS12: {"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
		TLS13: nil,
	})
	v := Enumerate(context.Background(), e, DefaultCandidateSet(), target())
	if v.Outcome != Enumerated {
		t.Fatalf("outcome = %q, want enumerated", v.Outcome)
	}
	if len(v.Versions) != 2 {
		t.Fatalf("accepted %d versions, want 2 (1.2 and 1.3)", len(v.Versions))
	}
	if v.Versions[0].Version != TLS12 || len(v.Versions[0].Ciphers) != 2 {
		t.Errorf("1.2 acceptance = %+v, want two suites", v.Versions[0])
	}
	if v.Versions[1].Version != TLS13 || len(v.Versions[1].Ciphers) != 0 {
		t.Errorf("1.3 must record version-only, no suites, got %+v", v.Versions[1])
	}
}

// Iterative narrowing costs accepted + 1 handshakes per 1.0–1.2 version, not one
// per candidate (measurement-offers §1.5). A listener accepting two 1.2 suites and
// refusing 1.0/1.1 costs: 1.0 → 1, 1.1 → 1, 1.2 → 3 (2 accepts + 1 refuse), 1.3 → 1.
func TestEnumerateCostIsAcceptedPlusOne(t *testing.T) {
	e := newScript(true, map[string][]string{
		TLS12: {"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"},
		TLS13: nil,
	})
	_ = Enumerate(context.Background(), e, DefaultCandidateSet(), target())
	if e.calls != 1+1+3+1 {
		t.Errorf("enumeration cost = %d handshakes, want 6 (1+1+3+1) — narrowing, not one per candidate", e.calls)
	}
}

// TLS 1.0 acceptance is a finding: the value carries version 1.0, which reads the
// v1 signal `tls-1.0-accepted` (measurement-offers §1.2).
func TestEnumerateTLS10AcceptedIsRecorded(t *testing.T) {
	e := newScript(true, map[string][]string{
		TLS10: {"TLS_RSA_WITH_AES_128_CBC_SHA"},
		TLS12: {"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
	})
	v := Enumerate(context.Background(), e, DefaultCandidateSet(), target())
	if v.Outcome != Enumerated {
		t.Fatalf("outcome = %q, want enumerated", v.Outcome)
	}
	var saw10 bool
	for _, ver := range v.Versions {
		if ver.Version == TLS10 {
			saw10 = true
		}
	}
	if !saw10 {
		t.Error("a TLS 1.0 accept must appear in the value — it reads tls-1.0-accepted")
	}
}

// A peer that spoke TLS but accepted nothing offered is `tls-refused`, distinct
// from a plaintext port. Read with the batch's candidate set it is the finding
// *the peer spoke TLS and refused all of this* (measurement-offers §1.2).
func TestEnumerateTLSRefused(t *testing.T) {
	e := newScript(true, map[string][]string{}) // spoke, accepts nothing
	v := Enumerate(context.Background(), e, DefaultCandidateSet(), target())
	if v.Outcome != TLSRefused {
		t.Fatalf("outcome = %q, want tls-refused", v.Outcome)
	}
	if len(v.Versions) != 0 {
		t.Errorf("a refusal carries no accepted versions, got %+v", v.Versions)
	}
}

// A port where nothing spoke TLS at all is `no-tls`, a value distinct from a
// refusal — collapsing the two files a plaintext listener under *TLS server*.
func TestEnumerateNoTLS(t *testing.T) {
	e := newScript(false, nil)
	v := Enumerate(context.Background(), e, DefaultCandidateSet(), target())
	if v.Outcome != NoTLS {
		t.Fatalf("outcome = %q, want no-tls", v.Outcome)
	}
}

// The enumeration is deterministic: two runs against the same scripted listener
// fold to the identical value, so an unstable value never reads as drift.
func TestEnumerateDeterministic(t *testing.T) {
	mk := func() *scriptEnumerator {
		return newScript(true, map[string][]string{
			TLS12: {"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256"},
			TLS13: nil,
		})
	}
	a := Enumerate(context.Background(), mk(), DefaultCandidateSet(), target())
	b := Enumerate(context.Background(), mk(), DefaultCandidateSet(), target())
	if string(mustJSON(acceptanceValue{Outcome: a.Outcome, Versions: a.Versions})) !=
		string(mustJSON(acceptanceValue{Outcome: b.Outcome, Versions: b.Versions})) {
		t.Error("enumeration is not deterministic across two runs")
	}
}
