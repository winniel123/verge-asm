package queue

import (
	"net/netip"
	"testing"

	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/measure/resolutionwalk"
	"github.com/winniel123/verge-asm/internal/wire"
)

func subjectsOf(obs []wire.Observation) []string {
	out := make([]string, 0, len(obs))
	for _, o := range obs {
		out = append(out, o.Subject)
	}
	return out
}

func TestGateAdmitsInScopeAndDropsOutOfScopeAddresses(t *testing.T) {
	// These fixtures copy the real producers by content, so a producer change must reach here too.
	scope := parseAuthorizedScope([]byte(`{"vantage":"v1","addresses":["198.51.100.1"],"tcp_ports":[443]}`))

	inScope := connectoutcome.EmitService(
		"b1", "v1", netip.MustParseAddrPort("198.51.100.1:443"),
		connectoutcome.Reached, connectoutcome.ConnResult("open"),
	)
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

func TestGateAdmitsInScopeAndDropsOutOfScopeNames(t *testing.T) {
	scope := parseAuthorizedScope([]byte(`{"vantage":"v1","names":["a.example.com","b.example.com"]}`))

	inScope := wire.Observation{
		Kind: resolutionwalk.Kind, Facet: resolutionwalk.FacetResolution,
		Subject: "A.example.com.", Vantage: "v1",
	}
	inScopeRecord := wire.Observation{
		Kind: resolutionwalk.Kind, Facet: resolutionwalk.FacetDNSRecord,
		Subject: "b.example.com", Discriminator: "A", Vantage: "v1",
	}
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

func TestGateLeavesUndenotedDimensionsUngated(t *testing.T) {
	hot := parseAuthorizedScope([]byte(`{"vantage":"v1","addresses":["198.51.100.1"]}`))
	if hot.names != nil {
		t.Errorf("hot scope should denote no names")
	}
	nameObs := wire.Observation{Facet: resolutionwalk.FacetResolution, Subject: "anything.example"}
	if got := hot.gate([]wire.Observation{nameObs}, nil, 1); len(got) != 1 {
		t.Errorf("name-facet observation dropped against an address-only scope: %v", subjectsOf(got))
	}

	empty := parseAuthorizedScope([]byte(`{"vantage":"v1","addresses":[]}`))
	svc := connectoutcome.EmitService("b", "v1", netip.MustParseAddrPort("203.0.113.9:443"),
		connectoutcome.Reached, connectoutcome.ConnResult("open"))
	if got := empty.gate([]wire.Observation{svc}, nil, 2); len(got) != 1 {
		t.Errorf("empty-denotation scope dropped an observation instead of no-op: %v", subjectsOf(got))
	}
}

func TestGateAdmitsInScopeAndDropsOutOfScopeEdgeFanoutAddresses(t *testing.T) {
	// A fabricated row feeds the custody-extension veto an answer nothing measured (#985).
	scope := parseAuthorizedScope([]byte(`{"addresses":["93.184.216.34"]}`))

	inScope := edgefanout.Emit("b1", netip.MustParseAddrPort("93.184.216.34:443"),
		edgefanout.Result{Outcome: edgefanout.TLSRefused})
	outOfScope := edgefanout.Emit("b1", netip.MustParseAddrPort("104.16.132.229:443"),
		edgefanout.Result{Outcome: edgefanout.TLSRefused})

	got := scope.gate([]wire.Observation{inScope, outOfScope}, nil, 7)
	if len(got) != 1 || got[0].Address != "93.184.216.34" {
		t.Fatalf("gate admitted %d line(s) %v, want the in-scope address alone", len(got), got)
	}
}

func TestGateAdmitsEdgeFanoutAddressUnderNetipNormalisation(t *testing.T) {
	scope := parseAuthorizedScope([]byte(`{"addresses":["93.184.216.34"]}`))
	line := wire.Observation{Kind: edgefanout.Kind, Address: "::ffff:93.184.216.34"}
	if got := scope.gate([]wire.Observation{line}, nil, 7); len(got) != 1 {
		t.Fatalf("gate dropped the mapped spelling of an authorised address")
	}
}

func TestGateDropsEdgeFanoutUnderAScopeDenotingNoAddress(t *testing.T) {
	// This arm alone fails closed: BuildEdgeFanoutJobs enqueues no job over an empty candidate set.
	scope := parseAuthorizedScope([]byte(`{"vantage":"v1","names":["www.example.com"]}`))
	line := edgefanout.Emit("b1", netip.MustParseAddrPort("104.16.132.229:443"),
		edgefanout.Result{Outcome: edgefanout.TLSRefused})
	if got := scope.gate([]wire.Observation{line}, nil, 7); len(got) != 0 {
		t.Fatalf("gate admitted %v under a name-only scope, want none", got)
	}
}

func TestGateLeavesOtherFacetlessKindsUngated(t *testing.T) {
	scope := parseAuthorizedScope([]byte(`{"addresses":["93.184.216.34"]}`))
	line := wire.Observation{Kind: "some-later-kind", Address: "104.16.132.229"}
	if got := scope.gate([]wire.Observation{line}, nil, 7); len(got) != 1 {
		t.Fatalf("gate dropped a facet-less line of another kind")
	}
}
