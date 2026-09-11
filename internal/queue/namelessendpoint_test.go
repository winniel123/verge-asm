package queue

import "testing"

// Every Endpoint a dispatcher opens carries an absent Name leg (ADR-0205, #1774).

func namelessBeneath(addr, port string) []spanChange {
	svc := addr + ":" + port + "/tcp"
	return []spanChange{
		{SubjectKind: "service", SubjectKey: svc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
		{SubjectKind: "endpoint", SubjectKey: "@" + svc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
	}
}

func TestRePointResidueCountsTheNamelessEndpoint(t *testing.T) {
	store := knownAddressStore(moveRow(rpName, resolved(rpOld), resolved(rpNew)))
	store.opened = beneath("", rpNew, "443")

	msgs, _ := settleFrom(t, store)
	if len(msgs) != 1 {
		t.Fatalf("the move admits ground beneath a known address, so §2 fires; got %+v", msgs)
	}
	if k := censusKinds(msgs[0]); k["endpoint"] != 1 {
		t.Errorf("the residue is the nameless Endpoint beneath the newly cited address, got %v", k)
	}
}

func TestRePointResidueCountsAForeignNamedEndpointOnTheNewGround(t *testing.T) {
	store := knownAddressStore(moveRow(rpName, resolved(rpOld), resolved(rpNew)))
	store.opened = beneath(rpOther, rpNew, "443")

	msgs, _ := settleFrom(t, store)
	if len(msgs) != 1 {
		t.Fatalf("the move admits ground beneath a known address, so §2 fires; got %+v", msgs)
	}
	if k := censusKinds(msgs[0]); k["endpoint"] != 1 {
		t.Errorf("ADR-0026 §2 residues every Endpoint no membership message covers, whatever its Name leg; got %v", k)
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

func TestTheFoldStillRootsOnTheNamelessEndpointsAddress(t *testing.T) {
	// #1812 dropped the owner test from the residue. The Address root keeps its own census.
	changes := append([]spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpNew))}, namelessBeneath(rpNew, "443")...)
	store := &fakeMessageStore{}
	got := byKind(rePointFrom(t, store, changes, membershipInputs{}))["address"]
	if len(got) != 1 {
		t.Fatalf("an address no other Name cites is the root, got %+v", got)
	}
	if k := censusKinds(got[0]); k["endpoint"] != 1 || k["service"] != 1 {
		t.Errorf("the root's census carries the nameless Endpoint and its Service, got %v", k)
	}
}
