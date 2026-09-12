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

func TestTheAddressRootDefersTheNamelessEndpointToTheRelease(t *testing.T) {
	// The root is the Address, and that Endpoint opens in a later hot fold, so the census is
	// computed at release from the basis the move froze (ADR-1806 §4, #1774).
	changes := []spanChange{rePointMove(rpName, resolved(rpOld), resolved(rpNew))}
	store := &fakeMessageStore{}
	got := byKind(rePointFrom(t, store, changes, membershipInputs{}))["address"]
	if len(got) != 1 {
		t.Fatalf("an address no other Name cites is the root, got %+v", got)
	}
	var subjects []subjectRef
	for _, c := range namelessBeneath(rpNew, "443") {
		subjects = append(subjects, subjectRef{kind: c.SubjectKind, key: c.SubjectKey})
	}
	kinds := map[string]int{}
	for _, e := range censusBeneathRoot(basisRoot(heldBasis(t, got[0])), subjects).Entries {
		kinds[e.Kind]++
	}
	if kinds["endpoint"] != 1 || kinds["service"] != 1 {
		t.Errorf("the release names the nameless Endpoint and its Service, got %v", kinds)
	}
}
