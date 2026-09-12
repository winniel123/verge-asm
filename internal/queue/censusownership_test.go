package queue

import (
	"strings"
	"testing"

	"github.com/winniel123/verge-asm/internal/message"
)

const secondName = "second.test"

// #1809's failing input: a second Name cites one address, and both roots census all of it.

func sharedAddressFold() []spanChange {
	return append(apexFold(), spanChange{
		SubjectKind: "name", SubjectKey: secondName, Facet: "resolution",
		Opened: true, Value: resolved(apexAddr),
	})
}

func censusKeySet(c message.Census) map[string]bool {
	out := map[string]bool{}
	for _, e := range c.Entries {
		out[e.Key] = true
	}
	return out
}

func TestTwoNamesOnOneAddressCensusTheSameSubjects(t *testing.T) {
	changes := sharedAddressFold()
	apex := censusKeySet(membershipCensus(changes, changes[0]))
	second := censusKeySet(membershipCensus(changes, changes[len(changes)-1]))

	// Both censuses attribute on the citation axis, so neither drops a member (#1774, #1776).
	if len(apex) != 4 || len(second) != 4 {
		t.Fatalf("each root censuses the Service and the three Endpoints on it, got %v and %v", apex, second)
	}
	for k := range apex {
		if !second[k] {
			t.Errorf("the second Name cites the same address, so %q is beneath it too", k)
		}
	}
	// The fan-out is the ground of #1809: k roots × n subjects, with no root owning them.
	if !second[apexName+"@"+apexSvc] {
		t.Errorf("an Endpoint the apex named still rides the second Name's census, got %v", second)
	}
}

func TestNeitherNameClaimsTheSharedAddressBeneathIt(t *testing.T) {
	changes := sharedAddressFold()
	for _, root := range []spanChange{changes[0], changes[len(changes)-1]} {
		m := message.Membership(message.EntryAppeared, root.SubjectKind, root.SubjectKey, "",
			membershipCensus(changes, root), produceT0)
		if m == nil {
			t.Fatalf("a Name root fires a membership message, got nil for %q", root.SubjectKey)
		}
		if !strings.Contains(m.Headline, "4 timelines opened on an address it cites") {
			t.Errorf("the headline counts the sub-tree and names the citation as its ground, got %q", m.Headline)
		}
	}
}

func TestAnAddressRootStillReadsBeneathIt(t *testing.T) {
	changes := []spanChange{
		{SubjectKind: "address", SubjectKey: apexAddr, Facet: "resolution", Opened: true, Value: []byte(`{}`)},
		{SubjectKind: "service", SubjectKey: apexSvc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
	}
	m := message.Membership(message.EntryAppeared, changes[0].SubjectKind, changes[0].SubjectKey, "",
		membershipCensus(changes, changes[0]), produceT0)
	if m == nil {
		t.Fatal("an Address root fires a membership message")
	}
	if !strings.Contains(m.Headline, "1 timeline opened beneath it") {
		t.Errorf("the address is the ground its sub-tree sits on, got %q", m.Headline)
	}
}
