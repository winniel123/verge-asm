package tlsacceptance

import "testing"

// The shipped candidate set is measurement-offers §1.2/§1.3 exactly: four versions
// TLS 1.0–1.3, and the nineteen declared TLS 1.0–1.2 suites (13 limb-1 findings +
// 6 limb-2 modal suites). Widening it is a Break on every tls-acceptance AND every
// certificate timeline, so the count is pinned here.
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
	// No duplicate suite may sit in the offer — a doubled candidate would make the
	// narrowing loop count it twice.
	seen := map[string]bool{}
	for _, c := range set.Ciphers {
		if seen[c] {
			t.Errorf("duplicate declared suite %q", c)
		}
		seen[c] = true
	}
}

// The §1.4 build gate, exercised: every declared candidate must be offerable by the
// linked crypto/tls. A Go release dropping a declared suite fails HERE — a
// deliberate, priced narrowing rather than a silent one (measurement-offers §1.4).
func TestDeclaredSetIsOfferable(t *testing.T) {
	if missing := OfferableCiphers(DefaultCandidateSet().Ciphers); len(missing) != 0 {
		t.Errorf("declared suites not offerable by the linked library: %v\n"+
			"the linked crypto/tls stopped offering these — narrow the declared set deliberately and bump the leaf version", missing)
	}
}

// The candidate-set digest is stable across marshals — the golden-corpus lock binds
// a declared-parameter change to a version bump through it, so it must not depend on
// map iteration order.
func TestCandidateSetDigestStable(t *testing.T) {
	if DefaultCandidateSet().Digest() != DefaultCandidateSet().Digest() {
		t.Error("candidate-set digest is not stable across two calls")
	}
}
