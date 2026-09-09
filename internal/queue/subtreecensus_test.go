package queue

import (
	"context"
	"testing"

	"github.com/winniel123/verge-asm/internal/message"
)

const (
	apexName  = "example.com"
	apexAddr  = "203.0.113.7"
	apexSvc   = apexAddr + ":443/tcp"
	subName   = "www." + apexName
	lookalike = "notexample.com"
)

func apexFold() []spanChange {
	return []spanChange{
		{SubjectKind: "name", SubjectKey: apexName, Facet: "resolution", Opened: true, Value: resolved(apexAddr)},
		{SubjectKind: "service", SubjectKey: apexSvc, Facet: "reachability", Opened: true, Value: reachValue("reached")},
		{SubjectKind: "endpoint", SubjectKey: apexName + "@" + apexSvc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
		{SubjectKind: "endpoint", SubjectKey: subName + "@" + apexSvc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
		{SubjectKind: "endpoint", SubjectKey: lookalike + "@" + apexSvc, Facet: "certificate", Opened: true, Value: []byte(`{}`)},
	}
}

func TestMembershipCensusExcludesAnotherNamesEndpoint(t *testing.T) {
	msgs := membershipMessages(produceT0, apexFold(), membershipInputs{})
	if len(msgs) != 1 {
		t.Fatalf("the apex is the one root that opened a resolution, got %+v", msgs)
	}
	keys := map[string]bool{}
	for _, e := range msgs[0].Census.Entries {
		keys[e.Key] = true
	}
	if !keys[apexSvc] || !keys[apexName+"@"+apexSvc] {
		t.Errorf("the cited Service and the apex's own Endpoint are beneath it, got %+v", msgs[0].Census.Entries)
	}
	if keys[subName+"@"+apexSvc] {
		t.Errorf("a distinct Name entering in the same fold is its own root (ADR-0031 §1), got %+v", msgs[0].Census.Entries)
	}
	if keys[lookalike+"@"+apexSvc] {
		t.Errorf("a Name that merely ends in the apex is no sub-name, got %+v", msgs[0].Census.Entries)
	}
	if msgs[0].Census.Len() != 2 {
		t.Errorf("census = %+v, want exactly the Service and the apex's Endpoint", msgs[0].Census.Entries)
	}
}

func TestAForeignEndpointKeepsItsOwnMessageBeneathAnApexRoot(t *testing.T) {
	ep := subName + "@" + sensitiveSvc
	changes := []spanChange{
		{SubjectKind: "name", SubjectKey: apexName, Facet: "resolution", Opened: true, Value: resolved("198.51.100.1")},
		{SubjectKind: "endpoint", SubjectKey: ep, Facet: "certificate", Value: []byte(certExpired), Previous: []byte(certNoTLS)},
		{SubjectKind: "endpoint", SubjectKey: ep, Facet: "certificate", Opened: true, Value: []byte(certExpired)},
	}
	store := &fakeMessageStore{}
	var log []routed
	if err := produceMessages(context.Background(), store, 31, produceT0, changes, nil, nil, membershipInputs{}, fakeEnqueuer(1, &log), false); err != nil {
		t.Fatalf("produce: %v", err)
	}
	var membership, own int
	for _, m := range store.inserted {
		switch {
		case m.SubjectKind == "name" && m.FiredAt == apexName:
			membership++
			c, err := message.ParseCensus(m.Census)
			if err != nil {
				t.Fatalf("parse census: %v", err)
			}
			if c.Len() != 0 {
				t.Errorf("nothing of the apex's own opened, so its census is empty; got %+v", c.Entries)
			}
		case m.FiredAt == ep:
			own++
		}
	}
	if membership != 1 {
		t.Fatalf("the apex's opening is one membership message, got %+v", store.inserted)
	}
	if own == 0 {
		t.Errorf("a foreign Endpoint's move is no residue of the apex, so it keeps its message; got %+v", store.inserted)
	}
}
