package tlsacceptance

import (
	"context"
	"net/netip"
	"testing"
)

type scriptEnumerator struct {
	spoke   bool
	accepts map[string][]string
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
	offeredSet := map[string]bool{}
	for _, c := range offered {
		offeredSet[c] = true
	}
	for _, pref := range s.accepts[version] {
		if offeredSet[pref] {
			return Attempt{Spoke: true, Accepted: true, SelectedCipher: pref}
		}
	}
	return Attempt{Spoke: true}
}

func target() netip.AddrPort { return netip.MustParseAddrPort("198.51.100.1:443") }

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

func TestEnumerateTLSRefused(t *testing.T) {
	e := newScript(true, map[string][]string{})
	v := Enumerate(context.Background(), e, DefaultCandidateSet(), target())
	if v.Outcome != TLSRefused {
		t.Fatalf("outcome = %q, want tls-refused", v.Outcome)
	}
	if len(v.Versions) != 0 {
		t.Errorf("a refusal carries no accepted versions, got %+v", v.Versions)
	}
}

func TestEnumerateNoTLS(t *testing.T) {
	e := newScript(false, nil)
	v := Enumerate(context.Background(), e, DefaultCandidateSet(), target())
	if v.Outcome != NoTLS {
		t.Fatalf("outcome = %q, want no-tls", v.Outcome)
	}
}

func TestEnumerateDeterministic(t *testing.T) {
	// An unstable value would read as drift on the timeline, not as a listener change.
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
