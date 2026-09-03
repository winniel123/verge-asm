package tlsacceptance

import "testing"

func TestDefaultCandidateSet(t *testing.T) {
	set := DefaultCandidateSet()
	if len(set.Versions) != 4 {
		t.Errorf("versions = %d, want 4 (TLS 1.0–1.3)", len(set.Versions))
	}
	if len(set.Ciphers) != 19 {
		t.Errorf("ciphers = %d, want 19 (13 limb-1 + 6 limb-2)", len(set.Ciphers))
	}
	if set.MaxHandshakesPerSecPerHost != 5 {
		t.Errorf("per-host handshake ceiling = %d, want 5 (#4 §6)", set.MaxHandshakesPerSecPerHost)
	}
	// A doubled candidate would make the narrowing loop count it twice.
	seen := map[string]bool{}
	for _, c := range set.Ciphers {
		if seen[c] {
			t.Errorf("duplicate declared suite %q", c)
		}
		seen[c] = true
	}
}

func TestDeclaredSetIsOfferable(t *testing.T) {
	if missing := OfferableCiphers(DefaultCandidateSet().Ciphers); len(missing) != 0 {
		t.Errorf("declared suites not offerable by the linked library: %v\n"+
			"the linked crypto/tls stopped offering these — narrow the declared set deliberately and bump the leaf version", missing)
	}
}

func TestCandidateSetDigestStable(t *testing.T) {
	if DefaultCandidateSet().Digest() != DefaultCandidateSet().Digest() {
		t.Error("candidate-set digest is not stable across two calls")
	}
}
