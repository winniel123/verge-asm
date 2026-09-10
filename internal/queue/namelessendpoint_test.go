package queue

import (
	"testing"

	"github.com/winniel123/verge-asm/internal/db"
)

// Every Endpoint a dispatcher opens carries an absent Name leg (ADR-0205, #1774).

func namelessBeneath(addr, port string) []spanChange {
	svc := addr + ":" + port + "/tcp"
	return []spanChange{
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
		{SubjectKind: "endpoint", SubjectKey: "@" + svc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
	}
}

func TestRePointResidueCountsTheNamelessEndpoint(t *testing.T) {
	changes := append([]spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpNew))}, namelessBeneath(rpNew, "443")...)
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpOther)}}}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	got := byKind(msgs)
	if len(got["name"]) != 1 {
		t.Fatalf("the move admits ground beneath a known address, so §2 fires; got %+v", msgs)
	}
	if k := censusKinds(got["name"][0]); k["endpoint"] != 1 {
		t.Errorf("the residue is the nameless Endpoint beneath the newly cited address, got %v", k)
	}
}

func TestRePointResidueExcludesAForeignNamedEndpoint(t *testing.T) {
	changes := append([]spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpNew))}, openedBeneath(rpOther, rpNew, "443")...)
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer(rpOther)}}}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	for _, m := range byKind(msgs)["name"] {
		if censusKinds(m)["endpoint"] != 0 {
			t.Errorf("another Name's Endpoint is not this move's consequence (ADR-0026 §2), got %+v", m.Census)
		}
	}
}

func TestRePointResidueNamesOneDispatcherEndpointUnderEveryMove(t *testing.T) {
	changes := []spanChange{
		rePointMove(rpName, resolved(rpOld), resolved(rpNew)),
		rePointMove(rpOther, resolved(rpOld), resolved(rpNew)),
	}
	changes = append(changes, namelessBeneath(rpNew, "443")...)
	store := &fakeMessageStore{citers: map[string][]db.ListResolutionCitersForAddressesRow{rpNew: {citer("cdn.example.com")}}}
	msgs := rePointFrom(t, store, changes, membershipInputs{})
	got := byKind(msgs)["name"]
	if len(got) != 2 {
		t.Fatalf("each Name's move is its own cause, so each fires (ADR-0026 §2); got %+v", msgs)
	}
	for _, m := range got {
		if censusKinds(m)["endpoint"] != 1 {
			t.Errorf("both Names gained the dispatcher's ground, so both residues name it, got %+v", m.Census)
		}
	}
}

func TestMembershipCensusCountsTheNamelessEndpointBeneathACitedAddress(t *testing.T) {
	root := spanChange{SubjectKind: "name", SubjectKey: rpName, Facet: "resolution", Opened: true, Value: resolved(rpNew)}
	changes := append([]spanChange{root}, namelessBeneath(rpNew, "443")...)
	kinds := map[string]int{}
	for _, e := range membershipCensus(changes, root).Entries {
		kinds[e.Kind]++
	}
	if kinds["service"] != 1 || kinds["endpoint"] != 1 {
		t.Errorf("the census enumerates the Service and the nameless Endpoint beneath the cited address, got %v", kinds)
	}
}
