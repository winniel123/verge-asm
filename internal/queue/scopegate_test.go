package queue

import (
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

// subjectsOf lists the surviving observations' subjects, for order-preserving
// assertions on what the gate admitted.
func subjectsOf(obs []wire.Observation) []string {
	out := make([]string, 0, len(obs))
	for _, o := range obs {
		out = append(out, o.Subject)
	}
	return out
}

// TestGateAdmitsInScopeAndDropsOutOfScopeAddresses proves the recording-side re-gate
// (#773) on the address dimension: a reachability observation whose Service subject
// is one of the job's authorised addresses is kept and would be written, while one
// for an address the job never dispatched is dropped before any DB write or fold. The
// subjects are produced by the REAL prober emit helper, so the format the gate parses
// is exactly the one a legitimate prober emits.
func TestGateAdmitsInScopeAndDropsOutOfScopeAddresses(t *testing.T) {
	// A hot job's AttemptedScope, by content, exactly as scan.HotJob.AttemptedScope
	// marshals it (internal/scan/hot.go hotScopeRecord json tags).
	scope := parseAuthorizedScope([]byte(`{"vantage":"v1","addresses":["198.51.100.1"],"tcp_ports":[443]}`))

	inScope := connectoutcome.EmitService(
		"b1", "v1", netip.MustParseAddrPort("198.51.100.1:443"),
		connectoutcome.Reached, connectoutcome.ConnResult("open"),
	)
	// A compromised prober fabricating a Service for an address it was never
	// dispatched to measure (outside the authorised address set).
	outOfScope := connectoutcome.EmitService(
		"b1", "v1", netip.MustParseAddrPort("203.0.113.9:443"),
		connectoutcome.Reached, connectoutcome.ConnResult("open"),
	)

	got := scope.gate([]wire.Observation{inScope, outOfScope}, nil, 7)

	if len(got) != 1 {
		t.Fatalf("expected exactly the in-scope observation to survive, got %d: %v", len(got), subjectsOf(got))
	}
	if got[0].Subject != inScope.Subject {
		t.Errorf("kept the wrong subject: got %q, want %q", got[0].Subject, inScope.Subject)
	}
	if got[0].Subject != "198.51.100.1:443/tcp" {
		t.Errorf("unexpected in-scope subject spelling: %q", got[0].Subject)
	}
}

// TestGateAdmitsInScopeAndDropsOutOfScopeNames proves the same on the name dimension:
// a resolution observation whose Name subject is one the dns job resolved is kept,
// while a Name the job never resolved (fabricated by a compromised prober) is dropped.
// A resolution subject is exactly CanonicalName(one of the job's Names), so exact
// normalised membership neither over-rejects a legitimate name (case / trailing dot
// fold together) nor admits a name outside the authorised set.
func TestGateAdmitsInScopeAndDropsOutOfScopeNames(t *testing.T) {
	// A dns job's AttemptedScope, by content, as scan.Job.AttemptedScope marshals it
	// (internal/scan/scan.go scopeRecord json tags).
	scope := parseAuthorizedScope([]byte(`{"vantage":"v1","names":["a.example.com","b.example.com"]}`))

	// Legitimate: subject is a resolved Name, admitted even with a trailing dot and
	// mixed case (CanonicalName / normalizeDomain fold both away).
	inScope := wire.Observation{
		Kind: resolutionwalk.Kind, Facet: resolutionwalk.FacetResolution,
		Subject: "A.example.com.", Vantage: "v1",
	}
	inScopeRecord := wire.Observation{
		Kind: resolutionwalk.Kind, Facet: resolutionwalk.FacetDNSRecord,
		Subject: "b.example.com", Discriminator: "A", Vantage: "v1",
	}
	// A compromised prober asserting drift for a Name the job never resolved.
	outOfScope := wire.Observation{
		Kind: resolutionwalk.Kind, Facet: resolutionwalk.FacetResolution,
		Subject: "evil.attacker.example", Vantage: "v1",
	}

	got := scope.gate([]wire.Observation{inScope, inScopeRecord, outOfScope}, nil, 9)

	want := []string{"A.example.com.", "b.example.com"}
	gotSubjects := subjectsOf(got)
	if len(gotSubjects) != len(want) {
		t.Fatalf("expected both in-scope names to survive and the out-of-scope one dropped, got %v", gotSubjects)
	}
	for i := range want {
		if gotSubjects[i] != want[i] {
			t.Errorf("survivor %d: got %q, want %q", i, gotSubjects[i], want[i])
		}
	}
	for _, o := range got {
		if o.Subject == outOfScope.Subject {
			t.Fatalf("out-of-scope name %q was NOT dropped", outOfScope.Subject)
		}
	}
}

// TestGateLeavesUndenotedDimensionsUngated guards against over-rejection: an
// address-scoped job's scope denotes no names, so a name-subject facet is not gated
// against it (and vice versa), and an all-empty/unparseable scope gates nothing.
func TestGateLeavesUndenotedDimensionsUngated(t *testing.T) {
	// Address-scoped hot scope: no names denoted. A (hypothetical) name-subject line
	// is admitted, not dropped against an absent name denotation.
	hot := parseAuthorizedScope([]byte(`{"vantage":"v1","addresses":["198.51.100.1"]}`))
	if hot.names != nil {
		t.Errorf("hot scope should denote no names")
	}
	nameObs := wire.Observation{Facet: resolutionwalk.FacetResolution, Subject: "anything.example"}
	if got := hot.gate([]wire.Observation{nameObs}, nil, 1); len(got) != 1 {
		t.Errorf("name-facet observation dropped against an address-only scope: %v", subjectsOf(got))
	}

	// A scope that parses to nothing gateable is a no-op (pre-#773 behaviour), never a
	// mass drop.
	empty := parseAuthorizedScope([]byte(`{"vantage":"v1","addresses":[]}`))
	svc := connectoutcome.EmitService("b", "v1", netip.MustParseAddrPort("203.0.113.9:443"),
		connectoutcome.Reached, connectoutcome.ConnResult("open"))
	if got := empty.gate([]wire.Observation{svc}, nil, 2); len(got) != 1 {
		t.Errorf("empty-denotation scope dropped an observation instead of no-op: %v", subjectsOf(got))
	}
}
